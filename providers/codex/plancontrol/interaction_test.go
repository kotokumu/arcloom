package codexplancontrol

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/providers/codex/appserver"
)

func TestNewAssessor_mapsExactMaterialToPlanControlResponses(t *testing.T) {
	type capturedMaterial struct {
		CurrentPlan struct {
			Name                 string   `json:"name"`
			Goal                 string   `json:"goal"`
			AcceptanceConditions []string `json:"acceptanceConditions"`
			Tasks                []string `json:"tasks"`
			TargetDate           *string  `json:"targetDate"`
		} `json:"currentPlan"`
		Observations string `json:"observations"`
	}
	type requestObservation struct {
		Model                string
		ReasoningEffort      string
		WorkingDirectory     string
		HasInstructions      bool
		HasValidOutputSchema bool
		Material             capturedMaterial
	}

	completeCurrent := must(plan.New(
		"Release readiness",
		must(plan.NewGoal("Release after outcome evidence is established")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("All release evidence is named"))},
		[]plan.Task{must(plan.NewTask("Collect release evidence"))},
		nil,
	))
	retainDate := must(plan.ParseTargetDate("2026-09-12"))
	retainCurrent := must(plan.New(
		"Keep delivery plan",
		must(plan.NewGoal("Preserve the current delivery direction")),
		[]plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("Current direction remains justified")),
			must(plan.NewAcceptanceCondition("No revision evidence exists")),
		},
		[]plan.Task{must(plan.NewTask("Continue implementation")), must(plan.NewTask("Review evidence"))},
		&retainDate,
	))
	reviseDate := must(plan.ParseTargetDate("2026-09-18"))
	reviseCurrent := must(plan.New(
		"Initial rollout",
		must(plan.NewGoal("Roll out the initial capability")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("One environment is verified"))},
		[]plan.Task{must(plan.NewTask("Implement rollout"))},
		&reviseDate,
	))
	proposedDate := must(plan.ParseTargetDate("2026-09-20"))
	proposed := must(plan.New(
		"Staged rollout",
		must(plan.NewGoal("Roll out safely in stages")),
		[]plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("One environment is verified")),
			must(plan.NewAcceptanceCondition("Rollback evidence is recorded")),
		},
		[]plan.Task{must(plan.NewTask("Verify canary")), must(plan.NewTask("Expand rollout"))},
		&proposedDate,
	))
	insufficientCurrent := must(plan.New(
		"Unknown readiness",
		must(plan.NewGoal("Establish whether release is ready")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Readiness evidence is available"))},
		nil,
		nil,
	))

	tests := []struct {
		name         string
		current      plan.Plan
		observations string
		output       string
		want         plancontrol.AssessorResponse
	}{
		{name: "complete", current: completeCurrent, observations: `{"outcomeEvidence":["release-approved"]}`, output: `{"outcome":"complete","proposedPlan":null}`, want: plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Complete}}},
		{name: "retain", current: retainCurrent, observations: `{"progress":"on-track","coverage":100}`, output: `{"outcome":"retain","proposedPlan":null}`, want: plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Retain}}},
		{name: "revise", current: reviseCurrent, observations: `{"risk":"rollout-too-broad"}`, output: `{"outcome":"revise","proposedPlan":{"name":"Staged rollout","goal":"Roll out safely in stages","acceptanceConditions":["One environment is verified","Rollback evidence is recorded"],"tasks":["Verify canary","Expand rollout"],"targetDate":"2026-09-20"}}`, want: plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.Revise}, ProposedPlans: []plan.Plan{proposed}}},
		{name: "insufficient information", current: insufficientCurrent, observations: `{"missing":["readiness evidence"]}`, output: `{"outcome":"insufficient_information","proposedPlan":null}`, want: plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{plancontrol.InsufficientInformation}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDirectory := t.TempDir()
			client := &recordingClient{complete: func(_ context.Context, _ codexappserver.ReadOnlyTurnRequest) (codexappserver.CompletedTurn, error) {
				return codexappserver.CompletedTurn{FinalOutput: tt.output}, nil
			}}
			configuration := NewConfiguration(
				must(NewModel("gpt-5.6-sol")),
				must(NewReasoningEffort("high")),
				must(NewWorkingDirectory(workingDirectory)),
			)
			assessor := must(NewAssessor(client, configuration, func(_ context.Context, got string) (string, error) {
				if diff := cmp.Diff(tt.observations, got); diff != "" {
					t.Errorf("encoder input mismatch (-want +got):\n%s", diff)
				}
				return got, nil
			}))

			got, err := assessor(context.Background(), tt.current, tt.observations)
			if err != nil {
				t.Fatalf("Assessor() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got, cmp.Comparer(func(left, right plan.Plan) bool { return left.Equal(right) })); diff != "" {
				t.Errorf("Assessor() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(1, len(client.requests)); diff != "" {
				t.Fatalf("Client invocation count mismatch (-want +got):\n%s", diff)
			}
			request := client.requests[0]
			var material capturedMaterial
			if err := json.Unmarshal([]byte(request.Input()), &material); err != nil {
				t.Fatal(err)
			}
			gotRequest := requestObservation{
				Model:                request.Model(),
				ReasoningEffort:      request.ReasoningEffort(),
				WorkingDirectory:     request.WorkingDirectory(),
				HasInstructions:      request.DeveloperInstructions() != "",
				HasValidOutputSchema: json.Valid([]byte(request.OutputSchema())),
				Material:             material,
			}
			wantRequest := requestObservation{
				Model:                "gpt-5.6-sol",
				ReasoningEffort:      "high",
				WorkingDirectory:     workingDirectory,
				HasInstructions:      true,
				HasValidOutputSchema: true,
			}
			wantRequest.Material.Observations = tt.observations
			wantRequest.Material.CurrentPlan.Name = tt.current.Name()
			wantRequest.Material.CurrentPlan.Goal = tt.current.Goal().Text()
			for _, condition := range tt.current.AcceptanceConditions() {
				wantRequest.Material.CurrentPlan.AcceptanceConditions = append(wantRequest.Material.CurrentPlan.AcceptanceConditions, condition.Statement())
			}
			for _, task := range tt.current.Tasks() {
				wantRequest.Material.CurrentPlan.Tasks = append(wantRequest.Material.CurrentPlan.Tasks, task.Name())
			}
			if targetDate, ok := tt.current.TargetDate(); ok {
				value := targetDate.String()
				wantRequest.Material.CurrentPlan.TargetDate = &value
			}
			if diff := cmp.Diff(wantRequest, gotRequest); diff != "" {
				t.Errorf("ReadOnlyTurnRequest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
