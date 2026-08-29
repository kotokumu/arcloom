package codexplancontrol

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/codexappserver"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

type isolatedClient struct {
	received chan codexappserver.ReadOnlyTurnRequest
}

type isolatedAssessmentMaterial struct {
	CurrentPlan struct {
		Name                 string   `json:"name"`
		Goal                 string   `json:"goal"`
		AcceptanceConditions []string `json:"acceptanceConditions"`
		Tasks                []string `json:"tasks"`
		TargetDate           *string  `json:"targetDate"`
	} `json:"currentPlan"`
	Observations string `json:"observations"`
}

func (c isolatedClient) CompleteReadOnlyTurn(
	ctx context.Context,
	request codexappserver.ReadOnlyTurnRequest,
) (codexappserver.CompletedTurn, error) {
	c.received <- request
	var material struct {
		Observations string `json:"observations"`
	}
	if err := json.Unmarshal([]byte(request.Input()), &material); err != nil {
		return codexappserver.CompletedTurn{}, err
	}
	switch material.Observations {
	case "hang":
		<-ctx.Done()
		return codexappserver.CompletedTurn{}, ctx.Err()
	case "first":
		return codexappserver.CompletedTurn{FinalOutput: `{"outcome":"retain","proposedPlan":null}`}, nil
	default:
		return codexappserver.CompletedTurn{FinalOutput: `{"outcome":"complete","proposedPlan":null}`}, nil
	}
}

func TestNewAssessor_propagatesCancellationAndIsolatesConcurrentAssessments(t *testing.T) {
	client := isolatedClient{received: make(chan codexappserver.ReadOnlyTurnRequest, 2)}
	configuration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		must(NewWorkingDirectory(t.TempDir())),
	)
	assessor := must(NewAssessor(client, configuration, func(_ context.Context, observation string) (string, error) { return observation, nil }))
	hangingPlan := must(plan.New(
		"Hanging Plan",
		must(plan.NewGoal("Remain isolated while cancelled")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("No result crosses calls"))},
		nil,
		nil,
	))
	successPlan := must(plan.New(
		"Successful Plan",
		must(plan.NewGoal("Complete independently")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Success is unaffected"))},
		[]plan.Task{must(plan.NewTask("Return complete"))},
		nil,
	))

	hangingContext, cancelHanging := context.WithCancel(context.Background())
	hangingResult := make(chan error, 1)
	go func() {
		_, err := plancontrol.Assess(hangingContext, hangingPlan, "hang", assessor)
		hangingResult <- err
	}()
	type successResult struct {
		assessment plancontrol.Assessment
		err        error
	}
	successfulResult := make(chan successResult, 1)
	go func() {
		assessment, err := plancontrol.Assess(context.Background(), successPlan, "success", assessor)
		successfulResult <- successResult{assessment: assessment, err: err}
	}()

	received := make(map[string]isolatedAssessmentMaterial, 2)
	for range 2 {
		select {
		case request := <-client.received:
			var material isolatedAssessmentMaterial
			if err := json.Unmarshal([]byte(request.Input()), &material); err != nil {
				t.Fatal(err)
			}
			received[material.Observations] = material
		case <-time.After(time.Second):
			t.Fatal("Client did not receive both isolated requests")
		}
	}
	wantReceived := make(map[string]isolatedAssessmentMaterial, 2)
	wantHanging := isolatedAssessmentMaterial{Observations: "hang"}
	wantHanging.CurrentPlan.Name = hangingPlan.Name()
	wantHanging.CurrentPlan.Goal = hangingPlan.Goal().Text()
	wantHanging.CurrentPlan.AcceptanceConditions = []string{"No result crosses calls"}
	wantReceived["hang"] = wantHanging
	wantSuccessful := isolatedAssessmentMaterial{Observations: "success"}
	wantSuccessful.CurrentPlan.Name = successPlan.Name()
	wantSuccessful.CurrentPlan.Goal = successPlan.Goal().Text()
	wantSuccessful.CurrentPlan.AcceptanceConditions = []string{"Success is unaffected"}
	wantSuccessful.CurrentPlan.Tasks = []string{"Return complete"}
	wantReceived["success"] = wantSuccessful
	if diff := cmp.Diff(wantReceived, received); diff != "" {
		t.Errorf("concurrent Client material mismatch (-want +got):\n%s", diff)
	}
	select {
	case completed := <-successfulResult:
		if completed.err != nil {
			t.Fatalf("successful Assess() error = %v", completed.err)
		}
		if completed.assessment.Outcome() != plancontrol.Complete || !completed.assessment.AssessedPlan().Equal(successPlan) {
			t.Errorf("successful Assessment = %#v, want Complete for successful Plan", completed.assessment)
		}
	case <-time.After(time.Second):
		t.Fatal("successful concurrent assessment did not complete")
	}
	cancelHanging()
	select {
	case err := <-hangingResult:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("hanging Assess() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled assessment did not return through the Client contract")
	}
}

func TestNewAssessor_repeatedAssessmentsRetainNoPriorJudgment(t *testing.T) {
	client := isolatedClient{received: make(chan codexappserver.ReadOnlyTurnRequest, 2)}
	configuration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		must(NewWorkingDirectory(t.TempDir())),
	)
	assessor := must(NewAssessor(client, configuration, func(_ context.Context, observation string) (string, error) { return observation, nil }))
	firstPlan := must(plan.New("First Plan", must(plan.NewGoal("First goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("First accepted"))}, nil, nil))
	secondPlan := must(plan.New("Second Plan", must(plan.NewGoal("Second goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Second accepted"))}, nil, nil))

	first, err := plancontrol.Assess(context.Background(), firstPlan, "first", assessor)
	if err != nil || first.Outcome() != plancontrol.Retain || !first.AssessedPlan().Equal(firstPlan) {
		t.Fatalf("first Assessment = %#v, error = %v", first, err)
	}
	second, err := plancontrol.Assess(context.Background(), secondPlan, "second", assessor)
	if err != nil || second.Outcome() != plancontrol.Complete || !second.AssessedPlan().Equal(secondPlan) {
		t.Fatalf("second Assessment = %#v, error = %v", second, err)
	}
	var received []isolatedAssessmentMaterial
	for range 2 {
		select {
		case request := <-client.received:
			var material isolatedAssessmentMaterial
			if err := json.Unmarshal([]byte(request.Input()), &material); err != nil {
				t.Fatal(err)
			}
			received = append(received, material)
		case <-time.After(time.Second):
			t.Fatal("Client did not receive repeated requests")
		}
	}
	wantFirst := isolatedAssessmentMaterial{Observations: "first"}
	wantFirst.CurrentPlan.Name = firstPlan.Name()
	wantFirst.CurrentPlan.Goal = firstPlan.Goal().Text()
	wantFirst.CurrentPlan.AcceptanceConditions = []string{"First accepted"}
	wantSecond := isolatedAssessmentMaterial{Observations: "second"}
	wantSecond.CurrentPlan.Name = secondPlan.Name()
	wantSecond.CurrentPlan.Goal = secondPlan.Goal().Text()
	wantSecond.CurrentPlan.AcceptanceConditions = []string{"Second accepted"}
	if diff := cmp.Diff([]isolatedAssessmentMaterial{wantFirst, wantSecond}, received); diff != "" {
		t.Errorf("repeated Client material mismatch (-want +got):\n%s", diff)
	}
}
