package githubplanning_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/githubplanning"
	"github.com/kotokumu/arcloom/plan"
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
		want    githubplanning.Repository
		wantErr bool
	}{
		{name: "normal", args: args{owner: "owner", name: "repo"}, want: must(githubplanning.NewRepository("owner", "repo"))},
		{name: "unusual accepted text", args: args{owner: " owner+.~ ", name: "repo_1"}, want: must(githubplanning.NewRepository(" owner+.~ ", "repo_1"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplanning.NewRepository(tt.args.owner, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(githubplanning.Repository{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewRepositoryValidation(t *testing.T) {
	tests := []struct {
		name, owner, repository string
		wantField               githubplanning.Field
	}{
		{name: "blank owner", owner: " \u2003", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "invalid owner utf8", owner: string([]byte{0xff}), repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "slash owner", owner: "owner/name", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "owner CR", owner: "owner\r", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "owner LF", owner: "owner\n", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "owner NEL", owner: "owner\u0085", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "owner LS", owner: "owner\u2028", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "owner PS", owner: "owner\u2029", repository: "repo", wantField: githubplanning.RepositoryOwnerField},
		{name: "blank repository", owner: "owner", repository: "\u2029", wantField: githubplanning.RepositoryNameField},
		{name: "invalid repository utf8", owner: "owner", repository: string([]byte{0xff}), wantField: githubplanning.RepositoryNameField},
		{name: "slash repository", owner: "owner", repository: "repo/name", wantField: githubplanning.RepositoryNameField},
		{name: "repository CR", owner: "owner", repository: "repo\r", wantField: githubplanning.RepositoryNameField},
		{name: "repository LF", owner: "owner", repository: "repo\n", wantField: githubplanning.RepositoryNameField},
		{name: "repository NEL", owner: "owner", repository: "repo\u0085", wantField: githubplanning.RepositoryNameField},
		{name: "repository LS", owner: "owner", repository: "repo\u2028", wantField: githubplanning.RepositoryNameField},
		{name: "repository PS", owner: "owner", repository: "repo\u2029", wantField: githubplanning.RepositoryNameField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplanning.NewRepository(tt.owner, tt.repository)
			var validation *githubplanning.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(githubplanning.InvalidRepository, validation.Code()); diff != "" {
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
		repository     githubplanning.Repository
		representation githubplanning.Representation
		value          plan.Plan
	}
	tests := []struct {
		name    string
		args    args
		want    githubplanning.RequestPlan
		wantErr bool
	}{
		{name: "milestone minimum", args: args{repository: must(githubplanning.NewRepository("owner", "repo")), representation: githubplanning.MilestoneRepresentation, value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))}, want: must(githubplanning.NewCreationRequestPlan(must(githubplanning.NewRepository("owner", "repo")), githubplanning.MilestoneRepresentation, must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplanning.NewCreationRequestPlan(tt.args.repository, tt.args.representation, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewCreationRequestPlan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(githubplanning.RequestPlan{}, githubplanning.Repository{}, githubplanning.CreateMilestoneRequest{}, githubplanning.CreateIssueRequest{}, githubplanning.AddSubIssueRequest{}, githubplanning.ResultReference{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
			repository := must(githubplanning.NewRepository("owner", "repo"))
			got := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.MilestoneRepresentation, value))
			if diff := cmp.Diff(githubplanning.MilestoneRepresentation, got.Representation()); diff != "" {
				t.Errorf("representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplanning.RESTAPIVersion, got.APIVersion()); diff != "" {
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
			root := requests[0].(githubplanning.CreateMilestoneRequest)
			if diff := cmp.Diff("Plan", root.Title()); diff != "" {
				t.Errorf("title mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantDescription, root.Description()); diff != "" {
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
				issue := requests[index+1].(githubplanning.CreateIssueRequest)
				if diff := cmp.Diff(name, issue.Title()); diff != "" {
					t.Errorf("task title mismatch (-want +got):\n%s", diff)
				}
				body, hasBody := issue.Body()
				if diff := cmp.Diff("", body); diff != "" || hasBody {
					t.Errorf("body = %q, %v", body, hasBody)
				}
				ref, hasReference := issue.Milestone()
				if diff := cmp.Diff(githubplanning.RequestPosition(0), ref.Source()); diff != "" || !hasReference {
					t.Errorf("milestone source = %d, %v", ref.Source(), hasReference)
				}
				if diff := cmp.Diff(githubplanning.MilestoneNumber, ref.Kind()); diff != "" {
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
			repository := must(githubplanning.NewRepository("owner", "repo"))
			got := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value))
			if diff := cmp.Diff(githubplanning.IssueRepresentation, got.Representation()); diff != "" {
				t.Errorf("representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplanning.RESTAPIVersion, got.APIVersion()); diff != "" {
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
			root := requests[0].(githubplanning.CreateIssueRequest)
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
				child := requests[1+index*2].(githubplanning.CreateIssueRequest)
				relation := requests[2+index*2].(githubplanning.AddSubIssueRequest)
				if diff := cmp.Diff(name, child.Title()); diff != "" {
					t.Errorf("child title mismatch (-want +got):\n%s", diff)
				}
				childBody, hasChildBody := child.Body()
				if diff := cmp.Diff("", childBody); diff != "" || hasChildBody {
					t.Errorf("child body = %q, %v", childBody, hasChildBody)
				}
				childMilestone, hasMilestone := child.Milestone()
				if diff := cmp.Diff(githubplanning.RequestPosition(0), childMilestone.Source()); diff != "" {
					t.Errorf("child milestone source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.ResultKind(0), childMilestone.Kind()); diff != "" {
					t.Errorf("child milestone kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(false, hasMilestone); diff != "" {
					t.Errorf("child milestone presence mismatch (-want +got):\n%s", diff)
				}
				parent := relation.ParentIssueNumber()
				childReference := relation.SubIssueID()
				if diff := cmp.Diff(githubplanning.RequestPosition(0), parent.Source()); diff != "" {
					t.Errorf("parent source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.IssueNumber, parent.Kind()); diff != "" {
					t.Errorf("parent kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.RequestPosition(1+index*2), childReference.Source()); diff != "" {
					t.Errorf("child source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.IssueID, childReference.Kind()); diff != "" {
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
	repository := must(githubplanning.NewRepository("owner", "repo"))
	got := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.MilestoneRepresentation, value))
	request := got.Requests()[0].(githubplanning.CreateMilestoneRequest)
	if diff := cmp.Diff("## Goal\n\ngoal\n\n## Acceptance Conditions\n\n### 1\n\naccept\n", request.Description()); diff != "" {
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
	repository := must(githubplanning.NewRepository("owner", "repo"))
	got := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value))
	request := got.Requests()[0].(githubplanning.CreateIssueRequest)
	body, hasBody := request.Body()
	if diff := cmp.Diff("## Goal\n\ngoal\n\n## Acceptance Conditions\n\n### 1\n\naccept\n", body); diff != "" {
		t.Errorf("narrative mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, hasBody); diff != "" {
		t.Errorf("body presence mismatch (-want +got):\n%s", diff)
	}
	reference, hasMilestone := request.Milestone()
	if diff := cmp.Diff(githubplanning.RequestPosition(0), reference.Source()); diff != "" {
		t.Errorf("milestone source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), reference.Kind()); diff != "" {
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
			repository := must(githubplanning.NewRepository("owner", "repo"))
			first := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.MilestoneRepresentation, value))
			second := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.MilestoneRepresentation, value))
			firstRequests := first.Requests()
			secondRequests := second.Requests()
			if diff := cmp.Diff(tt.wantCount, len(firstRequests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(len(firstRequests), len(secondRequests)); diff != "" {
				t.Errorf("repeatability count mismatch (-want +got):\n%s", diff)
			}
			firstRoot := firstRequests[0].(githubplanning.CreateMilestoneRequest)
			secondRoot := secondRequests[0].(githubplanning.CreateMilestoneRequest)
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
				firstIssue := firstRequests[index].(githubplanning.CreateIssueRequest)
				secondIssue := secondRequests[index].(githubplanning.CreateIssueRequest)
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
				if diff := cmp.Diff(githubplanning.RequestPosition(0), firstReference.Source()); diff != "" {
					t.Errorf("literal reference source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.MilestoneNumber, firstReference.Kind()); diff != "" {
					t.Errorf("literal reference kind mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(true, firstReference.Source() < githubplanning.RequestPosition(index)); diff != "" {
					t.Errorf("reference ordering mismatch (-want +got):\n%s", diff)
				}
				_, sourceIsMilestone := firstRequests[firstReference.Source()].(githubplanning.CreateMilestoneRequest)
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
			repository := must(githubplanning.NewRepository("owner", "repo"))
			first := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value))
			second := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value))
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
				case githubplanning.CreateIssueRequest:
					secondRequest := secondRequests[index].(githubplanning.CreateIssueRequest)
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
				case githubplanning.AddSubIssueRequest:
					secondRequest := secondRequests[index].(githubplanning.AddSubIssueRequest)
					firstParent, secondParent := firstRequest.ParentIssueNumber(), secondRequest.ParentIssueNumber()
					firstChild, secondChild := firstRequest.SubIssueID(), secondRequest.SubIssueID()
					if diff := cmp.Diff(githubplanning.RequestPosition(0), firstParent.Source()); diff != "" {
						t.Errorf("literal parent source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplanning.IssueNumber, firstParent.Kind()); diff != "" {
						t.Errorf("literal parent kind mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplanning.RequestPosition(index-1), firstChild.Source()); diff != "" {
						t.Errorf("literal child source mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(githubplanning.IssueID, firstChild.Kind()); diff != "" {
						t.Errorf("literal child kind mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, firstParent.Source() < githubplanning.RequestPosition(index)); diff != "" {
						t.Errorf("parent ordering mismatch (-want +got):\n%s", diff)
					}
					if diff := cmp.Diff(true, firstChild.Source() < githubplanning.RequestPosition(index)); diff != "" {
						t.Errorf("child ordering mismatch (-want +got):\n%s", diff)
					}
					_, parentIsIssue := firstRequests[firstParent.Source()].(githubplanning.CreateIssueRequest)
					_, childIsIssue := firstRequests[firstChild.Source()].(githubplanning.CreateIssueRequest)
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
	repository := must(githubplanning.NewRepository("owner", "repo"))
	got, err := githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value)
	var validation *githubplanning.ValidationError
	if !errors.As(err, &validation) || validation.Code() != githubplanning.UnsupportedRepresentation || validation.Field() != githubplanning.RepresentationField {
		t.Fatalf("validation = %v", err)
	}
	if diff := cmp.Diff("", got.Repository().Owner()); diff != "" {
		t.Errorf("repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", got.Repository().Name()); diff != "" {
		t.Errorf("repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.Representation(0), got.Representation()); diff != "" {
		t.Errorf("representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", got.APIVersion()); diff != "" {
		t.Errorf("version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplanning.Request(nil), got.Requests()); diff != "" {
		t.Errorf("requests mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneRequestsDefensiveCopy(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	repository := must(githubplanning.NewRepository("owner", "repo"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, []plan.Task{must(plan.NewTask("task"))}, nil))
	requestPlan := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.MilestoneRepresentation, value))
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
	wantRoot := want[0].(githubplanning.CreateMilestoneRequest)
	gotRoot := gotRequests[0].(githubplanning.CreateMilestoneRequest)
	if diff := cmp.Diff(wantRoot.Title(), gotRoot.Title()); diff != "" {
		t.Errorf("title mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantRoot.Description(), gotRoot.Description()); diff != "" {
		t.Errorf("description mismatch (-want +got):\n%s", diff)
	}
	wantTask := want[1].(githubplanning.CreateIssueRequest)
	gotTask := gotRequests[1].(githubplanning.CreateIssueRequest)
	if diff := cmp.Diff(wantTask.Title(), gotTask.Title()); diff != "" {
		t.Errorf("task title mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueRequestsDefensiveCopy(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	task := must(plan.NewTask("task"))
	repository := must(githubplanning.NewRepository("owner", "repo"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, []plan.Task{task}, nil))
	requestPlan := must(githubplanning.NewCreationRequestPlan(repository, githubplanning.IssueRepresentation, value))
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
	wantRoot := want[0].(githubplanning.CreateIssueRequest)
	gotRoot := gotRequests[0].(githubplanning.CreateIssueRequest)
	if diff := cmp.Diff(wantRoot.Title(), gotRoot.Title()); diff != "" {
		t.Errorf("root title mismatch (-want +got):\n%s", diff)
	}
	wantBody, wantHasBody := wantRoot.Body()
	gotBody, gotHasBody := gotRoot.Body()
	if diff := cmp.Diff(wantBody, gotBody); diff != "" || wantHasBody != gotHasBody {
		t.Errorf("root body = %q/%v, %q/%v", wantBody, wantHasBody, gotBody, gotHasBody)
	}
	wantChild := want[1].(githubplanning.CreateIssueRequest)
	gotChild := gotRequests[1].(githubplanning.CreateIssueRequest)
	if diff := cmp.Diff(wantChild.Title(), gotChild.Title()); diff != "" {
		t.Errorf("child title mismatch (-want +got):\n%s", diff)
	}
	wantRelation := want[2].(githubplanning.AddSubIssueRequest)
	gotRelation := gotRequests[2].(githubplanning.AddSubIssueRequest)
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
		repository     githubplanning.Repository
		representation githubplanning.Representation
		value          plan.Plan
		wantCode       githubplanning.ViolationCode
		wantField      githubplanning.Field
	}{
		{name: "zero repository", representation: githubplanning.MilestoneRepresentation, value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplanning.InvalidRepository, wantField: githubplanning.RepositoryOwnerField},
		{name: "zero representation", repository: must(githubplanning.NewRepository("owner", "repo")), value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplanning.InvalidRepresentation, wantField: githubplanning.RepresentationField},
		{name: "unsupported representation", repository: must(githubplanning.NewRepository("owner", "repo")), representation: githubplanning.Representation(99), value: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), wantCode: githubplanning.InvalidRepresentation, wantField: githubplanning.RepresentationField},
		{name: "zero plan", repository: must(githubplanning.NewRepository("owner", "repo")), representation: githubplanning.IssueRepresentation, value: plan.Plan{}, wantCode: githubplanning.InvalidPlan, wantField: githubplanning.PlanField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := githubplanning.NewCreationRequestPlan(tt.repository, tt.representation, tt.value)
			var validation *githubplanning.ValidationError
			if !errors.As(err, &validation) || validation.Code() != tt.wantCode || validation.Field() != tt.wantField {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.Repository().Owner()); diff != "" {
				t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Repository().Name()); diff != "" {
				t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(githubplanning.Representation(0), got.Representation()); diff != "" {
				t.Errorf("zero representation mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.APIVersion()); diff != "" {
				t.Errorf("zero API version mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]githubplanning.Request(nil), got.Requests()); diff != "" {
				t.Errorf("zero requests mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestPlanZeroValueAccessors(t *testing.T) {
	var zero githubplanning.RequestPlan
	if diff := cmp.Diff("", zero.Repository().Owner()); diff != "" {
		t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", zero.Repository().Name()); diff != "" {
		t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.Representation(0), zero.Representation()); diff != "" {
		t.Errorf("zero representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", zero.APIVersion()); diff != "" {
		t.Errorf("zero version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplanning.Request(nil), zero.Requests()); diff != "" {
		t.Errorf("zero requests mismatch (-want +got):\n%s", diff)
	}
	zeroReference, hasReference := (githubplanning.CreateIssueRequest{}).Milestone()
	if diff := cmp.Diff(githubplanning.RequestPosition(0), zeroReference.Source()); diff != "" {
		t.Errorf("zero issue reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), zeroReference.Kind()); diff != "" {
		t.Errorf("zero issue reference kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasReference); diff != "" {
		t.Errorf("zero issue reference presence mismatch (-want +got):\n%s", diff)
	}
}

func TestAllGitHubZeroValueAccessorsArePanicFree(t *testing.T) {
	var repository githubplanning.Repository
	if diff := cmp.Diff("", repository.Owner()); diff != "" {
		t.Errorf("zero repository owner mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", repository.Name()); diff != "" {
		t.Errorf("zero repository name mismatch (-want +got):\n%s", diff)
	}
	var reference githubplanning.ResultReference
	if diff := cmp.Diff(githubplanning.RequestPosition(0), reference.Source()); diff != "" {
		t.Errorf("zero reference source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), reference.Kind()); diff != "" {
		t.Errorf("zero reference kind mismatch (-want +got):\n%s", diff)
	}
	var milestone githubplanning.CreateMilestoneRequest
	if diff := cmp.Diff("", milestone.Title()); diff != "" {
		t.Errorf("zero milestone title mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", milestone.Description()); diff != "" {
		t.Errorf("zero milestone description mismatch (-want +got):\n%s", diff)
	}
	if dueOn, ok := milestone.DueOn(); ok || dueOn != "" {
		t.Errorf("zero milestone due_on = %q, %v", dueOn, ok)
	}
	var issue githubplanning.CreateIssueRequest
	if diff := cmp.Diff("", issue.Title()); diff != "" {
		t.Errorf("zero issue title mismatch (-want +got):\n%s", diff)
	}
	if body, ok := issue.Body(); ok || body != "" {
		t.Errorf("zero issue body = %q, %v", body, ok)
	}
	issueReference, hasIssueMilestone := issue.Milestone()
	if diff := cmp.Diff(githubplanning.RequestPosition(0), issueReference.Source()); diff != "" {
		t.Errorf("zero issue milestone source mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), issueReference.Kind()); diff != "" {
		t.Errorf("zero issue milestone kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, hasIssueMilestone); diff != "" {
		t.Errorf("zero issue milestone presence mismatch (-want +got):\n%s", diff)
	}
	var subIssue githubplanning.AddSubIssueRequest
	if diff := cmp.Diff(githubplanning.RequestPosition(0), subIssue.ParentIssueNumber().Source()); diff != "" {
		t.Errorf("zero parent reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), subIssue.ParentIssueNumber().Kind()); diff != "" {
		t.Errorf("zero parent reference kind mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.RequestPosition(0), subIssue.SubIssueID().Source()); diff != "" {
		t.Errorf("zero child reference mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ResultKind(0), subIssue.SubIssueID().Kind()); diff != "" {
		t.Errorf("zero child reference kind mismatch (-want +got):\n%s", diff)
	}
	var requestPlan githubplanning.RequestPlan
	if diff := cmp.Diff("", requestPlan.Repository().Owner()); diff != "" {
		t.Errorf("zero request repository mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", requestPlan.Repository().Name()); diff != "" {
		t.Errorf("zero request repository name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.Representation(0), requestPlan.Representation()); diff != "" {
		t.Errorf("zero request representation mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", requestPlan.APIVersion()); diff != "" {
		t.Errorf("zero request version mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]githubplanning.Request(nil), requestPlan.Requests()); diff != "" {
		t.Errorf("zero request list mismatch (-want +got):\n%s", diff)
	}
	var validation *githubplanning.ValidationError
	if diff := cmp.Diff("github planning validation error", validation.Error()); diff != "" {
		t.Errorf("nil validation error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.ViolationCode(""), validation.Code()); diff != "" {
		t.Errorf("nil validation code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(githubplanning.Field(0), validation.Field()); diff != "" {
		t.Errorf("nil validation field mismatch (-want +got):\n%s", diff)
	}
}

func TestRepositoryPreservesUnusualText(t *testing.T) {
	value := must(githubplanning.NewRepository(" owner+.~ ", "repo_1"))
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
			requestPlan := must(githubplanning.NewCreationRequestPlan(must(githubplanning.NewRepository("owner", "repo")), githubplanning.MilestoneRepresentation, value))
			requests := requestPlan.Requests()
			for consumer := 1; consumer < len(requests); consumer++ {
				request := requests[consumer].(githubplanning.CreateIssueRequest)
				reference, hasReference := request.Milestone()
				if diff := cmp.Diff(true, hasReference); diff != "" {
					t.Errorf("milestone presence mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.RequestPosition(0), reference.Source()); diff != "" {
					t.Errorf("milestone source mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(githubplanning.MilestoneNumber, reference.Kind()); diff != "" {
					t.Errorf("milestone kind mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
