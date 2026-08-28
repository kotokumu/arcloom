package codexplancontrol

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

func TestNewAssessor_failsClosedWithoutProviderDetails(t *testing.T) {
	stubDirectory := t.TempDir()
	stubPath := filepath.Join(stubDirectory, "codex")
	build := exec.Command("go", "build", "-o", stubPath, "./testdata/codexappserverstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex app-server protocol stub: %v\n%s", err, output)
	}

	current := must(plan.New(
		"Current Plan",
		must(plan.NewGoal("Establish the exact current outcome")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Outcome evidence is named"))},
		[]plan.Task{must(plan.NewTask("Collect evidence"))},
		nil,
	))
	equalOutput := `{"outcome":"revise","proposedPlan":{"name":"Current Plan","goal":"Establish the exact current outcome","acceptanceConditions":["Outcome evidence is named"],"tasks":["Collect evidence"],"targetDate":null}}`
	validOutput := `{"outcome":"retain","proposedPlan":null}`
	encoderFailure := errors.New("private encoder failure")

	tests := []struct {
		name         string
		mode         string
		output       string
		path         string
		encoder      ObservationEncoder[string]
		wantCode     plancontrol.FailureCode
		privateError error
	}{
		{name: "empty completed output", output: "", path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "malformed completed output", output: "{", path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "unknown output field", output: `{"outcome":"retain","proposedPlan":null,"provider":"codex"}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "case variant output field", output: `{"Outcome":"retain","proposedPlan":null}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate outcome field", output: `{"outcome":"complete","outcome":"retain","proposedPlan":null}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate proposal field", output: `{"outcome":"revise","proposedPlan":null,"proposedPlan":{"name":"Other","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate nested proposal field", output: `{"outcome":"revise","proposedPlan":{"name":"Other","name":"Conflicting","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "missing proposed Plan field", output: `{"outcome":"retain"}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "unknown outcome", output: `{"outcome":"continue","proposedPlan":null}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "non revise with proposal", output: `{"outcome":"complete","proposedPlan":{"name":"Other","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "revise without proposal", output: `{"outcome":"revise","proposedPlan":null}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "revise with invalid proposal", output: `{"outcome":"revise","proposedPlan":{"name":"","goal":"Goal","acceptanceConditions":[],"tasks":[],"targetDate":null}}`, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "revise with equal proposal", output: equalOutput, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate final messages", mode: "duplicate_final", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "missing final message", mode: "missing_final", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "unknown message phase", mode: "unknown_phase", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIContractFailure},
		{name: "missing JSON-RPC version", mode: "missing_jsonrpc", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "wrong JSON-RPC version", mode: "wrong_jsonrpc", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate JSON-RPC version", mode: "duplicate_jsonrpc", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "missing RPC result and error", mode: "missing_result", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "RPC result and error conflict", mode: "result_and_error", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "notification carries response ID", mode: "notification_with_id", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "completed turn contains tool activity", mode: "tool_item", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "tool activity notification", mode: "tool_notification", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate thread start ID", mode: "duplicate_thread_id", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate turn start ID", mode: "duplicate_turn_id", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate completed thread ID", mode: "duplicate_completed_thread_id", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate completed turn ID", mode: "duplicate_completed_turn_id", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate completed status", mode: "duplicate_completed_status", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "duplicate final item text", mode: "duplicate_final_text", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "initialize RPC failure", mode: "initialize_rpc_error", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "thread RPC failure", mode: "thread_rpc_error", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "turn RPC failure", mode: "turn_rpc_error", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "malformed protocol", mode: "malformed_protocol", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "unexpected server request", mode: "server_request", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "failed turn", mode: "turn_failed", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "wrong thread correlation", mode: "wrong_thread", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "wrong turn correlation", mode: "wrong_turn", output: validOutput, path: stubDirectory, wantCode: plancontrol.AIBoundaryFailure},
		{name: "process start failure", output: validOutput, path: t.TempDir(), wantCode: plancontrol.AIBoundaryFailure},
		{name: "observation encoder failure", output: validOutput, path: stubDirectory, encoder: func(context.Context, string) (string, error) { return "", encoderFailure }, wantCode: plancontrol.AIBoundaryFailure, privateError: encoderFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", tt.path)
			t.Setenv("CODEX_STUB_MODE", tt.mode)
			t.Setenv("CODEX_STUB_OUTPUT", tt.output)
			t.Setenv("CODEX_STUB_CAPTURE", filepath.Join(t.TempDir(), "capture.json"))
			encoder := tt.encoder
			if encoder == nil {
				encoder = func(_ context.Context, observation string) (string, error) { return observation, nil }
			}
			configuration := NewConfiguration(
				must(NewModel("gpt-5.6-sol")),
				must(NewReasoningEffort("high")),
				must(NewWorkingDirectory(t.TempDir())),
				must(NewShutdownGrace(100*time.Millisecond)),
			)
			assessor := must(NewAssessor(configuration, encoder))

			assessment, err := plancontrol.Assess(context.Background(), current, "observation", assessor)
			if assessment.Outcome() != 0 || assessment.AssessedPlan().IsValid() {
				t.Errorf("Assessment = %#v, want zero assessment", assessment)
			}
			var failure *plancontrol.FailureError
			if !errors.As(err, &failure) {
				t.Fatalf("Assess() error = %v, want *plancontrol.FailureError", err)
			}
			if failure.Code() != tt.wantCode {
				t.Errorf("FailureCode = %q, want %q", failure.Code(), tt.wantCode)
			}
			if errors.Is(err, plancontrol.ErrUntranslatableAIResponse) || (tt.privateError != nil && errors.Is(err, tt.privateError)) {
				t.Errorf("Assess() exposed private error: %v", err)
			}
		})
	}
}
