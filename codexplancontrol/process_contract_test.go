package codexplancontrol

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

func TestNewAssessor_mapsExactMaterialToPlanControlResponses(t *testing.T) {
	stubDirectory := t.TempDir()
	stubPath := filepath.Join(stubDirectory, "codex")
	build := exec.Command("go", "build", "-o", stubPath, "./testdata/codexappserverstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex app-server protocol stub: %v\n%s", err, output)
	}
	t.Setenv("PATH", stubDirectory)

	type assessmentMaterial struct {
		CurrentPlan struct {
			Name                 string   `json:"name"`
			Goal                 string   `json:"goal"`
			AcceptanceConditions []string `json:"acceptanceConditions"`
			Tasks                []string `json:"tasks"`
			TargetDate           *string  `json:"targetDate"`
		} `json:"currentPlan"`
		Observations string `json:"observations"`
	}
	type processCapture struct {
		Args             []string        `json:"args"`
		WorkingDirectory string          `json:"workingDirectory"`
		ThreadStart      json.RawMessage `json:"threadStart"`
		TurnStart        json.RawMessage `json:"turnStart"`
	}
	type threadStartParams struct {
		ApprovalPolicy          string `json:"approvalPolicy"`
		Sandbox                 string `json:"sandbox"`
		Model                   string `json:"model"`
		CWD                     string `json:"cwd"`
		Ephemeral               bool   `json:"ephemeral"`
		DeveloperInstructions   string `json:"developerInstructions"`
		DynamicTools            []any  `json:"dynamicTools"`
		Environments            []any  `json:"environments"`
		SelectedCapabilityRoots []any  `json:"selectedCapabilityRoots"`
	}
	type turnStartParams struct {
		ThreadID       string `json:"threadId"`
		Effort         string `json:"effort"`
		ApprovalPolicy string `json:"approvalPolicy"`
		SandboxPolicy  struct {
			Type          string `json:"type"`
			NetworkAccess bool   `json:"networkAccess"`
		} `json:"sandboxPolicy"`
		Environments []any `json:"environments"`
		Input        []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"input"`
		OutputSchema json.RawMessage `json:"outputSchema"`
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
			capturePath := filepath.Join(t.TempDir(), "capture.json")
			t.Setenv("CODEX_STUB_CAPTURE", capturePath)
			t.Setenv("CODEX_STUB_OUTPUT", tt.output)
			configuration := NewConfiguration(
				must(NewModel("gpt-5.6-sol")),
				must(NewReasoningEffort("high")),
				must(NewWorkingDirectory(workingDirectory)),
				must(NewShutdownGrace(time.Second)),
			)
			assessor := must(NewAssessor(configuration, func(_ context.Context, got string) (string, error) {
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

			capturedBytes, err := os.ReadFile(capturePath)
			if err != nil {
				t.Fatal(err)
			}
			var captured processCapture
			if err := json.Unmarshal(capturedBytes, &captured); err != nil {
				t.Fatal(err)
			}
			wantArgs := []string{"-s", "read-only", "-a", "never", "--disable", "apps", "--disable", "browser_use", "--disable", "computer_use", "--disable", "hooks", "--disable", "image_generation", "--disable", "multi_agent", "--disable", "plugins", "--disable", "shell_tool", "--disable", "unified_exec", "--disable", "code_mode_host", "--disable", "js_repl", "-c", "mcp_servers={}", "app-server", "--stdio"}
			if diff := cmp.Diff(wantArgs, captured.Args); diff != "" {
				t.Errorf("Codex arguments mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(workingDirectory, captured.WorkingDirectory); diff != "" {
				t.Errorf("Codex working directory mismatch (-want +got):\n%s", diff)
			}

			var threadStart threadStartParams
			if err := json.Unmarshal(captured.ThreadStart, &threadStart); err != nil {
				t.Fatal(err)
			}
			wantThreadStart := threadStartParams{
				ApprovalPolicy:          "never",
				Sandbox:                 "read-only",
				Model:                   "gpt-5.6-sol",
				CWD:                     workingDirectory,
				Ephemeral:               true,
				DeveloperInstructions:   threadStart.DeveloperInstructions,
				DynamicTools:            []any{},
				Environments:            []any{},
				SelectedCapabilityRoots: []any{},
			}
			if threadStart.DeveloperInstructions == "" {
				t.Error("developer instructions are empty")
			}
			if diff := cmp.Diff(wantThreadStart, threadStart); diff != "" {
				t.Errorf("thread/start safety mismatch (-want +got):\n%s", diff)
			}

			var turnStart turnStartParams
			if err := json.Unmarshal(captured.TurnStart, &turnStart); err != nil {
				t.Fatal(err)
			}
			if turnStart.ThreadID != "thread-stub" || turnStart.Effort != "high" || turnStart.ApprovalPolicy != "never" {
				t.Errorf("turn/start identity or policy = %#v", turnStart)
			}
			if turnStart.SandboxPolicy.Type != "readOnly" || turnStart.SandboxPolicy.NetworkAccess {
				t.Errorf("turn/start sandbox = %#v", turnStart.SandboxPolicy)
			}
			if len(turnStart.Environments) != 0 || len(turnStart.Input) != 1 || turnStart.Input[0].Type != "text" || len(turnStart.OutputSchema) == 0 {
				t.Errorf("turn/start boundary = %#v", turnStart)
			}
			var material assessmentMaterial
			if err := json.Unmarshal([]byte(turnStart.Input[0].Text), &material); err != nil {
				t.Fatal(err)
			}
			wantMaterial := assessmentMaterial{Observations: tt.observations}
			wantMaterial.CurrentPlan.Name = tt.current.Name()
			wantMaterial.CurrentPlan.Goal = tt.current.Goal().Text()
			for _, condition := range tt.current.AcceptanceConditions() {
				wantMaterial.CurrentPlan.AcceptanceConditions = append(wantMaterial.CurrentPlan.AcceptanceConditions, condition.Statement())
			}
			for _, task := range tt.current.Tasks() {
				wantMaterial.CurrentPlan.Tasks = append(wantMaterial.CurrentPlan.Tasks, task.Name())
			}
			if targetDate, ok := tt.current.TargetDate(); ok {
				value := targetDate.String()
				wantMaterial.CurrentPlan.TargetDate = &value
			}
			if diff := cmp.Diff(wantMaterial, material); diff != "" {
				t.Errorf("assessment material mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
