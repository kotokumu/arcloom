package githubplan_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/githubplan"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
	"github.com/kotokumu/arcloom/reconciliation"
)

type paginationObservationRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *paginationObservationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type paginationObservationBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *paginationObservationBody) Read(p []byte) (int, error) {
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
func (b *paginationObservationBody) Close() error { *b.closed = true; return nil }

func TestPaginationPublicResultAndRequestMatrix(t *testing.T) {
	milestonePage2 := "https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all"
	issuePage2 := "https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100"
	tests := []struct {
		name              string
		representation    githubplan.Representation
		pageBodies        []string
		pageStatuses      []int
		links             []string
		expectedTasks     []string
		wantDetermination reconciliation.Determination
		wantUnavailable   []planrepresentation.LocationKind
		wantViolation     plan.ViolationCode
		wantRequests      int
		wantLastURL       string
	}{
		{name: "milestone multi-page complete", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"},{"id":8,"title":"Second"}]`, `[{"id":9,"title":"Third"}]`}, links: []string{`<` + milestonePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"First", "Second", "Third"}, wantDetermination: reconciliation.Satisfied, wantRequests: 3, wantLastURL: milestonePage2},
		{name: "issue multi-page complete", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`, `[{"id":9,"title":"Third"}]`}, links: []string{`<` + issuePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"First", "Third"}, wantDetermination: reconciliation.Satisfied, wantRequests: 3, wantLastURL: issuePage2},
		{name: "later page failure preserves coherent member", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`, `provider detail`}, pageStatuses: []int{http.StatusOK, http.StatusInternalServerError}, links: []string{`<` + milestonePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 3},
		{name: "same id same title is one incomplete member", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"A"}]`, `[{"id":7,"title":"A"}]`}, links: []string{`<` + issuePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"A"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 3},
		{name: "same id conflict A A B", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"A"}]`, `[{"id":7,"title":"A"}]`, `[{"id":7,"title":"B"}]`}, links: []string{`<` + milestonePage2 + `>; rel="next"`, `<https://api.github.com/repos/owner/repo/issues?milestone=42&page=3&per_page=100&state=all>; rel="next"`, ""}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 4},
		{name: "same id conflict A B A", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"A"}]`, `[{"id":7,"title":"B"}]`, `[{"id":7,"title":"A"}]`}, links: []string{`<` + milestonePage2 + `>; rel="next"`, `<https://api.github.com/repos/owner/repo/issues?milestone=42&page=3&per_page=100&state=all>; rel="next"`, ""}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 4},
		{name: "same id conflict reversed page order", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"B"}]`, `[{"id":7,"title":"A"}]`, `[{"id":7,"title":"A"}]`}, links: []string{`<` + issuePage2 + `>; rel="next"`, `<https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=3&per_page=100>; rel="next"`, ""}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 4},
		{name: "distinct ids equal title are Plan duplicate violation", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"Same"}]`, `[{"id":8,"title":"Same"}]`}, links: []string{`<` + milestonePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"Same"}, wantDetermination: reconciliation.NotSatisfied, wantViolation: plan.DuplicateTask, wantRequests: 3},
		{name: "valid next among other relations", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`, `[]`}, links: []string{`<` + issuePage2 + `>; rel="last", <` + issuePage2 + `>; rel="next"`, ""}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Satisfied, wantRequests: 3, wantLastURL: issuePage2},
		{name: "duplicate rel next", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<` + milestonePage2 + `>; rel="next", <` + milestonePage2 + `>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "milestone 41", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues?milestone=41&page=2&per_page=100&state=all>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "missing milestone", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues?page=2&per_page=100&state=all>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "duplicate milestone", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<` + milestonePage2 + `&milestone=42>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "missing state", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "duplicate state", representation: githubplan.MilestoneRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<` + milestonePage2 + `&state=all>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "missing per page", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=2>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "duplicate per page", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<` + issuePage2 + `&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "changed scheme", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<http://api.github.com/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "changed host", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://evil.example/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "changed port", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com:8443/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "userinfo", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://user@api.github.com/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "changed path", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/other/issues/42/sub_issues?page=2&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "additional selection query", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<` + issuePage2 + `&labels=bug>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "zero page", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=0&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "negative page", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=-1&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
		{name: "revisited page rejected before request", representation: githubplan.IssueRepresentation, pageBodies: []string{`[{"id":7,"title":"First"}]`}, links: []string{`<https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=1&per_page=100>; rel="next"`}, expectedTasks: []string{"First"}, wantDetermination: reconciliation.Undecidable, wantUnavailable: []planrepresentation.LocationKind{planrepresentation.TaskCollectionLocation}, wantRequests: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
			if tt.representation == githubplan.IssueRepresentation {
				root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
			}
			closed := make([]bool, 1+len(tt.pageBodies))
			responses := []*http.Response{{StatusCode: http.StatusOK, Header: make(http.Header), Body: &paginationObservationBody{data: []byte(root), closed: &closed[0]}}}
			for index, body := range tt.pageBodies {
				status := http.StatusOK
				if index < len(tt.pageStatuses) && tt.pageStatuses[index] != 0 {
					status = tt.pageStatuses[index]
				}
				header := make(http.Header)
				if index < len(tt.links) && tt.links[index] != "" {
					header.Set("Link", tt.links[index])
				}
				responses = append(responses, &http.Response{StatusCode: status, Header: header, Body: &paginationObservationBody{data: []byte(body), closed: &closed[index+1]}})
			}
			transport := &paginationObservationRoundTripper{responses: responses}
			client := &http.Client{Transport: transport}
			repository := must(githubplan.NewRepository("owner", "repo"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			tasks := make([]plan.Task, len(tt.expectedTasks))
			for index, title := range tt.expectedTasks {
				tasks[index] = must(plan.NewTask(title))
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, tasks, nil))
			controller := must(planrepresentation.NewController(observer))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			var unavailable []planrepresentation.LocationKind
			for _, fact := range result.UnavailableInformation() {
				unavailable = append(unavailable, fact.Location().Kind())
			}
			if diff := cmp.Diff(tt.wantDetermination, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantUnavailable, unavailable); diff != "" {
				t.Errorf("unavailable locations mismatch (-want +got):\n%s", diff)
			}
			if tt.wantViolation == "" {
				if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
					t.Errorf("differences mismatch (-want +got):\n%s", diff)
				}
			} else {
				differences := result.Differences()
				if len(differences) != 1 {
					t.Fatalf("differences = %v, want one", differences)
				}
				invalid, ok := differences[0].(planrepresentation.InvalidObservedDifference)
				if !ok || differences[0].Location().Kind() != planrepresentation.TaskCollectionLocation || invalid.Violation() != tt.wantViolation {
					t.Errorf("difference = %v, want Task collection violation %v", differences[0], tt.wantViolation)
				}
			}
			if diff := cmp.Diff(tt.wantRequests, len(transport.requests)); diff != "" {
				t.Errorf("request count mismatch (-want +got):\n%s", diff)
			}
			if tt.wantLastURL != "" && len(transport.requests) > 0 {
				if diff := cmp.Diff(tt.wantLastURL, transport.requests[len(transport.requests)-1].URL.String()); diff != "" {
					t.Errorf("last request URL mismatch (-want +got):\n%s", diff)
				}
			}
			for index, wasClosed := range closed {
				if !wasClosed {
					t.Errorf("response %d body was not closed", index)
				}
			}
		})
	}
}

func TestPagination101ResourcesIncludesLaterPage(t *testing.T) {
	var firstItems []string
	var expectedTasks []plan.Task
	for index := 1; index <= 100; index++ {
		firstItems = append(firstItems, fmt.Sprintf(`{"id":%d,"title":"Task %03d"}`, index, index))
		expectedTasks = append(expectedTasks, must(plan.NewTask(fmt.Sprintf("Task %03d", index))))
	}
	expectedTasks = append(expectedTasks, must(plan.NewTask("Task 101")))
	rootClosed, firstClosed, secondClosed := false, false, false
	transport := &paginationObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &paginationObservationBody{data: []byte(`{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`), closed: &rootClosed}},
		{StatusCode: http.StatusOK, Header: http.Header{"Link": []string{`<https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all>; rel="next"`}}, Body: &paginationObservationBody{data: []byte("[" + strings.Join(firstItems, ",") + "]"), closed: &firstClosed}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &paginationObservationBody{data: []byte(`[{"id":101,"title":"Task 101"}]`), closed: &secondClosed}},
	}}
	observer := must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, expectedTasks, nil))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.Determination() != reconciliation.Satisfied || len(result.Differences()) != 0 || len(result.UnavailableInformation()) != 0 {
		t.Fatalf("Result = (%v, %v, %v), want complete satisfied", result.Determination(), result.Differences(), result.UnavailableInformation())
	}
	if diff := cmp.Diff(3, len(transport.requests)); diff != "" {
		t.Errorf("request count mismatch (-want +got):\n%s", diff)
	}
	if !rootClosed || !firstClosed || !secondClosed {
		t.Errorf("body close = (%v,%v,%v), want all true", rootClosed, firstClosed, secondClosed)
	}
}
