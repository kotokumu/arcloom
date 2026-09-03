package plansnapshot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
)

func TestNew(t *testing.T) {
	type args struct {
		current  plan.Plan
		progress ProgressEvidence
	}
	tests := []struct {
		name    string
		args    args
		want    Snapshot
		wantErr bool
	}{
		{name: "valid current", args: args{current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), progress: must(CompleteProgress(Open, nil))}, want: Snapshot{current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), hasCurrent: true, progress: must(CompleteProgress(Open, nil)), hasProgress: true, valid: true}},
		{name: "zero current", args: args{current: plan.Plan{}, progress: must(CompleteProgress(Open, nil))}, want: Snapshot{}, wantErr: true},
		{name: "zero progress", args: args{current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), progress: ProgressEvidence{}}, want: Snapshot{}, wantErr: true},
		{name: "incomplete progress", args: args{current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), progress: must(IncompleteProgress(Open, nil))}, want: Snapshot{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.current, tt.args.progress)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("New error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Snapshot{}, ProgressEvidence{}, TaskProgress{}), cmp.Comparer(func(left, right plan.Plan) bool { return (!left.IsValid() && !right.IsValid()) || left.Equal(right) })); diff != "" {
				t.Errorf("New mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWithoutCurrent(t *testing.T) {
	type args struct{ progress ProgressEvidence }
	tests := []struct {
		name    string
		args    args
		want    Snapshot
		wantErr bool
	}{
		{name: "complete duplicate members", args: args{progress: must(CompleteProgress(Closed, []TaskProgress{{name: "A", state: Open, valid: true}, {name: "A", state: Closed, valid: true}}))}, want: Snapshot{progress: must(CompleteProgress(Closed, []TaskProgress{{name: "A", state: Open, valid: true}, {name: "A", state: Closed, valid: true}})), hasProgress: true, valid: true}},
		{name: "incomplete members", args: args{progress: must(IncompleteProgress(Unknown, []TaskProgress{{name: "A", state: Unknown, valid: true}}))}, want: Snapshot{progress: must(IncompleteProgress(Unknown, []TaskProgress{{name: "A", state: Unknown, valid: true}})), hasProgress: true, valid: true}},
		{name: "zero progress", args: args{progress: ProgressEvidence{}}, want: Snapshot{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WithoutCurrent(tt.args.progress)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("WithoutCurrent error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Snapshot{}, ProgressEvidence{}, TaskProgress{}), cmp.Comparer(func(left, right plan.Plan) bool { return (!left.IsValid() && !right.IsValid()) || left.Equal(right) })); diff != "" {
				t.Errorf("WithoutCurrent mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_cloneProgress(t *testing.T) {
	type args struct{ progress ProgressEvidence }
	tests := []struct {
		name string
		args args
		want ProgressEvidence
	}{
		{name: "progress", args: args{progress: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, valid: true}}, want: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, valid: true}},
		{name: "zero", args: args{progress: ProgressEvidence{}}, want: ProgressEvidence{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, cloneProgress(tt.args.progress), cmp.AllowUnexported(ProgressEvidence{}, TaskProgress{})); diff != "" {
				t.Errorf("cloneProgress mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnapshot_CurrentPlan(t *testing.T) {
	type fields struct {
		current     plan.Plan
		hasCurrent  bool
		progress    ProgressEvidence
		hasProgress bool
		valid       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   plan.Plan
		want1  bool
	}{
		{name: "current", fields: fields{current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), hasCurrent: true, valid: true}, want: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), want1: true},
		{name: "zero", fields: fields{}, want: plan.Plan{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{current: tt.fields.current, hasCurrent: tt.fields.hasCurrent, progress: tt.fields.progress, hasProgress: tt.fields.hasProgress, valid: tt.fields.valid}
			got, got1 := s.CurrentPlan()
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(func(left, right plan.Plan) bool { return (!left.IsValid() && !right.IsValid()) || left.Equal(right) })); diff != "" {
				t.Errorf("Snapshot.CurrentPlan value mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("Snapshot.CurrentPlan presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnapshot_Progress(t *testing.T) {
	type fields struct {
		current     plan.Plan
		hasCurrent  bool
		progress    ProgressEvidence
		hasProgress bool
		valid       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   ProgressEvidence
		want1  bool
	}{
		{name: "progress", fields: fields{progress: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, valid: true}, hasProgress: true, valid: true}, want: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, valid: true}, want1: true},
		{name: "zero", fields: fields{}, want: ProgressEvidence{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{current: tt.fields.current, hasCurrent: tt.fields.hasCurrent, progress: tt.fields.progress, hasProgress: tt.fields.hasProgress, valid: tt.fields.valid}
			got, got1 := s.Progress()
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ProgressEvidence{}, TaskProgress{})); diff != "" {
				t.Errorf("Snapshot.Progress value mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("Snapshot.Progress presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
