package plan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlan_Equal(t *testing.T) {
	type fields struct {
		name       string
		goal       Goal
		conditions []AcceptanceCondition
		tasks      []Task
		targetDate *TargetDate
		valid      bool
	}
	type args struct {
		other Plan
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "identical", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "task"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "task"}}, valid: true}}, want: true},
		{name: "name differs", fields: fields{name: "A", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}, args: args{other: Plan{name: "B", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}}, want: false},
		{name: "goal differs", fields: fields{name: "Plan", goal: Goal{text: "A"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "B"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}}, want: false},
		{name: "condition value differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "A"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "B"}}, valid: true}}, want: false},
		{name: "condition order differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "A"}, {statement: "B"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "B"}, {statement: "A"}}, valid: true}}, want: false},
		{name: "condition count differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "A"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "A"}, {statement: "B"}}, valid: true}}, want: false},
		{name: "task value differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "A"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "B"}}, valid: true}}, want: false},
		{name: "task order differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "A"}, {name: "B"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "B"}, {name: "A"}}, valid: true}}, want: false},
		{name: "task count differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, tasks: []Task{{name: "A"}}, valid: true}}, want: false},
		{name: "target date presence differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, targetDate: &TargetDate{text: "2028-01-01"}, valid: true}}, want: false},
		{name: "target date value differs", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, targetDate: &TargetDate{text: "2028-01-01"}, valid: true}, args: args{other: Plan{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, targetDate: &TargetDate{text: "2028-01-02"}, valid: true}}, want: false},
		{name: "valid versus invalid", fields: fields{name: "Plan", goal: Goal{text: "goal"}, conditions: []AcceptanceCondition{{statement: "accept"}}, valid: true}, args: args{other: Plan{}}, want: false},
		{name: "invalid versus invalid", fields: fields{}, args: args{other: Plan{}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Plan{
				name:       tt.fields.name,
				goal:       tt.fields.goal,
				conditions: tt.fields.conditions,
				tasks:      tt.fields.tasks,
				targetDate: tt.fields.targetDate,
				valid:      tt.fields.valid,
			}
			if diff := cmp.Diff(tt.want, p.Equal(tt.args.other)); diff != "" {
				t.Errorf("Plan.Equal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
