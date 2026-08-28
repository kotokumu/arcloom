package codexplancontrol

import (
	"context"
	"errors"
	"testing"

	"github.com/kotokumu/arcloom/codexappserver"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

func TestNewAssessor_failsClosedWithoutSDKDetails(t *testing.T) {
	current := must(plan.New(
		"Current Plan",
		must(plan.NewGoal("Establish the exact current outcome")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Outcome evidence is named"))},
		[]plan.Task{must(plan.NewTask("Collect evidence"))},
		nil,
	))
	equalOutput := `{"outcome":"revise","proposedPlan":{"name":"Current Plan","goal":"Establish the exact current outcome","acceptanceConditions":["Outcome evidence is named"],"tasks":["Collect evidence"],"targetDate":null}}`
	sdkFailure := errors.New("private SDK failure")
	encoderFailure := errors.New("private encoder failure")

	tests := []struct {
		name         string
		output       string
		clientErr    error
		encoder      ObservationEncoder[string]
		wantCode     plancontrol.FailureCode
		privateError error
	}{
		{name: "empty completed output", output: "", wantCode: plancontrol.AIContractFailure},
		{name: "malformed completed output", output: "{", wantCode: plancontrol.AIContractFailure},
		{name: "unknown output field", output: `{"outcome":"retain","proposedPlan":null,"provider":"codex"}`, wantCode: plancontrol.AIContractFailure},
		{name: "case variant output field", output: `{"Outcome":"retain","proposedPlan":null}`, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate outcome field", output: `{"outcome":"complete","outcome":"retain","proposedPlan":null}`, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate proposal field", output: `{"outcome":"revise","proposedPlan":null,"proposedPlan":{"name":"Other","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, wantCode: plancontrol.AIContractFailure},
		{name: "duplicate nested proposal field", output: `{"outcome":"revise","proposedPlan":{"name":"Other","name":"Conflicting","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, wantCode: plancontrol.AIContractFailure},
		{name: "missing proposed Plan field", output: `{"outcome":"retain"}`, wantCode: plancontrol.AIContractFailure},
		{name: "unknown outcome", output: `{"outcome":"continue","proposedPlan":null}`, wantCode: plancontrol.AIContractFailure},
		{name: "non revise with proposal", output: `{"outcome":"complete","proposedPlan":{"name":"Other","goal":"Other goal","acceptanceConditions":["Other condition"],"tasks":[],"targetDate":null}}`, wantCode: plancontrol.AIContractFailure},
		{name: "revise without proposal", output: `{"outcome":"revise","proposedPlan":null}`, wantCode: plancontrol.AIContractFailure},
		{name: "revise with invalid proposal", output: `{"outcome":"revise","proposedPlan":{"name":"","goal":"Goal","acceptanceConditions":[],"tasks":[],"targetDate":null}}`, wantCode: plancontrol.AIContractFailure},
		{name: "revise with equal proposal", output: equalOutput, wantCode: plancontrol.AIContractFailure},
		{name: "SDK interaction failure", clientErr: sdkFailure, wantCode: plancontrol.AIBoundaryFailure, privateError: sdkFailure},
		{name: "observation encoder failure", encoder: func(context.Context, string) (string, error) { return "", encoderFailure }, wantCode: plancontrol.AIBoundaryFailure, privateError: encoderFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &recordingClient{complete: func(context.Context, codexappserver.ReadOnlyTurnRequest) (codexappserver.CompletedTurn, error) {
				return codexappserver.CompletedTurn{FinalOutput: tt.output}, tt.clientErr
			}}
			encoder := tt.encoder
			if encoder == nil {
				encoder = func(_ context.Context, observation string) (string, error) { return observation, nil }
			}
			configuration := NewConfiguration(
				must(NewModel("gpt-5.6-sol")),
				must(NewReasoningEffort("high")),
				must(NewWorkingDirectory(t.TempDir())),
			)
			assessor := must(NewAssessor(client, configuration, encoder))

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
