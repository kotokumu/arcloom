package githubplan_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

func TestCanonicalMilestoneNativeNarrative(t *testing.T) {
	goal := "goal"
	condition := "accept"
	value := must(plan.New(
		"Plan",
		must(plan.NewGoal(goal)),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(condition))},
		nil,
		nil,
	))
	requestPlan := must(githubplan.NewCreationRequestPlan(
		must(githubplan.NewRepository("owner", "repo")),
		githubplan.MilestoneRepresentation,
		value,
	))
	request := requestPlan.Requests()[0].(githubplan.CreateMilestoneRequest)
	want := "## Goal\n\n" + goal + "\n\n" +
		"## Acceptance Conditions\n\n### 1\n\n" + condition + "\n"
	if diff := cmp.Diff(want, request.Description()); diff != "" {
		t.Errorf("Milestone description mismatch (-want +got):\n%s", diff)
	}
}

func TestCanonicalIssueNativeNarrative(t *testing.T) {
	goal := "goal"
	condition := "accept"
	date := must(plan.ParseTargetDate("2028-02-29"))
	value := must(plan.New(
		"Plan",
		must(plan.NewGoal(goal)),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(condition))},
		nil,
		&date,
	))
	requestPlan := must(githubplan.NewCreationRequestPlan(
		must(githubplan.NewRepository("owner", "repo")),
		githubplan.IssueRepresentation,
		value,
	))
	request := requestPlan.Requests()[0].(githubplan.CreateIssueRequest)
	body, hasBody := request.Body()
	if diff := cmp.Diff(true, hasBody); diff != "" {
		t.Fatalf("Issue body presence mismatch (-want +got):\n%s", diff)
	}
	want := "## Goal\n\n" + goal + "\n\n" +
		"## Acceptance Conditions\n\n### 1\n\n" + condition + "\n\n" +
		"## Target Date\n\n2028-02-29\n"
	if diff := cmp.Diff(want, body); diff != "" {
		t.Errorf("Issue body mismatch (-want +got):\n%s", diff)
	}
}

func TestNativeNarrativePreservesExactPlanTextBytes(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		date           string
		wantTarget     string
	}{
		{
			name:           "milestone",
			representation: githubplan.MilestoneRepresentation,
		},
		{
			name:           "issue",
			representation: githubplan.IssueRepresentation,
			date:           "2028-02-29",
			wantTarget:     "\n## Target Date\n\n2028-02-29\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal := " leading\r\n<!-- legacy marker -->\ntrailing\n"
			conditions := []string{"first\r\nline", "  second --> <!--\n"}
			var targetDate *plan.TargetDate
			if tt.date != "" {
				parsed := must(plan.ParseTargetDate(tt.date))
				targetDate = &parsed
			}
			value := must(plan.New(
				"Plan",
				must(plan.NewGoal(goal)),
				[]plan.AcceptanceCondition{
					must(plan.NewAcceptanceCondition(conditions[0])),
					must(plan.NewAcceptanceCondition(conditions[1])),
				},
				nil,
				targetDate,
			))
			requestPlan := must(githubplan.NewCreationRequestPlan(
				must(githubplan.NewRepository("owner", "repo")),
				tt.representation,
				value,
			))
			want := "## Goal\n\n" + goal + "\n\n" +
				"## Acceptance Conditions\n\n### 1\n\n" + conditions[0] +
				"\n\n### 2\n\n" + conditions[1] + "\n" + tt.wantTarget
			requests := requestPlan.Requests()
			var got string
			if tt.representation == githubplan.MilestoneRepresentation {
				got = requests[0].(githubplan.CreateMilestoneRequest).Description()
			} else {
				got, _ = requests[0].(githubplan.CreateIssueRequest).Body()
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("complete content mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
