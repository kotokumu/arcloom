package githubplan_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type failureObservationRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *failureObservationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, r)
	if len(t.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

type failureObservationBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *failureObservationBody) Read(p []byte) (int, error) {
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
func (b *failureObservationBody) Close() error { *b.closed = true; return nil }

func TestGitHubHTTPFailuresAtRootLocalizeOnlyPlanRoot(t *testing.T) {
	failureStatuses := []struct {
		name   string
		status int
	}{
		{name: "redirect", status: http.StatusFound},
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "forbidden", status: http.StatusForbidden},
		{name: "not found", status: http.StatusNotFound},
		{name: "gone", status: http.StatusGone},
		{name: "unprocessable", status: http.StatusUnprocessableEntity},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "server error", status: http.StatusInternalServerError},
		{name: "service unavailable", status: http.StatusServiceUnavailable},
	}
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		for _, tt := range failureStatuses {
			t.Run(string(representation)+"/"+tt.name, func(t *testing.T) {
				closed := false
				response := &http.Response{StatusCode: tt.status, Header: make(http.Header), Body: &failureObservationBody{data: []byte("provider detail must not cross"), closed: &closed}}
				if tt.status == http.StatusFound {
					response.Header.Set("Location", "https://elsewhere.example/rebind")
				}
				transport := &failureObservationRoundTripper{responses: []*http.Response{response}}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				result, err := controller.Reconcile(context.Background(), expected)
				if err != nil {
					t.Fatalf("Reconcile() error = %v, want nil", err)
				}
				unavailable := result.UnavailableInformation()
				if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.PlanRootLocation || len(result.Differences()) != 0 {
					t.Errorf("Result evidence = (%v, %v), want Plan root unavailable only", result.Differences(), unavailable)
				}
				if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
					t.Errorf("determination mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(1, len(transport.requests)); diff != "" {
					t.Errorf("root failure request count mismatch (-want +got):\n%s", diff)
				}
				if !closed {
					t.Error("root response body was not closed")
				}
			})
		}
	}
}

func TestGitHubHTTPFailuresAtCollectionLocalizeTaskCollection(t *testing.T) {
	failureStatuses := []struct {
		name   string
		status int
	}{
		{name: "redirect", status: http.StatusFound},
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "forbidden", status: http.StatusForbidden},
		{name: "not found", status: http.StatusNotFound},
		{name: "gone", status: http.StatusGone},
		{name: "unprocessable", status: http.StatusUnprocessableEntity},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "server error", status: http.StatusInternalServerError},
		{name: "service unavailable", status: http.StatusServiceUnavailable},
	}
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		for _, tt := range failureStatuses {
			t.Run(string(representation)+"/"+tt.name, func(t *testing.T) {
				rootClosed, collectionClosed := false, false
				response := &http.Response{StatusCode: tt.status, Header: make(http.Header), Body: &failureObservationBody{data: []byte("provider detail must not cross"), closed: &collectionClosed}}
				if tt.status == http.StatusFound {
					response.Header.Set("Location", "https://elsewhere.example/rebind")
				}
				root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
				if representation == githubplan.IssueRepresentation {
					root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
				}
				transport := &failureObservationRoundTripper{responses: []*http.Response{
					{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(root), closed: &rootClosed}},
					response,
				}}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				result, err := controller.Reconcile(context.Background(), expected)
				if err != nil {
					t.Fatalf("Reconcile() error = %v, want nil", err)
				}
				unavailable := result.UnavailableInformation()
				if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.TaskCollectionLocation || len(result.Differences()) != 0 || result.Determination() != planrepresentation.Undecidable {
					t.Errorf("Result = (%v, %v, %v), want Task collection unavailable only", result.Determination(), result.Differences(), unavailable)
				}
				if diff := cmp.Diff(2, len(transport.requests)); diff != "" {
					t.Errorf("collection failure request count mismatch (-want +got):\n%s", diff)
				}
				if !rootClosed || !collectionClosed {
					t.Errorf("response bodies were not closed: root=%v collection=%v", rootClosed, collectionClosed)
				}
			})
		}
	}
}

func TestGitHubMalformedJSONAndTransportFailuresLocalizeKnowledge(t *testing.T) {
	providerSentinel := errors.New("provider secret transport failure")
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		t.Run(string(representation)+"/malformed root", func(t *testing.T) {
			closed := false
			transport := &failureObservationRoundTripper{responses: []*http.Response{{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte("not json"), closed: &closed}}}}
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := must(planrepresentation.NewController(observer)).Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.PlanRootLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Plan root unavailable", result.Differences(), unavailable)
			}
		})
		t.Run(string(representation)+"/malformed collection", func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
			if representation == githubplan.IssueRepresentation {
				root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
			}
			transport := &failureObservationRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(root), closed: &rootClosed}},
				{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte("not json"), closed: &collectionClosed}},
			}}
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := must(planrepresentation.NewController(observer)).Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.TaskCollectionLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Task collection unavailable", result.Differences(), unavailable)
			}
		})
		t.Run(string(representation)+"/transport root", func(t *testing.T) {
			transport := &errorRoundTripper{err: providerSentinel}
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.PlanRootLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Plan root unavailable", result.Differences(), unavailable)
			}
			if transport.lastError != providerSentinel {
				t.Fatalf("transport did not return provider sentinel")
			}
		})
		t.Run(string(representation)+"/transport collection", func(t *testing.T) {
			root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
			if representation == githubplan.IssueRepresentation {
				root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
			}
			transport := &errorAfterRootRoundTripper{root: root, err: providerSentinel}
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			controller := must(planrepresentation.NewController(observer))
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.TaskCollectionLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Task collection unavailable", result.Differences(), unavailable)
			}
		})
	}
}

func TestGitHubObserverDirectNilContextReturnsRootUnavailableWithoutRequest(t *testing.T) {
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		t.Run(string(representation), func(t *testing.T) {
			transport := &failureObservationRoundTripper{}
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			} else {
				observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
			}
			observation, err := observer(nil) //nolint:staticcheck // nil context is the documented direct Observer contract.
			if err != nil {
				t.Fatalf("Observer(nil) error = %v, want nil", err)
			}
			if diff := cmp.Diff(0, len(transport.requests)); diff != "" {
				t.Errorf("nil-context request count mismatch (-want +got):\n%s", diff)
			}
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				return observation, nil
			}))
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.PlanRootLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Plan root unavailable", result.Differences(), unavailable)
			}
		})
	}
}

func TestGitHubRedirectDoesNotFollowRebindOrInvokeHostCallback(t *testing.T) {
	for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
		t.Run(string(representation), func(t *testing.T) {
			closed := false
			callbackCalled := false
			transport := &failureObservationRoundTripper{responses: []*http.Response{{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://elsewhere.example/rebind"}},
				Body:       &failureObservationBody{data: []byte("redirect"), closed: &closed},
			}}}
			client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
				callbackCalled = true
				return nil
			}}
			repository := must(githubplan.NewRepository("owner", "repo"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			observation, err := observer(context.Background())
			if err != nil {
				t.Fatalf("Observer() error = %v, want nil", err)
			}
			if callbackCalled {
				t.Fatal("host redirect callback was invoked")
			}
			if diff := cmp.Diff(1, len(transport.requests)); diff != "" {
				t.Errorf("redirect request count mismatch (-want +got):\n%s", diff)
			}
			if !closed {
				t.Error("redirect body was not closed")
			}
			controller := must(planrepresentation.NewController(func(context.Context) (planrepresentation.Observation, error) {
				return observation, nil
			}))
			expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
			result, err := controller.Reconcile(context.Background(), expected)
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}
			unavailable := result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.PlanRootLocation || len(result.Differences()) != 0 {
				t.Errorf("evidence = (%v,%v), want Plan root unavailable", result.Differences(), unavailable)
			}
		})
	}
}

func TestGitHubContextCancellationAndDeadlineBeforeRoot(t *testing.T) {
	for _, wanted := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
			t.Run(string(representation)+"/"+wanted.Error(), func(t *testing.T) {
				ctx := &synchronizableContext{base: context.Background(), done: make(chan struct{})}
				ctx.trigger(wanted)
				transport := &failureObservationRoundTripper{}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				result, err := controller.Reconcile(ctx, expected)
				if err != wanted {
					t.Fatalf("error = %v, want exact %v", err, wanted)
				}
				if result.Determination() != planrepresentation.Determination(0) || result.Differences() != nil || result.UnavailableInformation() != nil {
					t.Fatalf("context result = (%v, %v, %v), want complete zero Result", result.Determination(), result.Differences(), result.UnavailableInformation())
				}
				if diff := cmp.Diff(0, len(transport.requests)); diff != "" {
					t.Errorf("before-root request count mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestGitHubContextCancellationAndDeadlineDuringRoot(t *testing.T) {
	for _, wanted := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
			t.Run(string(representation)+"/"+wanted.Error(), func(t *testing.T) {
				ctx := &synchronizableContext{base: context.Background(), done: make(chan struct{})}
				transport := &blockingRoundTripper{started: make(chan struct{})}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				resultChannel := make(chan contextResult, 1)
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				go func() {
					result, err := controller.Reconcile(ctx, expected)
					resultChannel <- contextResult{result: result, err: err}
				}()
				<-transport.started
				ctx.trigger(wanted)
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
				var outcome contextResult
				select {
				case outcome = <-resultChannel:
				case <-timer.C:
					t.Fatal("context-bound observation did not return")
				}
				if outcome.err != wanted {
					t.Fatalf("error = %v, want exact %v", outcome.err, wanted)
				}
				if outcome.result.Determination() != planrepresentation.Determination(0) || outcome.result.Differences() != nil || outcome.result.UnavailableInformation() != nil {
					t.Fatalf("context result = (%v, %v, %v), want complete zero Result", outcome.result.Determination(), outcome.result.Differences(), outcome.result.UnavailableInformation())
				}
			})
		}
	}
}

func TestGitHubContextCancellationAfterRootSuccessStopsBeforeCollection(t *testing.T) {
	for _, wanted := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
			t.Run(string(representation)+"/"+wanted.Error(), func(t *testing.T) {
				ctx := &synchronizableContext{base: context.Background(), done: make(chan struct{})}
				root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
				if representation == githubplan.IssueRepresentation {
					root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
				}
				body := &callbackBody{data: []byte(root), close: func() { ctx.trigger(wanted) }}
				transport := &singleBodyRoundTripper{body: body}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				result, err := controller.Reconcile(ctx, expected)
				if err != wanted {
					t.Fatalf("error = %v, want exact %v", err, wanted)
				}
				if result.Determination() != planrepresentation.Determination(0) || result.Differences() != nil || result.UnavailableInformation() != nil {
					t.Fatalf("context result = (%v, %v, %v), want complete zero Result", result.Determination(), result.Differences(), result.UnavailableInformation())
				}
				if diff := cmp.Diff(1, transport.requests); diff != "" {
					t.Errorf("after-root request count mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestGitHubContextCancellationAndDeadlineDuringLaterPage(t *testing.T) {
	for _, wanted := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
			t.Run(string(representation)+"/"+wanted.Error(), func(t *testing.T) {
				ctx := &synchronizableContext{base: context.Background(), done: make(chan struct{})}
				root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
				if representation == githubplan.IssueRepresentation {
					root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
				}
				transport := &blockingAfterPagesRoundTripper{representation: representation, root: root, started: make(chan struct{})}
				var observer planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					observer = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					observer = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				controller := must(planrepresentation.NewController(observer))
				resultChannel := make(chan contextResult, 1)
				go func() {
					expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
					result, err := controller.Reconcile(ctx, expected)
					resultChannel <- contextResult{result: result, err: err}
				}()
				<-transport.started
				ctx.trigger(wanted)
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
				var outcome contextResult
				select {
				case outcome = <-resultChannel:
				case <-timer.C:
					t.Fatal("context-bound observation did not return")
				}
				if outcome.err != wanted {
					t.Fatalf("error = %v, want exact %v", outcome.err, wanted)
				}
				if outcome.result.Determination() != planrepresentation.Determination(0) || outcome.result.Differences() != nil || outcome.result.UnavailableInformation() != nil {
					t.Fatalf("context result = (%v, %v, %v), want complete zero Result", outcome.result.Determination(), outcome.result.Differences(), outcome.result.UnavailableInformation())
				}
				if diff := cmp.Diff(3, transport.page); diff != "" {
					t.Errorf("later-page request count mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}

func TestGitHubContextCancellationAtFinalCompletionCheckIgnoresObservation(t *testing.T) {
	for _, wanted := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, representation := range []githubplan.Representation{githubplan.MilestoneRepresentation, githubplan.IssueRepresentation} {
			t.Run(string(representation)+"/"+wanted.Error(), func(t *testing.T) {
				ctx := &synchronizableContext{base: context.Background(), done: make(chan struct{})}
				rootClosed, collectionClosed := false, false
				root := `{"number":42,"title":"Plan","description":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl19\n-->"}`
				if representation == githubplan.IssueRepresentation {
					root = `{"number":42,"title":"Plan","body":"<!-- arcloom-plan:v1\neyJnb2FsIjoiR29hbCIsImFjY2VwdGFuY2VfY29uZGl0aW9ucyI6WyJBIl0sInRhcmdldF9kYXRlIjpudWxsfQ\n-->"}`
				}
				transport := &failureObservationRoundTripper{responses: []*http.Response{
					{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(root), closed: &rootClosed}},
					{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(`[{"id":7,"title":"Task"}]`), closed: &collectionClosed}},
				}}
				var baseObserver planrepresentation.Observer
				if representation == githubplan.MilestoneRepresentation {
					baseObserver = must(githubplan.NewMilestoneObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				} else {
					baseObserver = must(githubplan.NewIssueObserver(&http.Client{Transport: transport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
				}
				observer := func(callContext context.Context) (planrepresentation.Observation, error) {
					observation, err := baseObserver(callContext)
					ctx.trigger(wanted)
					return observation, err
				}
				controller := must(planrepresentation.NewController(observer))
				expected := must(plan.New("Plan", must(plan.NewGoal("Goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A"))}, []plan.Task{must(plan.NewTask("Task"))}, nil))
				result, err := controller.Reconcile(ctx, expected)
				if err != wanted {
					t.Fatalf("error = %v, want exact %v", err, wanted)
				}
				if result.Determination() != planrepresentation.Determination(0) || result.Differences() != nil || result.UnavailableInformation() != nil {
					t.Fatalf("context result = (%v, %v, %v), want complete zero Result", result.Determination(), result.Differences(), result.UnavailableInformation())
				}
			})
		}
	}
}

type contextResult struct {
	result planrepresentation.Result
	err    error
}

type errorRoundTripper struct {
	err       error
	lastError error
}

func (t *errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	t.lastError = t.err
	return nil, t.err
}

type errorAfterRootRoundTripper struct {
	root  string
	err   error
	count int
}

type singleBodyRoundTripper struct {
	body     io.ReadCloser
	requests int
}

func (t *singleBodyRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	t.requests++
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: t.body}, nil
}

func (t *errorAfterRootRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.count++
	if t.count == 1 {
		closed := false
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(t.root), closed: &closed}}, nil
	}
	return nil, t.err
}

type callbackBody struct {
	data   []byte
	close  func()
	closed bool
}

func (b *callbackBody) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *callbackBody) Close() error {
	b.closed = true
	b.close()
	return nil
}

type synchronizableContext struct {
	base     context.Context
	done     chan struct{}
	mu       sync.RWMutex
	err      error
	deadline time.Time
}

func (c *synchronizableContext) Deadline() (time.Time, bool) {
	return c.deadline, !c.deadline.IsZero()
}

func (c *synchronizableContext) Done() <-chan struct{} { return c.done }

func (c *synchronizableContext) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *synchronizableContext) Value(key any) any { return c.base.Value(key) }

func (c *synchronizableContext) trigger(err error) {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
		close(c.done)
	}
	c.mu.Unlock()
}

type blockingRoundTripper struct {
	started chan struct{}
	once    sync.Once
}

type blockingAfterPagesRoundTripper struct {
	representation githubplan.Representation
	root           string
	started        chan struct{}
	page           int
	once           sync.Once
}

func (t *blockingAfterPagesRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.page++
	if t.page == 1 {
		closed := false
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: &failureObservationBody{data: []byte(t.root), closed: &closed}}, nil
	}
	if t.page == 2 {
		closed := false
		target := "https://api.github.com/repos/owner/repo/issues/42/sub_issues?page=2&per_page=100"
		if t.representation == githubplan.MilestoneRepresentation {
			target = "https://api.github.com/repos/owner/repo/issues?milestone=42&page=2&per_page=100&state=all"
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Link": []string{`<` + target + `>; rel="next"`}}, Body: &failureObservationBody{data: []byte(`[{"id":7,"title":"First"}]`), closed: &closed}}, nil
	}
	t.once.Do(func() { close(t.started) })
	<-request.Context().Done()
	return nil, request.Context().Err()
}

func (t *blockingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.once.Do(func() { close(t.started) })
	<-request.Context().Done()
	return nil, request.Context().Err()
}
