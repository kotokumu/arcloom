package githubplan_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type issueObservationRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *issueObservationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, request)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type issueObservationBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *issueObservationBody) Read(destination []byte) (int, error) {
	if len(b.data) > 0 {
		count := copy(destination, b.data)
		b.data = b.data[count:]
		return count, nil
	}
	if b.readError != nil {
		return 0, b.readError
	}
	return 0, io.EOF
}

func (b *issueObservationBody) Close() error {
	*b.closed = true
	return nil
}

func TestIssueNativeObservationMapsParentAndSubIssues(t *testing.T) {
	rootClosed, tasksClosed := false, false
	transport := &issueObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`{"number":42,"title":"Parent","body":""}`), closed: &rootClosed}, Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`[{"id":1007,"number":7,"title":"First task","repository":{"full_name":"other/repo"}},{"id":1008,"number":8,"title":"Second task","repository":{"full_name":"owner/repo"}}]`), closed: &tasksClosed}, Header: make(http.Header)},
	}}
	observer := must(githubplan.NewIssueObserver(
		&http.Client{Transport: transport},
		must(githubplan.NewRepository("owner", "repo")),
		must(githubplan.NewResourceNumber(42)),
	))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New(
		"Parent",
		must(plan.NewGoal("goal")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))},
		[]plan.Task{must(plan.NewTask("First task")), must(plan.NewTask("Second task"))},
		nil,
	))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
		t.Errorf("determination mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
		t.Errorf("difference count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(3, len(result.UnavailableInformation())); diff != "" {
		t.Errorf("payload unavailable count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, len(transport.requests)); diff != "" {
		t.Errorf("request count mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueSubIssuesRequirePositiveRESTIDWithoutRepositoryRestriction(t *testing.T) {
	rootClosed, tasksClosed := false, false
	transport := &issueObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`{"number":42,"title":"Parent"}`), closed: &rootClosed}, Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`[{"id":1007,"number":7,"title":"Accepted","repository":{"full_name":"other/repo"}},{"id":0,"number":8,"title":"Zero id","repository":{"full_name":"owner/repo"}},{"id":"1009","number":9,"title":"String id","repository":{"full_name":"owner/repo"}}]`), closed: &tasksClosed}, Header: make(http.Header)},
	}}
	observer := must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New("Parent", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("Accepted"))}, nil))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
		t.Errorf("difference count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(4, len(result.UnavailableInformation())); diff != "" {
		t.Errorf("invalid identity unavailable count mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueParentPullRequestMakesRootUnavailable(t *testing.T) {
	closed := false
	transport := &issueObservationRoundTripper{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Body:       &issueObservationBody{data: []byte(`{"number":42,"title":"Parent","pull_request":{"url":"https://api.github.com/repos/owner/repo/pulls/42"}}`), closed: &closed},
		Header:     make(http.Header),
	}}}
	observer := must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New("Parent", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(1, len(result.UnavailableInformation())); diff != "" {
		t.Errorf("root unavailable count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(planrepresentation.PlanRootLocation, result.UnavailableInformation()[0].Location().Kind()); diff != "" {
		t.Errorf("root unavailable location mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, len(transport.requests)); diff != "" {
		t.Errorf("root PR request count mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueUnexpectedSubIssuePullRequestIsExcludedAndIncomplete(t *testing.T) {
	rootClosed, tasksClosed := false, false
	transport := &issueObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`{"number":42,"title":"Parent","body":""}`), closed: &rootClosed}, Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`[{"id":1007,"number":7,"title":"Task"},{"id":1008,"number":8,"title":"PR","pull_request":{"url":"https://api.github.com/repos/owner/repo/pulls/8"}}]`), closed: &tasksClosed}, Header: make(http.Header)},
	}}
	observer := must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New("Parent", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
		t.Errorf("difference count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(4, len(result.UnavailableInformation())); diff != "" {
		t.Errorf("incomplete observation unavailable count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, len(transport.requests)); diff != "" {
		t.Errorf("request count mismatch (-want +got):\n%s", diff)
	}
}

func TestIssueFirstPageTaskBoundaries(t *testing.T) {
	for _, count := range []int{0, 1, 100} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var collection strings.Builder
			collection.WriteByte('[')
			tasks := make([]plan.Task, 0, count)
			for index := 0; index < count; index++ {
				if index > 0 {
					collection.WriteByte(',')
				}
				title := "Task " + strconv.Itoa(index+1)
				_, _ = fmt.Fprintf(&collection, `{"id":%d,"number":%d,"title":%q}`, index+1, index+1, title)
				tasks = append(tasks, must(plan.NewTask(title)))
			}
			collection.WriteByte(']')
			rootClosed, tasksClosed := false, false
			transport := &issueObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(`{"number":42,"title":"Parent"}`), closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &issueObservationBody{data: []byte(collection.String()), closed: &tasksClosed}, Header: make(http.Header)},
			}}
			observer := must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Parent", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, tasks, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("difference count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(3, len(result.UnavailableInformation())); diff != "" {
				t.Errorf("payload unavailable count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIssueResponseShapeFailuresLocalizeKnowledge(t *testing.T) {
	tests := []struct {
		name             string
		rootStatus       int
		rootBody         string
		collectionStatus int
		collectionBody   string
		rootFailure      bool
	}{
		{name: "malformed root", rootStatus: http.StatusOK, rootBody: "not json", rootFailure: true},
		{name: "root http error", rootStatus: http.StatusForbidden, rootBody: "denied", rootFailure: true},
		{name: "malformed collection", rootStatus: http.StatusOK, rootBody: `{"number":42,"title":"Parent"}`, collectionStatus: http.StatusOK, collectionBody: "not json"},
		{name: "collection http error", rootStatus: http.StatusOK, rootBody: `{"number":42,"title":"Parent"}`, collectionStatus: http.StatusInternalServerError, collectionBody: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			responses := []*http.Response{{StatusCode: tt.rootStatus, Body: &issueObservationBody{data: []byte(tt.rootBody), closed: &rootClosed}, Header: make(http.Header)}}
			if !tt.rootFailure {
				responses = append(responses, &http.Response{StatusCode: tt.collectionStatus, Body: &issueObservationBody{data: []byte(tt.collectionBody), closed: &collectionClosed}, Header: make(http.Header)})
			}
			transport := &issueObservationRoundTripper{responses: responses}
			observer := must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Parent", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			wantUnavailable := 4
			if tt.rootFailure {
				wantUnavailable = 1
			}
			if diff := cmp.Diff(wantUnavailable, len(result.UnavailableInformation())); diff != "" {
				t.Errorf("unavailable count mismatch (-want +got):\n%s", diff)
			}
			if tt.rootFailure {
				if diff := cmp.Diff(planrepresentation.PlanRootLocation, result.UnavailableInformation()[0].Location().Kind()); diff != "" {
					t.Errorf("root unavailable location mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
