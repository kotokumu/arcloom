package planrepresentation_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
)

func TestNewControllerMissingObserver(t *testing.T) {
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		wantErr  bool
	}{
		{name: "missing observer", observer: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := planrepresentation.NewController(tt.observer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewController() error = %v, wantErr %v", err, tt.wantErr)
			}
			if controller != nil {
				t.Fatalf("controller = %v, want nil", controller)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObserver, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewControllerConfigured(t *testing.T) {
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		wantErr  bool
	}{
		{name: "configured observer", observer: func(context.Context) (planrepresentation.Observation, error) {
			return planrepresentation.UnavailableObservation(), nil
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := planrepresentation.NewController(tt.observer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewController() error = %v, wantErr %v", err, tt.wantErr)
			}
			if controller == nil {
				t.Fatal("NewController() returned nil controller")
			}
		})
	}
}

func TestReconcileNilContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Determination
		wantErr bool
	}{
		{name: "nil context", args: args{ctx: nil}, want: planrepresentation.Determination(0), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var controller planrepresentation.Controller
			result, err := controller.Reconcile(tt.args.ctx, plan.Plan{}) //nolint:staticcheck // SA1012: nil context intentionally verifies the documented InvalidContext contract.
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.InvalidContext, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCancelledContext(t *testing.T) {
	tests := []struct{ name string }{{name: "cancelled context"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var controller planrepresentation.Controller
			result, err := controller.Reconcile(ctx, plan.Plan{})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileExpiredContext(t *testing.T) {
	tests := []struct{ name string }{{name: "expired context"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			var controller planrepresentation.Controller
			result, err := controller.Reconcile(ctx, plan.Plan{})
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want context.DeadlineExceeded", err)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileZeroController(t *testing.T) {
	tests := []struct{ name string }{{name: "zero controller"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			var controller planrepresentation.Controller
			result, err := controller.Reconcile(context.Background(), expected)
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObserver, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidExpectedPlan(t *testing.T) {
	tests := []struct{ name string }{{name: "invalid expected Plan"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observerCalled := false
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				observerCalled = true
				return planrepresentation.UnavailableObservation(), nil
			}))
			result, err := controller.Reconcile(context.Background(), plan.Plan{})
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			if observerCalled {
				t.Fatal("observer called for invalid expected Plan")
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.InvalidExpectedPlan, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidObservationContract(t *testing.T) {
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		want     planrepresentation.Result
		wantErr  bool
	}{
		{name: "invalid observation", observer: func(context.Context) (planrepresentation.Observation, error) {
			return planrepresentation.Observation{}, nil
		}, want: planrepresentation.Result{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(tt.observer))
			result, err := controller.Reconcile(context.Background(), validPlan)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), result.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileObserverNonContextError(t *testing.T) {
	providerSentinel := errors.New("provider failure")
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		want     planrepresentation.Result
		wantErr  bool
	}{
		{name: "non context error", observer: func(context.Context) (planrepresentation.Observation, error) {
			return planrepresentation.Observation{}, providerSentinel
		}, want: planrepresentation.Result{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(tt.observer))
			result, err := controller.Reconcile(context.Background(), validPlan)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), result.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
			if errors.Is(err, providerSentinel) {
				t.Errorf("error exposes Provider sentinel: %v", err)
			}
		})
	}
}

func TestReconcileObserverUnrelatedContextError(t *testing.T) {
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		want     planrepresentation.Result
		wantErr  bool
	}{
		{name: "unrelated context error", observer: func(context.Context) (planrepresentation.Observation, error) {
			return planrepresentation.Observation{}, context.Canceled
		}, want: planrepresentation.Result{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(tt.observer))
			result, err := controller.Reconcile(context.Background(), validPlan)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), result.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileObserverObservationPlusError(t *testing.T) {
	providerSentinel := errors.New("provider failure")
	tests := []struct {
		name     string
		observer planrepresentation.Observer
		want     planrepresentation.Result
		wantErr  bool
	}{
		{name: "observation plus error", observer: func(context.Context) (planrepresentation.Observation, error) {
			return planrepresentation.UnavailableObservation(), providerSentinel
		}, want: planrepresentation.Result{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(tt.observer))
			result, err := controller.Reconcile(context.Background(), validPlan)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), result.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			var failure *planrepresentation.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("error = %v, want *FailureError", err)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
			if errors.Is(err, providerSentinel) {
				t.Errorf("error exposes Provider sentinel: %v", err)
			}
		})
	}
}

func TestReconcileMatchingObserverContextError(t *testing.T) {
	tests := []struct{ name string }{{name: "matching observer context error"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			ctx, cancel := context.WithCancel(context.Background())
			controller := must(planrepresentation.NewController(func(ctx context.Context) (planrepresentation.Observation, error) {
				cancel()
				return planrepresentation.Observation{}, ctx.Err()
			}))
			result, err := controller.Reconcile(ctx, validPlan)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCancellationBeforeSuccessfulReturn(t *testing.T) {
	tests := []struct{ name string }{{name: "cancellation before successful return"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			reached := make(chan struct{})
			release := make(chan struct{})
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				close(reached)
				<-release
				return planrepresentation.UnavailableObservation(), nil
			}))
			ctx, cancel := context.WithCancel(context.Background())
			resultCh := make(chan struct {
				result planrepresentation.Result
				err    error
			})
			go func() {
				result, err := controller.Reconcile(ctx, validPlan)
				resultCh <- struct {
					result planrepresentation.Result
					err    error
				}{result: result, err: err}
			}()
			<-reached
			cancel()
			close(release)
			returned := <-resultCh
			if !errors.Is(returned.err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", returned.err)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), returned.result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCancellationBeforeObserverFailureReturn(t *testing.T) {
	tests := []struct{ name string }{{name: "cancellation before observer failure return"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			reached := make(chan struct{})
			release := make(chan struct{})
			providerSentinel := errors.New("provider failure")
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				close(reached)
				<-release
				return planrepresentation.Observation{}, providerSentinel
			}))
			ctx, cancel := context.WithCancel(context.Background())
			resultCh := make(chan struct {
				result planrepresentation.Result
				err    error
			})
			go func() {
				result, err := controller.Reconcile(ctx, validPlan)
				resultCh <- struct {
					result planrepresentation.Result
					err    error
				}{result: result, err: err}
			}()
			<-reached
			cancel()
			close(release)
			returned := <-resultCh
			if !errors.Is(returned.err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", returned.err)
			}
			if errors.Is(returned.err, providerSentinel) {
				t.Fatalf("error exposes Provider sentinel: %v", returned.err)
			}
			if diff := cmp.Diff(planrepresentation.Determination(0), returned.result.Determination()); diff != "" {
				t.Errorf("zero result determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCancellationAfterResultReturn(t *testing.T) {
	tests := []struct{ name string }{{name: "cancellation after result return"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			ctx, cancel := context.WithCancel(context.Background())
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) { return observation, nil }))
			result, err := controller.Reconcile(ctx, validPlan)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			cancel()
			if diff := cmp.Diff(planrepresentation.Satisfied, result.Determination()); diff != "" {
				t.Errorf("determination after cancellation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("evidence after cancellation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCurrentObservationReconstruction(t *testing.T) {
	tests := []struct{ name string }{{name: "current observation reconstruction"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			first := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			second := planrepresentation.AbsentObservation()
			observations := []planrepresentation.Observation{first, second}
			var mu sync.Mutex
			index := 0
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				mu.Lock()
				defer mu.Unlock()
				observation := observations[index]
				index++
				return observation, nil
			}))
			firstResult := must(controller.Reconcile(context.Background(), validPlan))
			secondResult := must(controller.Reconcile(context.Background(), validPlan))
			if diff := cmp.Diff(planrepresentation.Satisfied, firstResult.Determination()); diff != "" {
				t.Errorf("first determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, secondResult.Determination()); diff != "" {
				t.Errorf("second determination mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileConcurrentCorrelationIsolation(t *testing.T) {
	type correlationKey struct{}
	tests := []struct{ name string }{{name: "independent callers"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPlan := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			arrived := make(chan struct{}, 2)
			release := make(chan struct{})
			controller := must(planrepresentation.NewController(func(ctx context.Context) (planrepresentation.Observation, error) {
				arrived <- struct{}{}
				<-release
				value := ctx.Value(correlationKey{})
				correlation, ok := value.(string)
				if !ok {
					return planrepresentation.Observation{}, errors.New("missing correlation")
				}
				return must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName(correlation), planrepresentation.ClassifyGoal("Ship-"+correlation), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.UnavailableTargetDate())), nil
			}))
			type returnedResult struct {
				correlation string
				result      planrepresentation.Result
				err         error
			}
			results := make(chan returnedResult, 2)
			mutationArrived := make(chan struct{}, 2)
			mutationRelease := make(chan struct{})
			mutationDone := make(chan struct{}, 2)
			mutationErrors := make(chan string, 2)
			for _, correlation := range []string{"Release", "Other"} {
				go func(correlation string) {
					ctx := context.WithValue(context.Background(), correlationKey{}, correlation)
					result, err := controller.Reconcile(ctx, validPlan)
					mutationArrived <- struct{}{}
					<-mutationRelease
					differences := result.Differences()
					unavailable := result.UnavailableInformation()
					if len(differences) == 0 || len(unavailable) == 0 {
						mutationErrors <- correlation
					} else {
						differences[0] = nil
						unavailable[0] = nil
					}
					mutationDone <- struct{}{}
					results <- returnedResult{correlation: correlation, result: result, err: err}
				}(correlation)
			}
			<-arrived
			<-arrived
			close(release)
			<-mutationArrived
			<-mutationArrived
			close(mutationRelease)
			<-mutationDone
			<-mutationDone
			if len(mutationErrors) != 0 {
				t.Fatalf("concurrent result lacked difference and unavailable evidence")
			}
			var releaseResult planrepresentation.Result
			var otherResult planrepresentation.Result
			for range 2 {
				returned := <-results
				if returned.err != nil {
					t.Fatalf("Reconcile() error for %q = %v", returned.correlation, returned.err)
				}
				switch returned.correlation {
				case "Release":
					releaseResult = returned.result
				case "Other":
					otherResult = returned.result
				default:
					t.Fatalf("unexpected correlation %q", returned.correlation)
				}
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, releaseResult.Determination()); diff != "" {
				t.Errorf("Release determination mismatch (-want +got):\n%s", diff)
			}
			releaseDifferences := releaseResult.Differences()
			if diff := cmp.Diff(1, len(releaseDifferences)); diff != "" {
				t.Fatalf("Release difference count mismatch (-want +got):\n%s", diff)
			}
			releaseDifference, ok := releaseDifferences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("Release difference type = %T, want ValueDifferentDifference", releaseDifferences[0])
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, releaseDifference.Location().Kind()); diff != "" {
				t.Errorf("Release difference location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ValueDifferentCategory, releaseDifference.Category()); diff != "" {
				t.Errorf("Release difference category mismatch (-want +got):\n%s", diff)
			}
			releaseExpected, ok := releaseDifference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Release expected meaning type = %T, want TextMeaning", releaseDifference.Expected())
			}
			if diff := cmp.Diff("Ship", releaseExpected.Text()); diff != "" {
				t.Errorf("Release expected payload mismatch (-want +got):\n%s", diff)
			}
			releaseObserved, ok := releaseDifference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Release observed meaning type = %T, want TextMeaning", releaseDifference.Observed())
			}
			if diff := cmp.Diff("Ship-Release", releaseObserved.Text()); diff != "" {
				t.Errorf("Release observed payload mismatch (-want +got):\n%s", diff)
			}
			releaseUnavailable := releaseResult.UnavailableInformation()
			if diff := cmp.Diff(1, len(releaseUnavailable)); diff != "" {
				t.Fatalf("Release unavailable count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, releaseUnavailable[0].Location().Kind()); diff != "" {
				t.Errorf("Release unavailable location mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(planrepresentation.Undecidable, otherResult.Determination()); diff != "" {
				t.Errorf("Other determination mismatch (-want +got):\n%s", diff)
			}
			otherDifferences := otherResult.Differences()
			if diff := cmp.Diff(2, len(otherDifferences)); diff != "" {
				t.Fatalf("Other difference count mismatch (-want +got):\n%s", diff)
			}
			otherName, ok := otherDifferences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("Other name difference type = %T, want ValueDifferentDifference", otherDifferences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, otherName.Location().Kind()); diff != "" {
				t.Errorf("Other name location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ValueDifferentCategory, otherName.Category()); diff != "" {
				t.Errorf("Other name category mismatch (-want +got):\n%s", diff)
			}
			otherNameExpected, ok := otherName.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Other name expected type = %T, want TextMeaning", otherName.Expected())
			}
			if diff := cmp.Diff("Release", otherNameExpected.Text()); diff != "" {
				t.Errorf("Other name expected payload mismatch (-want +got):\n%s", diff)
			}
			otherNameObserved, ok := otherName.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Other name observed type = %T, want TextMeaning", otherName.Observed())
			}
			if diff := cmp.Diff("Other", otherNameObserved.Text()); diff != "" {
				t.Errorf("Other name observed payload mismatch (-want +got):\n%s", diff)
			}
			otherGoal, ok := otherDifferences[1].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("Other goal difference type = %T, want ValueDifferentDifference", otherDifferences[1])
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, otherGoal.Location().Kind()); diff != "" {
				t.Errorf("Other goal location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ValueDifferentCategory, otherGoal.Category()); diff != "" {
				t.Errorf("Other goal category mismatch (-want +got):\n%s", diff)
			}
			otherGoalExpected, ok := otherGoal.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Other goal expected type = %T, want TextMeaning", otherGoal.Expected())
			}
			if diff := cmp.Diff("Ship", otherGoalExpected.Text()); diff != "" {
				t.Errorf("Other goal expected payload mismatch (-want +got):\n%s", diff)
			}
			otherGoalObserved, ok := otherGoal.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("Other goal observed type = %T, want TextMeaning", otherGoal.Observed())
			}
			if diff := cmp.Diff("Ship-Other", otherGoalObserved.Text()); diff != "" {
				t.Errorf("Other goal observed payload mismatch (-want +got):\n%s", diff)
			}
			otherUnavailable := otherResult.UnavailableInformation()
			if diff := cmp.Diff(1, len(otherUnavailable)); diff != "" {
				t.Fatalf("Other unavailable count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, otherUnavailable[0].Location().Kind()); diff != "" {
				t.Errorf("Other unavailable location mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
