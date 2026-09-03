package githubplan_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type nativeNarrativeRoundTripper struct {
	responses []*http.Response
}

func (t *nativeNarrativeRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type nativeNarrativeResult struct {
	determination planrepresentation.Determination
	categories    []planrepresentation.DifferenceCategory
	locations     []planrepresentation.LocationKind
	violations    []plan.ViolationCode
	members       []string
	unavailable   []planrepresentation.LocationKind
}

func observeNativeNarrative(
	t *testing.T,
	representation githubplan.Representation,
	content *string,
	dueOn *string,
	expected plan.Plan,
) nativeNarrativeResult {
	t.Helper()
	root := map[string]any{"number": 42, "title": expected.Name()}
	if representation == githubplan.MilestoneRepresentation {
		if content != nil {
			root["description"] = *content
		}
		if dueOn == nil {
			root["due_on"] = nil
		} else {
			root["due_on"] = *dueOn
		}
	} else if content != nil {
		root["body"] = *content
	}
	rootJSON := must(json.Marshal(root))
	transport := &nativeNarrativeRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(rootJSON))), Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("[]")), Header: make(http.Header)},
	}}
	repository := must(githubplan.NewRepository("owner", "repo"))
	number := must(githubplan.NewResourceNumber(42))
	var observer planrepresentation.Observer
	if representation == githubplan.MilestoneRepresentation {
		observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, repository, number))
	} else {
		observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, repository, number))
	}
	result, err := must(planrepresentation.NewController(observer)).Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	got := nativeNarrativeResult{determination: result.Determination()}
	for _, difference := range result.Differences() {
		got.categories = append(got.categories, difference.Category())
		got.locations = append(got.locations, difference.Location().Kind())
		violation := plan.ViolationCode("")
		if invalid, ok := difference.(planrepresentation.InvalidObservedDifference); ok {
			violation = invalid.Violation()
		}
		got.violations = append(got.violations, violation)
		member, _ := difference.Location().Member()
		got.members = append(got.members, member)
	}
	for _, unavailable := range result.UnavailableInformation() {
		got.unavailable = append(got.unavailable, unavailable.Location().Kind())
	}
	return got
}

func TestNativeNarrativeObservationAuthority(t *testing.T) {
	legacy := "<!-- arcloom-plan:v1\neyJnb2FsIjoiTGVnYWN5IiwiYWNjZXB0YW5jZV9jb25kaXRpb25zIjpbIkxlZ2FjeSJdfQ\n-->\n\n"
	legacyDated := "<!-- arcloom-plan:v1\neyJnb2FsIjoiTGVnYWN5IiwiYWNjZXB0YW5jZV9jb25kaXRpb25zIjpbIkxlZ2FjeSBjb25kaXRpb24iXSwidGFyZ2V0X2RhdGUiOiIyMDI3LTAxLTAxIn0\n-->\n\n"
	tests := []struct {
		name           string
		representation githubplan.Representation
		content        string
		goal           string
		conditions     []string
		date           string
		want           nativeNarrativeResult
	}{
		{
			name:           "native milestone",
			representation: githubplan.MilestoneRepresentation,
			content:        "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 2\n\nSecond\n",
			goal:           "Goal", conditions: []string{"First", "Second"},
			want: nativeNarrativeResult{determination: planrepresentation.Satisfied},
		},
		{
			name:           "native dated issue",
			representation: githubplan.IssueRepresentation,
			content:        "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n## Target Date\n\n2028-02-29\n",
			goal:           "Goal", conditions: []string{"First"}, date: "2028-02-29",
			want: nativeNarrativeResult{determination: planrepresentation.Satisfied},
		},
		{
			name:           "arbitrary preamble",
			representation: githubplan.MilestoneRepresentation,
			content:        "Operator notes without reserved headings.\n\n## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n",
			goal:           "Goal", conditions: []string{"First"},
			want: nativeNarrativeResult{determination: planrepresentation.Satisfied},
		},
		{
			name:           "conflicting legacy preamble",
			representation: githubplan.MilestoneRepresentation,
			content:        legacy + "## Goal\n\nNative\n\n## Acceptance Conditions\n\n### 1\n\nNative condition\n",
			goal:           "Native", conditions: []string{"Native condition"},
			want: nativeNarrativeResult{determination: planrepresentation.Satisfied},
		},
		{
			name:           "conflicting legacy dated issue preamble",
			representation: githubplan.IssueRepresentation,
			content:        legacyDated + "## Goal\n\nNative\n\n## Acceptance Conditions\n\n### 1\n\nNative condition\n\n## Target Date\n\n2028-02-29\n",
			goal:           "Native", conditions: []string{"Native condition"}, date: "2028-02-29",
			want: nativeNarrativeResult{determination: planrepresentation.Satisfied},
		},
		{
			name:           "marker only issue",
			representation: githubplan.IssueRepresentation,
			content:        legacy,
			goal:           "Goal", conditions: []string{"First"}, date: "2028-02-29",
			want: nativeNarrativeResult{
				determination: planrepresentation.Undecidable,
				unavailable: []planrepresentation.LocationKind{
					planrepresentation.GoalLocation,
					planrepresentation.AcceptanceConditionCollectionLocation,
					planrepresentation.TargetDateLocation,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions := make([]plan.AcceptanceCondition, len(tt.conditions))
			for index, statement := range tt.conditions {
				conditions[index] = must(plan.NewAcceptanceCondition(statement))
			}
			var date *plan.TargetDate
			if tt.date != "" {
				parsed := must(plan.ParseTargetDate(tt.date))
				date = &parsed
			}
			expected := must(plan.New("Plan", must(plan.NewGoal(tt.goal)), conditions, nil, date))
			got := observeNativeNarrative(t, tt.representation, &tt.content, nil, expected)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeNarrativeObservationPreambleRemovalIsMeaningPreserving(t *testing.T) {
	native := "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n"
	withPreamble := "plain preamble\n\n" + native
	expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("First"))}, nil, nil))
	want := observeNativeNarrative(t, githubplan.MilestoneRepresentation, &native, nil, expected)
	got := observeNativeNarrative(t, githubplan.MilestoneRepresentation, &withPreamble, nil, expected)
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
		t.Errorf("preamble removal mismatch (-want +got):\n%s", diff)
	}
}

func TestNativeNarrativeObservationCreationRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		date           string
	}{
		{name: "milestone", representation: githubplan.MilestoneRepresentation},
		{name: "undated issue", representation: githubplan.IssueRepresentation},
		{name: "dated issue", representation: githubplan.IssueRepresentation, date: "2028-02-29"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var date *plan.TargetDate
			if tt.date != "" {
				parsed := must(plan.ParseTargetDate(tt.date))
				date = &parsed
			}
			expected := must(plan.New(
				"Plan",
				must(plan.NewGoal(" Goal\r\nline ")),
				[]plan.AcceptanceCondition{
					must(plan.NewAcceptanceCondition("First\r\nline")),
					must(plan.NewAcceptanceCondition(" second \nline ")),
				},
				nil,
				date,
			))
			requestPlan := must(githubplan.NewCreationRequestPlan(
				must(githubplan.NewRepository("owner", "repo")),
				tt.representation,
				expected,
			))
			var content string
			if tt.representation == githubplan.MilestoneRepresentation {
				content = requestPlan.Requests()[0].(githubplan.CreateMilestoneRequest).Description()
			} else {
				content, _ = requestPlan.Requests()[0].(githubplan.CreateIssueRequest).Body()
			}
			got := observeNativeNarrative(t, tt.representation, &content, nil, expected)
			want := nativeNarrativeResult{determination: planrepresentation.Satisfied}
			if diff := cmp.Diff(want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeNarrativeObservationGlobalStructure(t *testing.T) {
	nativeDueOn := "2028-02-29T00:00:00Z"
	tests := []struct {
		name           string
		representation githubplan.Representation
		content        *string
		dueOn          *string
		date           string
		want           nativeNarrativeResult
	}{
		{
			name: "issue body unavailable", representation: githubplan.IssueRepresentation,
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}},
		},
		{
			name: "missing goal", representation: githubplan.MilestoneRepresentation,
			content: func() *string { value := "## Acceptance Conditions\n\n### 1\n\nA\n"; return &value }(),
			want:    nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
		{
			name: "duplicate goal", representation: githubplan.MilestoneRepresentation,
			content: func() *string {
				value := "## Goal\n\nGoal\n\n## Goal\n\nOther\n\n## Acceptance Conditions\n\n### 1\n\nA\n"
				return &value
			}(),
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
		{
			name: "duplicate acceptance heading", representation: githubplan.MilestoneRepresentation,
			content: func() *string {
				value := "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Acceptance Conditions\n"
				return &value
			}(),
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
		{
			name: "required headings out of order", representation: githubplan.MilestoneRepresentation,
			content: func() *string { value := "## Acceptance Conditions\n\n## Goal\n\nGoal\n"; return &value }(),
			want:    nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
		{
			name: "goal opening malformed", representation: githubplan.MilestoneRepresentation,
			content: func() *string { value := "## Goal\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n"; return &value }(),
			want:    nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
		{
			name: "issue target date before conditions", representation: githubplan.IssueRepresentation,
			content: func() *string {
				value := "## Goal\n\nGoal\n\n## Target Date\n\n2028-02-29\n\n## Acceptance Conditions\n\n### 1\n\nA\n"
				return &value
			}(),
			date: "2028-02-29",
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}},
		},
		{
			name: "issue target date duplicated", representation: githubplan.IssueRepresentation,
			content: func() *string {
				value := "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-02-29\n\n## Target Date\n\n2028-03-01\n"
				return &value
			}(),
			date: "2028-02-29",
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}},
		},
		{
			name: "milestone narrative target date preserves due on", representation: githubplan.MilestoneRepresentation,
			content: func() *string {
				value := "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-03-01\n"
				return &value
			}(),
			dueOn: &nativeDueOn, date: "2028-02-29",
			want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var date *plan.TargetDate
			if tt.date != "" {
				parsed := must(plan.ParseTargetDate(tt.date))
				date = &parsed
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, date))
			got := observeNativeNarrative(t, tt.representation, tt.content, tt.dueOn, expected)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeNarrativeObservationConditionFramingAndOrdinals(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    nativeNarrativeResult
	}{
		{
			name:    "known prefix remains observable",
			content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 4\n\nLater\n",
			want: nativeNarrativeResult{
				determination: planrepresentation.Undecidable,
				categories:    []planrepresentation.DifferenceCategory{planrepresentation.UnexpectedPresentCategory},
				locations:     []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation},
				violations:    []plan.ViolationCode{""}, members: []string{"First"},
				unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
			},
		},
		{name: "duplicate ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 1\n\nLater\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "wrong first ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 2\n\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "gap ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 3\n\nLater\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "does not resume after ordinal defect", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 3\n\nDefect\n\n### 2\n\nCanonical later\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "out of order ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 2\n\nSecond\n\n### 1\n\nLater\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, categories: []planrepresentation.DifferenceCategory{planrepresentation.UnexpectedPresentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{""}, members: []string{"Second"}, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "zero ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 0\n\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "leading zero ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 01\n\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "nonnumeric ordinal", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### one\n\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "ordinal opening malformed", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "first member delimiter malformed", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n### 1\n\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "later member delimiter malformed", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n### 2\n\nSecond\n", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "final framing missing", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}}},
		{name: "complete empty", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.ExpectedAbsentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{""}, members: []string{"First"}}},
		{name: "nonreserved headings remain member text", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\n## Notes\n###\tmember\nFirst\n", want: nativeNarrativeResult{determination: planrepresentation.Satisfied}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := "First"
			if tt.name == "known prefix remains observable" {
				condition = "Expected"
			}
			if tt.name == "nonreserved headings remain member text" {
				condition = "## Notes\n###\tmember\nFirst"
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition(condition))}, nil, nil))
			got := observeNativeNarrative(t, githubplan.MilestoneRepresentation, &tt.content, nil, expected)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeNarrativeObservationIssueTargetDateStates(t *testing.T) {
	tests := []struct {
		name    string
		content string
		date    string
		want    nativeNarrativeResult
	}{
		{name: "absent", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n", want: nativeNarrativeResult{determination: planrepresentation.Satisfied}},
		{name: "valid", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Satisfied}},
		{name: "complete empty dated", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n## Target Date\n\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.ExpectedAbsentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{""}, members: []string{"A"}}},
		{name: "empty", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, violations: []plan.ViolationCode{plan.InvalidTargetDate}, members: []string{""}}},
		{name: "whitespace", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n \t\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, violations: []plan.ViolationCode{plan.InvalidTargetDate}, members: []string{""}}},
		{name: "noncanonical", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-2-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, violations: []plan.ViolationCode{plan.InvalidTargetDate}, members: []string{""}}},
		{name: "impossible", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2027-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, violations: []plan.ViolationCode{plan.InvalidTargetDate}, members: []string{""}}},
		{name: "multiline", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-02-29\nextra\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, violations: []plan.ViolationCode{plan.InvalidTargetDate}, members: []string{""}}},
		{name: "opening boundary malformed", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n## Target Date\n\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening after heading malformed", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening truncated at heading", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening truncated after heading LF", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening malformed cannot establish empty collection", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n## Target Date\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening malformed preserves only preceding framed members", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 2\n\nUnclosed\n\n## Target Date\n2028-02-29\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, categories: []planrepresentation.DifferenceCategory{planrepresentation.UnexpectedPresentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{""}, members: []string{"First"}, unavailable: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation}}},
		{name: "opening without value or final framing", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}}},
		{name: "final framing missing", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n## Target Date\n\n2028-02-29", date: "2028-02-29", want: nativeNarrativeResult{determination: planrepresentation.Undecidable, unavailable: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var date *plan.TargetDate
			if tt.date != "" {
				parsed := must(plan.ParseTargetDate(tt.date))
				date = &parsed
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, nil, date))
			got := observeNativeNarrative(t, githubplan.IssueRepresentation, &tt.content, nil, expected)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNativeNarrativeObservationPlanValueClassification(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		conditions []string
		want       nativeNarrativeResult
	}{
		{name: "blank goal", content: "## Goal\n\n\n\n## Acceptance Conditions\n\n### 1\n\nA\n", conditions: []string{"A"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.GoalLocation}, violations: []plan.ViolationCode{plan.InvalidText}, members: []string{""}}},
		{name: "unicode whitespace goal", content: "## Goal\n\n \u2003\n\n## Acceptance Conditions\n\n### 1\n\nA\n", conditions: []string{"A"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.GoalLocation}, violations: []plan.ViolationCode{plan.InvalidText}, members: []string{""}}},
		{name: "blank condition", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\n\n", conditions: []string{"A"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory, planrepresentation.ExpectedAbsentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{plan.InvalidText, ""}, members: []string{"", "A"}}},
		{name: "unicode whitespace condition", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\n \u2003\n", conditions: []string{"A"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory, planrepresentation.ExpectedAbsentCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.AcceptanceConditionMemberLocation}, violations: []plan.ViolationCode{plan.InvalidText, ""}, members: []string{"", "A"}}},
		{name: "duplicate conditions", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nA\n\n### 2\n\nA\n", conditions: []string{"A"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation}, violations: []plan.ViolationCode{plan.DuplicateAcceptanceCondition}, members: []string{""}}},
		{name: "mixed valid and invalid members", content: "## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nBefore\n\n### 2\n\n\n\n### 3\n\nBefore\n\n### 4\n\nAfter\n", conditions: []string{"Before", "After"}, want: nativeNarrativeResult{determination: planrepresentation.NotSatisfied, categories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory, planrepresentation.InvalidObservedCategory}, locations: []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.AcceptanceConditionCollectionLocation}, violations: []plan.ViolationCode{plan.DuplicateAcceptanceCondition, plan.InvalidText}, members: []string{"", ""}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions := make([]plan.AcceptanceCondition, len(tt.conditions))
			for index, statement := range tt.conditions {
				conditions[index] = must(plan.NewAcceptanceCondition(statement))
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), conditions, nil, nil))
			got := observeNativeNarrative(t, githubplan.MilestoneRepresentation, &tt.content, nil, expected)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(nativeNarrativeResult{})); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
