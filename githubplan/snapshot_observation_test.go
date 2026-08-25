package githubplan_test

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kotokumu/arcloom/githubplan"
	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plansnapshot"
)

func TestMilestoneSnapshotObserverReconstructsCurrentPlanAndProgress(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"closed","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"A","state":"open"},{"id":2,"title":"B","state":"closed"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("A")), must(plan.NewTask("B"))}, nil))
	current, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(true, hasCurrent); diff != "" {
		t.Fatalf("current Plan presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, current.Equal(expected)); diff != "" {
		t.Errorf("current Plan mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.Closed, progress.OverallState()); diff != "" {
		t.Errorf("overall state mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"A", "B"}, []string{progress.Tasks()[0].Name(), progress.Tasks()[1].Name()}); diff != "" {
		t.Errorf("task order mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Open, plansnapshot.Closed}, []plansnapshot.ProgressState{progress.Tasks()[0].State(), progress.Tasks()[1].State()}); diff != "" {
		t.Errorf("task states mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, progress.MembershipComplete()); diff != "" {
		t.Errorf("membership completeness mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverPreservesUnknownDuplicateAndIncompleteProgress(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"paused","description":"unusable"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"same"},{"id":2,"title":"same","state":"future"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(false, hasCurrent); diff != "" {
		t.Errorf("current Plan presence mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.Unknown, progress.OverallState()); diff != "" {
		t.Errorf("unknown overall state mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"same", "same"}, []string{progress.Tasks()[0].Name(), progress.Tasks()[1].Name()}); diff != "" {
		t.Errorf("duplicate names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Unknown, plansnapshot.Unknown}, []plansnapshot.ProgressState{progress.Tasks()[0].State(), progress.Tasks()[1].State()}); diff != "" {
		t.Errorf("unknown task states mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, progress.MembershipComplete()); diff != "" {
		t.Errorf("duplicate membership completeness mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverUsesCompletePagination(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"open","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: http.Header{"Link": []string{`<https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all>; rel="next"`}}, Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"A","state":"open"}]`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":2,"title":"B","state":"closed"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, progress.MembershipComplete()); diff != "" {
		t.Errorf("pagination completeness mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"A", "B"}, []string{progress.Tasks()[0].Name(), progress.Tasks()[1].Name()}); diff != "" {
		t.Errorf("pagination order mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(true, hasCurrent); diff != "" {
		t.Errorf("paginated current Plan mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/repos/owner/repo/issues", transport.requests[2].URL.Path); diff != "" {
		t.Errorf("page two path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("milestone=42&page=2&per_page=100&state=all", transport.requests[2].URL.RawQuery); diff != "" {
		t.Errorf("page two query mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverRejectsReversedPageConflict(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"open","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: http.Header{"Link": []string{`<https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all>; rel="next"`}}, Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"same","state":"open"}]`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"same","state":"closed"},{"id":2,"title":"other","state":"open"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(false, hasCurrent); diff != "" {
		t.Errorf("conflict current Plan mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("conflict progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, progress.MembershipComplete()); diff != "" {
		t.Errorf("conflict completeness mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"other"}, []string{progress.Tasks()[0].Name()}); diff != "" {
		t.Errorf("conflict member retention mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverReobservesFreshFacts(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"First","state":"open","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"first","state":"open"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	first, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("first Observe() error mismatch (-want +got):\n%s", diff)
	}
	transport.responses = append(transport.responses,
		&http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Second","state":"closed","description":"bad"}`), closed: new(bool)}},
		&http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":2,"title":"second","state":"closed"}]`), closed: new(bool)}},
	)
	second, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("second Observe() error mismatch (-want +got):\n%s", diff)
	}
	firstProgress, firstOK := first.Progress()
	secondProgress, secondOK := second.Progress()
	if diff := cmp.Diff(true, firstOK); diff != "" {
		t.Fatalf("first progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, secondOK); diff != "" {
		t.Fatalf("second progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("first", firstProgress.Tasks()[0].Name()); diff != "" {
		t.Errorf("first fresh name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("second", secondProgress.Tasks()[0].Name()); diff != "" {
		t.Errorf("second fresh name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.Open, firstProgress.OverallState()); diff != "" {
		t.Errorf("first fresh state mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.Closed, secondProgress.OverallState()); diff != "" {
		t.Errorf("second fresh state mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverReturnsIncompleteProgressAfterMemberFailure(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"open","description":"unusable"}`), closed: new(bool)}},
		{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`failed`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(false, hasCurrent); diff != "" {
		t.Errorf("incomplete current Plan presence mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("incomplete progress presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, progress.MembershipComplete()); diff != "" {
		t.Errorf("incomplete membership mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverRetainsPageOneBeforeLaterFailure(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"closed","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: http.Header{"Link": []string{`<https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all>; rel="next"`}}, Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"A","state":"open"}]`), closed: new(bool)}},
		{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`provider detail`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	_, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(false, hasCurrent); diff != "" {
		t.Errorf("later failure current Plan mismatch (-want +got):\n%s", diff)
	}
	progress, hasProgress := got.Progress()
	if diff := cmp.Diff(true, hasProgress); diff != "" {
		t.Fatalf("later failure progress mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(false, progress.MembershipComplete()); diff != "" {
		t.Errorf("later failure completeness mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"A"}, []string{progress.Tasks()[0].Name()}); diff != "" {
		t.Errorf("later failure names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plansnapshot.ProgressState{plansnapshot.Open}, []plansnapshot.ProgressState{progress.Tasks()[0].State()}); diff != "" {
		t.Errorf("later failure states mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverRootFailuresAreStable(t *testing.T) {
	providerDetail := errors.New("provider detail must not cross snapshot boundary")
	closedMalformed, closedBody, closedStatus := false, false, false
	tests := []struct {
		name      string
		transport http.RoundTripper
	}{
		{name: "http failure", transport: &failureObservationRoundTripper{responses: []*http.Response{{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: &failureObservationBody{data: []byte(`denied`), closed: &closedStatus}}}}},
		{name: "malformed root", transport: &failureObservationRoundTripper{responses: []*http.Response{{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(`not json`), closed: &closedMalformed}}}}},
		{name: "unusable body", transport: &failureObservationRoundTripper{responses: []*http.Response{{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: nil, readError: providerDetail, closed: &closedBody}}}}},
		{name: "transport failure", transport: &errorRoundTripper{err: providerDetail}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: tt.transport}, target))
			got, err := plansnapshot.Observe(context.Background(), observer)
			var failure plansnapshot.ObservationFailure
			if diff := cmp.Diff(true, errors.As(err, &failure)); diff != "" {
				t.Fatalf("failure type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plansnapshot.ObservationUnavailable, failure.Code()); diff != "" {
				t.Errorf("failure code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, errors.Is(err, providerDetail)); diff != "" {
				t.Errorf("provider detail leakage mismatch (-want +got):\n%s", diff)
			}
			_, hasCurrent := got.CurrentPlan()
			_, hasProgress := got.Progress()
			if diff := cmp.Diff(false, hasCurrent); diff != "" {
				t.Errorf("failure current Plan mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasProgress); diff != "" {
				t.Errorf("failure progress mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMilestoneSnapshotObserverReconstructsValidDueOn(t *testing.T) {
	transport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Plan","state":"open","due_on":"2028-02-29T00:00:00Z","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"A","state":"open"}]`), closed: new(bool)}},
	}}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	got, err := plansnapshot.Observe(context.Background(), observer)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
	}
	current, hasCurrent := got.CurrentPlan()
	if diff := cmp.Diff(true, hasCurrent); diff != "" {
		t.Fatalf("due-on current Plan mismatch (-want +got):\n%s", diff)
	}
	expectedDate := must(plan.ParseTargetDate("2028-02-29"))
	date, hasDate := current.TargetDate()
	if diff := cmp.Diff(true, hasDate); diff != "" {
		t.Errorf("target date presence mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expectedDate.String(), date.String()); diff != "" {
		t.Errorf("target date mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverInvalidOrUnavailableDueOnKeepsProgressOnly(t *testing.T) {
	tests := []struct {
		name string
		root string
	}{
		{name: "invalid", root: `{"number":42,"title":"Plan","state":"open","due_on":"2028-02-30T00:00:00Z","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`},
		{name: "unavailable", root: `{"number":42,"title":"Plan","state":"open","due_on":123,"description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &milestoneObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(tt.root), closed: new(bool)}},
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"A","state":"open"}]`), closed: new(bool)}},
			}}
			target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
			got, err := plansnapshot.Observe(context.Background(), observer)
			if diff := cmp.Diff(nil, err); diff != "" {
				t.Fatalf("Observe() error mismatch (-want +got):\n%s", diff)
			}
			_, hasCurrent := got.CurrentPlan()
			if diff := cmp.Diff(false, hasCurrent); diff != "" {
				t.Errorf("date-only current Plan mismatch (-want +got):\n%s", diff)
			}
			progress, hasProgress := got.Progress()
			if diff := cmp.Diff(true, hasProgress); diff != "" {
				t.Fatalf("date-only progress mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, progress.MembershipComplete()); diff != "" {
				t.Errorf("date-only completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMilestoneSnapshotObserverCancellationAndConcurrentIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := &blockingRoundTripper{started: make(chan struct{})}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	result := make(chan struct {
		snapshot plansnapshot.Snapshot
		err      error
	}, 1)
	go func() {
		snapshot, err := plansnapshot.Observe(ctx, observer)
		result <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	<-transport.started
	cancel()
	got := <-result
	if diff := cmp.Diff(context.Canceled, got.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("cancelled error mismatch (-want +got):\n%s", diff)
	}
	_, hasProgress := got.snapshot.Progress()
	if diff := cmp.Diff(false, hasProgress); diff != "" {
		t.Errorf("cancelled snapshot mismatch (-want +got):\n%s", diff)
	}

	firstTransport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"First","state":"open","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"First task","state":"open"}]`), closed: new(bool)}},
	}}
	secondTransport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Second","state":"closed","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":2,"title":"Second task","state":"closed"}]`), closed: new(bool)}},
	}}
	first := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: firstTransport}, target))
	second := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: secondTransport}, target))
	var wait sync.WaitGroup
	results := make(chan struct {
		snapshot plansnapshot.Snapshot
		err      error
	}, 2)
	wait.Add(2)
	go func() {
		defer wait.Done()
		snapshot, err := plansnapshot.Observe(context.Background(), first)
		results <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	go func() {
		defer wait.Done()
		snapshot, err := plansnapshot.Observe(context.Background(), second)
		results <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	wait.Wait()
	close(results)
	seen := map[string]bool{}
	for result := range results {
		if diff := cmp.Diff(error(nil), result.err, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("unaffected observation error mismatch (-want +got):\n%s", diff)
		}
		snapshot := result.snapshot
		_, hasCurrent := snapshot.CurrentPlan()
		if diff := cmp.Diff(false, hasCurrent); diff != "" {
			t.Errorf("unexpected current Plan presence (-want +got):\n%s", diff)
		}
		progress, hasProgress := snapshot.Progress()
		if diff := cmp.Diff(true, hasProgress); diff != "" {
			t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(1, len(progress.Tasks())); diff != "" {
			t.Fatalf("task count mismatch (-want +got):\n%s", diff)
		}
		seen[progress.Tasks()[0].Name()] = true
	}
	if diff := cmp.Diff(map[string]bool{"First task": true, "Second task": true}, seen); diff != "" {
		t.Errorf("concurrent isolation mismatch (-want +got):\n%s", diff)
	}
}

type snapshotObservationCallKey struct{}

type sharedSnapshotRoundTripper struct {
	mu              sync.Mutex
	cancelStarted   chan struct{}
	cancelObserved  bool
	successRequests int
	requests        []*http.Request
}

func (t *sharedSnapshotRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.requests = append(t.requests, request)
	t.mu.Unlock()
	if request.Context().Value(snapshotObservationCallKey{}) == "cancel" {
		t.mu.Lock()
		if !t.cancelObserved {
			t.cancelObserved = true
			close(t.cancelStarted)
		}
		t.mu.Unlock()
		<-request.Context().Done()
		return nil, request.Context().Err()
	}
	t.mu.Lock()
	t.successRequests++
	requestNumber := t.successRequests
	t.mu.Unlock()
	if requestNumber == 1 {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":42,"title":"Success","state":"open","description":"bad"}`), closed: new(bool)}}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"success","state":"open"}]`), closed: new(bool)}}, nil
}

func TestMilestoneSnapshotObserverSameObserverCancellationDoesNotAffectSuccess(t *testing.T) {
	transport := &sharedSnapshotRoundTripper{cancelStarted: make(chan struct{})}
	target := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	observer := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: transport}, target))
	cancelContext, cancel := context.WithCancel(context.WithValue(context.Background(), snapshotObservationCallKey{}, "cancel"))
	successContext := context.WithValue(context.Background(), snapshotObservationCallKey{}, "success")
	type outcome struct {
		snapshot plansnapshot.Snapshot
		err      error
	}
	results := make(chan outcome, 2)
	go func() {
		snapshot, err := plansnapshot.Observe(cancelContext, observer)
		results <- outcome{snapshot: snapshot, err: err}
	}()
	go func() {
		snapshot, err := plansnapshot.Observe(successContext, observer)
		results <- outcome{snapshot: snapshot, err: err}
	}()
	<-transport.cancelStarted
	cancel()
	first := <-results
	second := <-results
	outcomes := []outcome{first, second}
	sort.Slice(outcomes, func(left, right int) bool { return errors.Is(outcomes[left].err, context.Canceled) })
	if diff := cmp.Diff(context.Canceled, outcomes[0].err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("cancelled error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(error(nil), outcomes[1].err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("successful error mismatch (-want +got):\n%s", diff)
	}
	_, cancelledHasProgress := outcomes[0].snapshot.Progress()
	successProgress, successHasProgress := outcomes[1].snapshot.Progress()
	if diff := cmp.Diff(false, cancelledHasProgress); diff != "" {
		t.Errorf("cancelled result snapshot mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, successHasProgress); diff != "" {
		t.Errorf("successful result progress mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"success"}, []string{successProgress.Tasks()[0].Name()}); diff != "" {
		t.Errorf("successful result material mismatch (-want +got):\n%s", diff)
	}
}

func TestMilestoneSnapshotObserverBindsDistinctTargetsAndRequests(t *testing.T) {
	firstTransport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":7,"title":"First","state":"open","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":1,"title":"first","state":"open"}]`), closed: new(bool)}},
	}}
	secondTransport := &milestoneObservationRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`{"number":9,"title":"Second","state":"closed","description":"bad"}`), closed: new(bool)}},
		{StatusCode: http.StatusOK, Header: make(http.Header), Body: &milestoneObservationBody{data: []byte(`[{"id":2,"title":"second","state":"closed"}]`), closed: new(bool)}},
	}}
	firstTarget := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("alice", "one")), must(githubplan.NewResourceNumber(7))))
	secondTarget := must(githubplan.NewMilestoneTarget(must(githubplan.NewRepository("bob", "two")), must(githubplan.NewResourceNumber(9))))
	firstObserver := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: firstTransport}, firstTarget))
	secondObserver := must(githubplan.NewMilestoneSnapshotObserver(&http.Client{Transport: secondTransport}, secondTarget))
	results := make(chan struct {
		snapshot plansnapshot.Snapshot
		err      error
	}, 2)
	go func() {
		snapshot, err := plansnapshot.Observe(context.Background(), firstObserver)
		results <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	go func() {
		snapshot, err := plansnapshot.Observe(context.Background(), secondObserver)
		results <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	firstResult := <-results
	secondResult := <-results
	resultByName := map[string]struct {
		snapshot plansnapshot.Snapshot
		err      error
	}{}
	for _, result := range []struct {
		snapshot plansnapshot.Snapshot
		err      error
	}{firstResult, secondResult} {
		progress, _ := result.snapshot.Progress()
		resultByName[progress.Tasks()[0].Name()] = result
	}
	firstResult = resultByName["first"]
	secondResult = resultByName["second"]
	if diff := cmp.Diff(error(nil), firstResult.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("first target error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(error(nil), secondResult.err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("second target error mismatch (-want +got):\n%s", diff)
	}
	firstProgress, firstOK := firstResult.snapshot.Progress()
	secondProgress, secondOK := secondResult.snapshot.Progress()
	if diff := cmp.Diff(true, firstOK); diff != "" {
		t.Errorf("first target progress mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(true, secondOK); diff != "" {
		t.Errorf("second target progress mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("first", firstProgress.Tasks()[0].Name()); diff != "" {
		t.Errorf("first target material mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("second", secondProgress.Tasks()[0].Name()); diff != "" {
		t.Errorf("second target material mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/repos/alice/one/milestones/7", firstTransport.requests[0].URL.Path); diff != "" {
		t.Errorf("first root path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/repos/alice/one/issues", firstTransport.requests[1].URL.Path); diff != "" {
		t.Errorf("first members path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("milestone=7&page=1&per_page=100&state=all", firstTransport.requests[1].URL.RawQuery); diff != "" {
		t.Errorf("first members query mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/repos/bob/two/milestones/9", secondTransport.requests[0].URL.Path); diff != "" {
		t.Errorf("second root path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("/repos/bob/two/issues", secondTransport.requests[1].URL.Path); diff != "" {
		t.Errorf("second members path mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("milestone=9&page=1&per_page=100&state=all", secondTransport.requests[1].URL.RawQuery); diff != "" {
		t.Errorf("second members query mismatch (-want +got):\n%s", diff)
	}
}
