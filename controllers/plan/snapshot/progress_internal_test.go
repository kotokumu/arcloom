package plansnapshot

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTaskProgress(t *testing.T) {
	type args struct {
		name  string
		state ProgressState
	}
	tests := []struct {
		name    string
		args    args
		want    TaskProgress
		wantErr bool
	}{
		{name: "open", args: args{name: "Build", state: Open}, want: TaskProgress{name: "Build", state: Open, valid: true}},
		{name: "unknown", args: args{name: "Build", state: Unknown}, want: TaskProgress{name: "Build", state: Unknown, valid: true}},
		{name: "blank name", args: args{name: " ", state: Open}, want: TaskProgress{}, wantErr: true},
		{name: "zero state", args: args{name: "Build", state: ProgressState(0)}, want: TaskProgress{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTaskProgress(tt.args.name, tt.args.state)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("NewTaskProgress() error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(TaskProgress{})); diff != "" {
				t.Errorf("NewTaskProgress() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompleteProgress(t *testing.T) {
	type args struct {
		overall ProgressState
		tasks   []TaskProgress
	}
	tests := []struct {
		name    string
		args    args
		want    ProgressEvidence
		wantErr bool
	}{
		{name: "open", args: args{overall: Open, tasks: []TaskProgress{{name: "Build", state: Open, valid: true}}}, want: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "Build", state: Open, valid: true}}, complete: true, valid: true}},
		{name: "unknown", args: args{overall: Unknown, tasks: nil}, want: ProgressEvidence{overall: Unknown, complete: true, valid: true}},
		{name: "zero overall", args: args{overall: ProgressState(0), tasks: nil}, want: ProgressEvidence{}, wantErr: true},
		{name: "zero task", args: args{overall: Open, tasks: []TaskProgress{{}}}, want: ProgressEvidence{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompleteProgress(tt.args.overall, tt.args.tasks)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("CompleteProgress() error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ProgressEvidence{}, TaskProgress{})); diff != "" {
				t.Errorf("CompleteProgress() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIncompleteProgress(t *testing.T) {
	type args struct {
		overall ProgressState
		tasks   []TaskProgress
	}
	tests := []struct {
		name    string
		args    args
		want    ProgressEvidence
		wantErr bool
	}{
		{name: "open", args: args{overall: Open, tasks: []TaskProgress{{name: "Build", state: Open, valid: true}}}, want: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "Build", state: Open, valid: true}}, complete: false, valid: true}},
		{name: "unknown", args: args{overall: Unknown, tasks: nil}, want: ProgressEvidence{overall: Unknown, complete: false, valid: true}},
		{name: "zero overall", args: args{overall: ProgressState(0), tasks: nil}, want: ProgressEvidence{}, wantErr: true},
		{name: "zero task", args: args{overall: Open, tasks: []TaskProgress{{}}}, want: ProgressEvidence{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncompleteProgress(tt.args.overall, tt.args.tasks)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("IncompleteProgress() error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ProgressEvidence{}, TaskProgress{})); diff != "" {
				t.Errorf("IncompleteProgress() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskProgress_Name(t *testing.T) {
	type fields struct {
		name  string
		state ProgressState
		valid bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "named", fields: fields{name: "Task", state: Open, valid: true}, want: "Task"},
		{name: "zero", fields: fields{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := TaskProgress{name: tt.fields.name, state: tt.fields.state, valid: tt.fields.valid}
			if diff := cmp.Diff(tt.want, p.Name()); diff != "" {
				t.Errorf("TaskProgress.Name() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskProgress_State(t *testing.T) {
	type fields struct {
		name  string
		state ProgressState
		valid bool
	}
	tests := []struct {
		name   string
		fields fields
		want   ProgressState
	}{
		{name: "open", fields: fields{name: "Task", state: Open, valid: true}, want: Open},
		{name: "unknown", fields: fields{name: "Task", state: Unknown, valid: true}, want: Unknown},
		{name: "zero", fields: fields{}, want: ProgressState(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := TaskProgress{name: tt.fields.name, state: tt.fields.state, valid: tt.fields.valid}
			if diff := cmp.Diff(tt.want, p.State()); diff != "" {
				t.Errorf("TaskProgress.State() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_newProgress(t *testing.T) {
	type args struct {
		overall  ProgressState
		tasks    []TaskProgress
		complete bool
	}
	tests := []struct {
		name    string
		args    args
		want    ProgressEvidence
		wantErr bool
	}{
		{name: "complete", args: args{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, complete: true}, want: ProgressEvidence{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, complete: true, valid: true}},
		{name: "incomplete", args: args{overall: Unknown, tasks: nil, complete: false}, want: ProgressEvidence{overall: Unknown, complete: false, valid: true}},
		{name: "invalid overall", args: args{overall: ProgressState(0)}, wantErr: true},
		{name: "invalid member", args: args{overall: Open, tasks: []TaskProgress{{name: "A", state: Open}}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newProgress(tt.args.overall, tt.args.tasks, tt.args.complete)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("newProgress error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ProgressEvidence{}, TaskProgress{})); diff != "" {
				t.Errorf("newProgress mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_validProgressState(t *testing.T) {
	type args struct{ state ProgressState }
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "open", args: args{state: Open}, want: true},
		{name: "closed", args: args{state: Closed}, want: true},
		{name: "unknown", args: args{state: Unknown}, want: true},
		{name: "zero", args: args{state: ProgressState(0)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, validProgressState(tt.args.state)); diff != "" {
				t.Errorf("validProgressState mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_cloneTasks(t *testing.T) {
	type args struct{ tasks []TaskProgress }
	tests := []struct {
		name string
		args args
		want []TaskProgress
	}{
		{name: "tasks", args: args{tasks: []TaskProgress{{name: "A", state: Open, valid: true}}}, want: []TaskProgress{{name: "A", state: Open, valid: true}}},
		{name: "nil", args: args{tasks: nil}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, cloneTasks(tt.args.tasks), cmp.AllowUnexported(TaskProgress{})); diff != "" {
				t.Errorf("cloneTasks mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProgressEvidence_OverallState(t *testing.T) {
	type fields struct {
		overall  ProgressState
		tasks    []TaskProgress
		complete bool
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   ProgressState
	}{
		{name: "closed", fields: fields{overall: Closed, valid: true}, want: Closed},
		{name: "zero", fields: fields{}, want: ProgressState(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ProgressEvidence{overall: tt.fields.overall, tasks: tt.fields.tasks, complete: tt.fields.complete, valid: tt.fields.valid}
			if diff := cmp.Diff(tt.want, p.OverallState()); diff != "" {
				t.Errorf("ProgressEvidence.OverallState mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProgressEvidence_Tasks(t *testing.T) {
	type fields struct {
		overall  ProgressState
		tasks    []TaskProgress
		complete bool
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   []TaskProgress
	}{
		{name: "tasks", fields: fields{overall: Open, tasks: []TaskProgress{{name: "A", state: Open, valid: true}}, valid: true}, want: []TaskProgress{{name: "A", state: Open, valid: true}}},
		{name: "zero", fields: fields{}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ProgressEvidence{overall: tt.fields.overall, tasks: tt.fields.tasks, complete: tt.fields.complete, valid: tt.fields.valid}
			if diff := cmp.Diff(tt.want, p.Tasks(), cmp.AllowUnexported(TaskProgress{})); diff != "" {
				t.Errorf("ProgressEvidence.Tasks mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProgressEvidence_MembershipComplete(t *testing.T) {
	type fields struct {
		overall  ProgressState
		tasks    []TaskProgress
		complete bool
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "complete", fields: fields{overall: Open, complete: true, valid: true}, want: true},
		{name: "incomplete", fields: fields{overall: Open, valid: true}, want: false},
		{name: "zero", fields: fields{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, ProgressEvidence{overall: tt.fields.overall, tasks: tt.fields.tasks, complete: tt.fields.complete, valid: tt.fields.valid}.MembershipComplete()); diff != "" {
				t.Errorf("ProgressEvidence.MembershipComplete mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
