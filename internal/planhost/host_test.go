package planhost_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/assessmentdelivery"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
	"github.com/kotokumu/arcloom/internal/planhost"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// planningContext is test-owned source authority. Only completeTask changes its
// revision. Observers and assessor derive outputs from current source facts.
type planningContext struct {
	mu          sync.Mutex
	revision    int
	closed      [2]bool
	accepted    bool
	revise      bool
	noCurrent   bool
	unavailable bool
	events      []string
	observed    []int
	assessed    []int
	writes      []string
}
type deliveryFacts struct {
	revision int
	closed   [2]bool
	accepted bool
	revise   bool
}

func (s *planningContext) binding(key string) planattempt.PlanAttemptBinding {
	identity := must(controlruntime.NewTargetIdentity("plan", key))
	current := must(plan.New(key, must(plan.NewGoal("Deliver both results")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Both results accepted"))}, []plan.Task{must(plan.NewTask("first")), must(plan.NewTask("second"))}, nil))
	target := must(planattempt.NewPlanTarget(identity,
		func(context.Context) (plansnapshot.Snapshot, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.events = append(s.events, "snapshot")
			s.observed = append(s.observed, s.revision)
			if s.unavailable {
				return plansnapshot.Snapshot{}, errors.New("source unavailable")
			}
			states := []plansnapshot.TaskProgress{}
			for i, name := range []string{"first", "second"} {
				state := plansnapshot.Open
				if s.closed[i] {
					state = plansnapshot.Closed
				}
				states = append(states, must(plansnapshot.NewTaskProgress(name, state)))
			}
			progress := must(plansnapshot.CompleteProgress(plansnapshot.Open, states))
			if s.noCurrent {
				return plansnapshot.WithoutCurrent(progress)
			}
			return plansnapshot.New(current, progress)
		},
		func(context.Context) (deliveryFacts, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.events = append(s.events, "delivery observation")
			return deliveryFacts{s.revision, s.closed, s.accepted, s.revise}, nil
		},
	))
	return must(planattempt.NewPlanAttemptBinding(target, func(_ context.Context, got plan.Plan, facts deliveryFacts) (plancontrol.AssessorResponse, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.events = append(s.events, "assessment")
		s.assessed = append(s.assessed, facts.revision)
		if !got.Equal(current) {
			return plancontrol.AssessorResponse{}, errors.New("exchanged Plan")
		}
		if facts.revise {
			proposed := must(plan.New("revised "+key, current.Goal(), current.AcceptanceConditions(), current.Tasks(), nil))
			return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Revise}, ProposedPlans: []plan.Plan{proposed}}, nil
		}
		outcome := plancontrol.Retain
		if facts.closed[0] && facts.closed[1] {
			outcome = plancontrol.InsufficientInformation
			if facts.accepted {
				outcome = plancontrol.Complete
			}
		}
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{outcome}}, nil
	}))
}

func (s *planningContext) completeTask(index int, accepted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed[index] = true
	s.accepted = accepted
	s.revision++
	s.writes = append(s.writes, []string{"Actor completes first", "Actor completes second with acceptance"}[index])
}

func TestStartRejectsInvalidConfiguration(t *testing.T) {
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	source := &planningContext{}
	binding := source.binding("startup")
	type args struct {
		ctx       context.Context
		binding   planattempt.PlanAttemptBinding
		recipient assessmentdelivery.Recipient
	}
	tests := []struct {
		name    string
		args    args
		want    *planhost.Host
		wantErr bool
	}{
		{name: "nil context", want: nil, wantErr: true},
		{name: "ended context", args: args{ctx: ended}, want: nil, wantErr: true},
		{name: "zero binding", args: args{ctx: context.Background()}, want: nil, wantErr: true},
		{name: "nil recipient", args: args{ctx: context.Background(), binding: binding}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planhost.Start(tt.args.ctx, tt.args.binding, tt.args.recipient)
			if (err != nil) != tt.wantErr {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
	var missingContext context.Context
	if _, err := planhost.Start(missingContext, planattempt.PlanAttemptBinding{}, nil); !errors.Is(err, controlruntime.ErrInvalidContext) {
		t.Fatal(err)
	}
	if _, err := planhost.Start(ended, planattempt.PlanAttemptBinding{}, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := planhost.Start(context.Background(), planattempt.PlanAttemptBinding{}, nil); !errors.Is(err, planattempt.ErrInvalidPlanAttemptBinding) {
		t.Fatal(err)
	}
	if _, err := planhost.Start(context.Background(), binding, nil); !errors.Is(err, assessmentdelivery.ErrInvalidRecipient) {
		t.Fatal(err)
	}
	if len(source.events) != 0 {
		t.Fatal("rejected startup did work")
	}
}

func TestHostZeroAndIdleLifecycle(t *testing.T) {
	for _, host := range []*planhost.Host{nil, {}} {
		if !errors.Is(host.Trigger(context.Background()), planhost.ErrHostNotStarted) || !errors.Is(host.Wait(), planhost.ErrHostNotStarted) || host.Reports() != nil {
			t.Fatal("zero Host contract")
		}
	}
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		source := &planningContext{}
		host := must(planhost.Start(ctx, source.binding("idle"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
			t.Error("idle delivered")
			return nil
		}))
		stream := host.Reports()
		if stream != host.Reports() {
			t.Fatal("unstable stream")
		}
		var missingContext context.Context
		if !errors.Is(host.Trigger(missingContext), controlruntime.ErrInvalidContext) {
			t.Fatal("nil submission accepted")
		}
		submission, end := context.WithCancel(ctx)
		end()
		if !errors.Is(host.Trigger(submission), context.Canceled) {
			t.Fatal("ended submission accepted")
		}
		synctest.Wait()
		if len(source.events) != 0 {
			t.Fatal("idle or rejected request did work")
		}
		cancel()
		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				if !errors.Is(host.Wait(), context.Canceled) || !errors.Is(host.Trigger(context.Background()), context.Canceled) {
					t.Error("unstable terminal outcome")
				}
			})
		}
		wg.Wait()
		if _, ok := <-host.Reports(); ok {
			t.Fatal("idle produced report")
		}
	})
}

func TestHostConvergesOnlyAfterActorChangesAndAcceptance(t *testing.T) {
	source := &planningContext{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	identity := must(controlruntime.NewTargetIdentity("plan", "convergence"))
	deliveries := []plancontrol.Outcome{}
	host := must(planhost.Start(ctx, source.binding("convergence"), func(_ context.Context, got controlruntime.TargetIdentity, a plancontrol.Assessment) error {
		if diff := cmp.Diff(identity, got, cmp.AllowUnexported(controlruntime.TargetIdentity{})); diff != "" {
			t.Error(diff)
		}
		source.mu.Lock()
		defer source.mu.Unlock()
		source.events = append(source.events, "handoff")
		deliveries = append(deliveries, a.Outcome())
		return nil
	}))
	for i, want := range []plancontrol.Outcome{plancontrol.Retain, plancontrol.Retain, plancontrol.Complete} {
		if i > 0 {
			source.completeTask(i-1, i == 2)
		}
		submission, end := context.WithCancel(ctx)
		if err := host.Trigger(submission); err != nil {
			t.Fatal(err)
		}
		end()
		report, ok := <-host.Reports()
		if !ok {
			t.Fatal(host.Wait())
		}
		result, directive, ok := report.Completion()
		if !ok {
			t.Fatal("no Completion")
		}
		assessment, ok := result.Assessment()
		if !ok || assessment.Outcome() != want || directive.Kind() != controlruntime.AwaitRequest {
			t.Fatal("wrong assessment/directive")
		}
		source.mu.Lock()
		if len(deliveries) != i+1 || deliveries[i] != want {
			t.Error("processed evidence preceded handoff")
		}
		source.events = append(source.events, "processed")
		source.mu.Unlock()
	}
	cancel()
	if !errors.Is(host.Wait(), context.Canceled) {
		t.Fatal("terminal")
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if diff := cmp.Diff([]int{0, 1, 2}, source.observed); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]int{0, 1, 2}, source.assessed); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]string{"Actor completes first", "Actor completes second with acceptance"}, source.writes); diff != "" {
		t.Fatal(diff)
	}
	wantEvents := []string{}
	for range 3 {
		wantEvents = append(wantEvents, "snapshot", "delivery observation", "assessment", "handoff", "processed")
	}
	if diff := cmp.Diff(wantEvents, source.events); diff != "" {
		t.Fatal(diff)
	}
}

func TestHostCurrentFactsDetermineAllOutcomes(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		source *planningContext
		want   plancontrol.Outcome
	}{
		{name: "open tasks", source: &planningContext{}, want: plancontrol.Retain},
		{name: "closed without acceptance", source: &planningContext{closed: [2]bool{true, true}}, want: plancontrol.InsufficientInformation},
		{name: "closed and accepted", source: &planningContext{closed: [2]bool{true, true}, accepted: true}, want: plancontrol.Complete},
		{name: "revision requested by current facts", source: &planningContext{revise: true}, want: plancontrol.Revise},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var delivered plancontrol.Assessment
			host := must(planhost.Start(ctx, scenario.source.binding(scenario.name), func(_ context.Context, _ controlruntime.TargetIdentity, a plancontrol.Assessment) error {
				delivered = a
				return nil
			}))
			if err := host.Trigger(ctx); err != nil {
				t.Fatal(err)
			}
			report := <-host.Reports()
			result, _, _ := report.Completion()
			a, _ := result.Assessment()
			if diff := cmp.Diff(scenario.want, delivered.Outcome()); diff != "" {
				t.Fatal(diff)
			}
			if !a.AssessedPlan().Equal(delivered.AssessedPlan()) || a.Outcome() != delivered.Outcome() {
				t.Fatal("changed assessment")
			}
			cancel()
			_ = host.Wait()
		})
	}
}

func TestHostRecoveryUsesFreshObservationsWithoutRestart(t *testing.T) {
	for _, failure := range []string{"no current", "source failure"} {
		t.Run(failure, func(t *testing.T) {
			source := &planningContext{noCurrent: failure == "no current", unavailable: failure == "source failure"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deliveries := 0
			host := must(planhost.Start(ctx, source.binding("recovery"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
				deliveries++
				return nil
			}))
			if err := host.Trigger(ctx); err != nil {
				t.Fatal(err)
			}
			report := <-host.Reports()
			if _, failed := report.Failure(); failed != (failure == "source failure") {
				t.Fatal("failure classification")
			}
			if deliveries != 0 {
				t.Fatal("non-assessed delivered")
			}
			source.mu.Lock()
			source.noCurrent = false
			source.unavailable = false
			source.revision = 1
			source.mu.Unlock()
			if err := host.Trigger(ctx); err != nil {
				t.Fatal(err)
			}
			report = <-host.Reports()
			result, _, ok := report.Completion()
			if !ok || result.Kind() != planattempt.CurrentPlanAssessed || deliveries != 1 {
				t.Fatal("recovery")
			}
			cancel()
			_ = host.Wait()
			if diff := cmp.Diff([]int{0, 1}, source.observed); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestHostFullBufferCancellationPreservesOnlyPublishedEvidence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		source := &planningContext{}
		calls := 0
		host := must(planhost.Start(ctx, source.binding("buffer"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
			calls++
			return nil
		}))
		if err := host.Trigger(ctx); err != nil {
			t.Fatal(err)
		}
		synctest.Wait() // A is published into the buffer, not read.
		if err := host.Trigger(ctx); err != nil {
			t.Fatal(err)
		}
		synctest.Wait() // B has returned successfully and is blocked on publication.
		if calls != 2 {
			t.Fatalf("handoffs=%d", calls)
		}
		cancel()
		if !errors.Is(host.Wait(), context.Canceled) {
			t.Fatal("wait outcome")
		}
		reports := 0
		for range host.Reports() {
			reports++
		}
		if diff := cmp.Diff(1, reports); diff != "" {
			t.Fatal(diff)
		}
		if calls != 2 || source.revision != 0 {
			t.Fatal("replay or source change")
		}
	})
}

func TestHostDeliveryFailureWinsBeforeLaterCancellationAndRestartIsFresh(t *testing.T) {
	source := &planningContext{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cause := errors.New("receipt uncertain")
	calls := 0
	releaseFailure := make(chan struct{})
	host := must(planhost.Start(ctx, source.binding("failstop"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		calls++
		<-releaseFailure
		return cause
	}))
	if err := host.Trigger(ctx); err != nil {
		t.Fatal(err)
	}
	close(releaseFailure)
	err := host.Wait() // Acceptance of failure is established before cancellation.
	var delivery *assessmentdelivery.DeliveryError
	if !errors.As(err, &delivery) || !errors.Is(err, cause) {
		t.Fatalf("failure=%v", err)
	}
	cancel()
	if host.Wait() != err || host.Trigger(context.Background()) != err {
		t.Fatal("failure replaced")
	}
	if _, ok := <-host.Reports(); ok {
		t.Fatal("failed report exposed as processed")
	}
	source.completeTask(0, false)
	nextCtx, nextCancel := context.WithCancel(context.Background())
	defer nextCancel()
	next := must(planhost.Start(nextCtx, source.binding("failstop"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
		calls++
		return nil
	}))
	if err := next.Trigger(nextCtx); err != nil {
		t.Fatal(err)
	}
	<-next.Reports()
	nextCancel()
	_ = next.Wait()
	if calls != 2 {
		t.Fatal("retry/replay")
	}
	if diff := cmp.Diff([]int{0, 1}, source.observed); diff != "" {
		t.Fatal(diff)
	}
}

func TestHostCancellationAtEveryCooperativeBoundary(t *testing.T) {
	for _, stage := range []string{"snapshot", "observations", "assessor", "recipient"} {
		t.Run(stage, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered := make(chan struct{})
				current := must(plan.New("cancel", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accepted"))}, nil, nil))
				snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
				target := must(planattempt.NewPlanTarget(must(controlruntime.NewTargetIdentity("plan", stage)),
					func(ctx context.Context) (plansnapshot.Snapshot, error) {
						if stage == "snapshot" {
							close(entered)
							<-ctx.Done()
						}
						return snapshot, nil
					},
					func(ctx context.Context) (string, error) {
						if stage == "observations" {
							close(entered)
							<-ctx.Done()
						}
						return "facts", nil
					},
				))
				binding := must(planattempt.NewPlanAttemptBinding(target, func(ctx context.Context, _ plan.Plan, _ string) (plancontrol.AssessorResponse, error) {
					if stage == "assessor" {
						close(entered)
						<-ctx.Done()
					}
					return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Retain}}, nil
				}))
				deliveries := 0
				host := must(planhost.Start(ctx, binding, func(ctx context.Context, _ controlruntime.TargetIdentity, _ plancontrol.Assessment) error {
					deliveries++
					close(entered)
					<-ctx.Done()
					return nil
				}))
				if err := host.Trigger(ctx); err != nil {
					t.Fatal(err)
				}
				<-entered
				cancel()
				synctest.Wait()
				if !errors.Is(host.Wait(), context.Canceled) {
					t.Fatal("cancellation lost")
				}
				if _, ok := <-host.Reports(); ok {
					t.Fatal("cancelled result published")
				}
				want := 0
				if stage == "recipient" {
					want = 1
				}
				if deliveries != want {
					t.Fatal("late handoff")
				}
			})
		})
	}
}

func TestHostPendingTriggersCoalesceDuringActiveAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		entered := make(chan struct{}, 1)
		release := make(chan struct{})
		observations := 0
		snapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
		target := must(planattempt.NewPlanTarget(must(controlruntime.NewTargetIdentity("plan", "coalescing")), func(ctx context.Context) (plansnapshot.Snapshot, error) {
			observations++
			entered <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
			}
			return snapshot, nil
		}, func(context.Context) (string, error) { return "", nil }))
		binding := must(planattempt.NewPlanAttemptBinding(target, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
			return plancontrol.AssessorResponse{}, nil
		}))
		host := must(planhost.Start(ctx, binding, func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
			t.Error("no current delivered")
			return nil
		}))
		if err := host.Trigger(ctx); err != nil {
			t.Fatal(err)
		}
		<-entered
		for range 8 {
			if err := host.Trigger(ctx); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		if observations != 1 {
			t.Fatal("overlapping attempt")
		}
		close(release)
		<-host.Reports()
		<-host.Reports()
		synctest.Wait()
		if observations != 2 {
			t.Fatalf("observations=%d", observations)
		}
		cancel()
		_ = host.Wait()
	})
}

func TestHostInvocationsAreIsolated(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		aSource, bSource := &planningContext{}, &planningContext{closed: [2]bool{true, true}, accepted: true}
		aCtx, aCancel := context.WithCancel(context.Background())
		defer aCancel()
		bCtx, bCancel := context.WithCancel(context.Background())
		defer bCancel()
		aCalls, bCalls := 0, 0
		a := must(planhost.Start(aCtx, aSource.binding("similar-1"), func(_ context.Context, id controlruntime.TargetIdentity, value plancontrol.Assessment) error {
			aCalls++
			if id.Key() != "similar-1" || value.Outcome() != plancontrol.Retain {
				t.Error("A contamination")
			}
			return nil
		}))
		b := must(planhost.Start(bCtx, bSource.binding("similar-10"), func(_ context.Context, id controlruntime.TargetIdentity, value plancontrol.Assessment) error {
			bCalls++
			if id.Key() != "similar-10" || value.Outcome() != plancontrol.Complete {
				t.Error("B contamination")
			}
			return nil
		}))
		var wg sync.WaitGroup
		wg.Go(func() {
			if err := a.Trigger(aCtx); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() {
			if err := b.Trigger(bCtx); err != nil {
				t.Error(err)
			}
		})
		wg.Wait()
		<-a.Reports()
		<-b.Reports()
		aCancel()
		_ = a.Wait()
		if err := b.Trigger(bCtx); err != nil {
			t.Fatal(err)
		}
		<-b.Reports()
		bCancel()
		_ = b.Wait()
		if diff := cmp.Diff([]int{1, 2}, []int{aCalls, bCalls}); diff != "" {
			t.Fatal(diff)
		}
		if len(aSource.writes) != 0 || len(bSource.writes) != 0 {
			t.Fatal("source mutation")
		}
	})
}

func TestHostTriggerRacingSubmissionAndParentCancellation(t *testing.T) {
	for _, endParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "submission", true: "both lifecycles"}[endParent], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				parent, stop := context.WithCancel(context.Background())
				defer stop()
				submission, cancel := context.WithCancel(parent)
				defer cancel()
				source := &planningContext{}
				handoffs := 0
				host := must(planhost.Start(parent, source.binding("intake-race"), func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error {
					handoffs++
					return nil
				}))
				gate := make(chan struct{})
				result := make(chan error, 1)
				go func() { <-gate; result <- host.Trigger(submission) }()
				go func() {
					<-gate
					cancel()
					if endParent {
						stop()
					}
				}()
				close(gate)
				err := <-result
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Fatalf("acceptance outcome=%v", err)
				}
				synctest.Wait()
				// Acceptance and cancellation are unordered. At most this one
				// request can deliver; cancellation alone supplies no wake-up.
				if handoffs > 1 {
					t.Fatal("extra handoff")
				}
				if !endParent && err == nil && handoffs != 1 {
					t.Fatal("accepted submission was cancelled by its context")
				}
				stop()
				if !errors.Is(host.Wait(), context.Canceled) || !errors.Is(host.Trigger(submission), context.Canceled) || !errors.Is(host.Trigger(context.Background()), context.Canceled) {
					t.Fatal("unstable parent termination")
				}
				before := handoffs
				for range host.Reports() {
				}
				synctest.Wait()
				if handoffs != before {
					t.Fatal("handoff after stop")
				}
			})
		})
	}
}
