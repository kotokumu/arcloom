package assessmentdelivery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/assessmentdelivery"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestDeliverValidation(t *testing.T) {
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	recipient := assessmentdelivery.Recipient(func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		t.Fatal("invalid input delivered")
		return nil
	})
	// gotests error-only scaffold; errors are checked independently below.
	type args struct {
		ctx       context.Context
		report    controlruntime.Report[planattempt.AttemptResult]
		recipient assessmentdelivery.Recipient
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil context and recipient", wantErr: true},
		{name: "nil context", args: args{recipient: recipient}, wantErr: true},
		{name: "ended context and nil recipient", args: args{ctx: ended}, wantErr: true},
		{name: "nil recipient", args: args{ctx: context.Background()}, wantErr: true},
		{name: "zero report", args: args{ctx: context.Background(), recipient: recipient}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := assessmentdelivery.Deliver(tt.args.ctx, tt.args.report, tt.args.recipient)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Fatal(diff)
			}
		})
	}
	var zero controlruntime.Report[planattempt.AttemptResult]
	var missingContext context.Context
	if err := assessmentdelivery.Deliver(missingContext, zero, nil); !errors.Is(err, controlruntime.ErrInvalidContext) {
		t.Fatal(err)
	}
	if err := assessmentdelivery.Deliver(ended, zero, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := assessmentdelivery.Deliver(context.Background(), zero, nil); !errors.Is(err, assessmentdelivery.ErrInvalidRecipient) {
		t.Fatal(err)
	}
	if err := assessmentdelivery.Deliver(context.Background(), zero, recipient); !errors.Is(err, assessmentdelivery.ErrInvalidReport) {
		t.Fatal(err)
	}
}

func TestDeliverPreservesEveryAssessmentOccurrence(t *testing.T) {
	for _, outcome := range []plancontrol.Outcome{plancontrol.Complete, plancontrol.Retain, plancontrol.Revise, plancontrol.InsufficientInformation} {
		t.Run(string(rune('0'+outcome)), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			identity := must(controlruntime.NewTargetIdentity("plan", "exact"))
			current := must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accepted"))}, nil, nil))
			proposed := must(plan.New("proposed", must(plan.NewGoal("new goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accepted"))}, nil, nil))
			response := plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{outcome}}
			if outcome == plancontrol.Revise {
				response.ProposedPlans = []plan.Plan{proposed}
			}
			snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
			target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "facts", nil }))
			binding := must(planattempt.NewPlanAttemptBinding(target, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) { return response, nil }))
			controller := must(controlruntime.Start(ctx, 1, binding.Attempt()))
			delivered := 0
			recipient := func(_ context.Context, gotTarget controlruntime.TargetIdentity, got plancontrol.Assessment) error {
				delivered++
				if diff := cmp.Diff(identity, gotTarget, cmp.AllowUnexported(controlruntime.TargetIdentity{})); diff != "" {
					t.Error(diff)
				}
				if got.Outcome() != outcome || !got.AssessedPlan().Equal(current) {
					t.Error("assessment changed")
				}
				p, ok := got.ProposedPlan()
				if ok != (outcome == plancontrol.Revise) || (ok && !p.Equal(proposed)) {
					t.Error("proposal changed")
				}
				return nil
			}
			for range 2 {
				if err := controller.Request(ctx, identity); err != nil {
					t.Fatal(err)
				}
				report := <-controller.Reports()
				if err := assessmentdelivery.Deliver(ctx, report, recipient); err != nil {
					t.Fatal(err)
				}
			}
			if diff := cmp.Diff(2, delivered); diff != "" {
				t.Fatal(diff)
			}
			cancel()
			_ = controller.Wait()
		})
	}
}

func TestDeliverSuppressesNonAssessedReports(t *testing.T) {
	for _, kind := range []string{"no current", "attempt failure", "directive failure", "invalid completion"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			identity := must(controlruntime.NewTargetIdentity("plan", kind))
			snapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
			target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "", nil }))
			binding := must(planattempt.NewPlanAttemptBinding(target, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
				return plancontrol.AssessorResponse{}, nil
			}))
			attempt := binding.Attempt()
			switch kind {
			case "attempt failure":
				attempt = func(context.Context, controlruntime.TargetIdentity) (planattempt.AttemptResult, controlruntime.Directive, error) {
					return planattempt.AttemptResult{}, controlruntime.Directive{}, errors.New("unavailable")
				}
			case "directive failure":
				attempt = func(context.Context, controlruntime.TargetIdentity) (planattempt.AttemptResult, controlruntime.Directive, error) {
					return planattempt.AttemptResult{}, controlruntime.Directive{}, nil
				}
			case "invalid completion":
				attempt = func(context.Context, controlruntime.TargetIdentity) (planattempt.AttemptResult, controlruntime.Directive, error) {
					return planattempt.AttemptResult{}, controlruntime.AwaitAnotherRequest(), nil
				}
			}
			controller := must(controlruntime.Start(ctx, 1, attempt))
			if err := controller.Request(ctx, identity); err != nil {
				t.Fatal(err)
			}
			report := <-controller.Reports()
			err := assessmentdelivery.Deliver(ctx, report, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
				t.Fatal("non-assessed handoff")
				return nil
			})
			if (kind == "invalid completion") != errors.Is(err, assessmentdelivery.ErrInvalidReport) {
				t.Fatalf("error=%v", err)
			}
			if kind != "invalid completion" && err != nil {
				t.Fatal(err)
			}
			cancel()
			_ = controller.Wait()
		})
	}
}

func TestDeliverFailureAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	identity := must(controlruntime.NewTargetIdentity("plan", "failure"))
	current := must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accepted"))}, nil, nil))
	snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "facts", nil }))
	binding := must(planattempt.NewPlanAttemptBinding(target, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Retain}}, nil
	}))
	controller := must(controlruntime.Start(ctx, 1, binding.Attempt()))
	if err := controller.Request(ctx, identity); err != nil {
		t.Fatal(err)
	}
	report := <-controller.Reports()
	cause := errors.New("possible effect")
	calls := 0
	err := assessmentdelivery.Deliver(ctx, report, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		calls++
		return cause
	})
	var failure *assessmentdelivery.DeliveryError
	if !errors.As(err, &failure) || !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("failure=%v calls=%d", err, calls)
	}
	cancelled, end := context.WithCancel(ctx)
	err = assessmentdelivery.Deliver(cancelled, report, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		end()
		return cause
	})
	if !errors.Is(err, context.Canceled) || errors.As(err, &failure) {
		t.Fatalf("cancel classified as %v", err)
	}
	calls = 0
	err = assessmentdelivery.Deliver(cancelled, report, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		calls++
		return nil
	})
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancel err=%v calls=%d", err, calls)
	}
	lateSuccess, stop := context.WithCancel(ctx)
	calls = 0
	err = assessmentdelivery.Deliver(lateSuccess, report, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		calls++
		stop()
		return nil
	})
	if !errors.Is(err, context.Canceled) || calls != 1 || errors.As(err, &failure) {
		t.Fatalf("late success err=%v calls=%d", err, calls)
	}
	cancel()
	_ = controller.Wait()
}
