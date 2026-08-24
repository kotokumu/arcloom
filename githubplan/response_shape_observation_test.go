package githubplan_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/githubplan"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
	"github.com/kotokumu/arcloom/reconciliation"
)

type responseShapeRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *responseShapeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type responseShapeBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *responseShapeBody) Read(p []byte) (int, error) {
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
func (b *responseShapeBody) Close() error { *b.closed = true; return nil }

func TestResponseShapePublicResultMatrix(t *testing.T) {
	type responseCase struct {
		name              string
		representation    githubplan.Representation
		root              string
		collection        string
		expectedTasks     []string
		expectedDate      string
		wantDetermination reconciliation.Determination
		wantCategories    []planrepresentation.DifferenceCategory
		wantLocations     []planrepresentation.LocationKind
		wantViolations    []plan.ViolationCode
		wantUnavailable   []planrepresentation.LocationKind
	}
	var tests []responseCase
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		for _, root := range []struct{ name, body string }{
			{name: "root null", body: "null"},
			{name: "root array", body: "[]"},
			{name: "root malformed", body: `{"number":42`},
			{name: "root trailing", body: `{"number":42,"title":"Plan"} true`},
		} {
			tests = append(tests, responseCase{name: string(representation) + "/" + root.name, representation: representation, root: root.body, collection: "[]", wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.PlanRootLocation}})
		}
		for _, number := range []struct{ name, value string }{
			{name: "number missing", value: ""},
			{name: "number null", value: "null"},
			{name: "number zero", value: "0"},
			{name: "number negative", value: "-1"},
			{name: "number fraction", value: "42.0"},
			{name: "number overflow", value: "9223372036854775808"},
			{name: "number binding mismatch", value: "41"},
		} {
			prefix := ""
			if number.value != "" {
				prefix = `"number":` + number.value + `,`
			}
			field := `"description":""`
			if representation == githubplan.IssueRepresentation {
				field = `"body":""`
			}
			tests = append(tests, responseCase{name: string(representation) + "/" + number.name, representation: representation, root: `{` + prefix + `"title":"Plan",` + field + `}`, collection: "[]", wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.PlanRootLocation}})
		}

		validRoot := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
		if representation == githubplan.IssueRepresentation {
			validRoot = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
		}
		for _, collection := range []struct{ name, body string }{
			{name: "collection null", body: "null"},
			{name: "collection object", body: `{}`},
			{name: "collection malformed", body: `[{"id":1}`},
			{name: "collection trailing", body: `[] false`},
		} {
			tests = append(tests, responseCase{name: string(representation) + "/" + collection.name, representation: representation, root: validRoot, collection: collection.body, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}})
		}
		for _, item := range []struct{ name, body string }{
			{name: "item id missing", body: `[{"number":7,"node_id":"N","title":"Task"}]`},
			{name: "item id null", body: `[{"id":null,"title":"Task"}]`},
			{name: "item id zero", body: `[{"id":0,"title":"Task"}]`},
			{name: "item id negative", body: `[{"id":-1,"title":"Task"}]`},
			{name: "item id fraction", body: `[{"id":1.5,"title":"Task"}]`},
			{name: "item id overflow", body: `[{"id":9223372036854775808,"title":"Task"}]`},
			{name: "item title missing", body: `[{"id":1}]`},
			{name: "item title null", body: `[{"id":1,"title":null}]`},
			{name: "item title wrong type", body: `[{"id":1,"title":123}]`},
		} {
			tests = append(tests, responseCase{name: string(representation) + "/" + item.name, representation: representation, root: validRoot, collection: item.body, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}})
		}
		tests = append(tests,
			responseCase{name: string(representation) + "/REST id alone determines identity", representation: representation, root: validRoot, collection: `[{"id":9001,"number":0,"node_id":"not-the-resource","title":"Task"}]`, expectedTasks: []string{"Task"}, wantDetermination: reconciliation.Satisfied},
			responseCase{name: string(representation) + "/same id same title incomplete", representation: representation, root: validRoot, collection: `[{"id":7,"number":1,"node_id":"A","title":"Task"},{"id":7,"number":2,"node_id":"B","title":"Task"}]`, expectedTasks: []string{"Task"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}},
			responseCase{name: string(representation) + "/same id conflicting title incomplete", representation: representation, root: validRoot, collection: `[{"id":7,"title":"First"},{"id":7,"title":"Second"}]`, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}},
			responseCase{name: string(representation) + "/distinct ids duplicate title", representation: representation, root: validRoot, collection: `[{"id":7,"title":"Task"},{"id":8,"title":"Task"}]`, expectedTasks: []string{"Task"}, wantDetermination: reconciliation.NotSatisfied, wantCategories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, wantLocations: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantViolations: []plan.ViolationCode{plan.DuplicateTask}},
		)
	}

	milestonePayload := `<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->`
	tests = append(tests,
		responseCase{name: "milestone due absent", representation: githubplan.MilestoneRepresentation, root: `{"number":42,"title":"Plan","description":"` + milestonePayload + `"}`, collection: "[]", wantDetermination: reconciliation.Satisfied},
		responseCase{name: "milestone due null", representation: githubplan.MilestoneRepresentation, root: `{"number":42,"title":"Plan","description":"` + milestonePayload + `","due_on":null}`, collection: "[]", wantDetermination: reconciliation.Satisfied},
		responseCase{name: "milestone due wrong type", representation: githubplan.MilestoneRepresentation, root: `{"number":42,"title":"Plan","description":"` + milestonePayload + `","due_on":123}`, collection: "[]", wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}},
		responseCase{name: "issue simultaneous payload and item faults", representation: githubplan.IssueRepresentation, root: `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJhY2NlcHRhbmNlX2NvbmRpdGlvbnMiOlsiQSIsMV0sInRhcmdldF9kYXRlIjoiMjAyOC0wMi0zMCJ9\n-->"}`, collection: `[{"id":0,"title":"Task"}]`, expectedTasks: []string{"Task"}, expectedDate: "2028-02-29", wantDetermination: reconciliation.Undecidable, wantCategories: []planrepresentation.DifferenceCategory{planrepresentation.InvalidObservedCategory}, wantLocations: []planrepresentation.LocationKind{planrepresentation.TargetDateLocation}, wantViolations: []plan.ViolationCode{plan.InvalidTargetDate}, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.GoalLocation, planrepresentation.AcceptanceConditionCollectionLocation, planrepresentation.TaskCollectionLocation}},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			transport := &responseShapeRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &responseShapeBody{data: []byte(tt.root), closed: &rootClosed}},
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &responseShapeBody{data: []byte(tt.collection), closed: &collectionClosed}},
			}}
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			tasks := make([]plan.Task, len(tt.expectedTasks))
			for index, task := range tt.expectedTasks {
				tasks[index] = must(plan.NewTask(task))
			}
			var date *plan.TargetDate
			if tt.expectedDate != "" {
				parsed := must(plan.ParseTargetDate(tt.expectedDate))
				date = &parsed
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, tasks, date))
			result, err := must(planrepresentation.NewController(observer)).Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			var categories []planrepresentation.DifferenceCategory
			var locations []planrepresentation.LocationKind
			var violations []plan.ViolationCode
			for _, difference := range result.Differences() {
				categories = append(categories, difference.Category())
				locations = append(locations, difference.Location().Kind())
				violation := plan.ViolationCode("")
				if invalid, ok := difference.(planrepresentation.InvalidObservedDifference); ok {
					violation = invalid.Violation()
				}
				violations = append(violations, violation)
			}
			var unavailable []planrepresentation.LocationKind
			for _, fact := range result.UnavailableInformation() {
				unavailable = append(unavailable, fact.Location().Kind())
			}
			if diff := cmp.Diff(tt.wantDetermination, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantCategories, categories); diff != "" {
				t.Errorf("categories mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantLocations, locations); diff != "" {
				t.Errorf("locations mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantViolations, violations); diff != "" {
				t.Errorf("violations mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantUnavailable, unavailable); diff != "" {
				t.Errorf("unavailable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
