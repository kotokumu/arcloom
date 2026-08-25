package plan_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
)

func TestPlanEqualSymmetry(t *testing.T) {
	tests := []struct {
		name  string
		left  plan.Plan
		right plan.Plan
		want  bool
	}{
		{name: "identical values", left: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)), right: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)), want: true},
		{name: "different names", left: must(plan.New("Plan A", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)), right: must(plan.New("Plan B", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.left.Equal(tt.right)); diff != "" {
				t.Errorf("left-to-right mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, tt.right.Equal(tt.left)); diff != "" {
				t.Errorf("right-to-left mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
