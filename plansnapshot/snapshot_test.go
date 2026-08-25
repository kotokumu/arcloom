package plansnapshot_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plansnapshot"
)

func TestSnapshotConstructorsAndEligibility(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	taskA := must(plan.NewTask("A"))
	taskB := must(plan.NewTask("B"))
	current := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, []plan.Task{taskA, taskB}, nil))
	progress := must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open)), must(plansnapshot.NewTaskProgress("B", plansnapshot.Unknown))}))
	incomplete := must(plansnapshot.IncompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))
	tests := []struct {
		name            string
		make            func() (plansnapshot.Snapshot, error)
		wantErr         bool
		wantPlanPresent bool
		wantPlanEqual   bool
		wantProgress    bool
		wantComplete    bool
	}{
		{name: "current plan with exact complete progress", make: func() (plansnapshot.Snapshot, error) { return plansnapshot.New(current, progress) }, wantPlanPresent: true, wantPlanEqual: true, wantProgress: true, wantComplete: true},
		{name: "without current complete progress", make: func() (plansnapshot.Snapshot, error) { return plansnapshot.WithoutCurrent(progress) }, wantProgress: true, wantComplete: true},
		{name: "without current incomplete progress", make: func() (plansnapshot.Snapshot, error) { return plansnapshot.WithoutCurrent(incomplete) }, wantProgress: true, wantComplete: false},
		{name: "incomplete progress cannot accompany current", make: func() (plansnapshot.Snapshot, error) { return plansnapshot.New(current, incomplete) }, wantErr: true},
		{name: "task correspondence is ordered", make: func() (plansnapshot.Snapshot, error) {
			first := must(plansnapshot.NewTaskProgress("B", plansnapshot.Open))
			second := must(plansnapshot.NewTaskProgress("A", plansnapshot.Unknown))
			swapped := must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{first, second}))
			return plansnapshot.New(current, swapped)
		}, wantErr: true},
		{name: "task correspondence is exact", make: func() (plansnapshot.Snapshot, error) {
			extra := must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))
			return plansnapshot.New(current, extra)
		}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.make()
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			gotPlan, hasPlan := got.CurrentPlan()
			gotProgress, hasProgress := got.Progress()
			if diff := cmp.Diff(tt.wantPlanPresent, hasPlan); diff != "" {
				t.Errorf("plan presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantPlanEqual, gotPlan.Equal(current)); diff != "" {
				t.Errorf("plan equality mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantProgress, hasProgress); diff != "" {
				t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantComplete, gotProgress.MembershipComplete()); diff != "" {
				t.Errorf("progress completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnapshotRejectsInvalidInputs(t *testing.T) {
	progress := must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))
	tests := []struct {
		name string
		make func() (plansnapshot.Snapshot, error)
	}{
		{name: "invalid current plan", make: func() (plansnapshot.Snapshot, error) { return plansnapshot.New(plan.Plan{}, progress) }},
		{name: "invalid progress", make: func() (plansnapshot.Snapshot, error) {
			return plansnapshot.WithoutCurrent(plansnapshot.ProgressEvidence{})
		}},
		{name: "zero progress for current Plan", make: func() (plansnapshot.Snapshot, error) {
			return plansnapshot.New(must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), plansnapshot.ProgressEvidence{})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.make()
			if diff := cmp.Diff(true, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			gotPlan, gotHasPlan := got.CurrentPlan()
			gotProgress, gotHasProgress := got.Progress()
			if diff := cmp.Diff(false, gotHasPlan); diff != "" {
				t.Errorf("invalid snapshot plan presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, gotHasProgress); diff != "" {
				t.Errorf("invalid snapshot progress presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, gotPlan.IsValid()); diff != "" {
				t.Errorf("invalid snapshot plan validity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, gotProgress.MembershipComplete()); diff != "" {
				t.Errorf("invalid snapshot progress completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnapshotCopiesProgress(t *testing.T) {
	progress := must(plansnapshot.CompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))
	snapshot := must(plansnapshot.WithoutCurrent(progress))
	got, ok := snapshot.Progress()
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Errorf("snapshot progress presence mismatch (-want +got):\n%s", diff)
	}
	tasks := got.Tasks()
	tasks[0] = must(plansnapshot.NewTaskProgress("changed", plansnapshot.Closed))
	gotAgain, ok := snapshot.Progress()
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Errorf("second snapshot progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("A", gotAgain.Tasks()[0].Name()); diff != "" {
		t.Errorf("snapshot progress changed through accessor (-want +got):\n%s", diff)
	}
}

func TestSnapshotPreservesCompleteDuplicateProgressWithoutCurrent(t *testing.T) {
	first := must(plansnapshot.NewTaskProgress("same", plansnapshot.Open))
	second := must(plansnapshot.NewTaskProgress("same", plansnapshot.Closed))
	progress := must(plansnapshot.CompleteProgress(plansnapshot.Unknown, []plansnapshot.TaskProgress{first, second}))
	snapshot := must(plansnapshot.WithoutCurrent(progress))
	got, ok := snapshot.Progress()
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
	}
	gotTasks := got.Tasks()
	gotNames := []string{gotTasks[0].Name(), gotTasks[1].Name()}
	gotStates := []plansnapshot.ProgressState{gotTasks[0].State(), gotTasks[1].State()}
	if diff := cmp.Diff([]string{"same", "same"}, gotNames); diff != "" {
		t.Errorf("duplicate progress names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Open, plansnapshot.Closed}, gotStates); diff != "" {
		t.Errorf("duplicate progress states mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.Unknown, got.OverallState()); diff != "" {
		t.Errorf("duplicate overall state mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, got.MembershipComplete()); diff != "" {
		t.Errorf("duplicate completeness mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := snapshot.CurrentPlan()
	if diff := cmp.Diff(false, hasCurrent); diff != "" {
		t.Errorf("duplicate current Plan mismatch (-want +got):\n%s", diff)
	}
}

func TestSnapshotPublicProgressSemantics(t *testing.T) {
	type expected struct {
		hasPlan  bool
		overall  plansnapshot.ProgressState
		tasks    []plansnapshot.TaskProgress
		complete bool
	}
	validPlan := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("A"))}, nil))
	tests := []struct {
		name string
		make func() (plansnapshot.Snapshot, error)
		want expected
	}{
		{name: "current", make: func() (plansnapshot.Snapshot, error) {
			return plansnapshot.New(validPlan, must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Unknown))})))
		}, want: expected{hasPlan: true, overall: plansnapshot.Closed, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Unknown))}, complete: true}},
		{name: "no current complete", make: func() (plansnapshot.Snapshot, error) {
			return plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("observed", plansnapshot.Closed))})))
		}, want: expected{overall: plansnapshot.Open, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("observed", plansnapshot.Closed))}, complete: true}},
		{name: "no current incomplete", make: func() (plansnapshot.Snapshot, error) {
			return plansnapshot.WithoutCurrent(must(plansnapshot.IncompleteProgress(plansnapshot.Unknown, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("partial", plansnapshot.Unknown))})))
		}, want: expected{overall: plansnapshot.Unknown, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("partial", plansnapshot.Unknown))}, complete: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.make()
			if diff := cmp.Diff(nil, err); diff != "" {
				t.Errorf("snapshot error mismatch (-want +got):\n%s", diff)
			}
			_, hasPlan := got.CurrentPlan()
			if diff := cmp.Diff(tt.want.hasPlan, hasPlan); diff != "" {
				t.Errorf("current Plan presence mismatch (-want +got):\n%s", diff)
			}
			progress, hasProgress := got.Progress()
			if diff := cmp.Diff(true, hasProgress); diff != "" {
				t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.overall, progress.OverallState()); diff != "" {
				t.Errorf("overall state mismatch (-want +got):\n%s", diff)
			}
			wantTasks := tt.want.tasks
			gotTasks := progress.Tasks()
			wantNames := make([]string, len(wantTasks))
			gotNames := make([]string, len(gotTasks))
			wantStates := make([]plansnapshot.ProgressState, len(wantTasks))
			gotStates := make([]plansnapshot.ProgressState, len(gotTasks))
			for i := range wantTasks {
				wantNames[i] = wantTasks[i].Name()
				wantStates[i] = wantTasks[i].State()
			}
			for i := range gotTasks {
				gotNames[i] = gotTasks[i].Name()
				gotStates[i] = gotTasks[i].State()
			}
			if diff := cmp.Diff(wantNames, gotNames); diff != "" {
				t.Errorf("task names mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantStates, gotStates); diff != "" {
				t.Errorf("task states mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.complete, progress.MembershipComplete()); diff != "" {
				t.Errorf("completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSnapshotNewDefensivelyCopiesProgress(t *testing.T) {
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("A"))}, nil))
	progress := must(plansnapshot.CompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))
	snapshot := must(plansnapshot.New(current, progress))
	got, ok := snapshot.Progress()
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
	}
	tasks := got.Tasks()
	tasks[0] = must(plansnapshot.NewTaskProgress("changed", plansnapshot.Closed))
	again, ok := snapshot.Progress()
	if diff := cmp.Diff(true, ok); diff != "" {
		t.Errorf("second progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("A", again.Tasks()[0].Name()); diff != "" {
		t.Errorf("New snapshot copy mismatch (-want +got):\n%s", diff)
	}
}
