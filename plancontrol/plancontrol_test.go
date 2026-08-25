// Package plancontrol establishes a Plan-specific assessment from a current
// Plan, caller-owned observations, and an external AI judgment.
package plancontrol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/plan"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type publicPlan struct {
	Valid      bool
	Name       string
	Goal       string
	Conditions []string
	Tasks      []string
	TargetDate string
	HasDate    bool
}

type publicAssessment struct {
	Outcome      Outcome
	AssessedPlan publicPlan
	ProposedPlan publicPlan
	HasProposed  bool
	HasError     bool
}

type failureObservation struct {
	FailureCode       FailureCode
	ContextCanceled   bool
	ContextDeadline   bool
	RawErrorUnwrapped bool
	Assessment        publicAssessment
	AssessorCalls     int
}

type cancelDuringError struct {
	cancel     context.CancelFunc
	classified chan struct{}
}

func (e cancelDuringError) Error() string { return "cancel during error classification" }

func (e cancelDuringError) Is(error) bool {
	close(e.classified)
	e.cancel()
	return false
}

func TestAssessment_Outcome(t *testing.T) {
	tests := []struct {
		name string
		a    Assessment
		want Outcome
	}{
		{name: "zero assessment", a: Assessment{}, want: Outcome(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Outcome()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Assessment.Outcome() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessment_AssessedPlan(t *testing.T) {
	tests := []struct {
		name string
		a    Assessment
		want plan.Plan
	}{
		{name: "zero assessment", a: Assessment{}, want: plan.Plan{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.AssessedPlan()
			gotDate, gotHasDate := got.TargetDate()
			wantDate, wantHasDate := tt.want.TargetDate()
			gotPublic := publicPlan{Valid: got.IsValid(), Name: got.Name(), Goal: got.Goal().Text(), Conditions: []string{}, Tasks: []string{}, TargetDate: gotDate.String(), HasDate: gotHasDate}
			wantPublic := publicPlan{Valid: tt.want.IsValid(), Name: tt.want.Name(), Goal: tt.want.Goal().Text(), Conditions: []string{}, Tasks: []string{}, TargetDate: wantDate.String(), HasDate: wantHasDate}
			for _, condition := range got.AcceptanceConditions() {
				gotPublic.Conditions = append(gotPublic.Conditions, condition.Statement())
			}
			for _, condition := range tt.want.AcceptanceConditions() {
				wantPublic.Conditions = append(wantPublic.Conditions, condition.Statement())
			}
			for _, task := range got.Tasks() {
				gotPublic.Tasks = append(gotPublic.Tasks, task.Name())
			}
			for _, task := range tt.want.Tasks() {
				wantPublic.Tasks = append(wantPublic.Tasks, task.Name())
			}
			if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
				t.Errorf("Assessment.AssessedPlan() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessment_ProposedPlan(t *testing.T) {
	tests := []struct {
		name  string
		a     Assessment
		want  plan.Plan
		want1 bool
	}{
		{name: "zero assessment", a: Assessment{}, want: plan.Plan{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.a.ProposedPlan()
			gotDate, gotHasDate := got.TargetDate()
			wantDate, wantHasDate := tt.want.TargetDate()
			gotPublic := publicPlan{Valid: got.IsValid(), Name: got.Name(), Goal: got.Goal().Text(), Conditions: []string{}, Tasks: []string{}, TargetDate: gotDate.String(), HasDate: gotHasDate}
			wantPublic := publicPlan{Valid: tt.want.IsValid(), Name: tt.want.Name(), Goal: tt.want.Goal().Text(), Conditions: []string{}, Tasks: []string{}, TargetDate: wantDate.String(), HasDate: wantHasDate}
			for _, condition := range got.AcceptanceConditions() {
				gotPublic.Conditions = append(gotPublic.Conditions, condition.Statement())
			}
			for _, condition := range tt.want.AcceptanceConditions() {
				wantPublic.Conditions = append(wantPublic.Conditions, condition.Statement())
			}
			for _, task := range got.Tasks() {
				gotPublic.Tasks = append(gotPublic.Tasks, task.Name())
			}
			for _, task := range tt.want.Tasks() {
				wantPublic.Tasks = append(wantPublic.Tasks, task.Name())
			}
			if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
				t.Errorf("Assessment.ProposedPlan() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("Assessment.ProposedPlan() presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailureError_Error(t *testing.T) {
	tests := []struct {
		name string
		f    *FailureError
		want string
	}{
		{name: "zero receiver", f: &FailureError{}, want: "plan control failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.f.Error()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FailureError.Error() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailureError_Code(t *testing.T) {
	tests := []struct {
		name string
		f    *FailureError
		want FailureCode
	}{
		{name: "zero receiver", f: &FailureError{}, want: FailureCode("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.f.Code()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FailureError.Code() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssess(t *testing.T) {
	validCurrent := must(plan.New("Generated valid current", must(plan.NewGoal("generated goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("generated acceptance"))}, []plan.Task{must(plan.NewTask("generated task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))
	type args struct {
		in0 context.Context
		in1 plan.Plan
		in2 int
		in3 Assessor[int]
	}
	tests := []struct {
		name    string
		args    args
		want    Assessment
		wantErr bool
	}{
		{
			name: "valid current plan retains",
			args: args{
				in0: context.Background(),
				in1: validCurrent,
				in2: 42,
				in3: func(_ context.Context, received plan.Plan, observations int) (AssessorResponse, error) {
					receivedConditions := received.AcceptanceConditions()
					expectedConditions := validCurrent.AcceptanceConditions()
					receivedTasks := received.Tasks()
					expectedTasks := validCurrent.Tasks()
					receivedDate, receivedHasDate := received.TargetDate()
					expectedDate, expectedHasDate := validCurrent.TargetDate()
					conditionsMatch := len(receivedConditions) == len(expectedConditions)
					if conditionsMatch {
						for i := range expectedConditions {
							conditionsMatch = conditionsMatch && receivedConditions[i].Statement() == expectedConditions[i].Statement()
						}
					}
					tasksMatch := len(receivedTasks) == len(expectedTasks)
					if tasksMatch {
						for i := range expectedTasks {
							tasksMatch = tasksMatch && receivedTasks[i].Name() == expectedTasks[i].Name()
						}
					}
					if observations != 42 || received.IsValid() != validCurrent.IsValid() || received.Name() != validCurrent.Name() || received.Goal().Text() != validCurrent.Goal().Text() || !conditionsMatch || !tasksMatch || receivedHasDate != expectedHasDate || (receivedHasDate && receivedDate.String() != expectedDate.String()) {
						return AssessorResponse{}, errors.New("generated valid input was not forwarded")
					}
					return AssessorResponse{Claims: []Outcome{Retain}}, nil
				},
			},
			want:    Assessment{outcome: Retain, assessedPlan: validCurrent},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			assessor := func(ctx context.Context, current plan.Plan, observations int) (AssessorResponse, error) {
				calls++
				return tt.args.in3(ctx, current, observations)
			}
			got, err := Assess[int](tt.args.in0, tt.args.in1, tt.args.in2, assessor)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Fatalf("Assess() error presence mismatch (-want +got):\n%s", diff)
			}
			gotAssessed := got.AssessedPlan()
			wantAssessed := tt.want.AssessedPlan()
			gotDate, gotHasDate := gotAssessed.TargetDate()
			wantDate, wantHasDate := wantAssessed.TargetDate()
			gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasError: err != nil}
			wantPublic := publicAssessment{Outcome: tt.want.Outcome(), AssessedPlan: publicPlan{Valid: wantAssessed.IsValid(), Name: wantAssessed.Name(), Goal: wantAssessed.Goal().Text(), TargetDate: wantDate.String(), HasDate: wantHasDate}}
			for _, condition := range gotAssessed.AcceptanceConditions() {
				gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
			}
			for _, condition := range wantAssessed.AcceptanceConditions() {
				wantPublic.AssessedPlan.Conditions = append(wantPublic.AssessedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotAssessed.Tasks() {
				gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
			}
			for _, task := range wantAssessed.Tasks() {
				wantPublic.AssessedPlan.Tasks = append(wantPublic.AssessedPlan.Tasks, task.Name())
			}
			gotProposed, gotHasProposed := got.ProposedPlan()
			wantProposed, wantHasProposed := tt.want.ProposedPlan()
			gotProposedDate, gotProposedHasDate := gotProposed.TargetDate()
			wantProposedDate, wantProposedHasDate := wantProposed.TargetDate()
			gotPublic.HasProposed = gotHasProposed
			gotPublic.ProposedPlan = publicPlan{Valid: gotProposed.IsValid(), Name: gotProposed.Name(), Goal: gotProposed.Goal().Text(), TargetDate: gotProposedDate.String(), HasDate: gotProposedHasDate}
			wantPublic.HasProposed = wantHasProposed
			wantPublic.ProposedPlan = publicPlan{Valid: wantProposed.IsValid(), Name: wantProposed.Name(), Goal: wantProposed.Goal().Text(), TargetDate: wantProposedDate.String(), HasDate: wantProposedHasDate}
			for _, condition := range gotProposed.AcceptanceConditions() {
				gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
			}
			for _, condition := range wantProposed.AcceptanceConditions() {
				wantPublic.ProposedPlan.Conditions = append(wantPublic.ProposedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotProposed.Tasks() {
				gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
			}
			for _, task := range wantProposed.Tasks() {
				wantPublic.ProposedPlan.Tasks = append(wantPublic.ProposedPlan.Tasks, task.Name())
			}
			if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
				t.Errorf("Assess() public observations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessPreservesAIJudgment(t *testing.T) {
	type observation struct {
		AllTasksComplete         bool
		ObsoleteTaskIncomplete   bool
		OutcomeEvidenceAvailable bool
	}
	tests := []struct {
		name         string
		observations observation
		response     AssessorResponse
		want         publicAssessment
	}{
		{name: "all tasks complete without outcome evidence", observations: observation{AllTasksComplete: true, OutcomeEvidenceAvailable: false}, response: AssessorResponse{Claims: []Outcome{InsufficientInformation}}, want: publicAssessment{Outcome: InsufficientInformation, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"obsolete"}}}},
		{name: "obsolete task incomplete with achieved outcome", observations: observation{AllTasksComplete: false, ObsoleteTaskIncomplete: true, OutcomeEvidenceAvailable: true}, response: AssessorResponse{Claims: []Outcome{Complete}}, want: publicAssessment{Outcome: Complete, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"obsolete"}}}},
		{name: "retain", observations: observation{AllTasksComplete: false, ObsoleteTaskIncomplete: false, OutcomeEvidenceAvailable: false}, response: AssessorResponse{Claims: []Outcome{Retain}}, want: publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"obsolete"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("obsolete"))}, nil))
			got, err := Assess(context.Background(), current, tt.observations, func(_ context.Context, _ plan.Plan, observations observation) (AssessorResponse, error) {
				if diff := cmp.Diff(tt.observations, observations); diff != "" {
					t.Errorf("observations changed (-want +got):\n%s", diff)
				}
				return tt.response, nil
			})
			gotCurrent := got.AssessedPlan()
			gotDate, gotHasDate := gotCurrent.TargetDate()
			gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotCurrent.IsValid(), Name: gotCurrent.Name(), Goal: gotCurrent.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasError: err != nil}
			for _, condition := range gotCurrent.AcceptanceConditions() {
				gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotCurrent.Tasks() {
				gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
			}
			proposed, hasProposed := got.ProposedPlan()
			proposedDate, proposedHasDate := proposed.TargetDate()
			gotPublic.ProposedPlan = publicPlan{Valid: proposed.IsValid(), Name: proposed.Name(), Goal: proposed.Goal().Text(), TargetDate: proposedDate.String(), HasDate: proposedHasDate}
			gotPublic.HasProposed = hasProposed
			for _, condition := range proposed.AcceptanceConditions() {
				gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
			}
			for _, task := range proposed.Tasks() {
				gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
			}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("Assess() public observations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessInvokesAssessorExactlyOnce(t *testing.T) {
	current := must(plan.New("Plan once", must(plan.NewGoal("goal once")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept once"))}, []plan.Task{must(plan.NewTask("task once"))}, func() *plan.TargetDate { d := must(plan.ParseTargetDate("2026-09-01")); return &d }()))
	want := struct {
		Assessment publicAssessment
		Calls      int
		HasError   bool
	}{Assessment: publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan once", Goal: "goal once", Conditions: []string{"accept once"}, Tasks: []string{"task once"}, TargetDate: "2026-09-01", HasDate: true}}, Calls: 1}
	calls := 0
	got, err := Assess(context.Background(), current, "observation once", func(_ context.Context, received plan.Plan, _ string) (AssessorResponse, error) {
		calls++
		if diff := cmp.Diff(true, current.Equal(received)); diff != "" {
			t.Errorf("Assessor current Plan mismatch (-want +got):\n%s", diff)
		}
		return AssessorResponse{Claims: []Outcome{Retain}}, nil
	})
	gotDate, gotHasDate := got.AssessedPlan().TargetDate()
	gotPublic := struct {
		Assessment publicAssessment
		Calls      int
		HasError   bool
	}{Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid(), Name: got.AssessedPlan().Name(), Goal: got.AssessedPlan().Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}}, Calls: calls, HasError: err != nil}
	for _, condition := range got.AssessedPlan().AcceptanceConditions() {
		gotPublic.Assessment.AssessedPlan.Conditions = append(gotPublic.Assessment.AssessedPlan.Conditions, condition.Statement())
	}
	for _, task := range got.AssessedPlan().Tasks() {
		gotPublic.Assessment.AssessedPlan.Tasks = append(gotPublic.Assessment.AssessedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(want, gotPublic); diff != "" {
		t.Errorf("single-call observation mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessRevisionDifferences(t *testing.T) {
	tests := []struct {
		name     string
		current  plan.Plan
		proposed plan.Plan
		want     publicAssessment
	}{
		{name: "name", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("Plan 2", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan 2", Goal: "goal", Conditions: []string{"accept"}}, HasProposed: true}},
		{name: "goal", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal 2")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal 2", Conditions: []string{"accept"}}, HasProposed: true}},
		{name: "one acceptance-condition value", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept 2"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept 2"}}, HasProposed: true}},
		{name: "acceptance-condition order only", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("a")), must(plan.NewAcceptanceCondition("b"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("b")), must(plan.NewAcceptanceCondition("a"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"a", "b"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"b", "a"}}, HasProposed: true}},
		{name: "one acceptance condition added", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("a"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("a")), must(plan.NewAcceptanceCondition("b"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"a"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"a", "b"}}, HasProposed: true}},
		{name: "one acceptance condition removed while one remains", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("a")), must(plan.NewAcceptanceCondition("b"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("a"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"a", "b"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"a"}}, HasProposed: true}},
		{name: "one Task value", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a"))}, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("b"))}, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"b"}}, HasProposed: true}},
		{name: "Task order only", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a")), must(plan.NewTask("b"))}, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("b")), must(plan.NewTask("a"))}, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a", "b"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"b", "a"}}, HasProposed: true}},
		{name: "Task count from zero to one", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a"))}, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a"}}, HasProposed: true}},
		{name: "Task count from one to zero", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a"))}, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, HasProposed: true}},
		{name: "Task count from one to two", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a"))}, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("a")), must(plan.NewTask("b"))}, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"a", "b"}}, HasProposed: true}},
		{name: "target-date value", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, func() *plan.TargetDate { d := must(plan.ParseTargetDate("2026-09-01")); return &d }())), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, func() *plan.TargetDate { d := must(plan.ParseTargetDate("2026-09-02")); return &d }())), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, TargetDate: "2026-09-01", HasDate: true}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, TargetDate: "2026-09-02", HasDate: true}, HasProposed: true}},
		{name: "target date absent to present", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, func() *plan.TargetDate { d := must(plan.ParseTargetDate("2026-09-01")); return &d }())), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, TargetDate: "2026-09-01", HasDate: true}, HasProposed: true}},
		{name: "target date present to absent", current: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, func() *plan.TargetDate { d := must(plan.ParseTargetDate("2026-09-01")); return &d }())), proposed: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), want: publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, TargetDate: "2026-09-01", HasDate: true}, ProposedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, HasProposed: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Assess(context.Background(), tt.current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
				return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{tt.proposed}}, nil
			})
			gotAssessed := got.AssessedPlan()
			gotAssessedDate, gotAssessedHasDate := gotAssessed.TargetDate()
			proposed, hasProposed := got.ProposedPlan()
			gotProposedDate, gotProposedHasDate := proposed.TargetDate()
			gotObservation := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotAssessedDate.String(), HasDate: gotAssessedHasDate}, ProposedPlan: publicPlan{Valid: proposed.IsValid(), Name: proposed.Name(), Goal: proposed.Goal().Text(), TargetDate: gotProposedDate.String(), HasDate: gotProposedHasDate}, HasProposed: hasProposed, HasError: err != nil}
			for _, condition := range gotAssessed.AcceptanceConditions() {
				gotObservation.AssessedPlan.Conditions = append(gotObservation.AssessedPlan.Conditions, condition.Statement())
			}
			for _, condition := range proposed.AcceptanceConditions() {
				gotObservation.ProposedPlan.Conditions = append(gotObservation.ProposedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotAssessed.Tasks() {
				gotObservation.AssessedPlan.Tasks = append(gotObservation.AssessedPlan.Tasks, task.Name())
			}
			for _, task := range proposed.Tasks() {
				gotObservation.ProposedPlan.Tasks = append(gotObservation.ProposedPlan.Tasks, task.Name())
			}
			if diff := cmp.Diff(tt.want, gotObservation); diff != "" {
				t.Errorf("revision observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessRevisionRejectsSeparatelyConstructedExactEquivalentPlan(t *testing.T) {
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))
	proposed := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))
	got, err := Assess(context.Background(), current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
		return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposed}}, nil
	})
	var failure *FailureError
	gotCode := FailureCode("")
	if errors.As(err, &failure) {
		gotCode = failure.Code()
	}
	returnedProposed, hasProposed := got.ProposedPlan()
	gotPublic := failureObservation{FailureCode: gotCode, Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: returnedProposed.IsValid()}, HasProposed: hasProposed}}
	wantPublic := failureObservation{FailureCode: AIContractFailure, Assessment: publicAssessment{Outcome: 0, AssessedPlan: publicPlan{Valid: false}, ProposedPlan: publicPlan{Valid: false}, HasProposed: false}}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("exact-equivalent observation mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name     string
		current  plan.Plan
		response AssessorResponse
		want     failureObservation
	}{
		{name: "absent claim", current: must(plan.New("Plan absent", must(plan.NewGoal("goal absent")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept absent"))}, nil, nil)), response: AssessorResponse{}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "absent claim with proposal", current: must(plan.New("Plan absent proposal", must(plan.NewGoal("goal absent proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept absent proposal"))}, nil, nil)), response: AssessorResponse{ProposedPlans: []plan.Plan{must(plan.New("Proposed absent", must(plan.NewGoal("goal absent proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept absent proposal"))}, []plan.Task{must(plan.NewTask("task absent proposal"))}, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "unknown claim", current: must(plan.New("Plan unknown", must(plan.NewGoal("goal unknown")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unknown"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{99}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "unknown claim with proposal", current: must(plan.New("Plan unknown proposal", must(plan.NewGoal("goal unknown proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unknown proposal"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{99}, ProposedPlans: []plan.Plan{must(plan.New("Proposed unknown", must(plan.NewGoal("goal unknown proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unknown proposal"))}, []plan.Task{must(plan.NewTask("task unknown proposal"))}, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "duplicate claim", current: must(plan.New("Plan duplicate", must(plan.NewGoal("goal duplicate")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept duplicate"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Retain, Retain}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "contradictory claim", current: must(plan.New("Plan contradictory", must(plan.NewGoal("goal contradictory")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept contradictory"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Complete, Revise}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "retain with proposal", current: must(plan.New("Plan retain", must(plan.NewGoal("goal retain")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept retain"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Retain}, ProposedPlans: []plan.Plan{must(plan.New("Proposed retain", must(plan.NewGoal("goal retain")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept retain"))}, nil, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "complete with proposal", current: must(plan.New("Plan complete proposal", must(plan.NewGoal("goal complete proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept complete proposal"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Complete}, ProposedPlans: []plan.Plan{must(plan.New("Proposed complete", must(plan.NewGoal("goal complete proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept complete proposal"))}, []plan.Task{must(plan.NewTask("task complete proposal"))}, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "insufficient information with proposal", current: must(plan.New("Plan insufficient proposal", must(plan.NewGoal("goal insufficient proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept insufficient proposal"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{InsufficientInformation}, ProposedPlans: []plan.Plan{must(plan.New("Proposed insufficient", must(plan.NewGoal("goal insufficient proposal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept insufficient proposal"))}, []plan.Task{must(plan.NewTask("task insufficient proposal"))}, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "revise without proposal", current: must(plan.New("Plan missing", must(plan.NewGoal("goal missing")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept missing"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Revise}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "revise with multiple proposals", current: must(plan.New("Plan multiple", must(plan.NewGoal("goal multiple")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept multiple"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{must(plan.New("Proposed multiple one", must(plan.NewGoal("goal multiple")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept multiple"))}, nil, nil)), must(plan.New("Proposed multiple two", must(plan.NewGoal("goal multiple")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept multiple"))}, nil, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "revise with zero proposal", current: must(plan.New("Plan zero", must(plan.NewGoal("goal zero")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept zero"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{{}}}, want: failureObservation{FailureCode: AIContractFailure}},
		{name: "revise with exact equivalent proposal", current: must(plan.New("Plan equivalent", must(plan.NewGoal("goal equivalent")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept equivalent"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{must(plan.New("Plan equivalent", must(plan.NewGoal("goal equivalent")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept equivalent"))}, nil, nil))}}, want: failureObservation{FailureCode: AIContractFailure}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Assess(context.Background(), tt.current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
				return tt.response, nil
			})
			proposed, hasProposed := got.ProposedPlan()
			var failure *FailureError
			gotCode := FailureCode("")
			if errors.As(err, &failure) {
				gotCode = failure.Code()
			}
			gotPublic := failureObservation{FailureCode: gotCode, Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("invalid response observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessInputAndBoundaryFailures(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		current  plan.Plan
		assessor Assessor[int]
		want     failureObservation
	}{
		{name: "nil context", ctx: nil, current: must(plan.New("Plan nil context", must(plan.NewGoal("goal nil context")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept nil context"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{FailureCode: InvalidContext, AssessorCalls: 0}},
		{name: "invalid plan", ctx: context.Background(), current: plan.Plan{}, assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{FailureCode: InvalidCurrentPlan, AssessorCalls: 0}},
		{name: "nil assessor", ctx: context.Background(), current: must(plan.New("Plan nil assessor", must(plan.NewGoal("goal nil assessor")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept nil assessor"))}, nil, nil)), want: failureObservation{FailureCode: InvalidAssessor, AssessorCalls: 0}},
		{name: "provider unavailable", ctx: context.Background(), current: must(plan.New("Plan unavailable", must(plan.NewGoal("goal unavailable")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unavailable"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{}, errors.New("provider unavailable")
		}, want: failureObservation{FailureCode: AIBoundaryFailure, AssessorCalls: 1}},
		{name: "provider timeout", ctx: context.Background(), current: must(plan.New("Plan timeout", must(plan.NewGoal("goal timeout")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept timeout"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{}, context.DeadlineExceeded
		}, want: failureObservation{FailureCode: AIBoundaryFailure, AssessorCalls: 1}},
		{name: "unrelated context error", ctx: context.Background(), current: must(plan.New("Plan unrelated context", must(plan.NewGoal("goal unrelated context")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unrelated context"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{}, context.Canceled
		}, want: failureObservation{FailureCode: AIBoundaryFailure, AssessorCalls: 1}},
		{name: "untranslatable", ctx: context.Background(), current: must(plan.New("Plan untranslatable", must(plan.NewGoal("goal untranslatable")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept untranslatable"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{}, fmt.Errorf("decode: %w", ErrUntranslatableAIResponse)
		}, want: failureObservation{FailureCode: AIContractFailure, AssessorCalls: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			assessor := tt.assessor
			if assessor != nil {
				original := assessor
				assessor = func(ctx context.Context, current plan.Plan, observations int) (AssessorResponse, error) {
					calls++
					return original(ctx, current, observations)
				}
			}
			got, err := Assess(tt.ctx, tt.current, 0, assessor)
			proposed, hasProposed := got.ProposedPlan()
			var failure *FailureError
			gotCode := FailureCode("")
			if errors.As(err, &failure) {
				gotCode = failure.Code()
			}
			gotPublic := failureObservation{FailureCode: gotCode, Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}, AssessorCalls: calls}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("input/boundary observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessInputPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		ctx      func() context.Context
		current  plan.Plan
		assessor Assessor[int]
		want     failureObservation
	}{
		{name: "pre-cancel wins over invalid plan", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }, current: plan.Plan{}, assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{ContextCanceled: true}},
		{name: "pre-expiry wins over invalid plan", ctx: func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}, current: plan.Plan{}, assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{ContextDeadline: true}},
		{name: "pre-cancel wins over nil assessor", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }, current: must(plan.New("Plan cancelled", must(plan.NewGoal("goal cancelled")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept cancelled"))}, nil, nil)), want: failureObservation{ContextCanceled: true}},
		{name: "pre-cancel valid input", ctx: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }, current: must(plan.New("Plan pre-cancel valid", must(plan.NewGoal("goal pre-cancel valid")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept pre-cancel valid"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{ContextCanceled: true}},
		{name: "pre-expiry valid input", ctx: func() context.Context {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return ctx
		}, current: must(plan.New("Plan pre-expiry valid", must(plan.NewGoal("goal pre-expiry valid")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept pre-expiry valid"))}, nil, nil)), assessor: func(context.Context, plan.Plan, int) (AssessorResponse, error) {
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		}, want: failureObservation{ContextDeadline: true}},
		{name: "invalid plan plus nil assessor", ctx: func() context.Context { return context.Background() }, current: plan.Plan{}, want: failureObservation{FailureCode: InvalidCurrentPlan}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			assessor := tt.assessor
			if assessor != nil {
				original := assessor
				assessor = func(ctx context.Context, current plan.Plan, observations int) (AssessorResponse, error) {
					calls++
					return original(ctx, current, observations)
				}
			}
			got, err := Assess(tt.ctx(), tt.current, 0, assessor)
			proposed, hasProposed := got.ProposedPlan()
			var failure *FailureError
			gotCode := FailureCode("")
			if errors.As(err, &failure) {
				gotCode = failure.Code()
			}
			gotPublic := failureObservation{FailureCode: gotCode, ContextCanceled: errors.Is(err, context.Canceled), ContextDeadline: errors.Is(err, context.DeadlineExceeded), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}, AssessorCalls: calls}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("input precedence observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessCancellationBeforeFinalReturnOverridesBoundaryFailure(t *testing.T) {
	current := must(plan.New("Plan return cancellation", must(plan.NewGoal("goal return cancellation")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept return cancellation"))}, nil, nil))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	classified := make(chan struct{})
	providerErr := cancelDuringError{cancel: cancel, classified: classified}
	got, err := Assess(ctx, current, "observation return cancellation", func(context.Context, plan.Plan, string) (AssessorResponse, error) {
		return AssessorResponse{}, providerErr
	})
	select {
	case <-classified:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for cancellation classification")
	}
	proposed, hasProposed := got.ProposedPlan()
	gotPublic := failureObservation{ContextCanceled: errors.Is(err, context.Canceled), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
	wantPublic := failureObservation{ContextCanceled: true, Assessment: publicAssessment{Outcome: 0, AssessedPlan: publicPlan{Valid: false}, ProposedPlan: publicPlan{Valid: false}, HasProposed: false}}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("cancellation classification mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessBoundaryErrorPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		current plan.Plan
		err     error
		want    failureObservation
	}{
		{name: "joined untranslatable wins", current: must(plan.New("Plan joined", must(plan.NewGoal("goal joined")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept joined"))}, nil, nil)), err: errors.Join(errors.New("provider joined"), ErrUntranslatableAIResponse), want: failureObservation{FailureCode: AIContractFailure}},
		{name: "response plus error ignores response", current: must(plan.New("Plan response error", must(plan.NewGoal("goal response error")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept response error"))}, nil, nil)), err: errors.New("provider response error"), want: failureObservation{FailureCode: AIBoundaryFailure}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Assess(context.Background(), tt.current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
				return AssessorResponse{Claims: []Outcome{Complete}}, tt.err
			})
			proposed, hasProposed := got.ProposedPlan()
			var failure *FailureError
			gotCode := FailureCode("")
			if errors.As(err, &failure) {
				gotCode = failure.Code()
			}
			gotPublic := failureObservation{FailureCode: gotCode, RawErrorUnwrapped: errors.Is(err, tt.err), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("boundary observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessJoinedUntranslatableDoesNotExposeProviderError(t *testing.T) {
	current := must(plan.New("Plan joined raw", must(plan.NewGoal("goal joined raw")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept joined raw"))}, nil, nil))
	providerErr := errors.New("provider raw failure")
	got, err := Assess(context.Background(), current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
		return AssessorResponse{}, errors.Join(providerErr, ErrUntranslatableAIResponse)
	})
	var failure *FailureError
	gotCode := FailureCode("")
	if errors.As(err, &failure) {
		gotCode = failure.Code()
	}
	proposed, hasProposed := got.ProposedPlan()
	gotPublic := failureObservation{FailureCode: gotCode, RawErrorUnwrapped: errors.Is(err, providerErr) || errors.Is(err, ErrUntranslatableAIResponse), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
	wantPublic := failureObservation{FailureCode: AIContractFailure, RawErrorUnwrapped: false, Assessment: publicAssessment{Outcome: 0, AssessedPlan: publicPlan{Valid: false}, ProposedPlan: publicPlan{Valid: false}, HasProposed: false}}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("joined-error observation mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessProviderFailuresDoNotUnwrapRawErrors(t *testing.T) {
	tests := []struct {
		name     string
		current  plan.Plan
		rawError error
		want     failureObservation
	}{
		{name: "provider unavailable", current: must(plan.New("Plan unavailable raw", must(plan.NewGoal("goal unavailable raw")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unavailable raw"))}, nil, nil)), rawError: errors.New("provider unavailable raw"), want: failureObservation{FailureCode: AIBoundaryFailure}},
		{name: "provider timeout", current: must(plan.New("Plan timeout raw", must(plan.NewGoal("goal timeout raw")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept timeout raw"))}, nil, nil)), rawError: context.DeadlineExceeded, want: failureObservation{FailureCode: AIBoundaryFailure}},
		{name: "unrelated context error", current: must(plan.New("Plan unrelated raw", must(plan.NewGoal("goal unrelated raw")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept unrelated raw"))}, nil, nil)), rawError: context.Canceled, want: failureObservation{FailureCode: AIBoundaryFailure}},
		{name: "wrapped untranslatable", current: must(plan.New("Plan untranslatable raw", must(plan.NewGoal("goal untranslatable raw")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept untranslatable raw"))}, nil, nil)), rawError: fmt.Errorf("decode raw: %w", ErrUntranslatableAIResponse), want: failureObservation{FailureCode: AIContractFailure}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Assess(context.Background(), tt.current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
				return AssessorResponse{}, tt.rawError
			})
			proposed, hasProposed := got.ProposedPlan()
			var failure *FailureError
			gotCode := FailureCode("")
			if errors.As(err, &failure) {
				gotCode = failure.Code()
			}
			gotPublic := failureObservation{FailureCode: gotCode, RawErrorUnwrapped: errors.Is(err, tt.rawError), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("provider failure observation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssessObservationTypeAndResponseSnapshot(t *testing.T) {
	type observations struct {
		Schedule string
		Progress int
		Coverage []string
		TaskSize int
	}
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))
	proposedPlan := must(plan.New("Revised", must(plan.NewGoal("goal 2")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept")), must(plan.NewAcceptanceCondition("accept 2"))}, []plan.Task{must(plan.NewTask("task revised"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-02")); return &date }()))
	wantObservations := observations{Schedule: "delayed", Progress: 3, Coverage: []string{"goal", "accept"}, TaskSize: 2}
	response := AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposedPlan}}
	got, err := Assess(context.Background(), current, wantObservations, func(_ context.Context, _ plan.Plan, got observations) (AssessorResponse, error) {
		if diff := cmp.Diff(wantObservations, got); diff != "" {
			t.Errorf("observations changed (-want +got):\n%s", diff)
		}
		return response, nil
	})
	response.Claims[0] = Complete
	response.ProposedPlans[0] = plan.Plan{}
	gotAssessed := got.AssessedPlan()
	gotAssessedDate, gotAssessedHasDate := gotAssessed.TargetDate()
	returnedProposed, hasProposed := got.ProposedPlan()
	gotDate, gotHasDate := returnedProposed.TargetDate()
	wantAssessedDate, wantAssessedHasDate := current.TargetDate()
	wantDate, wantHasDate := proposedPlan.TargetDate()
	gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotAssessedDate.String(), HasDate: gotAssessedHasDate}, ProposedPlan: publicPlan{Valid: returnedProposed.IsValid(), Name: returnedProposed.Name(), Goal: returnedProposed.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasProposed: hasProposed, HasError: err != nil}
	wantPublic := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: current.IsValid(), Name: current.Name(), Goal: current.Goal().Text(), TargetDate: wantAssessedDate.String(), HasDate: wantAssessedHasDate}, ProposedPlan: publicPlan{Valid: proposedPlan.IsValid(), Name: proposedPlan.Name(), Goal: proposedPlan.Goal().Text(), TargetDate: wantDate.String(), HasDate: wantHasDate}, HasProposed: true}
	for _, condition := range gotAssessed.AcceptanceConditions() {
		gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range current.AcceptanceConditions() {
		wantPublic.AssessedPlan.Conditions = append(wantPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, task := range gotAssessed.Tasks() {
		gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range current.Tasks() {
		wantPublic.AssessedPlan.Tasks = append(wantPublic.AssessedPlan.Tasks, task.Name())
	}
	for _, condition := range returnedProposed.AcceptanceConditions() {
		gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
	}
	for _, condition := range proposedPlan.AcceptanceConditions() {
		wantPublic.ProposedPlan.Conditions = append(wantPublic.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range returnedProposed.Tasks() {
		gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
	}
	for _, task := range proposedPlan.Tasks() {
		wantPublic.ProposedPlan.Tasks = append(wantPublic.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("assessment snapshot observation mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessForwardsScheduleAndProgressObservation(t *testing.T) {
	type scheduleProgress struct {
		Schedule string
		Progress int
	}
	current := must(plan.New("Plan schedule", must(plan.NewGoal("goal schedule")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept schedule"))}, nil, nil))
	proposed := must(plan.New("Plan schedule revised", must(plan.NewGoal("goal schedule")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept schedule"))}, nil, nil))
	want := scheduleProgress{Schedule: "delayed", Progress: 3}
	resultWant := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan schedule", Goal: "goal schedule", Conditions: []string{"accept schedule"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan schedule revised", Goal: "goal schedule", Conditions: []string{"accept schedule"}}, HasProposed: true}
	got, err := Assess(context.Background(), current, want, func(_ context.Context, _ plan.Plan, observations scheduleProgress) (AssessorResponse, error) {
		if diff := cmp.Diff(want, observations); diff != "" {
			t.Errorf("schedule/progress observation mismatch (-want +got):\n%s", diff)
		}
		return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposed}}, nil
	})
	gotAssessed := got.AssessedPlan()
	gotAssessedDate, gotAssessedHasDate := gotAssessed.TargetDate()
	returned, gotHasProposed := got.ProposedPlan()
	gotDate, gotHasDate := returned.TargetDate()
	gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotAssessedDate.String(), HasDate: gotAssessedHasDate}, ProposedPlan: publicPlan{Valid: returned.IsValid(), Name: returned.Name(), Goal: returned.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasProposed: gotHasProposed, HasError: err != nil}
	for _, condition := range gotAssessed.AcceptanceConditions() {
		gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range returned.AcceptanceConditions() {
		gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range gotAssessed.Tasks() {
		gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range returned.Tasks() {
		gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(resultWant, gotPublic); diff != "" {
		t.Errorf("schedule/progress result mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessForwardsPlanCoverageObservation(t *testing.T) {
	type coverage struct {
		Goal       string
		Conditions []string
		Tasks      []string
	}
	current := must(plan.New("Plan coverage", must(plan.NewGoal("goal coverage")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept coverage"))}, nil, nil))
	proposed := must(plan.New("Plan coverage revised", must(plan.NewGoal("goal coverage")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept coverage")), must(plan.NewAcceptanceCondition("extra coverage"))}, nil, nil))
	want := coverage{Goal: "goal coverage", Conditions: []string{"accept coverage"}, Tasks: []string{"task coverage"}}
	resultWant := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan coverage", Goal: "goal coverage", Conditions: []string{"accept coverage"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan coverage revised", Goal: "goal coverage", Conditions: []string{"accept coverage", "extra coverage"}}, HasProposed: true}
	got, err := Assess(context.Background(), current, want, func(_ context.Context, _ plan.Plan, observations coverage) (AssessorResponse, error) {
		if diff := cmp.Diff(want, observations); diff != "" {
			t.Errorf("coverage observation mismatch (-want +got):\n%s", diff)
		}
		return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposed}}, nil
	})
	gotAssessed := got.AssessedPlan()
	gotAssessedDate, gotAssessedHasDate := gotAssessed.TargetDate()
	returned, gotHasProposed := got.ProposedPlan()
	gotDate, gotHasDate := returned.TargetDate()
	gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotAssessedDate.String(), HasDate: gotAssessedHasDate}, ProposedPlan: publicPlan{Valid: returned.IsValid(), Name: returned.Name(), Goal: returned.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasProposed: gotHasProposed, HasError: err != nil}
	for _, condition := range gotAssessed.AcceptanceConditions() {
		gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range returned.AcceptanceConditions() {
		gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range gotAssessed.Tasks() {
		gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range returned.Tasks() {
		gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(resultWant, gotPublic); diff != "" {
		t.Errorf("coverage result mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessForwardsTaskAmountAndGranularityObservation(t *testing.T) {
	type taskAmount struct {
		TaskCount   int
		Granularity string
	}
	current := must(plan.New("Plan task amount", must(plan.NewGoal("goal task amount")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept task amount"))}, nil, nil))
	proposed := must(plan.New("Plan task amount revised", must(plan.NewGoal("goal task amount")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept task amount"))}, []plan.Task{must(plan.NewTask("split task"))}, nil))
	want := taskAmount{TaskCount: 12, Granularity: "too coarse"}
	resultWant := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan task amount", Goal: "goal task amount", Conditions: []string{"accept task amount"}}, ProposedPlan: publicPlan{Valid: true, Name: "Plan task amount revised", Goal: "goal task amount", Conditions: []string{"accept task amount"}, Tasks: []string{"split task"}}, HasProposed: true}
	got, err := Assess(context.Background(), current, want, func(_ context.Context, _ plan.Plan, observations taskAmount) (AssessorResponse, error) {
		if diff := cmp.Diff(want, observations); diff != "" {
			t.Errorf("task amount observation mismatch (-want +got):\n%s", diff)
		}
		return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposed}}, nil
	})
	gotAssessed := got.AssessedPlan()
	gotAssessedDate, gotAssessedHasDate := gotAssessed.TargetDate()
	returned, gotHasProposed := got.ProposedPlan()
	gotDate, gotHasDate := returned.TargetDate()
	gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotAssessedDate.String(), HasDate: gotAssessedHasDate}, ProposedPlan: publicPlan{Valid: returned.IsValid(), Name: returned.Name(), Goal: returned.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasProposed: gotHasProposed, HasError: err != nil}
	for _, condition := range gotAssessed.AcceptanceConditions() {
		gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range returned.AcceptanceConditions() {
		gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range gotAssessed.Tasks() {
		gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range returned.Tasks() {
		gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(resultWant, gotPublic); diff != "" {
		t.Errorf("task amount result mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessmentPublicAccessorsAndZeroContract(t *testing.T) {
	var zero Assessment
	zeroProposed, zeroHasProposed := zero.ProposedPlan()
	zeroPublic := publicAssessment{Outcome: zero.Outcome(), AssessedPlan: publicPlan{Valid: zero.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: zeroProposed.IsValid()}, HasProposed: zeroHasProposed}
	if diff := cmp.Diff(publicAssessment{}, zeroPublic); diff != "" {
		t.Errorf("zero Assessment mismatch (-want +got):\n%s", diff)
	}

	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	proposedPlan := must(plan.New("Revised", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	got, err := Assess(context.Background(), current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
		return AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{proposedPlan}}, nil
	})
	gotProposed, ok := got.ProposedPlan()
	gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid(), Name: got.AssessedPlan().Name(), Goal: got.AssessedPlan().Goal().Text()}, ProposedPlan: publicPlan{Valid: gotProposed.IsValid(), Name: gotProposed.Name(), Goal: gotProposed.Goal().Text()}, HasProposed: ok, HasError: err != nil}
	wantPublic := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}}, ProposedPlan: publicPlan{Valid: true, Name: "Revised", Goal: "goal", Conditions: []string{"accept"}}, HasProposed: true}
	for _, condition := range got.AssessedPlan().AcceptanceConditions() {
		gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range gotProposed.AcceptanceConditions() {
		gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
	}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("public accessor mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessSuccessiveCallsDoNotReuseRuntimeState(t *testing.T) {
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))
	observation := struct {
		Schedule string
		Progress int
	}{Schedule: "delayed", Progress: 3}
	response := AssessorResponse{Claims: []Outcome{Revise}, ProposedPlans: []plan.Plan{must(plan.New("Revised", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }()))}}
	wantRepeat := publicAssessment{Outcome: Revise, AssessedPlan: publicPlan{Valid: true, Name: "Plan", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"task"}, TargetDate: "2026-09-01", HasDate: true}, ProposedPlan: publicPlan{Valid: true, Name: "Revised", Goal: "goal", Conditions: []string{"accept"}, Tasks: []string{"task"}, TargetDate: "2026-09-01", HasDate: true}, HasProposed: true}
	first, err := Assess(context.Background(), current, observation, func(context.Context, plan.Plan, struct {
		Schedule string
		Progress int
	}) (AssessorResponse, error) {
		return response, nil
	})
	firstAssessed := first.AssessedPlan()
	firstAssessedDate, firstAssessedHasDate := firstAssessed.TargetDate()
	firstProposed, firstHasProposed := first.ProposedPlan()
	firstProposedDate, firstProposedHasDate := firstProposed.TargetDate()
	firstGot := publicAssessment{Outcome: first.Outcome(), AssessedPlan: publicPlan{Valid: firstAssessed.IsValid(), Name: firstAssessed.Name(), Goal: firstAssessed.Goal().Text(), TargetDate: firstAssessedDate.String(), HasDate: firstAssessedHasDate}, ProposedPlan: publicPlan{Valid: firstProposed.IsValid(), Name: firstProposed.Name(), Goal: firstProposed.Goal().Text(), TargetDate: firstProposedDate.String(), HasDate: firstProposedHasDate}, HasProposed: firstHasProposed, HasError: err != nil}
	for _, condition := range firstAssessed.AcceptanceConditions() {
		firstGot.AssessedPlan.Conditions = append(firstGot.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range firstProposed.AcceptanceConditions() {
		firstGot.ProposedPlan.Conditions = append(firstGot.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range firstAssessed.Tasks() {
		firstGot.AssessedPlan.Tasks = append(firstGot.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range firstProposed.Tasks() {
		firstGot.ProposedPlan.Tasks = append(firstGot.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(wantRepeat, firstGot); diff != "" {
		t.Errorf("first call observation mismatch (-want +got):\n%s", diff)
	}
	_ = first
	second, err := Assess(context.Background(), current, observation, func(context.Context, plan.Plan, struct {
		Schedule string
		Progress int
	}) (AssessorResponse, error) {
		return response, nil
	})
	secondAssessed := second.AssessedPlan()
	secondAssessedDate, secondAssessedHasDate := secondAssessed.TargetDate()
	secondProposed, secondHasProposed := second.ProposedPlan()
	secondProposedDate, secondProposedHasDate := secondProposed.TargetDate()
	secondGot := publicAssessment{Outcome: second.Outcome(), AssessedPlan: publicPlan{Valid: secondAssessed.IsValid(), Name: secondAssessed.Name(), Goal: secondAssessed.Goal().Text(), TargetDate: secondAssessedDate.String(), HasDate: secondAssessedHasDate}, ProposedPlan: publicPlan{Valid: secondProposed.IsValid(), Name: secondProposed.Name(), Goal: secondProposed.Goal().Text(), TargetDate: secondProposedDate.String(), HasDate: secondProposedHasDate}, HasProposed: secondHasProposed, HasError: err != nil}
	for _, condition := range secondAssessed.AcceptanceConditions() {
		secondGot.AssessedPlan.Conditions = append(secondGot.AssessedPlan.Conditions, condition.Statement())
	}
	for _, condition := range secondProposed.AcceptanceConditions() {
		secondGot.ProposedPlan.Conditions = append(secondGot.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range secondAssessed.Tasks() {
		secondGot.AssessedPlan.Tasks = append(secondGot.AssessedPlan.Tasks, task.Name())
	}
	for _, task := range secondProposed.Tasks() {
		secondGot.ProposedPlan.Tasks = append(secondGot.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(wantRepeat, secondGot); diff != "" {
		t.Errorf("repeat call observation mismatch (-want +got):\n%s", diff)
	}

	distinctCurrent := must(plan.New("Plan B", must(plan.NewGoal("goal B")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept B")), must(plan.NewAcceptanceCondition("accept B2"))}, []plan.Task{must(plan.NewTask("task B"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-10-02")); return &date }()))
	distinctObservation := struct {
		Schedule string
		Progress int
	}{Schedule: "on track", Progress: 8}
	distinct, err := Assess(context.Background(), distinctCurrent, distinctObservation, func(context.Context, plan.Plan, struct {
		Schedule string
		Progress int
	}) (AssessorResponse, error) {
		return AssessorResponse{Claims: []Outcome{Complete}}, nil
	})
	distinctWant := publicAssessment{Outcome: Complete, AssessedPlan: publicPlan{Valid: true, Name: "Plan B", Goal: "goal B", Conditions: []string{"accept B", "accept B2"}, Tasks: []string{"task B"}, TargetDate: "2026-10-02", HasDate: true}}
	distinctAssessed := distinct.AssessedPlan()
	distinctAssessedDate, distinctAssessedHasDate := distinctAssessed.TargetDate()
	distinctProposed, distinctHasProposed := distinct.ProposedPlan()
	distinctProposedDate, distinctProposedHasDate := distinctProposed.TargetDate()
	distinctGot := publicAssessment{Outcome: distinct.Outcome(), AssessedPlan: publicPlan{Valid: distinctAssessed.IsValid(), Name: distinctAssessed.Name(), Goal: distinctAssessed.Goal().Text(), TargetDate: distinctAssessedDate.String(), HasDate: distinctAssessedHasDate}, ProposedPlan: publicPlan{Valid: distinctProposed.IsValid(), Name: distinctProposed.Name(), Goal: distinctProposed.Goal().Text(), TargetDate: distinctProposedDate.String(), HasDate: distinctProposedHasDate}, HasProposed: distinctHasProposed, HasError: err != nil}
	for _, condition := range distinctAssessed.AcceptanceConditions() {
		distinctGot.AssessedPlan.Conditions = append(distinctGot.AssessedPlan.Conditions, condition.Statement())
	}
	for _, task := range distinctAssessed.Tasks() {
		distinctGot.AssessedPlan.Tasks = append(distinctGot.AssessedPlan.Tasks, task.Name())
	}
	for _, condition := range distinctProposed.AcceptanceConditions() {
		distinctGot.ProposedPlan.Conditions = append(distinctGot.ProposedPlan.Conditions, condition.Statement())
	}
	for _, task := range distinctProposed.Tasks() {
		distinctGot.ProposedPlan.Tasks = append(distinctGot.ProposedPlan.Tasks, task.Name())
	}
	if diff := cmp.Diff(distinctWant, distinctGot); diff != "" {
		t.Errorf("distinct B call mismatch (-want +got):\n%s", diff)
	}
}

func TestAssessConcurrentCorrelationIsolation(t *testing.T) {
	entered := make(chan string, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup
	completed := make(chan struct{}, 2)
	errs := make(chan error, 2)
	for _, item := range []struct {
		current     plan.Plan
		observation string
		response    AssessorResponse
		want        publicAssessment
	}{
		{current: must(plan.New("Plan A", must(plan.NewGoal("goal A")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept A"))}, []plan.Task{must(plan.NewTask("task A"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-01")); return &date }())), observation: "retain A", response: AssessorResponse{Claims: []Outcome{Retain}}, want: publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan A", Goal: "goal A", Conditions: []string{"accept A"}, Tasks: []string{"task A"}, TargetDate: "2026-09-01", HasDate: true}}},
		{current: must(plan.New("Plan B", must(plan.NewGoal("goal B")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept B"))}, []plan.Task{must(plan.NewTask("task B"))}, func() *plan.TargetDate { date := must(plan.ParseTargetDate("2026-09-02")); return &date }())), observation: "complete B", response: AssessorResponse{Claims: []Outcome{Complete}}, want: publicAssessment{Outcome: Complete, AssessedPlan: publicPlan{Valid: true, Name: "Plan B", Goal: "goal B", Conditions: []string{"accept B"}, Tasks: []string{"task B"}, TargetDate: "2026-09-02", HasDate: true}}},
	} {
		item := item
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				completed <- struct{}{}
			}()
			got, err := Assess(context.Background(), item.current, item.observation, func(_ context.Context, current plan.Plan, observation string) (AssessorResponse, error) {
				select {
				case entered <- observation:
				case <-time.After(5 * time.Second):
					return AssessorResponse{}, errors.New("concurrent assessor entry timed out")
				}
				select {
				case <-release:
				case <-time.After(5 * time.Second):
					return AssessorResponse{}, errors.New("concurrent assessor release timed out")
				}
				if observation != item.observation || current.Name() != item.current.Name() {
					return AssessorResponse{}, errors.New("correlated input crossed calls")
				}
				return item.response, nil
			})
			gotAssessed := got.AssessedPlan()
			gotDate, gotHasDate := gotAssessed.TargetDate()
			gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasError: err != nil}
			for _, condition := range gotAssessed.AcceptanceConditions() {
				gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotAssessed.Tasks() {
				gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
			}
			proposed, hasProposed := got.ProposedPlan()
			proposedDate, proposedHasDate := proposed.TargetDate()
			gotPublic.HasProposed = hasProposed
			gotPublic.ProposedPlan = publicPlan{Valid: proposed.IsValid(), Name: proposed.Name(), Goal: proposed.Goal().Text(), TargetDate: proposedDate.String(), HasDate: proposedHasDate}
			for _, condition := range proposed.AcceptanceConditions() {
				gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
			}
			for _, task := range proposed.Tasks() {
				gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
			}
			if diff := cmp.Diff(item.want, gotPublic); diff != "" {
				errs <- fmt.Errorf("correlated public result mismatch: %s", diff)
			}
		}()
	}
	var firstEntered, secondEntered string
	select {
	case firstEntered = <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		for range 2 {
			select {
			case <-completed:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for concurrent assessors after release")
			}
		}
		wg.Wait()
		t.Fatal("timed out waiting for first concurrent assessor")
	}
	select {
	case secondEntered = <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		for range 2 {
			select {
			case <-completed:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for concurrent assessors after release")
			}
		}
		wg.Wait()
		t.Fatal("timed out waiting for second concurrent assessor")
	}
	if firstEntered == secondEntered {
		close(release)
		for range 2 {
			select {
			case <-completed:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for concurrent assessors after distinct-value failure")
			}
		}
		wg.Wait()
		t.Fatalf("concurrent rendezvous observations = %q and %q, want distinct calls", firstEntered, secondEntered)
	}
	close(release)
	for range 2 {
		select {
		case <-completed:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for concurrent assessors")
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestAssessCancellationAndStatelessConcurrentCalls(t *testing.T) {
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	result := make(chan struct {
		assessment Assessment
		err        error
	}, 1)
	go func() {
		assessment, err := Assess(ctx, current, 0, func(ctx context.Context, _ plan.Plan, _ int) (AssessorResponse, error) {
			close(started)
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				return AssessorResponse{}, errors.New("blocked cancellation assessor timed out")
			}
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		})
		result <- struct {
			assessment Assessment
			err        error
		}{assessment, err}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for blocked assessor to start")
	}
	cancel()
	var gotCanceled struct {
		assessment Assessment
		err        error
	}
	select {
	case gotCanceled = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for canceled assessment")
	}
	proposed, hasProposed := gotCanceled.assessment.ProposedPlan()
	gotPublic := failureObservation{ContextCanceled: errors.Is(gotCanceled.err, context.Canceled), Assessment: publicAssessment{Outcome: gotCanceled.assessment.Outcome(), AssessedPlan: publicPlan{Valid: gotCanceled.assessment.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
	wantPublic := failureObservation{ContextCanceled: true, Assessment: publicAssessment{Outcome: 0, AssessedPlan: publicPlan{Valid: false}, ProposedPlan: publicPlan{Valid: false}, HasProposed: false}}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("blocked cancellation observation mismatch (-want +got):\n%s", diff)
	}

	var wg sync.WaitGroup
	completed := make(chan struct{}, 2)
	errs := make(chan error, 2)
	for _, item := range []struct {
		current     plan.Plan
		observation string
		want        publicAssessment
	}{
		{current: must(plan.New("Plan concurrent first", must(plan.NewGoal("goal concurrent first")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept concurrent first"))}, []plan.Task{must(plan.NewTask("task concurrent first"))}, nil)), observation: "first", want: publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan concurrent first", Goal: "goal concurrent first", Conditions: []string{"accept concurrent first"}, Tasks: []string{"task concurrent first"}}}},
		{current: must(plan.New("Plan concurrent second", must(plan.NewGoal("goal concurrent second")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept concurrent second"))}, []plan.Task{must(plan.NewTask("task concurrent second"))}, nil)), observation: "second", want: publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan concurrent second", Goal: "goal concurrent second", Conditions: []string{"accept concurrent second"}, Tasks: []string{"task concurrent second"}}}},
	} {
		item := item
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				completed <- struct{}{}
			}()
			got, err := Assess(context.Background(), item.current, item.observation, func(context.Context, plan.Plan, string) (AssessorResponse, error) {
				return AssessorResponse{Claims: []Outcome{Retain}}, nil
			})
			gotAssessed := got.AssessedPlan()
			gotDate, gotHasDate := gotAssessed.TargetDate()
			gotPublic := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAssessed.IsValid(), Name: gotAssessed.Name(), Goal: gotAssessed.Goal().Text(), TargetDate: gotDate.String(), HasDate: gotHasDate}, HasError: err != nil}
			for _, condition := range gotAssessed.AcceptanceConditions() {
				gotPublic.AssessedPlan.Conditions = append(gotPublic.AssessedPlan.Conditions, condition.Statement())
			}
			for _, task := range gotAssessed.Tasks() {
				gotPublic.AssessedPlan.Tasks = append(gotPublic.AssessedPlan.Tasks, task.Name())
			}
			proposed, hasProposed := got.ProposedPlan()
			proposedDate, proposedHasDate := proposed.TargetDate()
			gotPublic.HasProposed = hasProposed
			gotPublic.ProposedPlan = publicPlan{Valid: proposed.IsValid(), Name: proposed.Name(), Goal: proposed.Goal().Text(), TargetDate: proposedDate.String(), HasDate: proposedHasDate}
			for _, condition := range proposed.AcceptanceConditions() {
				gotPublic.ProposedPlan.Conditions = append(gotPublic.ProposedPlan.Conditions, condition.Statement())
			}
			for _, task := range proposed.Tasks() {
				gotPublic.ProposedPlan.Tasks = append(gotPublic.ProposedPlan.Tasks, task.Name())
			}
			if diff := cmp.Diff(item.want, gotPublic); diff != "" {
				errs <- fmt.Errorf("concurrent public result mismatch: %s", diff)
			}
		}()
	}
	for range 2 {
		select {
		case <-completed:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for stateless concurrent assessors")
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestAssessCancellationSamplesOverrideBoundaryResults(t *testing.T) {
	tests := []struct {
		name     string
		current  plan.Plan
		response AssessorResponse
		err      error
		want     failureObservation
	}{
		{name: "cancel plus provider error", current: must(plan.New("Plan cancel provider", must(plan.NewGoal("goal cancel provider")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept cancel provider"))}, nil, nil)), err: errors.New("provider"), want: failureObservation{ContextCanceled: true}},
		{name: "cancel plus untranslatable", current: must(plan.New("Plan cancel untranslatable", must(plan.NewGoal("goal cancel untranslatable")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept cancel untranslatable"))}, nil, nil)), err: ErrUntranslatableAIResponse, want: failureObservation{ContextCanceled: true}},
		{name: "cancel plus valid response", current: must(plan.New("Plan cancel valid", must(plan.NewGoal("goal cancel valid")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept cancel valid"))}, nil, nil)), response: AssessorResponse{Claims: []Outcome{Retain}}, want: failureObservation{ContextCanceled: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			gotResult := make(chan error, 1)
			gotAssessment := make(chan Assessment, 1)
			go func() {
				got, err := Assess(ctx, tt.current, 0, func(ctx context.Context, _ plan.Plan, _ int) (AssessorResponse, error) {
					close(started)
					select {
					case <-ctx.Done():
					case <-time.After(5 * time.Second):
						return AssessorResponse{}, errors.New("cancellation assessor timed out")
					}
					return tt.response, tt.err
				})
				gotAssessment <- got
				gotResult <- err
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("timed out waiting for cancellation assessor to start")
			}
			cancel()
			var err error
			var got Assessment
			select {
			case err = <-gotResult:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for cancellation error")
			}
			select {
			case got = <-gotAssessment:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for cancellation assessment")
			}
			proposed, hasProposed := got.ProposedPlan()
			gotPublic := failureObservation{ContextCanceled: errors.Is(err, context.Canceled), ContextDeadline: errors.Is(err, context.DeadlineExceeded), Assessment: publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: got.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
			if diff := cmp.Diff(tt.want, gotPublic); diff != "" {
				t.Errorf("cancellation observation mismatch (-want +got):\n%s", diff)
			}
		})
	}

	current := must(plan.New("Plan success after cancellation", must(plan.NewGoal("goal success after cancellation")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept success after cancellation"))}, nil, nil))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got, err := Assess(ctx, current, 0, func(context.Context, plan.Plan, int) (AssessorResponse, error) {
		return AssessorResponse{Claims: []Outcome{Retain}}, nil
	})
	wantSuccess := publicAssessment{Outcome: Retain, AssessedPlan: publicPlan{Valid: true, Name: "Plan success after cancellation", Goal: "goal success after cancellation", Conditions: []string{"accept success after cancellation"}}}
	gotSuccessPlan := got.AssessedPlan()
	gotSuccess := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotSuccessPlan.IsValid(), Name: gotSuccessPlan.Name(), Goal: gotSuccessPlan.Goal().Text()}, HasError: err != nil}
	for _, condition := range gotSuccessPlan.AcceptanceConditions() {
		gotSuccess.AssessedPlan.Conditions = append(gotSuccess.AssessedPlan.Conditions, condition.Statement())
	}
	cancel()
	gotAfterCancelPlan := got.AssessedPlan()
	gotAfterCancel := publicAssessment{Outcome: got.Outcome(), AssessedPlan: publicPlan{Valid: gotAfterCancelPlan.IsValid(), Name: gotAfterCancelPlan.Name(), Goal: gotAfterCancelPlan.Goal().Text()}, HasError: err != nil}
	for _, condition := range gotAfterCancelPlan.AcceptanceConditions() {
		gotAfterCancel.AssessedPlan.Conditions = append(gotAfterCancel.AssessedPlan.Conditions, condition.Statement())
	}
	if diff := cmp.Diff(wantSuccess, gotSuccess); diff != "" {
		t.Errorf("success observation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantSuccess, gotAfterCancel); diff != "" {
		t.Errorf("post-cancel Assessment changed (-want +got):\n%s", diff)
	}
}

func TestAssessDeadlineWhileAssessorBlocked(t *testing.T) {
	current := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := make(chan struct{})
	deadlineObserved := make(chan struct{})
	release := make(chan struct{})
	result := make(chan struct {
		assessment Assessment
		err        error
	}, 1)
	go func() {
		assessment, err := Assess(ctx, current, 0, func(ctx context.Context, _ plan.Plan, _ int) (AssessorResponse, error) {
			close(started)
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				return AssessorResponse{}, errors.New("deadline assessor timed out")
			}
			close(deadlineObserved)
			select {
			case <-release:
			case <-time.After(5 * time.Second):
				return AssessorResponse{}, errors.New("deadline release timed out")
			}
			return AssessorResponse{Claims: []Outcome{Retain}}, nil
		})
		result <- struct {
			assessment Assessment
			err        error
		}{assessment, err}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		cancel()
		close(release)
		t.Fatal("timed out waiting for deadline assessor to start")
	}
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		cancel()
		close(release)
		t.Fatal("timed out waiting for context deadline")
	}
	select {
	case <-deadlineObserved:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("timed out waiting for deadline observation")
	}
	close(release)
	var got struct {
		assessment Assessment
		err        error
	}
	select {
	case got = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for deadline result")
	}
	proposed, hasProposed := got.assessment.ProposedPlan()
	gotPublic := failureObservation{ContextDeadline: errors.Is(got.err, context.DeadlineExceeded), Assessment: publicAssessment{Outcome: got.assessment.Outcome(), AssessedPlan: publicPlan{Valid: got.assessment.AssessedPlan().IsValid()}, ProposedPlan: publicPlan{Valid: proposed.IsValid()}, HasProposed: hasProposed}}
	wantPublic := failureObservation{ContextDeadline: true, Assessment: publicAssessment{Outcome: 0, AssessedPlan: publicPlan{Valid: false}, ProposedPlan: publicPlan{Valid: false}, HasProposed: false}}
	if diff := cmp.Diff(wantPublic, gotPublic); diff != "" {
		t.Errorf("deadline observation mismatch (-want +got):\n%s", diff)
	}
}
