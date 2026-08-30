package planrepresentation_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
)

func TestReconcileCorrespondence(t *testing.T) {
	type args struct {
		expected    plan.Plan
		observation planrepresentation.Observation
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "equal collections in different order", args: args{
			expected: must(plan.New(
				"Release",
				must(plan.NewGoal("Ship")),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A")), must(plan.NewAcceptanceCondition("B"))},
				[]plan.Task{must(plan.NewTask("Build")), must(plan.NewTask("Test"))},
				nil,
			)),
			observation: must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"),
				planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"B", "A"}),
				planrepresentation.CompleteTasks([]string{"Test", "Build"}),
				planrepresentation.AbsentTargetDate(),
			)),
		}, want: planrepresentation.Satisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return tt.args.observation, nil
			}))
			if err != nil {
				t.Fatalf("NewController() error = %v", err)
			}
			result, err := controller.Reconcile(context.Background(), tt.args.expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileScalarDifferenceEvidence(t *testing.T) {
	tests := []struct{ name string }{{name: "scalar value difference"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observed := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName(" release "),
				planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}),
				planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observed, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want ValueDifferentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			expectedMeaning, ok := difference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want TextMeaning", difference.Expected())
			}
			observedMeaning, ok := difference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("observed meaning type = %T, want TextMeaning", difference.Observed())
			}
			if diff := cmp.Diff("Release", expectedMeaning.Text()); diff != "" {
				t.Errorf("expected text mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(" release ", observedMeaning.Text()); diff != "" {
				t.Errorf("observed text mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileNameExactPayloadBoundaries(t *testing.T) {
	type args struct {
		expected string
		observed string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "case", args: args{expected: "Release", observed: "release"}, want: planrepresentation.NotSatisfied},
		{name: "whitespace", args: args{expected: " Release ", observed: "Release"}, want: planrepresentation.NotSatisfied},
		{name: "Unicode code points", args: args{expected: "é", observed: "e\u0301"}, want: planrepresentation.NotSatisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New(tt.args.expected, must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName(tt.args.observed), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want ValueDifferentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			expectedMeaning, ok := difference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want TextMeaning", difference.Expected())
			}
			observedMeaning, ok := difference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("observed meaning type = %T, want TextMeaning", difference.Observed())
			}
			if diff := cmp.Diff(tt.args.expected, expectedMeaning.Text()); diff != "" {
				t.Errorf("expected payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.args.observed, observedMeaning.Text()); diff != "" {
				t.Errorf("observed payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileGoalExactPayloadBoundaries(t *testing.T) {
	type args struct {
		expected string
		observed string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "case", args: args{expected: "Ship", observed: "ship"}, want: planrepresentation.NotSatisfied},
		{name: "whitespace", args: args{expected: " Ship ", observed: "Ship"}, want: planrepresentation.NotSatisfied},
		{name: "line ending", args: args{expected: "Ship\r\nGoal", observed: "Ship\nGoal"}, want: planrepresentation.NotSatisfied},
		{name: "Unicode code points", args: args{expected: "é", observed: "e\u0301"}, want: planrepresentation.NotSatisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal(tt.args.expected)), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal(tt.args.observed), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want ValueDifferentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			expectedMeaning, ok := difference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want TextMeaning", difference.Expected())
			}
			observedMeaning, ok := difference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("observed meaning type = %T, want TextMeaning", difference.Observed())
			}
			if diff := cmp.Diff(tt.args.expected, expectedMeaning.Text()); diff != "" {
				t.Errorf("expected payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.args.observed, observedMeaning.Text()); diff != "" {
				t.Errorf("observed payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCollectionDifferenceEvidence(t *testing.T) {
	tests := []struct{ name string }{{name: "collection member differences"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			observed := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"),
				planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"a"}),
				planrepresentation.CompleteTasks([]string{"build"}),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observed, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			differences := result.Differences()
			if diff := cmp.Diff(4, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			first, ok := differences[0].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("first difference type = %T, want ExpectedAbsentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, first.Location().Kind()); diff != "" {
				t.Errorf("first location mismatch (-want +got):\n%s", diff)
			}
			firstMember, firstHasMember := first.Location().Member()
			if diff := cmp.Diff("A", firstMember); diff != "" {
				t.Errorf("first member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, firstHasMember); diff != "" {
				t.Errorf("first member presence mismatch (-want +got):\n%s", diff)
			}
			second, ok := differences[1].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("second difference type = %T, want UnexpectedPresentDifference", differences[1])
			}
			secondMember, secondHasMember := second.Location().Member()
			if diff := cmp.Diff("a", secondMember); diff != "" {
				t.Errorf("second member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, secondHasMember); diff != "" {
				t.Errorf("second member presence mismatch (-want +got):\n%s", diff)
			}
			third, ok := differences[2].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("third difference type = %T, want ExpectedAbsentDifference", differences[2])
			}
			thirdMember, thirdHasMember := third.Location().Member()
			if diff := cmp.Diff("Build", thirdMember); diff != "" {
				t.Errorf("third member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, thirdHasMember); diff != "" {
				t.Errorf("third member presence mismatch (-want +got):\n%s", diff)
			}
			fourth, ok := differences[3].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("fourth difference type = %T, want UnexpectedPresentDifference", differences[3])
			}
			fourthMember, fourthHasMember := fourth.Location().Member()
			if diff := cmp.Diff("build", fourthMember); diff != "" {
				t.Errorf("fourth member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, fourthHasMember); diff != "" {
				t.Errorf("fourth member presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileAcceptanceExactPayloadBoundaries(t *testing.T) {
	type args struct {
		expected string
		observed string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "case", args: args{expected: "A", observed: "a"}, want: planrepresentation.NotSatisfied},
		{name: "whitespace", args: args{expected: "A", observed: " A "}, want: planrepresentation.NotSatisfied},
		{name: "line ending", args: args{expected: "A\r\nB", observed: "A\nB"}, want: planrepresentation.NotSatisfied},
		{name: "Unicode code points", args: args{expected: "é", observed: "e\u0301"}, want: planrepresentation.NotSatisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(tt.args.expected))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{tt.args.observed}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(2, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			expectedFound := false
			observedFound := false
			for index, value := range differences {
				switch difference := value.(type) {
				case planrepresentation.ExpectedAbsentDifference:
					if expectedFound {
						t.Fatalf("difference[%d] repeated ExpectedAbsentDifference", index)
					}
					expectedFound = true
					if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, difference.Location().Kind()); diff != "" {
						t.Errorf("expected difference location mismatch (-want +got):\n%s", diff)
					}
					member, hasMember := difference.Location().Member()
					if diff := cmp.Diff(tt.args.expected, member); diff != "" {
						t.Errorf("expected member mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, hasMember); diff != "" {
						t.Errorf("expected member presence mismatch (-want +got):\n%s", diff)
					}
					meaning, ok := difference.Expected().(planrepresentation.TextMeaning)
					if !ok {
						t.Fatalf("expected payload type = %T, want TextMeaning", difference.Expected())
					}
					if diff := cmp.Diff(tt.args.expected, meaning.Text()); diff != "" {
						t.Errorf("expected payload mismatch (-want +got):\n%s", diff)
					}
				case planrepresentation.UnexpectedPresentDifference:
					if observedFound {
						t.Fatalf("difference[%d] repeated UnexpectedPresentDifference", index)
					}
					observedFound = true
					if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, difference.Location().Kind()); diff != "" {
						t.Errorf("observed difference location mismatch (-want +got):\n%s", diff)
					}
					member, hasMember := difference.Location().Member()
					if diff := cmp.Diff(tt.args.observed, member); diff != "" {
						t.Errorf("observed member mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, hasMember); diff != "" {
						t.Errorf("observed member presence mismatch (-want +got):\n%s", diff)
					}
					meaning, ok := difference.Observed().(planrepresentation.TextMeaning)
					if !ok {
						t.Fatalf("observed payload type = %T, want TextMeaning", difference.Observed())
					}
					if diff := cmp.Diff(tt.args.observed, meaning.Text()); diff != "" {
						t.Errorf("observed payload mismatch (-want +got):\n%s", diff)
					}
				default:
					t.Fatalf("difference[%d] type = %T, want expected or unexpected member", index, value)
				}
			}
			if diff := cmp.Diff(true, expectedFound); diff != "" {
				t.Errorf("expected difference presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, observedFound); diff != "" {
				t.Errorf("observed difference presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileTaskExactPayloadBoundaries(t *testing.T) {
	type args struct {
		expected string
		observed string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "case", args: args{expected: "Build", observed: "build"}, want: planrepresentation.NotSatisfied},
		{name: "whitespace", args: args{expected: "Build", observed: " Build "}, want: planrepresentation.NotSatisfied},
		{name: "Unicode code points", args: args{expected: "é", observed: "e\u0301"}, want: planrepresentation.NotSatisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask(tt.args.expected))}, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks([]string{tt.args.observed}), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(2, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			expectedFound := false
			observedFound := false
			for index, value := range differences {
				switch difference := value.(type) {
				case planrepresentation.ExpectedAbsentDifference:
					if expectedFound {
						t.Fatalf("difference[%d] repeated ExpectedAbsentDifference", index)
					}
					expectedFound = true
					if diff := cmp.Diff(planrepresentation.TaskMemberLocation, difference.Location().Kind()); diff != "" {
						t.Errorf("expected difference location mismatch (-want +got):\n%s", diff)
					}
					member, hasMember := difference.Location().Member()
					if diff := cmp.Diff(tt.args.expected, member); diff != "" {
						t.Errorf("expected member mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, hasMember); diff != "" {
						t.Errorf("expected member presence mismatch (-want +got):\n%s", diff)
					}
					meaning, ok := difference.Expected().(planrepresentation.TextMeaning)
					if !ok {
						t.Fatalf("expected payload type = %T, want TextMeaning", difference.Expected())
					}
					if diff := cmp.Diff(tt.args.expected, meaning.Text()); diff != "" {
						t.Errorf("expected payload mismatch (-want +got):\n%s", diff)
					}
				case planrepresentation.UnexpectedPresentDifference:
					if observedFound {
						t.Fatalf("difference[%d] repeated UnexpectedPresentDifference", index)
					}
					observedFound = true
					if diff := cmp.Diff(planrepresentation.TaskMemberLocation, difference.Location().Kind()); diff != "" {
						t.Errorf("observed difference location mismatch (-want +got):\n%s", diff)
					}
					member, hasMember := difference.Location().Member()
					if diff := cmp.Diff(tt.args.observed, member); diff != "" {
						t.Errorf("observed member mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, hasMember); diff != "" {
						t.Errorf("observed member presence mismatch (-want +got):\n%s", diff)
					}
					meaning, ok := difference.Observed().(planrepresentation.TextMeaning)
					if !ok {
						t.Fatalf("observed payload type = %T, want TextMeaning", difference.Observed())
					}
					if diff := cmp.Diff(tt.args.observed, meaning.Text()); diff != "" {
						t.Errorf("observed payload mismatch (-want +got):\n%s", diff)
					}
				default:
					t.Fatalf("difference[%d] type = %T, want expected or unexpected member", index, value)
				}
			}
			if diff := cmp.Diff(true, expectedFound); diff != "" {
				t.Errorf("expected difference presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, observedFound); diff != "" {
				t.Errorf("observed difference presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileEqualTargetDate(t *testing.T) {
	tests := []struct{ name string }{{name: "equal target date"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date := must(plan.ParseTargetDate("2026-09-01"))
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, &date))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.ClassifyTargetDate("2026-09-01"),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.Satisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileExpectedTargetDateAbsent(t *testing.T) {
	tests := []struct{ name string }{{name: "expected target date absent"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date := must(plan.ParseTargetDate("2026-09-01"))
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, &date))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
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
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			meaning, ok := difference.Expected().(planrepresentation.TargetDateMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want TargetDateMeaning", difference.Expected())
			}
			if diff := cmp.Diff("2026-09-01", meaning.TargetDate().String()); diff != "" {
				t.Errorf("expected date mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileUnexpectedTargetDatePresent(t *testing.T) {
	tests := []struct{ name string }{{name: "unexpected target date present"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.ClassifyTargetDate("2026-09-01"),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want UnexpectedPresentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			meaning, ok := difference.Observed().(planrepresentation.TargetDateMeaning)
			if !ok {
				t.Fatalf("observed meaning type = %T, want TargetDateMeaning", difference.Observed())
			}
			if diff := cmp.Diff("2026-09-01", meaning.TargetDate().String()); diff != "" {
				t.Errorf("observed date mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileDifferentTargetDate(t *testing.T) {
	tests := []struct{ name string }{{name: "different target date"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date := must(plan.ParseTargetDate("2026-09-01"))
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, &date))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.ClassifyTargetDate("2026-09-02"),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference type = %T, want ValueDifferentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			expectedMeaning, ok := difference.Expected().(planrepresentation.TargetDateMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want TargetDateMeaning", difference.Expected())
			}
			observedMeaning, ok := difference.Observed().(planrepresentation.TargetDateMeaning)
			if !ok {
				t.Fatalf("observed meaning type = %T, want TargetDateMeaning", difference.Observed())
			}
			if diff := cmp.Diff("2026-09-01", expectedMeaning.TargetDate().String()); diff != "" {
				t.Errorf("expected date mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("2026-09-02", observedMeaning.TargetDate().String()); diff != "" {
				t.Errorf("observed date mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidPlanName(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "carriage return", args: args{value: "Release\rName"}, wantErr: false},
		{name: "line feed", args: args{value: "Release\nName"}, wantErr: false},
		{name: "NEL", args: args{value: "Release\u0085Name"}, wantErr: false},
		{name: "line separator", args: args{value: "Release\u2028Name"}, wantErr: false},
		{name: "paragraph separator", args: args{value: "Release\u2029Name"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName(tt.args.value), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result, err := controller.Reconcile(context.Background(), expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.MultilineName, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidPlanNameText(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "blank", args: args{value: " \u2003"}, wantErr: false},
		{name: "invalid UTF-8", args: args{value: string([]byte{0xff})}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName(tt.args.value), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result, err := controller.Reconcile(context.Background(), expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidText, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidGoal(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "blank", args: args{value: " \u2003"}, wantErr: false},
		{name: "invalid UTF-8", args: args{value: string([]byte{0xff})}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal(tt.args.value),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result, err := controller.Reconcile(context.Background(), expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidText, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidAcceptanceCondition(t *testing.T) {
	tests := []struct{ name string }{{name: "invalid acceptance condition"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A", " "}), planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidText, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidTargetDate(t *testing.T) {
	tests := []struct{ name string }{{name: "invalid target date"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil),
				planrepresentation.ClassifyTargetDate("2027-02-29"),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidTargetDate, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileValidGoalBoundaries(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "surrounding whitespace and Unicode", args: args{value: "  出荷\t"}, want: planrepresentation.Satisfied},
		{name: "carriage return and line feed", args: args{value: "Ship\r\nGoal"}, want: planrepresentation.Satisfied},
		{name: "NEL", args: args{value: "Ship\u0085Goal"}, want: planrepresentation.Satisfied},
		{name: "line separator", args: args{value: "Ship\u2028Goal"}, want: planrepresentation.Satisfied},
		{name: "paragraph separator", args: args{value: "Ship\u2029Goal"}, want: planrepresentation.Satisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal(tt.args.value)), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal(tt.args.value), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileValidTargetDateBoundaries(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want planrepresentation.Determination
	}{
		{name: "year one lower bound", args: args{value: "0001-01-01"}, want: planrepresentation.Satisfied},
		{name: "year 9999 upper bound", args: args{value: "9999-12-31"}, want: planrepresentation.Satisfied},
		{name: "leap day", args: args{value: "2028-02-29"}, want: planrepresentation.Satisfied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date := must(plan.ParseTargetDate(tt.args.value))
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, &date))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.ClassifyTargetDate(tt.args.value)))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(tt.want, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileInvalidTargetDateBoundaries(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "year zero", args: args{value: "0000-01-01"}, wantErr: false},
		{name: "invalid leap day", args: args{value: "2027-02-29"}, wantErr: false},
		{name: "non padded month and day", args: args{value: "2028-2-9"}, wantErr: false},
		{name: "surrounding whitespace", args: args{value: " 2028-02-29 "}, wantErr: false},
		{name: "timestamp", args: args{value: "2028-02-29T00:00:00Z"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions([]string{"A"}), planrepresentation.CompleteTasks(nil), planrepresentation.ClassifyTargetDate(tt.args.value)))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result, err := controller.Reconcile(context.Background(), expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			difference, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidTargetDate, difference.Violation()); diff != "" {
				t.Errorf("violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileDuplicateAndRepeatedInvalidEvidence(t *testing.T) {
	tests := []struct{ name string }{{name: "duplicate and repeated invalid evidence"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			observed := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName("Release"),
				planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A", "A", " ", " "}),
				planrepresentation.CompleteTasks([]string{"Build", "Build", "bad\n", "bad\n"}),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observed, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			differences := result.Differences()
			if diff := cmp.Diff(4, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, differences[0].Location().Kind()); diff != "" {
				t.Errorf("acceptance collection location mismatch (-want +got):\n%s", diff)
			}
			first, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[0] type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(plan.DuplicateAcceptanceCondition, first.Violation()); diff != "" {
				t.Errorf("acceptance duplicate violation mismatch (-want +got):\n%s", diff)
			}
			second, ok := differences[1].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[1] type = %T, want InvalidObservedDifference", differences[1])
			}
			if diff := cmp.Diff(plan.InvalidText, second.Violation()); diff != "" {
				t.Errorf("acceptance invalid violation mismatch (-want +got):\n%s", diff)
			}
			third, ok := differences[2].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[2] type = %T, want InvalidObservedDifference", differences[2])
			}
			if diff := cmp.Diff(plan.DuplicateTask, third.Violation()); diff != "" {
				t.Errorf("Task duplicate violation mismatch (-want +got):\n%s", diff)
			}
			fourth, ok := differences[3].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[3] type = %T, want InvalidObservedDifference", differences[3])
			}
			if diff := cmp.Diff(plan.MultilineName, fourth.Violation()); diff != "" {
				t.Errorf("Task invalid violation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileUnavailableEvidenceOrderAndPartialPreservation(t *testing.T) {
	tests := []struct{ name string }{{name: "unavailable evidence order and partial preservation"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			observation := must(planrepresentation.NewPresentObservation(
				planrepresentation.UnavailablePlanName(),
				planrepresentation.UnavailableGoal(),
				planrepresentation.IncompleteAcceptanceConditions([]string{"unexpected", " "}),
				planrepresentation.IncompleteTasks([]string{"extra", "bad\n"}),
				planrepresentation.UnavailableTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observation, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(4, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			first, ok := differences[0].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[0] type = %T, want InvalidObservedDifference", differences[0])
			}
			if diff := cmp.Diff(plan.InvalidText, first.Violation()); diff != "" {
				t.Errorf("acceptance invalid violation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, first.Category()); diff != "" {
				t.Errorf("acceptance invalid category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, first.Location().Kind()); diff != "" {
				t.Errorf("acceptance invalid location mismatch (-want +got):\n%s", diff)
			}
			_, hasMember := first.Location().Member()
			if diff := cmp.Diff(false, hasMember); diff != "" {
				t.Errorf("acceptance invalid location member presence mismatch (-want +got):\n%s", diff)
			}
			second, ok := differences[1].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference[1] type = %T, want UnexpectedPresentDifference", differences[1])
			}
			if diff := cmp.Diff(planrepresentation.UnexpectedPresentCategory, second.Category()); diff != "" {
				t.Errorf("acceptance unexpected category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, second.Location().Kind()); diff != "" {
				t.Errorf("acceptance unexpected location mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := second.Location().Member()
			if diff := cmp.Diff("unexpected", member); diff != "" {
				t.Errorf("acceptance unexpected member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("acceptance unexpected member presence mismatch (-want +got):\n%s", diff)
			}
			secondMeaning, ok := second.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[1] meaning type = %T, want TextMeaning", second.Observed())
			}
			if diff := cmp.Diff("unexpected", secondMeaning.Text()); diff != "" {
				t.Errorf("acceptance unexpected payload mismatch (-want +got):\n%s", diff)
			}
			third, ok := differences[2].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[2] type = %T, want InvalidObservedDifference", differences[2])
			}
			if diff := cmp.Diff(plan.MultilineName, third.Violation()); diff != "" {
				t.Errorf("Task invalid violation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, third.Category()); diff != "" {
				t.Errorf("Task invalid category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TaskCollectionLocation, third.Location().Kind()); diff != "" {
				t.Errorf("Task invalid location mismatch (-want +got):\n%s", diff)
			}
			_, hasMember = third.Location().Member()
			if diff := cmp.Diff(false, hasMember); diff != "" {
				t.Errorf("Task invalid location member presence mismatch (-want +got):\n%s", diff)
			}
			fourth, ok := differences[3].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference[3] type = %T, want UnexpectedPresentDifference", differences[3])
			}
			if diff := cmp.Diff(planrepresentation.UnexpectedPresentCategory, fourth.Category()); diff != "" {
				t.Errorf("Task unexpected category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TaskMemberLocation, fourth.Location().Kind()); diff != "" {
				t.Errorf("Task unexpected location mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = fourth.Location().Member()
			if diff := cmp.Diff("extra", member); diff != "" {
				t.Errorf("Task unexpected member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("Task unexpected member presence mismatch (-want +got):\n%s", diff)
			}
			fourthMeaning, ok := fourth.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[3] meaning type = %T, want TextMeaning", fourth.Observed())
			}
			if diff := cmp.Diff("extra", fourthMeaning.Text()); diff != "" {
				t.Errorf("Task unexpected payload mismatch (-want +got):\n%s", diff)
			}
			unavailable := result.UnavailableInformation()
			if diff := cmp.Diff(5, len(unavailable)); diff != "" {
				t.Fatalf("unavailable count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, unavailable[0].Location().Kind()); diff != "" {
				t.Errorf("unavailable[0] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, unavailable[1].Location().Kind()); diff != "" {
				t.Errorf("unavailable[1] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, unavailable[2].Location().Kind()); diff != "" {
				t.Errorf("unavailable[2] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TaskCollectionLocation, unavailable[3].Location().Kind()); diff != "" {
				t.Errorf("unavailable[3] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, unavailable[4].Location().Kind()); diff != "" {
				t.Errorf("unavailable[4] location mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCompleteEmptyCollections(t *testing.T) {
	tests := []struct{ name string }{{name: "complete empty collections"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			observation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), planrepresentation.CompleteAcceptanceConditions(nil), planrepresentation.CompleteTasks(nil), planrepresentation.AbsentTargetDate()))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return observation, nil })))
			result := must(controller.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			differences := result.Differences()
			if diff := cmp.Diff(2, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			first, ok := differences[0].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[0] type = %T, want ExpectedAbsentDifference", differences[0])
			}
			second, ok := differences[1].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[1] type = %T, want ExpectedAbsentDifference", differences[1])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, first.Location().Kind()); diff != "" {
				t.Errorf("acceptance location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.TaskMemberLocation, second.Location().Kind()); diff != "" {
				t.Errorf("Task location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, first.Category()); diff != "" {
				t.Errorf("acceptance category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, second.Category()); diff != "" {
				t.Errorf("Task category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := first.Location().Member()
			if diff := cmp.Diff("A", member); diff != "" {
				t.Errorf("acceptance member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("acceptance member presence mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = second.Location().Member()
			if diff := cmp.Diff("Build", member); diff != "" {
				t.Errorf("Task member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("Task member presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileRootAbsence(t *testing.T) {
	tests := []struct{ name string }{{name: "root absence"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return planrepresentation.AbsentObservation(), nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
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
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, difference.Category()); diff != "" {
				t.Errorf("category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.PlanRootLocation, difference.Location().Kind()); diff != "" {
				t.Errorf("location mismatch (-want +got):\n%s", diff)
			}
			meaning, ok := difference.Expected().(planrepresentation.PlanMeaning)
			if !ok {
				t.Fatalf("expected meaning type = %T, want PlanMeaning", difference.Expected())
			}
			if diff := cmp.Diff(expected.Name(), meaning.Plan().Name()); diff != "" {
				t.Errorf("name payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(expected.Goal().Text(), meaning.Plan().Goal().Text()); diff != "" {
				t.Errorf("goal payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileRootUnavailable(t *testing.T) {
	tests := []struct{ name string }{{name: "root unavailable"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Build"))}, nil))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return planrepresentation.UnavailableObservation(), nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
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

func TestReconcileResultSnapshots(t *testing.T) {
	tests := []struct{ name string }{{name: "difference and Plan payload snapshots"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, nil))
			observed := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName(" release "),
				planrepresentation.ClassifyGoal("Ship"),
				planrepresentation.CompleteAcceptanceConditions([]string{"A"}),
				planrepresentation.CompleteTasks(nil),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observed, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			differences := result.Differences()
			if diff := cmp.Diff(1, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			differences[0] = nil
			rereadDifferences := result.Differences()
			if diff := cmp.Diff(1, len(rereadDifferences)); diff != "" {
				t.Fatalf("difference output snapshot count mismatch (-want +got):\n%s", diff)
			}
			if rereadDifferences[0] == nil {
				t.Fatal("difference output snapshot returned nil element")
			}
			rereadDifference, ok := rereadDifferences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference output snapshot type = %T, want ValueDifferentDifference", rereadDifferences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, rereadDifference.Location().Kind()); diff != "" {
				t.Errorf("difference output snapshot location mismatch (-want +got):\n%s", diff)
			}
			rereadExpected, ok := rereadDifference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference output snapshot expected type = %T, want TextMeaning", rereadDifference.Expected())
			}
			if diff := cmp.Diff("Release", rereadExpected.Text()); diff != "" {
				t.Errorf("difference output snapshot expected payload mismatch (-want +got):\n%s", diff)
			}
			rereadObserved, ok := rereadDifference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference output snapshot observed type = %T, want TextMeaning", rereadDifference.Observed())
			}
			if diff := cmp.Diff(" release ", rereadObserved.Text()); diff != "" {
				t.Errorf("difference output snapshot observed payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.NotSatisfied, result.Determination()); diff != "" {
				t.Errorf("difference determination after mutation mismatch (-want +got):\n%s", diff)
			}
			rootDate := must(plan.ParseTargetDate("2026-09-01"))
			rootExpected := must(plan.New("Root", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Accept"))}, []plan.Task{must(plan.NewTask("Task"))}, &rootDate))
			rootObservation := planrepresentation.AbsentObservation()
			rootController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return rootObservation, nil
			})))
			rootResult := must(rootController.Reconcile(context.Background(), rootExpected))
			rootDifferences := rootResult.Differences()
			if diff := cmp.Diff(1, len(rootDifferences)); diff != "" {
				t.Fatalf("root difference count mismatch (-want +got):\n%s", diff)
			}
			rootDifference, ok := rootDifferences[0].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("root difference type = %T, want ExpectedAbsentDifference", rootDifferences[0])
			}
			rootMeaning, ok := rootDifference.Expected().(planrepresentation.PlanMeaning)
			if !ok {
				t.Fatalf("root expected meaning type = %T, want PlanMeaning", rootDifference.Expected())
			}
			rootPlan := rootMeaning.Plan()
			if diff := cmp.Diff("Root", rootPlan.Name()); diff != "" {
				t.Errorf("root name snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("Goal", rootPlan.Goal().Text()); diff != "" {
				t.Errorf("root Goal snapshot mismatch (-want +got):\n%s", diff)
			}
			conditions := rootPlan.AcceptanceConditions()
			conditions[0] = plan.AcceptanceCondition{}
			if diff := cmp.Diff("Accept", rootPlan.AcceptanceConditions()[0].Statement()); diff != "" {
				t.Errorf("root Plan payload snapshot mismatch (-want +got):\n%s", diff)
			}
			tasks := rootPlan.Tasks()
			tasks[0] = plan.Task{}
			if diff := cmp.Diff("Task", rootPlan.Tasks()[0].Name()); diff != "" {
				t.Errorf("root Task payload snapshot mismatch (-want +got):\n%s", diff)
			}
			target, hasTarget := rootPlan.TargetDate()
			if diff := cmp.Diff(true, hasTarget); diff != "" {
				t.Errorf("root target date presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("2026-09-01", target.String()); diff != "" {
				t.Errorf("root target date snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("Accept", rootPlan.AcceptanceConditions()[0].Statement()); diff != "" {
				t.Errorf("root acceptance accessor snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("Task", rootPlan.Tasks()[0].Name()); diff != "" {
				t.Errorf("root Task accessor snapshot mismatch (-want +got):\n%s", diff)
			}
			unavailableController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return planrepresentation.UnavailableObservation(), nil
			})))
			unavailableResult := must(unavailableController.Reconcile(context.Background(), rootExpected))
			unavailableEvidence := unavailableResult.UnavailableInformation()
			if diff := cmp.Diff(1, len(unavailableEvidence)); diff != "" {
				t.Fatalf("unavailable evidence count mismatch (-want +got):\n%s", diff)
			}
			unavailableEvidence[0] = nil
			rereadUnavailable := unavailableResult.UnavailableInformation()
			if diff := cmp.Diff(1, len(rereadUnavailable)); diff != "" {
				t.Fatalf("unavailable output snapshot count mismatch (-want +got):\n%s", diff)
			}
			if rereadUnavailable[0] == nil {
				t.Fatal("unavailable output snapshot returned nil element")
			}
			if diff := cmp.Diff(planrepresentation.PlanRootLocation, rereadUnavailable[0].Location().Kind()); diff != "" {
				t.Errorf("unavailable output snapshot location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, unavailableResult.Determination()); diff != "" {
				t.Errorf("unavailable determination after mutation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObservationCollectionInputSnapshots(t *testing.T) {
	tests := []struct{ name string }{{name: "complete and incomplete collection input snapshots"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completeConditions := []string{"A", "B"}
			incompleteConditions := []string{"C"}
			completeTasks := []string{"Build", "Test"}
			incompleteTasks := []string{"Deploy"}
			completeConditionObservation := planrepresentation.CompleteAcceptanceConditions(completeConditions)
			incompleteConditionObservation := planrepresentation.IncompleteAcceptanceConditions(incompleteConditions)
			completeTaskObservation := planrepresentation.CompleteTasks(completeTasks)
			incompleteTaskObservation := planrepresentation.IncompleteTasks(incompleteTasks)
			completeConditions[0] = "changed"
			incompleteConditions[0] = "changed"
			completeTasks[0] = "changed"
			incompleteTasks[0] = "changed"
			expected := must(plan.New("Release", must(plan.NewGoal("Ship")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A")), must(plan.NewAcceptanceCondition("B"))}, []plan.Task{must(plan.NewTask("Build")), must(plan.NewTask("Test"))}, nil))
			completeObservation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), completeConditionObservation, completeTaskObservation, planrepresentation.AbsentTargetDate()))
			completeController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return completeObservation, nil })))
			completeResult := must(completeController.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.Satisfied, completeResult.Determination()); diff != "" {
				t.Errorf("complete snapshot determination mismatch (-want +got):\n%s", diff)
			}
			incompleteObservation := must(planrepresentation.NewPresentObservation(planrepresentation.ClassifyPlanName("Release"), planrepresentation.ClassifyGoal("Ship"), incompleteConditionObservation, incompleteTaskObservation, planrepresentation.AbsentTargetDate()))
			incompleteController := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) { return incompleteObservation, nil })))
			incompleteResult := must(incompleteController.Reconcile(context.Background(), expected))
			if diff := cmp.Diff(planrepresentation.Undecidable, incompleteResult.Determination()); diff != "" {
				t.Errorf("incomplete snapshot determination mismatch (-want +got):\n%s", diff)
			}
			incompleteDifferences := incompleteResult.Differences()
			if diff := cmp.Diff(2, len(incompleteDifferences)); diff != "" {
				t.Fatalf("incomplete snapshot difference count mismatch (-want +got):\n%s", diff)
			}
			incompleteCondition, ok := incompleteDifferences[0].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("incomplete condition difference type = %T, want UnexpectedPresentDifference", incompleteDifferences[0])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, incompleteCondition.Location().Kind()); diff != "" {
				t.Errorf("incomplete condition location mismatch (-want +got):\n%s", diff)
			}
			conditionMember, conditionHasMember := incompleteCondition.Location().Member()
			if diff := cmp.Diff("C", conditionMember); diff != "" {
				t.Errorf("incomplete condition member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, conditionHasMember); diff != "" {
				t.Errorf("incomplete condition member presence mismatch (-want +got):\n%s", diff)
			}
			incompleteTask, ok := incompleteDifferences[1].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("incomplete Task difference type = %T, want UnexpectedPresentDifference", incompleteDifferences[1])
			}
			if diff := cmp.Diff(planrepresentation.TaskMemberLocation, incompleteTask.Location().Kind()); diff != "" {
				t.Errorf("incomplete Task location mismatch (-want +got):\n%s", diff)
			}
			taskMember, taskHasMember := incompleteTask.Location().Member()
			if diff := cmp.Diff("Deploy", taskMember); diff != "" {
				t.Errorf("incomplete Task member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, taskHasMember); diff != "" {
				t.Errorf("incomplete Task member presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(2, len(incompleteResult.UnavailableInformation())); diff != "" {
				t.Errorf("incomplete snapshot unavailable count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileCanonicalDifferenceOrder(t *testing.T) {
	tests := []struct{ name string }{{name: "canonical 13-item difference order"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetDate := must(plan.ParseTargetDate("2026-09-01"))
			expected := must(plan.New(
				"Release",
				must(plan.NewGoal("Ship")),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A")), must(plan.NewAcceptanceCondition("e\u0301"))},
				[]plan.Task{must(plan.NewTask("Build")), must(plan.NewTask("試験"))},
				&targetDate,
			))
			observed := must(planrepresentation.NewPresentObservation(
				planrepresentation.ClassifyPlanName(" release "),
				planrepresentation.ClassifyGoal("Ship!"),
				planrepresentation.CompleteAcceptanceConditions([]string{"a", "é", " ", "a"}),
				planrepresentation.CompleteTasks([]string{" build ", "試験", "bad\n", " build "}),
				planrepresentation.AbsentTargetDate(),
			))
			controller := must(planrepresentation.NewController(planrepresentation.Observer(func(context.Context) (planrepresentation.Observation, error) {
				return observed, nil
			})))
			result := must(controller.Reconcile(context.Background(), expected))
			differences := result.Differences()
			if diff := cmp.Diff(13, len(differences)); diff != "" {
				t.Fatalf("difference count mismatch (-want +got):\n%s", diff)
			}
			nameDifference, ok := differences[0].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference[0] type = %T, want ValueDifferentDifference", differences[0])
			}
			if diff := cmp.Diff(planrepresentation.PlanNameLocation, nameDifference.Location().Kind()); diff != "" {
				t.Errorf("difference[0] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ValueDifferentCategory, nameDifference.Category()); diff != "" {
				t.Errorf("difference[0] category mismatch (-want +got):\n%s", diff)
			}
			nameExpected, ok := nameDifference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[0] expected type = %T, want TextMeaning", nameDifference.Expected())
			}
			nameObserved, ok := nameDifference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[0] observed type = %T, want TextMeaning", nameDifference.Observed())
			}
			if diff := cmp.Diff("Release", nameExpected.Text()); diff != "" {
				t.Errorf("difference[0] expected payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(" release ", nameObserved.Text()); diff != "" {
				t.Errorf("difference[0] observed payload mismatch (-want +got):\n%s", diff)
			}

			goalDifference, ok := differences[1].(planrepresentation.ValueDifferentDifference)
			if !ok {
				t.Fatalf("difference[1] type = %T, want ValueDifferentDifference", differences[1])
			}
			if diff := cmp.Diff(planrepresentation.GoalLocation, goalDifference.Location().Kind()); diff != "" {
				t.Errorf("difference[1] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ValueDifferentCategory, goalDifference.Category()); diff != "" {
				t.Errorf("difference[1] category mismatch (-want +got):\n%s", diff)
			}
			goalExpected, ok := goalDifference.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[1] expected type = %T, want TextMeaning", goalDifference.Expected())
			}
			goalObserved, ok := goalDifference.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[1] observed type = %T, want TextMeaning", goalDifference.Observed())
			}
			if diff := cmp.Diff("Ship", goalExpected.Text()); diff != "" {
				t.Errorf("difference[1] expected payload mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("Ship!", goalObserved.Text()); diff != "" {
				t.Errorf("difference[1] observed payload mismatch (-want +got):\n%s", diff)
			}

			acceptDuplicate, ok := differences[2].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[2] type = %T, want InvalidObservedDifference", differences[2])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, acceptDuplicate.Location().Kind()); diff != "" {
				t.Errorf("difference[2] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, acceptDuplicate.Category()); diff != "" {
				t.Errorf("difference[2] category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.DuplicateAcceptanceCondition, acceptDuplicate.Violation()); diff != "" {
				t.Errorf("difference[2] violation mismatch (-want +got):\n%s", diff)
			}

			acceptInvalid, ok := differences[3].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[3] type = %T, want InvalidObservedDifference", differences[3])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionCollectionLocation, acceptInvalid.Location().Kind()); diff != "" {
				t.Errorf("difference[3] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, acceptInvalid.Category()); diff != "" {
				t.Errorf("difference[3] category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.InvalidText, acceptInvalid.Violation()); diff != "" {
				t.Errorf("difference[3] violation mismatch (-want +got):\n%s", diff)
			}

			acceptAbsentA, ok := differences[4].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[4] type = %T, want ExpectedAbsentDifference", differences[4])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, acceptAbsentA.Location().Kind()); diff != "" {
				t.Errorf("difference[4] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, acceptAbsentA.Category()); diff != "" {
				t.Errorf("difference[4] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := acceptAbsentA.Location().Member()
			if diff := cmp.Diff("A", member); diff != "" {
				t.Errorf("difference[4] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[4] member presence mismatch (-want +got):\n%s", diff)
			}
			acceptAbsentMeaning, ok := acceptAbsentA.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[4] expected type = %T, want TextMeaning", acceptAbsentA.Expected())
			}
			if diff := cmp.Diff("A", acceptAbsentMeaning.Text()); diff != "" {
				t.Errorf("difference[4] expected payload mismatch (-want +got):\n%s", diff)
			}

			acceptUnexpectedA, ok := differences[5].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference[5] type = %T, want UnexpectedPresentDifference", differences[5])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, acceptUnexpectedA.Location().Kind()); diff != "" {
				t.Errorf("difference[5] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.UnexpectedPresentCategory, acceptUnexpectedA.Category()); diff != "" {
				t.Errorf("difference[5] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = acceptUnexpectedA.Location().Member()
			if diff := cmp.Diff("a", member); diff != "" {
				t.Errorf("difference[5] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[5] member presence mismatch (-want +got):\n%s", diff)
			}
			acceptUnexpectedMeaning, ok := acceptUnexpectedA.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[5] observed type = %T, want TextMeaning", acceptUnexpectedA.Observed())
			}
			if diff := cmp.Diff("a", acceptUnexpectedMeaning.Text()); diff != "" {
				t.Errorf("difference[5] observed payload mismatch (-want +got):\n%s", diff)
			}

			acceptAbsentDecomposed, ok := differences[6].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[6] type = %T, want ExpectedAbsentDifference", differences[6])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, acceptAbsentDecomposed.Location().Kind()); diff != "" {
				t.Errorf("difference[6] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, acceptAbsentDecomposed.Category()); diff != "" {
				t.Errorf("difference[6] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = acceptAbsentDecomposed.Location().Member()
			if diff := cmp.Diff("e\u0301", member); diff != "" {
				t.Errorf("difference[6] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[6] member presence mismatch (-want +got):\n%s", diff)
			}
			acceptAbsentDecomposedMeaning, ok := acceptAbsentDecomposed.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[6] expected type = %T, want TextMeaning", acceptAbsentDecomposed.Expected())
			}
			if diff := cmp.Diff("e\u0301", acceptAbsentDecomposedMeaning.Text()); diff != "" {
				t.Errorf("difference[6] expected payload mismatch (-want +got):\n%s", diff)
			}

			acceptUnexpectedComposed, ok := differences[7].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference[7] type = %T, want UnexpectedPresentDifference", differences[7])
			}
			if diff := cmp.Diff(planrepresentation.AcceptanceConditionMemberLocation, acceptUnexpectedComposed.Location().Kind()); diff != "" {
				t.Errorf("difference[7] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.UnexpectedPresentCategory, acceptUnexpectedComposed.Category()); diff != "" {
				t.Errorf("difference[7] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = acceptUnexpectedComposed.Location().Member()
			if diff := cmp.Diff("é", member); diff != "" {
				t.Errorf("difference[7] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[7] member presence mismatch (-want +got):\n%s", diff)
			}
			acceptUnexpectedComposedMeaning, ok := acceptUnexpectedComposed.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[7] observed type = %T, want TextMeaning", acceptUnexpectedComposed.Observed())
			}
			if diff := cmp.Diff("é", acceptUnexpectedComposedMeaning.Text()); diff != "" {
				t.Errorf("difference[7] observed payload mismatch (-want +got):\n%s", diff)
			}

			taskDuplicate, ok := differences[8].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[8] type = %T, want InvalidObservedDifference", differences[8])
			}
			if diff := cmp.Diff(planrepresentation.TaskCollectionLocation, taskDuplicate.Location().Kind()); diff != "" {
				t.Errorf("difference[8] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, taskDuplicate.Category()); diff != "" {
				t.Errorf("difference[8] category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.DuplicateTask, taskDuplicate.Violation()); diff != "" {
				t.Errorf("difference[8] violation mismatch (-want +got):\n%s", diff)
			}

			taskInvalid, ok := differences[9].(planrepresentation.InvalidObservedDifference)
			if !ok {
				t.Fatalf("difference[9] type = %T, want InvalidObservedDifference", differences[9])
			}
			if diff := cmp.Diff(planrepresentation.TaskCollectionLocation, taskInvalid.Location().Kind()); diff != "" {
				t.Errorf("difference[9] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.InvalidObservedCategory, taskInvalid.Category()); diff != "" {
				t.Errorf("difference[9] category mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.MultilineName, taskInvalid.Violation()); diff != "" {
				t.Errorf("difference[9] violation mismatch (-want +got):\n%s", diff)
			}

			taskUnexpected, ok := differences[10].(planrepresentation.UnexpectedPresentDifference)
			if !ok {
				t.Fatalf("difference[10] type = %T, want UnexpectedPresentDifference", differences[10])
			}
			if diff := cmp.Diff(planrepresentation.TaskMemberLocation, taskUnexpected.Location().Kind()); diff != "" {
				t.Errorf("difference[10] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.UnexpectedPresentCategory, taskUnexpected.Category()); diff != "" {
				t.Errorf("difference[10] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = taskUnexpected.Location().Member()
			if diff := cmp.Diff(" build ", member); diff != "" {
				t.Errorf("difference[10] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[10] member presence mismatch (-want +got):\n%s", diff)
			}
			taskUnexpectedMeaning, ok := taskUnexpected.Observed().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[10] observed type = %T, want TextMeaning", taskUnexpected.Observed())
			}
			if diff := cmp.Diff(" build ", taskUnexpectedMeaning.Text()); diff != "" {
				t.Errorf("difference[10] observed payload mismatch (-want +got):\n%s", diff)
			}

			taskAbsent, ok := differences[11].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[11] type = %T, want ExpectedAbsentDifference", differences[11])
			}
			if diff := cmp.Diff(planrepresentation.TaskMemberLocation, taskAbsent.Location().Kind()); diff != "" {
				t.Errorf("difference[11] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, taskAbsent.Category()); diff != "" {
				t.Errorf("difference[11] category mismatch (-want +got):\n%s", diff)
			}
			member, hasMember = taskAbsent.Location().Member()
			if diff := cmp.Diff("Build", member); diff != "" {
				t.Errorf("difference[11] member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasMember); diff != "" {
				t.Errorf("difference[11] member presence mismatch (-want +got):\n%s", diff)
			}
			taskAbsentMeaning, ok := taskAbsent.Expected().(planrepresentation.TextMeaning)
			if !ok {
				t.Fatalf("difference[11] expected type = %T, want TextMeaning", taskAbsent.Expected())
			}
			if diff := cmp.Diff("Build", taskAbsentMeaning.Text()); diff != "" {
				t.Errorf("difference[11] expected payload mismatch (-want +got):\n%s", diff)
			}

			targetAbsent, ok := differences[12].(planrepresentation.ExpectedAbsentDifference)
			if !ok {
				t.Fatalf("difference[12] type = %T, want ExpectedAbsentDifference", differences[12])
			}
			if diff := cmp.Diff(planrepresentation.TargetDateLocation, targetAbsent.Location().Kind()); diff != "" {
				t.Errorf("difference[12] location mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(planrepresentation.ExpectedAbsentCategory, targetAbsent.Category()); diff != "" {
				t.Errorf("difference[12] category mismatch (-want +got):\n%s", diff)
			}
			targetAbsentMeaning, ok := targetAbsent.Expected().(planrepresentation.TargetDateMeaning)
			if !ok {
				t.Fatalf("difference[12] expected type = %T, want TargetDateMeaning", targetAbsent.Expected())
			}
			if diff := cmp.Diff("2026-09-01", targetAbsentMeaning.TargetDate().String()); diff != "" {
				t.Errorf("difference[12] expected payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
