package githubplan_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestNewRepository(t *testing.T) {
	type args struct {
		owner string
		name  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "normal", args: args{owner: "owner", name: "repo"}},
		{name: "unusual accepted text", args: args{owner: " owner+.~ ", name: "repo_1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewRepository(tt.args.owner, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.args.owner, got.Owner()); diff != "" {
				t.Errorf("owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.args.name, got.Name()); diff != "" {
				t.Errorf("name mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewRepositoryValidation(t *testing.T) {
	tests := []struct {
		name, owner, repository string
		wantField               githubplan.Field
	}{
		{name: "blank owner", owner: " \u2003", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "invalid owner utf8", owner: string([]byte{0xff}), repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "slash owner", owner: "owner/name", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "owner CR", owner: "owner\r", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "owner LF", owner: "owner\n", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "owner NEL", owner: "owner\u0085", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "owner LS", owner: "owner\u2028", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "owner PS", owner: "owner\u2029", repository: "repo", wantField: githubplan.RepositoryOwnerField},
		{name: "blank repository", owner: "owner", repository: "\u2029", wantField: githubplan.RepositoryNameField},
		{name: "invalid repository utf8", owner: "owner", repository: string([]byte{0xff}), wantField: githubplan.RepositoryNameField},
		{name: "slash repository", owner: "owner", repository: "repo/name", wantField: githubplan.RepositoryNameField},
		{name: "repository CR", owner: "owner", repository: "repo\r", wantField: githubplan.RepositoryNameField},
		{name: "repository LF", owner: "owner", repository: "repo\n", wantField: githubplan.RepositoryNameField},
		{name: "repository NEL", owner: "owner", repository: "repo\u0085", wantField: githubplan.RepositoryNameField},
		{name: "repository LS", owner: "owner", repository: "repo\u2028", wantField: githubplan.RepositoryNameField},
		{name: "repository PS", owner: "owner", repository: "repo\u2029", wantField: githubplan.RepositoryNameField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewRepository(tt.owner, tt.repository)
			var validation *githubplan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(githubplan.InvalidRepository, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantField, validation.Field()); diff != "" {
				t.Errorf("field mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Owner()); diff != "" {
				t.Errorf("zero owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Name()); diff != "" {
				t.Errorf("zero name mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCreationRequestPlan(t *testing.T) {
	type args struct {
		repository     githubplan.Repository
		representation githubplan.Representation
		value          plan.Plan
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "milestone minimum", args: args{repository: must(githubplan.NewRepository("owner", "repo")), representation: githubplan.MilestoneRepresentation, value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewCreationRequestPlan(tt.args.repository, tt.args.representation, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewCreationRequestPlan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff("owner", got.Repository().Owner()); diff != "" {
				t.Errorf("owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("repo", got.Repository().Name()); diff != "" {
				t.Errorf("repository mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplan.MilestoneRepresentation, got.Representation()); diff != "" {
				t.Errorf("representation mismatch (-want +got):\n%s", diff)
			}
			requests := got.Requests()
			if diff := cmp.Diff(1, len(requests)); diff != "" {
				t.Fatalf("request count mismatch (-want +got):\n%s", diff)
			}
			request, ok := requests[0].(githubplan.CreateMilestoneRequest)
			if !ok {
				t.Fatalf("request type = %T, want CreateMilestoneRequest", requests[0])
			}
			if diff := cmp.Diff("Plan", request.Title()); diff != "" {
				t.Errorf("title mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCreationRequestPlanMilestone(t *testing.T) {
	tests := []struct {
		name, goalText, dateText string
		conditions               []string
		tasks                    []string
		wantCount                int
		wantDescription          string
		wantDueOn                string
	}{
		{name: "conditions and target date", goalText: " Goal\r\n", conditions: []string{" # heading-like\n", " trailing \n"}, tasks: []string{"Task A", "Task B"}, dateText: "2028-02-29", wantCount: 3, wantDescription: "## Goal\n\n Goal\r\n\n\n## Acceptance Conditions\n\n### 1\n\n # heading-like\n\n\n### 2\n\n trailing \n\n", wantDueOn: "2028-02-29T00:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := must(plan.NewGoal(tt.goalText))
			conditions := make([]plan.AcceptanceCondition, len(tt.conditions))
			for index, text := range tt.conditions {
				conditions[index] = must(plan.NewAcceptanceCondition(text))
			}
			tasks := make([]plan.Task, len(tt.tasks))
			for index, name := range tt.tasks {
				tasks[index] = must(plan.NewTask(name))
			}
			date := must(plan.ParseTargetDate(tt.dateText))
			value := must(plan.New("Plan", goal, conditions, tasks, &date))
			repository := must(githubplan.NewRepository("owner", "repo"))
			got := must(githubplan.NewCreationRequestPlan(repository, githubplan.MilestoneRepresentation, value))
			if diff := cmp.Diff(githubplan.MilestoneRepresentation, got.Representation()); diff != "" {
				t.Errorf("representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplan.RESTAPIVersion, got.APIVersion()); diff != "" {
				t.Errorf("version mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("owner", got.Repository().Owner()); diff != "" {
				t.Errorf("repository owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("repo", got.Repository().Name()); diff != "" {
				t.Errorf("repository name mismatch (-want +got):\n%s", diff)
			}
			requests := got.Requests()
			if diff := cmp.Diff(tt.wantCount, len(requests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			root := requests[0].(githubplan.CreateMilestoneRequest)
			if diff := cmp.Diff("Plan", root.Title()); diff != "" {
				t.Errorf("title mismatch (-want +got):\n%s", diff)
			}
			description := root.Description()
			if diff := cmp.Diff(tt.wantDescription, description); diff != "" {
				t.Errorf("description mismatch (-want +got):\n%s", diff)
			}
			dueOn, hasDueOn := root.DueOn()
			if diff := cmp.Diff(tt.wantDueOn, dueOn); diff != "" {
				t.Errorf("due_on mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasDueOn); diff != "" {
				t.Errorf("due_on presence mismatch (-want +got):\n%s", diff)
			}
			for index, name := range tt.tasks {
				issue := requests[index+1].(githubplan.CreateIssueRequest)
				if diff := cmp.Diff(name, issue.Title()); diff != "" {
					t.Errorf("task title mismatch (-want +got):\n%s", diff)
				}
				body, hasBody := issue.Body()
				if diff := cmp.Diff("", body); diff != "" || hasBody {
					t.Errorf("body = %q, %v", body, hasBody)
				}
				ref, hasReference := issue.Milestone()
				if diff := cmp.Diff(githubplan.RequestPosition(0), ref.Source()); diff != "" || !hasReference {
					t.Errorf("milestone source = %d, %v", ref.Source(), hasReference)
				}
				if diff := cmp.Diff(githubplan.MilestoneNumber, ref.Kind()); diff != "" {
					t.Errorf("milestone kind mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNewCreationRequestPlanIssue(t *testing.T) {
	tests := []struct {
		name, goalText, dateText string
		conditions               []string
		tasks                    []string
		wantCount                int
		wantBody                 string
	}{
		{name: "conditions and target date", goalText: " Goal\r\n", conditions: []string{" # heading-like\n", " trailing \n"}, tasks: []string{"Task A", "Task B"}, dateText: "2028-02-29", wantCount: 5, wantBody: "## Goal\n\n Goal\r\n\n\n## Acceptance Conditions\n\n### 1\n\n # heading-like\n\n\n### 2\n\n trailing \n\n\n## Target Date\n\n2028-02-29\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := must(plan.NewGoal(tt.goalText))
			conditions := make([]plan.AcceptanceCondition, len(tt.conditions))
			for index, text := range tt.conditions {
				conditions[index] = must(plan.NewAcceptanceCondition(text))
			}
			tasks := make([]plan.Task, len(tt.tasks))
			for index, name := range tt.tasks {
				tasks[index] = must(plan.NewTask(name))
			}
			date := must(plan.ParseTargetDate(tt.dateText))
			value := must(plan.New("Plan", goal, conditions, tasks, &date))
			repository := must(githubplan.NewRepository("owner", "repo"))
			got := must(githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value))
			if diff := cmp.Diff(githubplan.IssueRepresentation, got.Representation()); diff != "" {
				t.Errorf("representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplan.RESTAPIVersion, got.APIVersion()); diff != "" {
				t.Errorf("version mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("owner", got.Repository().Owner()); diff != "" {
				t.Errorf("repository owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("repo", got.Repository().Name()); diff != "" {
				t.Errorf("repository name mismatch (-want +got):\n%s", diff)
			}
			requests := got.Requests()
			if diff := cmp.Diff(tt.wantCount, len(requests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			root := requests[0].(githubplan.CreateIssueRequest)
			if diff := cmp.Diff("Plan", root.Title()); diff != "" {
				t.Errorf("root title mismatch (-want +got):\n%s", diff)
			}
			body, hasBody := root.Body()
			if diff := cmp.Diff(tt.wantBody, body); diff != "" {
				t.Errorf("body mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasBody); diff != "" {
				t.Errorf("body presence mismatch (-want +got):\n%s", diff)
			}
			for index, name := range tt.tasks {
				child := requests[1+index*2].(githubplan.CreateIssueRequest)
				relation := requests[2+index*2].(githubplan.AddSubIssueRequest)
				if diff := cmp.Diff(name, child.Title()); diff != "" {
					t.Errorf("child title mismatch (-want +got):\n%s", diff)
				}
				childBody, hasChildBody := child.Body()
				if diff := cmp.Diff("", childBody); diff != "" || hasChildBody {
					t.Errorf("child body = %q, %v", childBody, hasChildBody)
				}
				childMilestone, hasMilestone := child.Milestone()
				if diff := cmp.Diff(githubplan.RequestPosition(0), childMilestone.Source()); diff != "" {
					t.Errorf("child milestone source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.ResultKind(0), childMilestone.Kind()); diff != "" {
					t.Errorf("child milestone kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(false, hasMilestone); diff != "" {
					t.Errorf("child milestone presence mismatch (-want +got):\n%s", diff)
				}
				parent := relation.ParentIssueNumber()
				childReference := relation.SubIssueID()
				if diff := cmp.Diff(githubplan.RequestPosition(0), parent.Source()); diff != "" {
					t.Errorf("parent source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.IssueNumber, parent.Kind()); diff != "" {
					t.Errorf("parent kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.RequestPosition(1+index*2), childReference.Source()); diff != "" {
					t.Errorf("child source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.IssueID, childReference.Kind()); diff != "" {
					t.Errorf("child kind mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCanonicalNarrativeMilestoneWithoutDate(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, nil, nil))
	repository := must(githubplan.NewRepository("owner", "repo"))
	got := must(githubplan.NewCreationRequestPlan(repository, githubplan.MilestoneRepresentation, value))
	request := got.Requests()[0].(githubplan.CreateMilestoneRequest)
	description := request.Description()
	if diff := cmp.Diff("## Goal\n\ngoal\n\n## Acceptance Conditions\n\n### 1\n\naccept\n", description); diff != "" {
		t.Errorf("narrative mismatch (-want +got):\n%s", diff)
	}
	dueOn, hasDueOn := request.DueOn()
	if diff := cmp.Diff("", dueOn); diff != "" {
		t.Errorf("due_on mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasDueOn); diff != "" {
		t.Errorf("due_on presence mismatch (-want +got):\n%s", diff)
	}
}

func TestCanonicalNarrativeIssueWithoutDate(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, nil, nil))
	repository := must(githubplan.NewRepository("owner", "repo"))
	got := must(githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value))
	request := got.Requests()[0].(githubplan.CreateIssueRequest)
	body, hasBody := request.Body()
	if diff := cmp.Diff("## Goal\n\ngoal\n\n## Acceptance Conditions\n\n### 1\n\naccept\n", body); diff != "" {
		t.Errorf("narrative mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, hasBody); diff != "" {
		t.Errorf("body presence mismatch (-want +got):\n%s", diff)
	}
	reference, hasMilestone := request.Milestone()
	if diff := cmp.Diff(githubplan.RequestPosition(0), reference.Source()); diff != "" {
		t.Errorf("milestone source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), reference.Kind()); diff != "" {
		t.Errorf("milestone kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasMilestone); diff != "" {
		t.Errorf("milestone presence mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneRequestPlanBoundariesAndRepeatability(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		wantCount int
	}{
		{name: "zero tasks", count: 0, wantCount: 1},
		{name: "one task", count: 1, wantCount: 2},
		{name: "representative tasks", count: 3, wantCount: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			tasks := make([]plan.Task, tt.count)
			for index := range tasks {
				tasks[index] = must(plan.NewTask("task-" + string(rune('a'+index))))
			}
			value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tasks, nil))
			repository := must(githubplan.NewRepository("owner", "repo"))
			first := must(githubplan.NewCreationRequestPlan(repository, githubplan.MilestoneRepresentation, value))
			second := must(githubplan.NewCreationRequestPlan(repository, githubplan.MilestoneRepresentation, value))
			firstRequests := first.Requests()
			secondRequests := second.Requests()
			if diff := cmp.Diff(tt.wantCount, len(firstRequests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(len(firstRequests), len(secondRequests)); diff != "" {
				t.Errorf("repeatability count mismatch (-want +got):\n%s", diff)
			}
			firstRoot := firstRequests[0].(githubplan.CreateMilestoneRequest)
			secondRoot := secondRequests[0].(githubplan.CreateMilestoneRequest)
			if diff := cmp.Diff(firstRoot.Title(), secondRoot.Title()); diff != "" {
				t.Errorf("repeatability title mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(firstRoot.Description(), secondRoot.Description()); diff != "" {
				t.Errorf("repeatability description mismatch (-want +got):\n%s", diff)
			}
			firstDueOn, firstHasDueOn := firstRoot.DueOn()
			secondDueOn, secondHasDueOn := secondRoot.DueOn()
			if diff := cmp.Diff(firstDueOn, secondDueOn); diff != "" {
				t.Errorf("repeatability due_on mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(firstHasDueOn, secondHasDueOn); diff != "" {
				t.Errorf("repeatability due_on presence mismatch (-want +got):\n%s", diff)
			}
			for index := 1; index < len(firstRequests); index++ {
				firstIssue := firstRequests[index].(githubplan.CreateIssueRequest)
				secondIssue := secondRequests[index].(githubplan.CreateIssueRequest)
				if diff := cmp.Diff(firstIssue.Title(), secondIssue.Title()); diff != "" {
					t.Errorf("repeatability task title mismatch (-want +got):\n%s", diff)
				}
				firstBody, firstHasBody := firstIssue.Body()
				secondBody, secondHasBody := secondIssue.Body()
				if diff := cmp.Diff(firstBody, secondBody); diff != "" {
					t.Errorf("repeatability body mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(firstHasBody, secondHasBody); diff != "" {
					t.Errorf("repeatability body presence mismatch (-want +got):\n%s", diff)
				}
				firstReference, firstHasReference := firstIssue.Milestone()
				secondReference, secondHasReference := secondIssue.Milestone()
				if diff := cmp.Diff(githubplan.RequestPosition(0), firstReference.Source()); diff != "" {
					t.Errorf("literal reference source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.MilestoneNumber, firstReference.Kind()); diff != "" {
					t.Errorf("literal reference kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(true, firstReference.Source() < githubplan.RequestPosition(index)); diff != "" {
					t.Errorf("reference ordering mismatch (-want +got):\n%s", diff)
				}
				_, sourceIsMilestone := firstRequests[firstReference.Source()].(githubplan.CreateMilestoneRequest)
				if diff := cmp.Diff(true, sourceIsMilestone); diff != "" {
					t.Errorf("reference source type mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(firstReference.Source(), secondReference.Source()); diff != "" {
					t.Errorf("repeatability reference source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(firstReference.Kind(), secondReference.Kind()); diff != "" {
					t.Errorf("repeatability reference kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(firstHasReference, secondHasReference); diff != "" {
					t.Errorf("repeatability reference presence mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestIssueRequestPlanBoundariesAndRepeatability(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		wantCount int
	}{
		{name: "zero tasks", count: 0, wantCount: 1},
		{name: "one task", count: 1, wantCount: 3},
		{name: "two tasks", count: 2, wantCount: 5},
		{name: "one hundred tasks", count: 100, wantCount: 201},
	}
	for count := 3; count < 100; count++ {
		tests = append(tests, struct {
			name      string
			count     int
			wantCount int
		}{name: "task count " + string(rune('0'+count/100)) + string(rune('0'+(count/10)%10)) + string(rune('0'+count%10)), count: count, wantCount: 1 + count*2})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			tasks := make([]plan.Task, tt.count)
			for index := range tasks {
				tasks[index] = must(plan.NewTask("task-" + string(rune('a'+index/26)) + string(rune('a'+index%26))))
			}
			value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tasks, nil))
			repository := must(githubplan.NewRepository("owner", "repo"))
			first := must(githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value))
			second := must(githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value))
			firstRequests := first.Requests()
			secondRequests := second.Requests()
			if diff := cmp.Diff(tt.wantCount, len(firstRequests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(len(firstRequests), len(secondRequests)); diff != "" {
				t.Errorf("repeatability count mismatch (-want +got):\n%s", diff)
			}
			for index, request := range firstRequests {
				switch firstRequest := request.(type) {
				case githubplan.CreateIssueRequest:
					secondRequest := secondRequests[index].(githubplan.CreateIssueRequest)
					if diff := cmp.Diff(firstRequest.Title(), secondRequest.Title()); diff != "" {
						t.Errorf("repeatability title mismatch (-want +got):\n%s", diff)
					}
					firstBody, firstHasBody := firstRequest.Body()
					secondBody, secondHasBody := secondRequest.Body()
					if diff := cmp.Diff(firstBody, secondBody); diff != "" || firstHasBody != secondHasBody {
						t.Errorf("repeatability body = %q/%v, %q/%v", firstBody, firstHasBody, secondBody, secondHasBody)
					}
					_, hasMilestone := firstRequest.Milestone()
					if diff := cmp.Diff(false, hasMilestone); diff != "" {
						t.Errorf("issue milestone presence mismatch (-want +got):\n%s", diff)
					}
				case githubplan.AddSubIssueRequest:
					secondRequest := secondRequests[index].(githubplan.AddSubIssueRequest)
					firstParent, secondParent := firstRequest.ParentIssueNumber(), secondRequest.ParentIssueNumber()
					firstChild, secondChild := firstRequest.SubIssueID(), secondRequest.SubIssueID()
					if diff := cmp.Diff(githubplan.RequestPosition(0), firstParent.Source()); diff != "" {
						t.Errorf("literal parent source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplan.IssueNumber, firstParent.Kind()); diff != "" {
						t.Errorf("literal parent kind mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplan.RequestPosition(index-1), firstChild.Source()); diff != "" {
						t.Errorf("literal child source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplan.IssueID, firstChild.Kind()); diff != "" {
						t.Errorf("literal child kind mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, firstParent.Source() < githubplan.RequestPosition(index)); diff != "" {
						t.Errorf("parent ordering mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, firstChild.Source() < githubplan.RequestPosition(index)); diff != "" {
						t.Errorf("child ordering mismatch (-want +got):\n%s", diff)
					}
					_, parentIsIssue := firstRequests[firstParent.Source()].(githubplan.CreateIssueRequest)
					_, childIsIssue := firstRequests[firstChild.Source()].(githubplan.CreateIssueRequest)
					if diff := cmp.Diff(true, parentIsIssue); diff != "" {
						t.Errorf("parent source type mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, childIsIssue); diff != "" {
						t.Errorf("child source type mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(firstParent.Source(), secondParent.Source()); diff != "" {
						t.Errorf("repeatability parent source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(firstParent.Kind(), secondParent.Kind()); diff != "" {
						t.Errorf("repeatability parent kind mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(firstChild.Source(), secondChild.Source()); diff != "" {
						t.Errorf("repeatability child source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(firstChild.Kind(), secondChild.Kind()); diff != "" {
						t.Errorf("repeatability child kind mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestIssueRequestPlanUnsupportedTaskBoundary(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	tasks := make([]plan.Task, 101)
	for index := range tasks {
		tasks[index] = must(plan.NewTask("task-" + string(rune('a'+index/26)) + string(rune('a'+index%26))))
	}
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tasks, nil))
	repository := must(githubplan.NewRepository("owner", "repo"))
	got, err := githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value)
	var validation *githubplan.ValidationError
	if !errors.As(err, &validation) || validation.Code() != githubplan.UnsupportedRepresentation || validation.Field() != githubplan.RepresentationField {
		t.Fatalf("validation = %v", err)
	}
	if diff := cmp.Diff("", got.Repository().Owner()); diff != "" {
		t.Errorf("repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", got.Repository().Name()); diff != "" {
		t.Errorf("repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.Representation(0), got.Representation()); diff != "" {
		t.Errorf("representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", got.APIVersion()); diff != "" {
		t.Errorf("version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplan.Request(nil), got.Requests()); diff != "" {
		t.Errorf("requests mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneRequestsDefensiveCopy(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	repository := must(githubplan.NewRepository("owner", "repo"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, []plan.Task{must(plan.NewTask("task"))}, nil))
	requestPlan := must(githubplan.NewCreationRequestPlan(repository, githubplan.MilestoneRepresentation, value))
	want := requestPlan.Requests()
	if diff := cmp.Diff(2, len(want)); diff != "" {
		t.Errorf("request count mismatch (-want +got):\n%s", diff)
	}
	returned := requestPlan.Requests()
	returned[0] = nil
	gotRequests := requestPlan.Requests()
	if diff := cmp.Diff(len(want), len(gotRequests)); diff != "" {
		t.Errorf("requests defensive-copy count mismatch (-want +got):\n%s", diff)
	}
	wantRoot := want[0].(githubplan.CreateMilestoneRequest)
	gotRoot := gotRequests[0].(githubplan.CreateMilestoneRequest)
	if diff := cmp.Diff(wantRoot.Title(), gotRoot.Title()); diff != "" {
		t.Errorf("title mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantRoot.Description(), gotRoot.Description()); diff != "" {
		t.Errorf("description mismatch (-want +got):\n%s", diff)
	}
	wantTask := want[1].(githubplan.CreateIssueRequest)
	gotTask := gotRequests[1].(githubplan.CreateIssueRequest)
	if diff := cmp.Diff(wantTask.Title(), gotTask.Title()); diff != "" {
		t.Errorf("task title mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueRequestsDefensiveCopy(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	task := must(plan.NewTask("task"))
	repository := must(githubplan.NewRepository("owner", "repo"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, []plan.Task{task}, nil))
	requestPlan := must(githubplan.NewCreationRequestPlan(repository, githubplan.IssueRepresentation, value))
	want := requestPlan.Requests()
	if diff := cmp.Diff(3, len(want)); diff != "" {
		t.Errorf("request count mismatch (-want +got):\n%s", diff)
	}
	returned := requestPlan.Requests()
	returned[0] = nil
	gotRequests := requestPlan.Requests()
	if diff := cmp.Diff(len(want), len(gotRequests)); diff != "" {
		t.Errorf("requests defensive-copy count mismatch (-want +got):\n%s", diff)
	}
	wantRoot := want[0].(githubplan.CreateIssueRequest)
	gotRoot := gotRequests[0].(githubplan.CreateIssueRequest)
	if diff := cmp.Diff(wantRoot.Title(), gotRoot.Title()); diff != "" {
		t.Errorf("root title mismatch (-want +got):\n%s", diff)
	}
	wantBody, wantHasBody := wantRoot.Body()
	gotBody, gotHasBody := gotRoot.Body()
	if diff := cmp.Diff(wantBody, gotBody); diff != "" || wantHasBody != gotHasBody {
		t.Errorf("root body = %q/%v, %q/%v", wantBody, wantHasBody, gotBody, gotHasBody)
	}
	wantChild := want[1].(githubplan.CreateIssueRequest)
	gotChild := gotRequests[1].(githubplan.CreateIssueRequest)
	if diff := cmp.Diff(wantChild.Title(), gotChild.Title()); diff != "" {
		t.Errorf("child title mismatch (-want +got):\n%s", diff)
	}
	wantRelation := want[2].(githubplan.AddSubIssueRequest)
	gotRelation := gotRequests[2].(githubplan.AddSubIssueRequest)
	if diff := cmp.Diff(wantRelation.ParentIssueNumber().Source(), gotRelation.ParentIssueNumber().Source()); diff != "" {
		t.Errorf("parent source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantRelation.SubIssueID().Source(), gotRelation.SubIssueID().Source()); diff != "" {
		t.Errorf("child source mismatch (-want +got):\n%s", diff)
	}
}

func TestNewCreationRequestPlanValidation(t *testing.T) {
	tests := []struct {
		name           string
		repository     githubplan.Repository
		representation githubplan.Representation
		value          plan.Plan
		wantCode       githubplan.ViolationCode
		wantField      githubplan.Field
	}{
		{name: "zero repository", representation: githubplan.MilestoneRepresentation, value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplan.InvalidRepository, wantField: githubplan.RepositoryOwnerField},
		{name: "zero representation", repository: must(githubplan.NewRepository("owner", "repo")), value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplan.InvalidRepresentation, wantField: githubplan.RepresentationField},
		{name: "unsupported representation", repository: must(githubplan.NewRepository("owner", "repo")), representation: githubplan.Representation(99), value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplan.InvalidRepresentation, wantField: githubplan.RepresentationField},
		{name: "zero plan", repository: must(githubplan.NewRepository("owner", "repo")), representation: githubplan.IssueRepresentation, value: plan.Plan{}, wantCode: githubplan.InvalidPlan, wantField: githubplan.PlanField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplan.NewCreationRequestPlan(tt.repository, tt.representation, tt.value)
			var validation *githubplan.ValidationError
			if !errors.As(err, &validation) || validation.Code() != tt.wantCode || validation.Field() != tt.wantField {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.Repository().Owner()); diff != "" {
				t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Repository().Name()); diff != "" {
				t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplan.Representation(0), got.Representation()); diff != "" {
				t.Errorf("zero representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.APIVersion()); diff != "" {
				t.Errorf("zero API version mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]githubplan.Request(nil), got.Requests()); diff != "" {
				t.Errorf("zero requests mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCreationRequestPlanNativeNarrativeCollision(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		goal           string
		condition      string
	}{
		{name: "milestone goal heading at start", representation: githubplan.MilestoneRepresentation, goal: "## Goal\ntext", condition: "accept"},
		{name: "milestone goal heading in middle", representation: githubplan.MilestoneRepresentation, goal: "text\n## Acceptance Conditions\ntext", condition: "accept"},
		{name: "milestone goal heading at end", representation: githubplan.MilestoneRepresentation, goal: "text\n## Target Date", condition: "accept"},
		{name: "issue goal heading at start", representation: githubplan.IssueRepresentation, goal: "## Goal\ntext", condition: "accept"},
		{name: "issue goal heading in middle", representation: githubplan.IssueRepresentation, goal: "text\n## Acceptance Conditions\ntext", condition: "accept"},
		{name: "issue goal heading at end", representation: githubplan.IssueRepresentation, goal: "text\n## Target Date", condition: "accept"},
		{name: "milestone condition heading at start", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "## Goal\ntext"},
		{name: "milestone condition heading in middle", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "text\n## Acceptance Conditions\ntext"},
		{name: "milestone condition heading at end", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "text\n## Target Date"},
		{name: "issue condition heading at start", representation: githubplan.IssueRepresentation, goal: "goal", condition: "## Goal\ntext"},
		{name: "issue condition heading in middle", representation: githubplan.IssueRepresentation, goal: "goal", condition: "text\n## Acceptance Conditions\ntext"},
		{name: "issue condition heading at end", representation: githubplan.IssueRepresentation, goal: "goal", condition: "text\n## Target Date"},
		{name: "milestone condition ordinal at start", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "### 1\ntext"},
		{name: "milestone condition ordinal in middle", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "text\n### anything\ntext"},
		{name: "milestone condition ordinal at end", representation: githubplan.MilestoneRepresentation, goal: "goal", condition: "text\n### "},
		{name: "issue condition ordinal at start", representation: githubplan.IssueRepresentation, goal: "goal", condition: "### 1\ntext"},
		{name: "issue condition ordinal in middle", representation: githubplan.IssueRepresentation, goal: "goal", condition: "text\n### anything\ntext"},
		{name: "issue condition ordinal at end", representation: githubplan.IssueRepresentation, goal: "goal", condition: "text\n### "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := must(plan.New(
				"Plan",
				must(plan.NewGoal(tt.goal)),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(tt.condition))},
				nil,
				nil,
			))
			got, err := githubplan.NewCreationRequestPlan(
				must(githubplan.NewRepository("owner", "repo")),
				tt.representation,
				value,
			)
			var validation *githubplan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(githubplan.UnsupportedRepresentation, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplan.RepresentationField, validation.Field()); diff != "" {
				t.Errorf("field mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]githubplan.Request(nil), got.Requests()); diff != "" {
				t.Errorf("requests mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCreationRequestPlanNativeNarrativeNearMiss(t *testing.T) {
	tests := []struct {
		name      string
		goal      string
		condition string
	}{
		{name: "goal ordinal-like line", goal: "### 1", condition: "accept"},
		{name: "heading with trailing space", goal: "goal", condition: "## Goal "},
		{name: "tab after ordinal marker", goal: "goal", condition: "###\t1"},
		{name: "indented ordinal marker", goal: "goal", condition: " ### 1"},
		{name: "embedded heading text", goal: "goal", condition: "text ## Acceptance Conditions text"},
		{name: "reserved bytes on CRLF line", goal: "goal", condition: "before\r\n## Target Date\r\nafter"},
		{name: "non-reserved heading", goal: "goal", condition: "## Notes\n###\tmember"},
	}
	representations := []struct {
		name  string
		value githubplan.Representation
	}{
		{name: "milestone", value: githubplan.MilestoneRepresentation},
		{name: "issue", value: githubplan.IssueRepresentation},
	}
	for _, representation := range representations {
		for _, tt := range tests {
			t.Run(representation.name+"/"+tt.name, func(t *testing.T) {
				value := must(plan.New(
					"Plan",
					must(plan.NewGoal(tt.goal)),
					[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(tt.condition))},
					nil,
					nil,
				))
				got := must(githubplan.NewCreationRequestPlan(
					must(githubplan.NewRepository("owner", "repo")),
					representation.value,
					value,
				))
				var content string
				if representation.value == githubplan.MilestoneRepresentation {
					content = got.Requests()[0].(githubplan.CreateMilestoneRequest).Description()
				} else {
					content, _ = got.Requests()[0].(githubplan.CreateIssueRequest).Body()
				}
				want := "## Goal\n\n" + tt.goal + "\n\n## Acceptance Conditions\n\n### 1\n\n" + tt.condition + "\n"
				if diff := cmp.Diff(want, content); diff != "" {
					t.Errorf("content mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestNewCreationRequestPlanSimultaneousInvalidInputsReturnNoPlan(t *testing.T) {
	got, err := githubplan.NewCreationRequestPlan(githubplan.Repository{}, githubplan.Representation(99), plan.Plan{})
	var validation *githubplan.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("validation = %v", err)
	}
	allowed := validation.Code() == githubplan.InvalidRepository ||
		validation.Code() == githubplan.InvalidRepresentation ||
		validation.Code() == githubplan.InvalidPlan
	if diff := cmp.Diff(true, allowed); diff != "" {
		t.Errorf("allowed validation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplan.Request(nil), got.Requests()); diff != "" {
		t.Errorf("requests mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestPlanZeroValueAccessors(t *testing.T) {
	var zero githubplan.RequestPlan
	if diff := cmp.Diff("", zero.Repository().Owner()); diff != "" {
		t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", zero.Repository().Name()); diff != "" {
		t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.Representation(0), zero.Representation()); diff != "" {
		t.Errorf("zero representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", zero.APIVersion()); diff != "" {
		t.Errorf("zero version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplan.Request(nil), zero.Requests()); diff != "" {
		t.Errorf("zero requests mismatch (-want +got):\n%s", diff)
	}
	zeroReference, hasReference := (githubplan.CreateIssueRequest{}).Milestone()
	if diff := cmp.Diff(githubplan.RequestPosition(0), zeroReference.Source()); diff != "" {
		t.Errorf("zero issue reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), zeroReference.Kind()); diff != "" {
		t.Errorf("zero issue reference kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasReference); diff != "" {
		t.Errorf("zero issue reference presence mismatch (-want +got):\n%s", diff)
	}
}

func TestAllGitHubZeroValueAccessorsArePanicFree(t *testing.T) {
	var repository githubplan.Repository
	if diff := cmp.Diff("", repository.Owner()); diff != "" {
		t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", repository.Name()); diff != "" {
		t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
	}
	var reference githubplan.ResultReference
	if diff := cmp.Diff(githubplan.RequestPosition(0), reference.Source()); diff != "" {
		t.Errorf("zero reference source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), reference.Kind()); diff != "" {
		t.Errorf("zero reference kind mismatch (-want +got):\n%s", diff)
	}
	var milestone githubplan.CreateMilestoneRequest
	if diff := cmp.Diff("", milestone.Title()); diff != "" {
		t.Errorf("zero milestone title mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", milestone.Description()); diff != "" {
		t.Errorf("zero milestone description mismatch (-want +got):\n%s", diff)
	}
	if dueOn, ok := milestone.DueOn(); ok || dueOn != "" {
		t.Errorf("zero milestone due_on = %q, %v", dueOn, ok)
	}
	var issue githubplan.CreateIssueRequest
	if diff := cmp.Diff("", issue.Title()); diff != "" {
		t.Errorf("zero issue title mismatch (-want +got):\n%s", diff)
	}
	if body, ok := issue.Body(); ok || body != "" {
		t.Errorf("zero issue body = %q, %v", body, ok)
	}
	issueReference, hasIssueMilestone := issue.Milestone()
	if diff := cmp.Diff(githubplan.RequestPosition(0), issueReference.Source()); diff != "" {
		t.Errorf("zero issue milestone source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), issueReference.Kind()); diff != "" {
		t.Errorf("zero issue milestone kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasIssueMilestone); diff != "" {
		t.Errorf("zero issue milestone presence mismatch (-want +got):\n%s", diff)
	}
	var subIssue githubplan.AddSubIssueRequest
	if diff := cmp.Diff(githubplan.RequestPosition(0), subIssue.ParentIssueNumber().Source()); diff != "" {
		t.Errorf("zero parent reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), subIssue.ParentIssueNumber().Kind()); diff != "" {
		t.Errorf("zero parent reference kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.RequestPosition(0), subIssue.SubIssueID().Source()); diff != "" {
		t.Errorf("zero child reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ResultKind(0), subIssue.SubIssueID().Kind()); diff != "" {
		t.Errorf("zero child reference kind mismatch (-want +got):\n%s", diff)
	}
	var requestPlan githubplan.RequestPlan
	if diff := cmp.Diff("", requestPlan.Repository().Owner()); diff != "" {
		t.Errorf("zero request repository mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", requestPlan.Repository().Name()); diff != "" {
		t.Errorf("zero request repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.Representation(0), requestPlan.Representation()); diff != "" {
		t.Errorf("zero request representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", requestPlan.APIVersion()); diff != "" {
		t.Errorf("zero request version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplan.Request(nil), requestPlan.Requests()); diff != "" {
		t.Errorf("zero request list mismatch (-want +got):\n%s", diff)
	}
	var validation *githubplan.ValidationError
	if diff := cmp.Diff("github planning validation error", validation.Error()); diff != "" {
		t.Errorf("nil validation error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.ViolationCode(""), validation.Code()); diff != "" {
		t.Errorf("nil validation code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplan.Field(0), validation.Field()); diff != "" {
		t.Errorf("nil validation field mismatch (-want +got):\n%s", diff)
	}
}

func TestRepositoryPreservesUnusualText(t *testing.T) {
	value := must(githubplan.NewRepository(" owner+.~ ", "repo_1"))
	if diff := cmp.Diff(" owner+.~ ", value.Owner()); diff != "" {
		t.Errorf("owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("repo_1", value.Name()); diff != "" {
		t.Errorf("name mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneTaskConsumersHaveLiteralMilestonePresence(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{name: "zero tasks", count: 0},
		{name: "one task", count: 1},
		{name: "representative multiple tasks", count: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := make([]plan.Task, tt.count)
			for index := range tasks {
				tasks[index] = must(plan.NewTask("task-" + string(rune('a'+index))))
			}
			value := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, tasks, nil))
			requestPlan := must(githubplan.NewCreationRequestPlan(must(githubplan.NewRepository("owner", "repo")), githubplan.MilestoneRepresentation, value))
			requests := requestPlan.Requests()
			for consumer := 1; consumer < len(requests); consumer++ {
				request := requests[consumer].(githubplan.CreateIssueRequest)
				reference, hasReference := request.Milestone()
				if diff := cmp.Diff(true, hasReference); diff != "" {
					t.Errorf("milestone presence mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.RequestPosition(0), reference.Source()); diff != "" {
					t.Errorf("milestone source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplan.MilestoneNumber, reference.Kind()); diff != "" {
					t.Errorf("milestone kind mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
