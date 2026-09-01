package githubplan

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

func TestNewMilestoneTarget(t *testing.T) {
	type args struct {
		repository Repository
		number     ResourceNumber
	}
	tests := []struct {
		name    string
		args    args
		want    MilestoneTarget
		wantErr bool
	}{
		{name: "valid target", args: args{repository: Repository{owner: "owner", name: "repo"}, number: ResourceNumber{value: 7}}, want: MilestoneTarget{repository: Repository{owner: "owner", name: "repo"}, number: ResourceNumber{value: 7}, valid: true}},
		{name: "invalid repository", args: args{number: ResourceNumber{value: 7}}, want: MilestoneTarget{}, wantErr: true},
		{name: "invalid number", args: args{repository: Repository{owner: "owner", name: "repo"}}, want: MilestoneTarget{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMilestoneTarget(tt.args.repository, tt.args.number)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(MilestoneTarget{}, Repository{}, ResourceNumber{})); diff != "" {
				t.Errorf("NewMilestoneTarget() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewMilestoneSnapshotObserver(t *testing.T) {
	type args struct {
		client *http.Client
		target MilestoneTarget
	}
	tests := []struct {
		name    string
		args    args
		want    plansnapshot.Observer
		wantErr bool
	}{
		{name: "nil client", args: args{target: MilestoneTarget{}}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMilestoneSnapshotObserver(tt.args.client, tt.args.target)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plansnapshot.Snapshot{}, plansnapshot.ProgressEvidence{}, plansnapshot.TaskProgress{}), cmp.Comparer(func(left, right plan.Plan) bool {
				return left.IsValid() == right.IsValid() && (!left.IsValid() || left.Equal(right))
			})); diff != "" {
				t.Errorf("NewMilestoneSnapshotObserver() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_scheme_snapshot(t *testing.T) {
	type args struct {
		facts githubFactSet
	}
	tests := []struct {
		name    string
		s       scheme
		args    args
		want    plansnapshot.Snapshot
		wantErr bool
	}{
		{name: "invalid facts", s: milestoneScheme, args: args{facts: githubFactSet{}}, want: plansnapshot.Snapshot{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.snapshot(tt.args.facts)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plansnapshot.Snapshot{}, plansnapshot.ProgressEvidence{}, plansnapshot.TaskProgress{}), cmp.Comparer(func(left, right plansnapshot.Snapshot) bool {
				leftPlan, leftHas := left.CurrentPlan()
				rightPlan, rightHas := right.CurrentPlan()
				return leftHas == rightHas && (!leftHas || leftPlan.Equal(rightPlan))
			})); diff != "" {
				t.Errorf("scheme.snapshot() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_progressState(t *testing.T) {
	type args struct {
		state nativeState
	}
	tests := []struct {
		name string
		args args
		want plansnapshot.ProgressState
	}{
		{name: "open", args: args{state: nativeStateOpen}, want: plansnapshot.Open},
		{name: "closed", args: args{state: nativeStateClosed}, want: plansnapshot.Closed},
		{name: "unsupported", args: args{state: nativeState(99)}, want: plansnapshot.Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, progressState(tt.args.state)); diff != "" {
				t.Errorf("progressState() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_scheme_currentPlan(t *testing.T) {
	type args struct {
		facts githubFactSet
	}
	tests := []struct {
		name  string
		s     scheme
		args  args
		want  plan.Plan
		want1 bool
	}{
		{name: "invalid facts", s: milestoneScheme, args: args{facts: githubFactSet{}}, want: plan.Plan{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.s.currentPlan(tt.args.facts)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.Plan{}, plan.Goal{}, plan.AcceptanceCondition{}, plan.Task{}, plan.TargetDate{}), cmp.Comparer(func(left, right plan.Plan) bool {
				return left.IsValid() == right.IsValid() && (!left.IsValid() || left.Equal(right))
			})); diff != "" {
				t.Errorf("scheme.currentPlan() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("scheme.currentPlan() presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
