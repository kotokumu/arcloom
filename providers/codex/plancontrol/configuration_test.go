package codexplancontrol

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewModel(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    Model
		wantErr bool
	}{
		{name: "supported model identifier", args: args{value: "gpt-5.6-sol"}, want: Model{value: "gpt-5.6-sol", valid: true}},
		{name: "empty identifier", args: args{value: ""}, wantErr: true},
		{name: "blank identifier", args: args{value: " \t"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewModel(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewModel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !cmp.Equal(tt.want, got, cmp.AllowUnexported(Model{})) {
				t.Errorf("NewModel() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got, cmp.AllowUnexported(Model{})))
			}
		})
	}
}

func TestNewReasoningEffort(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    ReasoningEffort
		wantErr bool
	}{
		{name: "low", args: args{value: "low"}, want: ReasoningEffort{value: "low", valid: true}},
		{name: "medium", args: args{value: "medium"}, want: ReasoningEffort{value: "medium", valid: true}},
		{name: "high", args: args{value: "high"}, want: ReasoningEffort{value: "high", valid: true}},
		{name: "xhigh", args: args{value: "xhigh"}, want: ReasoningEffort{value: "xhigh", valid: true}},
		{name: "max", args: args{value: "max"}, want: ReasoningEffort{value: "max", valid: true}},
		{name: "ultra", args: args{value: "ultra"}, want: ReasoningEffort{value: "ultra", valid: true}},
		{name: "empty effort", args: args{value: ""}, wantErr: true},
		{name: "unsupported effort", args: args{value: "minimal"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReasoningEffort(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewReasoningEffort() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !cmp.Equal(tt.want, got, cmp.AllowUnexported(ReasoningEffort{})) {
				t.Errorf("NewReasoningEffort() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got, cmp.AllowUnexported(ReasoningEffort{})))
			}
		})
	}
}

func TestNewWorkingDirectory(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    WorkingDirectory
		wantErr bool
	}{
		{name: "existing absolute directory", args: args{value: t.TempDir()}, want: WorkingDirectory{}},
		{name: "relative directory", args: args{value: "."}, wantErr: true},
		{name: "missing absolute directory", args: args{value: filepath.Join(t.TempDir(), "missing")}, wantErr: true},
		{name: "absolute file", args: args{value: func() string {
			file, err := os.CreateTemp(t.TempDir(), "working-directory-file")
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			return file.Name()
		}()}, wantErr: true},
	}
	tests[0].want = WorkingDirectory{value: tests[0].args.value, valid: true}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWorkingDirectory(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewWorkingDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !cmp.Equal(tt.want, got, cmp.AllowUnexported(WorkingDirectory{})) {
				t.Errorf("NewWorkingDirectory() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got, cmp.AllowUnexported(WorkingDirectory{})))
			}
		})
	}
}

func TestNewConfiguration(t *testing.T) {
	type args struct {
		model            Model
		reasoningEffort  ReasoningEffort
		workingDirectory WorkingDirectory
	}
	tests := []struct {
		name string
		args args
		want Configuration
	}{
		{
			name: "validated values",
			args: args{
				model:            Model{value: "gpt-5.6-sol", valid: true},
				reasoningEffort:  ReasoningEffort{value: "high", valid: true},
				workingDirectory: WorkingDirectory{value: "/work", valid: true},
			},
			want: Configuration{
				model:            Model{value: "gpt-5.6-sol", valid: true},
				reasoningEffort:  ReasoningEffort{value: "high", valid: true},
				workingDirectory: WorkingDirectory{value: "/work", valid: true},
			},
		},
		{name: "zero values remain invalid", args: args{}, want: Configuration{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConfiguration(tt.args.model, tt.args.reasoningEffort, tt.args.workingDirectory); !cmp.Equal(tt.want, got, cmp.AllowUnexported(Configuration{}, Model{}, ReasoningEffort{}, WorkingDirectory{})) {
				t.Errorf("NewConfiguration() = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got, cmp.AllowUnexported(Configuration{}, Model{}, ReasoningEffort{}, WorkingDirectory{})))
			}
		})
	}
}
