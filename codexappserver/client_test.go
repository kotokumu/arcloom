package codexappserver

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewReadOnlyTurnRequest(t *testing.T) {
	type args struct {
		model                 string
		reasoningEffort       string
		workingDirectory      string
		developerInstructions string
		input                 string
		outputSchema          string
	}
	tests := []struct {
		name    string
		args    args
		want    ReadOnlyTurnRequest
		wantErr bool
	}{
		{
			name: "valid immutable material",
			args: args{
				model:                 "gpt-5.6-sol",
				reasoningEffort:       "high",
				workingDirectory:      t.TempDir(),
				developerInstructions: "Assess only supplied material.",
				input:                 `{"currentPlan":{"name":"Release"}}`,
				outputSchema:          `{"type":"object"}`,
			},
		},
		{name: "blank model", args: args{model: "\t", reasoningEffort: "high", workingDirectory: t.TempDir(), developerInstructions: "Assess.", input: `{}`, outputSchema: `{}`}, wantErr: true},
		{name: "blank reasoning effort", args: args{model: "gpt-5.6-sol", reasoningEffort: " ", workingDirectory: t.TempDir(), developerInstructions: "Assess.", input: `{}`, outputSchema: `{}`}, wantErr: true},
		{name: "relative working directory", args: args{model: "gpt-5.6-sol", reasoningEffort: "high", workingDirectory: filepath.Join("relative", "workspace"), developerInstructions: "Assess.", input: `{}`, outputSchema: `{}`}, wantErr: true},
		{name: "blank developer instructions", args: args{model: "gpt-5.6-sol", reasoningEffort: "high", workingDirectory: t.TempDir(), developerInstructions: "\n", input: `{}`, outputSchema: `{}`}, wantErr: true},
		{name: "blank input", args: args{model: "gpt-5.6-sol", reasoningEffort: "high", workingDirectory: t.TempDir(), developerInstructions: "Assess.", input: " ", outputSchema: `{}`}, wantErr: true},
		{name: "malformed output schema", args: args{model: "gpt-5.6-sol", reasoningEffort: "high", workingDirectory: t.TempDir(), developerInstructions: "Assess.", input: `{}`, outputSchema: `{"type":`}, wantErr: true},
		{name: "non-object output schema", args: args{model: "gpt-5.6-sol", reasoningEffort: "high", workingDirectory: t.TempDir(), developerInstructions: "Assess.", input: `{}`, outputSchema: `true`}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				tt.want = ReadOnlyTurnRequest{
					model:                 tt.args.model,
					reasoningEffort:       tt.args.reasoningEffort,
					workingDirectory:      tt.args.workingDirectory,
					developerInstructions: tt.args.developerInstructions,
					input:                 tt.args.input,
					outputSchema:          tt.args.outputSchema,
				}
			}
			got, err := NewReadOnlyTurnRequest(tt.args.model, tt.args.reasoningEffort, tt.args.workingDirectory, tt.args.developerInstructions, tt.args.input, tt.args.outputSchema)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewReadOnlyTurnRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ReadOnlyTurnRequest{})); diff != "" {
				t.Errorf("NewReadOnlyTurnRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadOnlyTurnRequest_accessorsReturnConstructionMaterial(t *testing.T) {
	request := ReadOnlyTurnRequest{
		model:                 "model",
		reasoningEffort:       "effort",
		workingDirectory:      "/workspace",
		developerInstructions: "instructions",
		input:                 "input",
		outputSchema:          "schema",
	}

	got := []string{
		request.Model(),
		request.ReasoningEffort(),
		request.WorkingDirectory(),
		request.DeveloperInstructions(),
		request.Input(),
		request.OutputSchema(),
	}
	want := []string{"model", "effort", "/workspace", "instructions", "input", "schema"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("accessor results mismatch (-want +got):\n%s", diff)
	}
}
