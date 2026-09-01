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

type milestoneObservationRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *milestoneObservationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type milestoneObservationBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *milestoneObservationBody) Read(p []byte) (int, error) {
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
func (b *milestoneObservationBody) Close() error { *b.closed = true; return nil }

func TestMilestoneNativeObservationMapsTitleTasksAndPullRequests(t *testing.T) {
	rootClosed, tasksClosed := false, false
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","description":"","due_on":null}`), closed: &rootClosed}, Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: []byte(`[{"id":101,"number":7,"title":"Open task","state":"open"},{"id":102,"number":8,"title":"Closed task","state":"closed"},{"id":103,"number":9,"title":"Pull request task","state":"closed","pull_request":{"url":"https://api.github.com/repos/owner/repo/pulls/9"}}]`), closed: &tasksClosed}, Header: make(http.Header)},
	}}
	observer := must(githubplan.NewMilestoneObserver(
		&http.Client{Transport: transport},
		must(githubplan.NewRepository("owner", "repo")),
		must(githubplan.NewResourceNumber(42)),
	))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New(
		"Plan",
		must(plan.NewGoal("goal")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))},
		[]plan.Task{must(plan.NewTask("Open task")), must(plan.NewTask("Closed task"))},
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
		t.Errorf("native difference count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, len(result.UnavailableInformation())); diff != "" {
		t.Fatalf("payload unavailable count mismatch (-want +got):\n%s", diff)
	}
	for _, unavailable := range result.UnavailableInformation() {
		if kind := unavailable.Location().Kind(); kind != planrepresentation.GoalLocation && kind != planrepresentation.AcceptanceConditionCollectionLocation {
			t.Errorf("unexpected unavailable location kind = %v", kind)
		}
	}
}

func TestMilestoneDueOnObservationStates(t *testing.T) {
	tests := []struct {
		name            string
		dueField        string
		expectedDate    *plan.TargetDate
		wantDifference  planrepresentation.DifferenceCategory
		wantViolation   plan.ViolationCode
		wantUnavailable bool
	}{
		{name: "absent", dueField: "", wantDifference: 0},
		{name: "null", dueField: `,"due_on":null`, wantDifference: 0},
		{name: "canonical", dueField: `,"due_on":"2028-02-29T00:00:00Z"`, expectedDate: func() *plan.TargetDate { date := must(plan.ParseTargetDate("2028-02-29")); return &date }(), wantDifference: 0},
		{name: "noncanonical time", dueField: `,"due_on":"2028-02-29T01:00:00Z"`, wantDifference: planrepresentation.InvalidObservedCategory, wantViolation: plan.InvalidTargetDate},
		{name: "wrong type", dueField: `,"due_on":123`, wantDifference: 0, wantUnavailable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, tasksClosed := false, false
			rootBody := []byte(`{"number":42,"title":"Plan","description":""` + tt.dueField + `}`)
			transport := &milestoneObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: rootBody, closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: []byte("[]"), closed: &tasksClosed}, Header: make(http.Header)},
			}}
			observer := must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, tt.expectedDate))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			dateFound := false
			for _, difference := range result.Differences() {
				if difference.Location().Kind() != planrepresentation.TargetDateLocation {
					continue
				}
				dateFound = true
				if diff := cmp.Diff(tt.wantDifference, difference.Category()); diff != "" {
					t.Errorf("target-date category mismatch (-want +got):\n%s", diff)
				}
				if invalid, ok := difference.(planrepresentation.InvalidObservedDifference); ok {
					if diff := cmp.Diff(tt.wantViolation, invalid.Violation()); diff != "" {
						t.Errorf("target-date violation mismatch (-want +got):\n%s", diff)
					}
				}
			}
			if diff := cmp.Diff(tt.wantDifference != 0, dateFound); diff != "" {
				t.Errorf("target-date difference presence mismatch (-want +got):\n%s", diff)
			}
			unavailableDate := false
			for _, unavailable := range result.UnavailableInformation() {
				if unavailable.Location().Kind() == planrepresentation.TargetDateLocation {
					unavailableDate = true
				}
			}
			if diff := cmp.Diff(tt.wantUnavailable, unavailableDate); diff != "" {
				t.Errorf("target-date unavailable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMilestoneFirstPageTaskBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{name: "zero tasks", count: 0},
		{name: "one task", count: 1},
		{name: "one hundred tasks", count: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var collection strings.Builder
			collection.WriteByte('[')
			tasks := make([]plan.Task, 0, tt.count)
			for index := 0; index < tt.count; index++ {
				if index > 0 {
					collection.WriteByte(',')
				}
				name := "Task " + strconv.Itoa(index+1)
				_, _ = fmt.Fprintf(&collection, `{"id":%d,"number":%d,"title":%q,"state":"open"}`, index+1, index+1, name)
				tasks = append(tasks, must(plan.NewTask(name)))
			}
			collection.WriteByte(']')
			rootClosed, tasksClosed := false, false
			transport := &milestoneObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","description":"","due_on":null}`), closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &milestoneObservationBody{data: []byte(collection.String()), closed: &tasksClosed}, Header: make(http.Header)},
			}}
			observer := must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, tasks, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
				t.Errorf("Task boundary difference count mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(2, len(result.UnavailableInformation())); diff != "" {
				t.Errorf("payload unavailable count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMilestoneResponseShapeFailuresLocalizeKnowledge(t *testing.T) {
	tests := []struct {
		name             string
		rootStatus       int
		rootBody         string
		collectionStatus int
		collectionBody   string
		wantRoot         bool
	}{
		{name: "malformed root", rootStatus: http.StatusOK, rootBody: "not json", wantRoot: true},
		{name: "root http error", rootStatus: http.StatusForbidden, rootBody: "denied", wantRoot: true},
		{name: "malformed collection", rootStatus: http.StatusOK, rootBody: `{"number":42,"title":"Plan","description":"","due_on":null}`, collectionStatus: http.StatusOK, collectionBody: "not json"},
		{name: "collection http error", rootStatus: http.StatusOK, rootBody: `{"number":42,"title":"Plan","description":"","due_on":null}`, collectionStatus: http.StatusInternalServerError, collectionBody: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			responses := []*http.Response{{StatusCode: tt.rootStatus, Body: &milestoneObservationBody{data: []byte(tt.rootBody), closed: &rootClosed}, Header: make(http.Header)}}
			if !tt.wantRoot {
				responses = append(responses, &http.Response{StatusCode: tt.collectionStatus, Body: &milestoneObservationBody{data: []byte(tt.collectionBody), closed: &collectionClosed}, Header: make(http.Header)})
			}
			transport := &milestoneObservationRoundTripper{responses: responses}
			observer := must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if tt.wantRoot {
				if diff := cmp.Diff(1, len(result.UnavailableInformation())); diff != "" {
					t.Errorf("root unavailable count mismatch (-want +got):\n%s", diff)
				}
				if kind := result.UnavailableInformation()[0].Location().Kind(); kind != planrepresentation.PlanRootLocation {
					t.Errorf("root unavailable location = %v", kind)
				}
			} else {
				if diff := cmp.Diff(3, len(result.UnavailableInformation())); diff != "" {
					t.Errorf("localized unavailable count mismatch (-want +got):\n%s", diff)
				}
				for _, unavailable := range result.UnavailableInformation() {
					if unavailable.Location().Kind() == planrepresentation.PlanRootLocation {
						t.Errorf("collection failure widened to root unavailable")
					}
				}
			}
		})
	}
}
