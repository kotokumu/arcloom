package planrepresentation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
)

func TestAbsentObservation(t *testing.T) {
	tests := []struct {
		name string
		want planrepresentation.Observation
	}{
		{name: "authoritative absence", want: planrepresentation.AbsentObservation()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return tt.want, nil
			})))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want ExpectedAbsentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanRootLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			meaning, ok := difference.Expected().(planrepresentation.PlanMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want PlanMeaning", difference.Expected())
			}
			if diff := cmp.Diff(expected.Name(), meaning.Plan().Name()); diff != "" {
				t.Errorf("root payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUnavailableObservation(t *testing.T) {
	tests := []struct {
		name string
		want planrepresentation.Observation
	}{
		{name: "unavailable existence", want: planrepresentation.UnavailableObservation()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return tt.want, nil
			})))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
			unavailable := result.UnavailableInformation()
			if diff := cmp.Diff(1, len(unavailable)); diff != "" {
				t.Fatalf("unavailable count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.PlanRootLocation, unavailable[0].Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPresentObservationInvalid(t *testing.T) {
	type args struct {
		name       planrepresentation.PlanNameObservation
		goal       planrepresentation.GoalObservation
		conditions planrepresentation.AcceptanceConditionsObservation
		tasks      planrepresentation.TasksObservation
		targetDate planrepresentation.TargetDateObservation
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Observation
		wantErr bool
	}{
		{name: "zero name", args: args{goal: planrepresentation.ClassifyGoal("Ship"), conditions: planrepresentation.CompleteAcceptanceConditions([]string{"A"}), tasks: planrepresentation.CompleteTasks(nil), targetDate: planrepresentation.AbsentTargetDate()}, want: planrepresentation.Observation{}, wantErr: true},
		{name: "zero goal", args: args{name: planrepresentation.ClassifyPlanName("Release"), conditions: planrepresentation.CompleteAcceptanceConditions([]string{"A"}), tasks: planrepresentation.CompleteTasks(nil), targetDate: planrepresentation.AbsentTargetDate()}, want: planrepresentation.Observation{}, wantErr: true},
		{name: "zero acceptance collection", args: args{name: planrepresentation.ClassifyPlanName("Release"), goal: planrepresentation.ClassifyGoal("Ship"), tasks: planrepresentation.CompleteTasks(nil), targetDate: planrepresentation.AbsentTargetDate()}, want: planrepresentation.Observation{}, wantErr: true},
		{name: "zero Task collection", args: args{name: planrepresentation.ClassifyPlanName("Release"), goal: planrepresentation.ClassifyGoal("Ship"), conditions: planrepresentation.CompleteAcceptanceConditions([]string{"A"}), targetDate: planrepresentation.AbsentTargetDate()}, want: planrepresentation.Observation{}, wantErr: true},
		{name: "zero target date", args: args{name: planrepresentation.ClassifyPlanName("Release"), goal: planrepresentation.ClassifyGoal("Ship"), conditions: planrepresentation.CompleteAcceptanceConditions([]string{"A"}), tasks: planrepresentation.CompleteTasks(nil)}, want: planrepresentation.Observation{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.NewPresentObservation(tt.args.name, tt.args.goal, tt.args.conditions, tt.args.tasks, tt.args.targetDate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewPresentObservation() error = %v, wantErr %v", err, tt.wantErr)
			}
			var validation *planrepresentation.ObservationValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *ObservationValidationError", err)
			}

			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			wantController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return tt.want, nil
			})))
			wantResult, wantErrValue := wantController.Reconcile(context.Background(), expected)
			if diff := cmp.Diff(planrepresentation.Determination(0), wantResult.Determination()); diff != "" {
				t.Errorf("zero want determination mismatch (-want +got):\n%s", diff)
			}
			var wantFailure *planrepresentation.FailureError
			if !errors.As(wantErrValue, &wantFailure) {
				t.Fatalf("zero want error = %v, want *FailureError", wantErrValue)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, wantFailure.Code()); diff != "" {
				t.Errorf("zero want failure code mismatch (-want +got):\n%s", diff)
			}

			gotController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return got, nil
			})))
			gotResult, gotErr := gotController.Reconcile(context.Background(), expected)
			if diff := cmp.Diff(planrepresentation.Determination(0), gotResult.Determination()); diff != "" {
				t.Errorf("returned observation determination mismatch (-want +got):\n%s", diff)
			}
			var gotFailure *planrepresentation.FailureError
			if !errors.As(gotErr, &gotFailure) {
				t.Fatalf("returned observation error = %v, want *FailureError", gotErr)
			}
			if diff := cmp.Diff(planrepresentation.ObservationContract, gotFailure.Code()); diff != "" {
				t.Errorf("returned observation failure code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
