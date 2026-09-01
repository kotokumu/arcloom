package planattempt_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func TestPlanAttemptSetupValidation(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "one"))
	observer := plansnapshot.Observer(func(context.Context) (plansnapshot.Snapshot, error) { return plansnapshot.Snapshot{}, nil })
	delivery := planattempt.DeliveryObserver[string](func(context.Context) (string, error) { return "", nil })
	assessor := plancontrol.Assessor[string](func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
		return plancontrol.AssessorResponse{}, nil
	})
	resolver := planattempt.TargetResolver[string](func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
		return planattempt.NewPlanTarget(identity, observer, delivery)
	})

	targetTests := []struct {
		name     string
		identity controlruntime.TargetIdentity
		observer plansnapshot.Observer
		delivery planattempt.DeliveryObserver[string]
	}{
		{name: "zero identity", observer: observer, delivery: delivery},
		{name: "nil snapshot observer", identity: identity, delivery: delivery},
		{name: "nil delivery observer", identity: identity, observer: observer},
	}
	for _, tt := range targetTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planattempt.NewPlanTarget(tt.identity, tt.observer, tt.delivery)
			if !errors.Is(err, planattempt.ErrInvalidPlanTarget) {
				t.Errorf("NewPlanTarget() error = %v", err)
			}
		})
	}

	attemptTests := []struct {
		name     string
		kind     string
		resolver planattempt.TargetResolver[string]
		assessor plancontrol.Assessor[string]
		want     error
	}{
		{name: "invalid kind", kind: " plan", resolver: resolver, assessor: assessor, want: planattempt.ErrInvalidTargetKind},
		{name: "nil resolver", kind: "plan", assessor: assessor, want: planattempt.ErrInvalidTargetResolver},
		{name: "nil assessor", kind: "plan", resolver: resolver, want: planattempt.ErrInvalidAssessor},
	}
	for _, tt := range attemptTests {
		t.Run(tt.name, func(t *testing.T) {
			attempt, err := planattempt.NewAttempt(tt.kind, tt.resolver, tt.assessor)
			if !errors.Is(err, tt.want) {
				t.Errorf("NewAttempt() error = %v, want %v", err, tt.want)
			}
			if attempt != nil {
				t.Error("NewAttempt() returned an Attempt for invalid setup")
			}
		})
	}
}

func TestPlanAttemptWithoutCurrentPlanIsSuccessfulObservationOnly(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "without-current"))
	observedTask := must(plansnapshot.NewTaskProgress("Observed task", plansnapshot.Closed))
	snapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Unknown, []plansnapshot.TaskProgress{observedTask}))))
	deliveryCalls := 0
	assessorCalls := 0
	target := must(planattempt.NewPlanTarget(identity,
		func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil },
		func(context.Context) (string, error) { deliveryCalls++; return "delivery", nil },
	))
	attempt := must(planattempt.NewAttempt("plan",
		func(_ context.Context, requested controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			if requested != identity {
				t.Errorf("Resolver identity = %v/%v", requested.Kind(), requested.Key())
			}
			return target, nil
		},
		func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
			assessorCalls++
			return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
		},
	))

	result, directive, err := attempt(context.Background(), identity)
	if err != nil {
		t.Fatalf("Attempt() error = %v", err)
	}
	if result.Kind() != planattempt.CurrentPlanNotEstablished {
		t.Errorf("AttemptResult Kind = %v", result.Kind())
	}
	if _, ok := result.Assessment(); ok {
		t.Error("no-current success exposed an Assessment")
	}
	if _, ok := result.Snapshot().CurrentPlan(); ok {
		t.Error("preserved Snapshot unexpectedly has a current Plan")
	}
	progress, ok := result.Snapshot().Progress()
	if !ok || progress.OverallState() != plansnapshot.Unknown || !progress.MembershipComplete() {
		t.Errorf("preserved progress = (%v, %v, %v)", progress.OverallState(), progress.MembershipComplete(), ok)
	}
	if tasks := progress.Tasks(); len(tasks) != 1 || tasks[0].Name() != observedTask.Name() || tasks[0].State() != observedTask.State() {
		t.Errorf("preserved task progress = %v", tasks)
	}
	if directive.Kind() != controlruntime.AwaitRequest {
		t.Errorf("Directive = %v, want AwaitRequest", directive.Kind())
	}
	if deliveryCalls != 0 || assessorCalls != 0 {
		t.Errorf("later boundary calls = delivery %d, assessor %d", deliveryCalls, assessorCalls)
	}
}

func TestPlanAttemptWithCurrentPlanAssessesInOrder(t *testing.T) {
	currentTask := must(plan.NewTask("Current task"))
	current := must(plan.New("current", must(plan.NewGoal("Deliver the goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("The outcome is accepted"))}, []plan.Task{currentTask}, nil))
	proposed := must(plan.New("proposed", must(plan.NewGoal("Deliver the revised goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("The revised outcome is accepted"))}, nil, nil))
	currentTaskProgress := must(plansnapshot.NewTaskProgress(currentTask.Name(), plansnapshot.Closed))
	snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{currentTaskProgress}))))
	identity := must(controlruntime.NewTargetIdentity("plan", "with-current"))
	tests := []struct {
		name    string
		outcome plancontrol.Outcome
	}{
		{name: "complete", outcome: plancontrol.Complete},
		{name: "retain", outcome: plancontrol.Retain},
		{name: "revise", outcome: plancontrol.Revise},
		{name: "insufficient information", outcome: plancontrol.InsufficientInformation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := []string{}
			target := must(planattempt.NewPlanTarget(identity,
				func(context.Context) (plansnapshot.Snapshot, error) {
					order = append(order, "snapshot")
					return snapshot, nil
				},
				func(context.Context) (string, error) {
					order = append(order, "delivery")
					return "observed delivery", nil
				},
			))
			attempt := must(planattempt.NewAttempt("plan",
				func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
					return target, nil
				},
				func(_ context.Context, gotPlan plan.Plan, observations string) (plancontrol.AssessorResponse, error) {
					order = append(order, "assessment")
					if !gotPlan.Equal(current) || observations != "observed delivery" {
						t.Errorf("Assessor inputs did not come from current boundaries")
					}
					response := plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{tt.outcome}}
					if tt.outcome == plancontrol.Revise {
						response.ProposedPlans = []plan.Plan{proposed}
					}
					return response, nil
				},
			))

			result, directive, err := attempt(context.Background(), identity)
			if err != nil {
				t.Fatalf("Attempt() error = %v", err)
			}
			if diff := cmp.Diff([]string{"snapshot", "delivery", "assessment"}, order); diff != "" {
				t.Errorf("call order mismatch (-want +got):\n%s", diff)
			}
			if result.Kind() != planattempt.CurrentPlanAssessed {
				t.Errorf("AttemptResult Kind = %v", result.Kind())
			}
			returnedPlan, ok := result.Snapshot().CurrentPlan()
			if !ok || !returnedPlan.Equal(current) {
				t.Error("AttemptResult did not preserve the assessed Snapshot")
			}
			returnedProgress, ok := result.Snapshot().Progress()
			if !ok || returnedProgress.OverallState() != plansnapshot.Open || !returnedProgress.MembershipComplete() {
				t.Errorf("AttemptResult progress = (%v, %v, %v)", returnedProgress.OverallState(), returnedProgress.MembershipComplete(), ok)
			}
			returnedTasks := returnedProgress.Tasks()
			if len(returnedTasks) != 1 || returnedTasks[0].Name() != currentTaskProgress.Name() || returnedTasks[0].State() != currentTaskProgress.State() {
				t.Errorf("AttemptResult task progress = %v", returnedTasks)
			}
			assessment, ok := result.Assessment()
			if !ok || assessment.Outcome() != tt.outcome || !assessment.AssessedPlan().Equal(current) {
				t.Errorf("Assessment = (%v, %v), want outcome %v", assessment.Outcome(), ok, tt.outcome)
			}
			if directive.Kind() != controlruntime.AwaitRequest {
				t.Errorf("Directive = %v, want AwaitRequest", directive.Kind())
			}
			if tt.outcome == plancontrol.Revise {
				gotProposed, present := assessment.ProposedPlan()
				if !present || !gotProposed.Equal(proposed) {
					t.Error("Revise Assessment did not preserve the exact Proposed Plan")
				}
			}
		})
	}
}

func TestPlanAttemptNormalizesBindingAndDeliveryFailures(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "failure"))
	other := must(controlruntime.NewTargetIdentity("plan", "other"))
	current := must(plan.New("current", must(plan.NewGoal("Deliver the goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("The outcome is accepted"))}, nil, nil))
	snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	providerErr := errors.New("provider detail")
	validAssessor := plancontrol.Assessor[string](func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
	})
	tests := []struct {
		name     string
		request  controlruntime.TargetIdentity
		resolver planattempt.TargetResolver[string]
		wantCode planattempt.FailureCode
	}{
		{name: "wrong kind", request: must(controlruntime.NewTargetIdentity("other", "failure")), resolver: func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			t.Fatal("resolver called")
			return planattempt.PlanTarget[string]{}, nil
		}, wantCode: planattempt.TargetBindingUnavailable},
		{name: "zero binding without resolver error", request: identity, resolver: func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return planattempt.PlanTarget[string]{}, nil
		}, wantCode: planattempt.TargetBindingUnavailable},
		{name: "resolver error", request: identity, resolver: func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return planattempt.PlanTarget[string]{}, providerErr
		}, wantCode: planattempt.TargetBindingUnavailable},
		{name: "mismatched binding", request: identity, resolver: func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return planattempt.NewPlanTarget(other, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "", nil })
		}, wantCode: planattempt.TargetBindingUnavailable},
		{name: "delivery error", request: identity, resolver: func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "", providerErr })
		}, wantCode: planattempt.DeliveryObservationUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := must(planattempt.NewAttempt("plan", tt.resolver, validAssessor))
			result, directive, err := attempt(context.Background(), tt.request)
			var failure *planattempt.FailureError
			if !errors.As(err, &failure) || failure.Code() != tt.wantCode {
				t.Errorf("Attempt() error = %v, want code %s", err, tt.wantCode)
			}
			if errors.Is(err, providerErr) {
				t.Error("Plan FailureError exposed provider error")
			}
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("failure returned non-zero result/directive: %v/%v", result.Kind(), directive.Kind())
			}
		})
	}
}

func TestPlanAttemptPreservesSnapshotAndAssessmentFailures(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "owned-failures"))
	current := must(plan.New("current", must(plan.NewGoal("Deliver the goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("The outcome is accepted"))}, nil, nil))
	snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	tests := []struct {
		name                string
		observer            plansnapshot.Observer
		assessor            plancontrol.Assessor[string]
		wantSnapshotFailure bool
		wantAssessmentCode  plancontrol.FailureCode
	}{
		{
			name: "snapshot failure",
			observer: func(context.Context) (plansnapshot.Snapshot, error) {
				return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
			},
			assessor: func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
				t.Fatal("assessor called")
				return plancontrol.AssessorResponse{}, nil
			},
			wantSnapshotFailure: true,
		},
		{
			name:     "assessment failure",
			observer: func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil },
			assessor: func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
				return plancontrol.AssessorResponse{}, errors.New("provider detail")
			},
			wantAssessmentCode: plancontrol.AIBoundaryFailure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := must(planattempt.NewPlanTarget(identity, tt.observer, func(context.Context) (string, error) { return "delivery", nil }))
			attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
				return target, nil
			}, tt.assessor))
			result, directive, err := attempt(context.Background(), identity)
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("failure returned non-zero result/directive: %v/%v", result.Kind(), directive.Kind())
			}
			if tt.wantSnapshotFailure {
				var failure plansnapshot.ObservationFailure
				if !errors.As(err, &failure) || failure.Code() != plansnapshot.ObservationUnavailable {
					t.Errorf("Attempt() error = %v, want Snapshot failure", err)
				}
			}
			if tt.wantAssessmentCode != "" {
				var failure *plancontrol.FailureError
				if !errors.As(err, &failure) || failure.Code() != tt.wantAssessmentCode {
					t.Errorf("Attempt() error = %v, want Assessment code %s", err, tt.wantAssessmentCode)
				}
			}
		})
	}
}

func TestPlanAttemptCallerCancellationWinsAtEveryBoundary(t *testing.T) {
	identity := must(controlruntime.NewTargetIdentity("plan", "cancellation"))
	current := must(plan.New("current", must(plan.NewGoal("Deliver the goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("The outcome is accepted"))}, nil, nil))
	snapshot := must(plansnapshot.New(current, must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	validAssessor := plancontrol.Assessor[string](func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
		return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
	})

	t.Run("nil context", func(t *testing.T) {
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			t.Fatal("resolver called")
			return planattempt.PlanTarget[string]{}, nil
		}, validAssessor))
		//nolint:staticcheck // The public Attempt contract explicitly rejects nil.
		result, directive, err := attempt(nil, identity)
		if !errors.Is(err, controlruntime.ErrInvalidContext) || result.Kind() != 0 || directive.Kind() != 0 {
			t.Errorf("Attempt() = %v/%v/%v", result.Kind(), directive.Kind(), err)
		}
	})

	t.Run("before resolver", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			t.Fatal("resolver called")
			return planattempt.PlanTarget[string]{}, nil
		}, validAssessor))
		result, directive, err := attempt(ctx, identity)
		if !errors.Is(err, context.Canceled) || result.Kind() != 0 || directive.Kind() != 0 {
			t.Errorf("Attempt() = %v/%v/%v", result.Kind(), directive.Kind(), err)
		}
	})

	t.Run("after resolver", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		observerCalls := 0
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { observerCalls++; return snapshot, nil }, func(context.Context) (string, error) { return "", nil }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			cancel()
			return target, errors.New("resolver detail")
		}, validAssessor))
		result, directive, err := attempt(ctx, identity)
		if !errors.Is(err, context.Canceled) || result.Kind() != 0 || directive.Kind() != 0 || observerCalls != 0 {
			t.Errorf("Attempt() = %v/%v/%v, observer calls %d", result.Kind(), directive.Kind(), err, observerCalls)
		}
	})

	t.Run("during resolver", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		attempt := must(planattempt.NewAttempt("plan", func(ctx context.Context, _ controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			close(started)
			<-ctx.Done()
			return planattempt.PlanTarget[string]{}, errors.New("resolver detail")
		}, validAssessor))
		returned := make(chan error, 1)
		go func() {
			result, directive, err := attempt(ctx, identity)
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("Attempt() returned non-zero values: %v/%v", result.Kind(), directive.Kind())
			}
			returned <- err
		}()
		<-started
		cancel()
		if err := <-returned; !errors.Is(err, context.Canceled) {
			t.Errorf("Attempt() error = %v", err)
		}
	})

	t.Run("after snapshot", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		deliveryCalls := 0
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { cancel(); return snapshot, nil }, func(context.Context) (string, error) { deliveryCalls++; return "", nil }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, validAssessor))
		result, directive, err := attempt(ctx, identity)
		if !errors.Is(err, context.Canceled) || result.Kind() != 0 || directive.Kind() != 0 || deliveryCalls != 0 {
			t.Errorf("Attempt() = %v/%v/%v, delivery calls %d", result.Kind(), directive.Kind(), err, deliveryCalls)
		}
	})

	t.Run("during snapshot", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		target := must(planattempt.NewPlanTarget(identity, func(ctx context.Context) (plansnapshot.Snapshot, error) {
			close(started)
			<-ctx.Done()
			return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
		}, func(context.Context) (string, error) { return "", nil }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, validAssessor))
		returned := make(chan error, 1)
		go func() {
			result, directive, err := attempt(ctx, identity)
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("Attempt() returned non-zero values: %v/%v", result.Kind(), directive.Kind())
			}
			returned <- err
		}()
		<-started
		cancel()
		if err := <-returned; !errors.Is(err, context.Canceled) {
			t.Errorf("Attempt() error = %v", err)
		}
	})

	t.Run("after delivery", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assessorCalls := 0
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { cancel(); return "delivery", errors.New("delivery detail") }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
			assessorCalls++
			return plancontrol.AssessorResponse{}, nil
		}))
		result, directive, err := attempt(ctx, identity)
		if !errors.Is(err, context.Canceled) || result.Kind() != 0 || directive.Kind() != 0 || assessorCalls != 0 {
			t.Errorf("Attempt() = %v/%v/%v, assessor calls %d", result.Kind(), directive.Kind(), err, assessorCalls)
		}
	})

	t.Run("during delivery", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(deliveryCtx context.Context) (string, error) {
			if deliveryCtx != ctx {
				t.Error("Delivery Observer received another context")
			}
			close(started)
			<-deliveryCtx.Done()
			return "nominal delivery", errors.New("delivery detail")
		}))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, validAssessor))
		returned := make(chan error, 1)
		go func() {
			result, directive, err := attempt(ctx, identity)
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("Attempt() returned non-zero values: %v/%v", result.Kind(), directive.Kind())
			}
			returned <- err
		}()
		<-started
		cancel()
		select {
		case err := <-returned:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Attempt() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Attempt did not return after Delivery cancellation")
		}
	})

	t.Run("after assessment", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "delivery", nil }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, func(context.Context, plan.Plan, string) (plancontrol.AssessorResponse, error) {
			cancel()
			return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
		}))
		result, directive, err := attempt(ctx, identity)
		if !errors.Is(err, context.Canceled) || result.Kind() != 0 || directive.Kind() != 0 {
			t.Errorf("Attempt() = %v/%v/%v", result.Kind(), directive.Kind(), err)
		}
	})

	t.Run("during assessment", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		target := must(planattempt.NewPlanTarget(identity, func(context.Context) (plansnapshot.Snapshot, error) { return snapshot, nil }, func(context.Context) (string, error) { return "delivery", nil }))
		attempt := must(planattempt.NewAttempt("plan", func(context.Context, controlruntime.TargetIdentity) (planattempt.PlanTarget[string], error) {
			return target, nil
		}, func(assessmentCtx context.Context, _ plan.Plan, _ string) (plancontrol.AssessorResponse, error) {
			if assessmentCtx != ctx {
				t.Error("Assessor received another context")
			}
			close(started)
			<-assessmentCtx.Done()
			return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}, nil
		}))
		returned := make(chan error, 1)
		go func() {
			result, directive, err := attempt(ctx, identity)
			if result.Kind() != 0 || directive.Kind() != 0 {
				t.Errorf("Attempt() returned non-zero values: %v/%v", result.Kind(), directive.Kind())
			}
			returned <- err
		}()
		<-started
		cancel()
		select {
		case err := <-returned:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Attempt() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Attempt did not return after Assessment cancellation")
		}
	})
}
