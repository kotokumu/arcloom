package githubplan_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/githubplan"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
)

type payloadObservationRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *payloadObservationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type payloadObservationBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *payloadObservationBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		return n, nil
	}
	if b.readError != nil {
		return 0, b.readError
	}
	return 0, io.EOF
}
func (b *payloadObservationBody) Close() error { *b.closed = true; return nil }

func TestPayloadObservationPublicResultMatrix(t *testing.T) {
	tests := []struct {
		name              string
		representation    githubplan.Representation
		content           string
		goal              string
		conditions        []string
		targetDate        string
		wantDetermination planrepresentation.Determination
		wantCategories    []planrepresentation.DifferenceCategory
		wantLocations     []planrepresentation.LocationKind
		wantViolations    []plan.ViolationCode
		wantMembers       []string
		wantUnavailable   []planrepresentation.LocationKind
	}{
		{
			name:           "milestone accepts member order JSON whitespace escapes Unicode and ignored unknown nested duplicate",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte("\n\t{\r\n\t\t\"acceptance_conditions\" : [\"A\\nB\",\"slash \\/ \\u0041 \\u00a9 \\uD83D\\uDE00\"],\n\t\t\"unknown\":{\"duplicate\":1,\"duplicate\":2,\"bad\":\"\\uD800\"},\n\t\t\"goal\":\"Goal \\u0041 \\u00A9 \\uD83D\\uDE00\"\n\t}\r\n")) + "\n-->\n\nfirst narrative",
			goal:           "Goal A © 😀", conditions: []string{"A\nB", "slash / A © 😀"},
			wantDetermination: planrepresentation.Satisfied,
		},
		{
			name:           "issue accepts reordered fields and ignores human narrative",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(` { "target_date": "2028-02-29", "acceptance_conditions": ["A\/B"], "goal": "Goal" } `)) + "\n-->\n\nsecond narrative with \\uD800",
			goal:           "Goal", conditions: []string{"A/B"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Satisfied,
		},
		{
			name:           "absent issue envelope",
			representation: githubplan.IssueRepresentation,
			content:        "", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "unsupported milestone version",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v2\nabc\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "issue suffix absent",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\ne30=", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "padded base64url",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\ne30=\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "standard base64 alphabet",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n////\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "invalid encoded length",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\na\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "nonzero base64 unused bits",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\nZh\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "decoded payload invalid UTF-8",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte{0xff}) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "malformed JSON",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "non-object JSON",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`null`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "trailing non-whitespace",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["A"]}x`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "duplicate top-level member",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"A","goal":"B","acceptance_conditions":[],"target_date":null}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TargetDateLocation},
		},
		{
			name:           "unpaired surrogate top-level name",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"\uD800":1,"goal":"Goal","acceptance_conditions":["A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "missing goal is field-local",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"acceptance_conditions":["A"],"target_date":"2028-02-29"}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation},
		},
		{
			name:           "wrong goal type is field-local",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":null,"acceptance_conditions":["A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation},
		},
		{
			name:           "missing conditions is field-local",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","target_date":"2028-02-29"}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "wrong conditions type is field-local",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":null}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "missing issue target date is field-local",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.TargetDateLocation},
		},
		{
			name:           "wrong issue target date type is field-local",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["A"],"target_date":123}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.TargetDateLocation},
		},
		{
			name:           "mixed conditions keep local members and aggregate violations",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["before",1,"","before",{"nested":1,"nested":2},"after"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"before", "after"},
			wantDetermination: planrepresentation.Undecidable,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory, planrepresentation.InvalidObservedCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.AcceptanceConditionCollectionLocation},
			wantViolations:    []plan.ViolationCode{plan.DuplicateAcceptanceCondition, plan.InvalidText},
			wantMembers:       []string{"", ""},
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "duplicate conditions are incomplete and invalid",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["A","A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
			wantViolations:    []plan.ViolationCode{plan.DuplicateAcceptanceCondition},
			wantMembers:       []string{""},
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "known goal unpaired surrogate",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"\uD800","acceptance_conditions":["A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.GoalLocation},
		},
		{
			name:           "known condition unpaired surrogate",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["\uD800","A"]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.Undecidable,
			wantUnavailable:   []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "empty conditions are known absent",
			representation: githubplan.MilestoneRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":[]}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"},
			wantDetermination: planrepresentation.NotSatisfied,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.ExpectedAbsentCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.AcceptanceConditionMemberLocation},
			wantViolations:    []plan.ViolationCode{""}, wantMembers: []string{"A"},
		},
		{
			name:           "blank issue goal is Plan-owned violation",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"","acceptance_conditions":["A"],"target_date":"2028-02-29"}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.NotSatisfied,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.GoalLocation},
			wantViolations:    []plan.ViolationCode{plan.InvalidText}, wantMembers: []string{""},
		},
		{
			name:           "simultaneous issue faults remain independent",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"acceptance_conditions":["A",1],"target_date":"2028-02-30"}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.Undecidable,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.TargetDateLocation},
			wantViolations:    []plan.ViolationCode{plan.InvalidTargetDate}, wantMembers: []string{""},
			wantUnavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation},
		},
		{
			name:           "issue target date null is known absent",
			representation: githubplan.IssueRepresentation,
			content:        "<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(`{"goal":"Goal","acceptance_conditions":["A"],"target_date":null}`)) + "\n-->\n", goal: "Goal", conditions: []string{"A"}, targetDate: "2028-02-29",
			wantDetermination: planrepresentation.NotSatisfied,
			wantCategories:    []planrepresentation.DifferenceCategory{planrepresentation.ExpectedAbsentCategory},
			wantLocations:     []planrepresentation.LocationKind{planrepresentation.TargetDateLocation},
			wantViolations:    []plan.ViolationCode{""}, wantMembers: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := "description"
			if tt.representation == githubplan.IssueRepresentation {
				field = "body"
			}
			encodedContent := must(json.Marshal(tt.content))
			root := `{"number":42,"title":"Plan","` + field + `":` + string(encodedContent) + `}`
			rootClosed, tasksClosed := false, false
			transport := &payloadObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &payloadObservationBody{data: []byte(root), closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &payloadObservationBody{data: []byte("[]"), closed: &tasksClosed}, Header: make(http.Header)},
			}}
			client := &http.Client{Transport: transport}
			repository := must(githubplan.NewRepository("owner", "repo"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			accepted := make([]plan.AcceptanceCondition, len(tt.conditions))
			for index, condition := range tt.conditions {
				accepted[index] = must(plan.NewAcceptanceCondition(condition))
			}
			var targetDate *plan.TargetDate
			if tt.targetDate != "" {
				parsed := must(plan.ParseTargetDate(tt.targetDate))
				targetDate = &parsed
			}
			expected := must(plan.New("Plan", must(plan.NewGoal(tt.goal)), accepted, nil, targetDate))
			controller := must(planrepresentation.NewController(observer))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			var gotCategories []planrepresentation.DifferenceCategory
			var gotLocations []planrepresentation.LocationKind
			var gotViolations []plan.ViolationCode
			var gotMembers []string
			for _, difference := range result.Differences() {
				gotCategories = append(gotCategories, difference.Category())
				gotLocations = append(gotLocations, difference.Location().Kind())
				violation := plan.ViolationCode("")
				if invalid, ok := difference.(planrepresentation.InvalidObservedDifference); ok {
					violation = invalid.Violation()
				}
				gotViolations = append(gotViolations, violation)
				member, _ := difference.Location().Member()
				gotMembers = append(gotMembers, member)
			}
			var gotUnavailable []planrepresentation.LocationKind
			for _, unavailable := range result.UnavailableInformation() {
				gotUnavailable = append(gotUnavailable, unavailable.Location().Kind())
			}

			if diff := cmp.Diff(tt.wantDetermination, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantCategories, gotCategories); diff != "" {
				t.Errorf("difference categories mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantLocations, gotLocations); diff != "" {
				t.Errorf("difference locations mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantViolations, gotViolations); diff != "" {
				t.Errorf("difference violations mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantMembers, gotMembers); diff != "" {
				t.Errorf("difference members mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantUnavailable, gotUnavailable); diff != "" {
				t.Errorf("unavailable locations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
