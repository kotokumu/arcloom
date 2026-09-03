package planattempt_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

func TestPlanAttemptResolvesDistinctTargetBindings(t *testing.T) {
	firstIdentity := must(controlruntime.NewTargetIdentity("plan", "first"))
	secondIdentity := must(controlruntime.NewTargetIdentity("plan", "second"))
	firstPlan := must(plan.New("first plan", must(plan.NewGoal("Deliver first")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("First accepted"))}, nil, nil))
	secondPlan := must(plan.New("second plan", must(plan.NewGoal("Deliver second")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Second accepted"))}, nil, nil))
	firstSnapshot := must(plansnapshot.New(firstPlan, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	secondSnapshot := must(plansnapshot.New(secondPlan, must(plansnapshot.CompleteProgress(plansnapshot.Closed, nil))))
	firstTarget := must(planattempt.NewPlanTarget(firstIdentity, func(context.Context) (plansnapshot.Snapshot, error) { return firstSnapshot, nil }, func(context.Context) (string, error) { return "first delivery", nil }))
	secondTarget := must(planattempt.NewPlanTarget(secondIdentity, func(context.Context) (plansnapshot.Snapshot, error) { return secondSnapshot, nil }, func(context.Context) (string, error) { return "second delivery", nil }))
	attempt := must(planattempt.NewAttempt("plan", func(_ context.Context, requested controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
		if requested == firstIdentity {
			return firstTarget, nil
		}
		if requested == secondIdentity {
			return secondTarget, nil
		}
		return planattempt.PlanTarget[string]{}, errors.New("not configured")
	}, func(_ context.Context, current plan.Plan, delivery string) (plancontrol.AssessorResponse, error) {
		if current.Equal(firstPlan) && delivery != "first delivery" {
			t.Errorf("first binding delivery = %q", delivery)
		}
		if current.Equal(secondPlan) && delivery != "second delivery" {
			t.Errorf("second binding delivery = %q", delivery)
		}
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
	}))

	for _, identity := range []controlruntime.TargetIdentity{firstIdentity, secondIdentity} {
		result, _, err := attempt(context.Background(), identity)
		if err != nil {
			t.Fatalf("Attempt(%q) error = %v", identity.Key(), err)
		}
		assessment, ok := result.Assessment()
		if !ok || assessment.AssessedPlan().Name() != identity.Key()+" plan" {
			t.Errorf("Attempt(%q) used another binding", identity.Key())
		}
	}
}

func TestPlanAttemptControllerReentryAlwaysObservesFreshFacts(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "reentry"))
	before := must(plan.New("before interaction", must(plan.NewGoal("Deliver before")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Before accepted"))}, nil, nil))
	after := must(plan.New("after interaction", must(plan.NewGoal("Deliver after")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("After accepted"))}, nil, nil))
	plans := []plan.Plan{before, after}
	var observation atomic.Int64
	var currentFact atomic.Int64
	target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) {
		observation.Add(1)
		index := int(currentFact.Load())
		return plansnapshot.New(plans[index], must(plansnapshot.CompleteProgress(plansnapshot.Open, nil)))
	}, func(context.Context) (struct{}, error) { return struct{}{}, nil }))
	attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[struct{}], error) {
		return target, nil
	}, func(context.Context, plan.Plan, struct{}) (plancontrol.AssessorResponse, error) {
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Retain}}, nil
	}))
	ctx, cancel := context.WithCancel(context.Background())
	controller := must(controlruntime.Start(ctx, 1, attempt))

	if err := controller.Request(context.Background(), identity); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	firstReport := <-controller.Reports()
	firstResult, _, ok := firstReport.Completion()
	if !ok {
		t.Fatal("first Plan Attempt did not complete")
	}
	firstAssessment, _ := firstResult.Assessment()
	if !firstAssessment.AssessedPlan().Equal(before) {
		t.Error("first Attempt did not use pre-interaction facts")
	}

	// The external interaction changes only the externally observed facts. Its
	// result or receipt is deliberately not passed to either control API.
	currentFact.Store(1)
	if calls := observation.Load(); calls != 1 {
		t.Fatalf("observations before later Request = %d", calls)
	}
	if err := controller.Request(context.Background(), identity); err != nil {
		t.Fatalf("later Request() error = %v", err)
	}
	secondReport := <-controller.Reports()
	secondResult, _, ok := secondReport.Completion()
	if !ok {
		t.Fatal("second Plan Attempt did not complete")
	}
	secondAssessment, _ := secondResult.Assessment()
	if !secondAssessment.AssessedPlan().Equal(after) {
		t.Error("second Attempt did not reacquire post-interaction facts")
	}
	if calls := observation.Load(); calls != 2 {
		t.Errorf("observations after later Request = %d, want 2", calls)
	}
	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v", err)
	}
}
