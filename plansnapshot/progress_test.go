package plansnapshot_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plansnapshot"
)

func TestNewTaskProgress(t *testing.T) {
	type args struct {
		name  string
		state plansnapshot.ProgressState
	}
	tests := []struct {
		name    string
		args    args
		want    plansnapshot.TaskProgress
		wantErr bool
	}{
		{name: "open", args: args{name: "Build", state: plansnapshot.Open}, want: must(plansnapshot.NewTaskProgress("Build", plansnapshot.Open))},
		{name: "closed", args: args{name: "Build", state: plansnapshot.Closed}, want: must(plansnapshot.NewTaskProgress("Build", plansnapshot.Closed))},
		{name: "unknown", args: args{name: "Build", state: plansnapshot.Unknown}, want: must(plansnapshot.NewTaskProgress("Build", plansnapshot.Unknown))},
		{name: "blank name", args: args{name: " ", state: plansnapshot.Open}, wantErr: true},
		{name: "multiline name", args: args{name: "Build\nnext", state: plansnapshot.Open}, wantErr: true},
		{name: "invalid state", args: args{name: "Build", state: plansnapshot.ProgressState(99)}, wantErr: true},
		{name: "zero state", args: args{name: "Build", state: plansnapshot.ProgressState(0)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plansnapshot.NewTaskProgress(tt.args.name, tt.args.state)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("NewTaskProgress() error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Name(), got.Name()); diff != "" {
				t.Errorf("name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.State(), got.State()); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompleteProgress(t *testing.T) {
	type args struct {
		overall plansnapshot.ProgressState
		tasks   []plansnapshot.TaskProgress
	}
	tests := []struct {
		name    string
		args    args
		want    plansnapshot.ProgressEvidence
		wantErr bool
	}{
		{name: "unknown overall", args: args{overall: plansnapshot.Unknown, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}}, want: must(plansnapshot.CompleteProgress(plansnapshot.Unknown, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))},
		{name: "empty tasks", args: args{overall: plansnapshot.Open, tasks: nil}, want: must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))},
		{name: "duplicate names preserved", args: args{overall: plansnapshot.Closed, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("same", plansnapshot.Open)), must(plansnapshot.NewTaskProgress("same", plansnapshot.Closed))}}, want: must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("same", plansnapshot.Open)), must(plansnapshot.NewTaskProgress("same", plansnapshot.Closed))}))},
		{name: "invalid overall", args: args{overall: plansnapshot.ProgressState(0)}, wantErr: true},
		{name: "zero task", args: args{overall: plansnapshot.Open, tasks: []plansnapshot.TaskProgress{{}}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plansnapshot.CompleteProgress(tt.args.overall, tt.args.tasks)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.OverallState(), got.OverallState()); diff != "" {
				t.Errorf("overall state mismatch (-want +got):\n%s", diff)
			}
			wantTasks := tt.want.Tasks()
			gotTasks := got.Tasks()
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
			if diff := cmp.Diff(tt.want.MembershipComplete(), got.MembershipComplete()); diff != "" {
				t.Errorf("completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIncompleteProgress(t *testing.T) {
	type args struct {
		overall plansnapshot.ProgressState
		tasks   []plansnapshot.TaskProgress
	}
	tests := []struct {
		name    string
		args    args
		want    plansnapshot.ProgressEvidence
		wantErr bool
	}{
		{name: "open overall", args: args{overall: plansnapshot.Open, tasks: []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}}, want: must(plansnapshot.IncompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("A", plansnapshot.Open))}))},
		{name: "unknown overall", args: args{overall: plansnapshot.Unknown, tasks: nil}, want: must(plansnapshot.IncompleteProgress(plansnapshot.Unknown, nil))},
		{name: "invalid overall", args: args{overall: plansnapshot.ProgressState(99)}, wantErr: true},
		{name: "zero overall", args: args{overall: plansnapshot.ProgressState(0)}, wantErr: true},
		{name: "zero task", args: args{overall: plansnapshot.Open, tasks: []plansnapshot.TaskProgress{{}}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plansnapshot.IncompleteProgress(tt.args.overall, tt.args.tasks)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.OverallState(), got.OverallState()); diff != "" {
				t.Errorf("overall state mismatch (-want +got):\n%s", diff)
			}
			wantTasks := tt.want.Tasks()
			gotTasks := got.Tasks()
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
			if diff := cmp.Diff(tt.want.MembershipComplete(), got.MembershipComplete()); diff != "" {
				t.Errorf("completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProgressEvidencePreservesDuplicateNamesAndCopies(t *testing.T) {
	first := must(plansnapshot.NewTaskProgress("same", plansnapshot.Open))
	second := must(plansnapshot.NewTaskProgress("same", plansnapshot.Closed))
	input := []plansnapshot.TaskProgress{first, second}
	evidence := must(plansnapshot.CompleteProgress(plansnapshot.Unknown, input))
	input[0] = must(plansnapshot.NewTaskProgress("changed", plansnapshot.Unknown))
	got := evidence.Tasks()
	gotNames := []string{got[0].Name(), got[1].Name()}
	gotStates := []plansnapshot.ProgressState{got[0].State(), got[1].State()}
	if diff := cmp.Diff([]string{"same", "same"}, gotNames); diff != "" {
		t.Errorf("input copy names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Open, plansnapshot.Closed}, gotStates); diff != "" {
		t.Errorf("input copy states mismatch (-want +got):\n%s", diff)
	}
	got[1] = first
	gotAgain := evidence.Tasks()
	gotAgainNames := []string{gotAgain[0].Name(), gotAgain[1].Name()}
	gotAgainStates := []plansnapshot.ProgressState{gotAgain[0].State(), gotAgain[1].State()}
	if diff := cmp.Diff([]string{"same", "same"}, gotAgainNames); diff != "" {
		t.Errorf("output copy names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Open, plansnapshot.Closed}, gotAgainStates); diff != "" {
		t.Errorf("output copy states mismatch (-want +got):\n%s", diff)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
