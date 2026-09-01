package githubplan_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type scriptedRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
}

func (s *scriptedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	s.requests = append(s.requests, request)
	if len(s.responses) == 0 {
		return nil, errors.New("unexpected request")
	}
	response := s.responses[0]
	s.responses = s.responses[1:]
	return response, nil
}

type trackingBody struct {
	data      []byte
	readError error
	closed    *bool
}

func (b *trackingBody) Read(p []byte) (int, error) {
	if b.readError != nil {
		return 0, b.readError
	}
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *trackingBody) Close() error {
	*b.closed = true
	return nil
}

func TestObserverRESTRequestContract(t *testing.T) {
	tests := []struct {
		name           string
		representation githubplan.Representation
		rootPath       string
		rootBody       string
		collectionPath string
		collectionBody string
		wantTaskQuery  string
	}{
		{
			name:           "milestone",
			representation: githubplan.MilestoneRepresentation,
			rootPath:       "/repos/owner%20space/repo%25name/milestones/42",
			rootBody:       `{"number":42,"title":"Plan","description":""}`,
			collectionPath: "/repos/owner%20space/repo%25name/issues",
			collectionBody: "[]",
			wantTaskQuery:  "milestone=42&page=1&per_page=100&state=all",
		},
		{
			name:           "issue",
			representation: githubplan.IssueRepresentation,
			rootPath:       "/repos/owner%20space/repo%25name/issues/42",
			rootBody:       `{"number":42,"title":"Plan","body":""}`,
			collectionPath: "/repos/owner%20space/repo%25name/issues/42/sub_issues",
			collectionBody: "[]",
			wantTaskQuery:  "page=1&per_page=100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootClosed, collectionClosed := false, false
			transport := &scriptedRoundTripper{responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: &trackingBody{data: []byte(tt.rootBody), closed: &rootClosed}, Header: make(http.Header)},
				{StatusCode: http.StatusOK, Body: &trackingBody{data: []byte(tt.collectionBody), closed: &collectionClosed}, Header: make(http.Header)},
			}}
			client := &http.Client{Transport: transport}
			repository := must(githubplan.NewRepository("owner space", "repo%name"))
			number := must(githubplan.NewResourceNumber(42))
			var observer planrepresentation.Observer
			if tt.representation == githubplan.MilestoneRepresentation {
				observer = must(githubplan.NewMilestoneObserver(client, repository, number))
			} else {
				observer = must(githubplan.NewIssueObserver(client, repository, number))
			}
			_, err := observer(context.Background())
			if err != nil {
				t.Fatalf("Observer() error = %v", err)
			}
			if diff := cmp.Diff(2, len(transport.requests)); diff != "" {
				t.Fatalf("request count mismatch (-want +got):\n%s", diff)
			}
			for index, request := range transport.requests {
				if diff := cmp.Diff(http.MethodGet, request.Method); diff != "" {
					t.Errorf("request %d method mismatch (-want +got):\n%s", index, diff)
				}
				if diff := cmp.Diff("https", request.URL.Scheme); diff != "" {
					t.Errorf("request %d scheme mismatch (-want +got):\n%s", index, diff)
				}
				if diff := cmp.Diff("api.github.com", request.URL.Host); diff != "" {
					t.Errorf("request %d host mismatch (-want +got):\n%s", index, diff)
				}
				if diff := cmp.Diff("application/vnd.github.raw+json", request.Header.Get("Accept")); diff != "" {
					t.Errorf("request %d Accept mismatch (-want +got):\n%s", index, diff)
				}
				if diff := cmp.Diff("2022-11-28", request.Header.Get("X-GitHub-Api-Version")); diff != "" {
					t.Errorf("request %d API version mismatch (-want +got):\n%s", index, diff)
				}
				if diff := cmp.Diff("arcloom", request.Header.Get("User-Agent")); diff != "" {
					t.Errorf("request %d User-Agent mismatch (-want +got):\n%s", index, diff)
				}
			}
			rootRequest := transport.requests[0]
			if diff := cmp.Diff(http.MethodGet, rootRequest.Method); diff != "" {
				t.Errorf("root method mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("https", rootRequest.URL.Scheme); diff != "" {
				t.Errorf("root scheme mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("api.github.com", rootRequest.URL.Host); diff != "" {
				t.Errorf("root host mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.rootPath, rootRequest.URL.EscapedPath()); diff != "" {
				t.Errorf("root path mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("application/vnd.github.raw+json", rootRequest.Header.Get("Accept")); diff != "" {
				t.Errorf("Accept mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("2022-11-28", rootRequest.Header.Get("X-GitHub-Api-Version")); diff != "" {
				t.Errorf("API version mismatch (-want +got):\n%s", diff)
			}
			taskRequest := transport.requests[1]
			if diff := cmp.Diff(http.MethodGet, taskRequest.Method); diff != "" {
				t.Errorf("task method mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.collectionPath, taskRequest.URL.EscapedPath()); diff != "" {
				t.Errorf("task path mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantTaskQuery, taskRequest.URL.RawQuery); diff != "" {
				t.Errorf("task query mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, rootClosed); diff != "" {
				t.Errorf("root body close mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, collectionClosed); diff != "" {
				t.Errorf("collection body close mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserverRESTReturnsValidUnavailableObservation(t *testing.T) {
	rootClosed, collectionClosed := false, false
	transport := &scriptedRoundTripper{responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: &trackingBody{data: []byte(`{"number":42,"title":"Plan","description":""}`), closed: &rootClosed}, Header: make(http.Header)},
		{StatusCode: http.StatusOK, Body: &trackingBody{data: []byte("[]"), closed: &collectionClosed}, Header: make(http.Header)},
	}}
	observer := must(githubplan.NewMilestoneObserver(
		&http.Client{Transport: transport},
		must(githubplan.NewRepository("owner", "repo")),
		must(githubplan.NewResourceNumber(42)),
	))
	controller := must(planrepresentation.NewController(observer))
	expected := must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	result, err := controller.Reconcile(context.Background(), expected)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(planrepresentation.Undecidable, result.Determination()); diff != "" {
		t.Errorf("determination mismatch (-want +got):\n%s", diff)
	}
}

func TestObserverRESTResponseBodiesCloseOnAllRootOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		readError  error
		location   string
	}{
		{name: "success", statusCode: http.StatusOK, body: []byte(`{"number":42,"title":"Plan","description":""}`)},
		{name: "http error", statusCode: http.StatusForbidden, body: []byte("denied")},
		{name: "read error", statusCode: http.StatusOK, readError: errors.New("read failed")},
		{name: "decode error", statusCode: http.StatusOK, body: []byte("not json")},
		{name: "redirect", statusCode: http.StatusFound, body: []byte("redirect"), location: "https://elsewhere.example/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closed := false
			response := &http.Response{StatusCode: tt.statusCode, Body: &trackingBody{data: tt.body, readError: tt.readError, closed: &closed}, Header: make(http.Header)}
			if tt.location != "" {
				response.Header.Set("Location", tt.location)
			}
			transport := &scriptedRoundTripper{responses: []*http.Response{response, {StatusCode: http.StatusOK, Body: &trackingBody{data: []byte("[]"), closed: &closed}, Header: make(http.Header)}}}
			observer := must(githubplan.NewMilestoneObserver(
				&http.Client{Transport: transport},
				must(githubplan.NewRepository("owner", "repo")),
				must(githubplan.NewResourceNumber(42)),
			))
			_, _ = observer(context.Background())
			if diff := cmp.Diff(true, closed); diff != "" {
				t.Errorf("response body close mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserverRESTRefusesRedirectAndDoesNotInvokeHostCallback(t *testing.T) {
	closed := false
	transport := &scriptedRoundTripper{responses: []*http.Response{{StatusCode: http.StatusFound, Body: &trackingBody{data: []byte("redirect"), closed: &closed}, Header: http.Header{"Location": []string{"https://elsewhere.example/"}}}}}
	callbackCalled := false
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			callbackCalled = true
			return nil
		},
	}
	observer := must(githubplan.NewMilestoneObserver(client, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	_, _ = observer(context.Background())
	if diff := cmp.Diff(false, callbackCalled); diff != "" {
		t.Errorf("host redirect callback mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(1, len(transport.requests)); diff != "" {
		t.Errorf("redirect request count mismatch (-want +got):\n%s", diff)
	}
}

func TestObserverRESTUsesCopiedClientValue(t *testing.T) {
	firstRequests := 0
	secondRequests := 0
	first := &recordingRoundTripper{requests: &firstRequests}
	second := &recordingRoundTripper{requests: &secondRequests}
	client := &http.Client{Transport: first}
	observer := must(githubplan.NewMilestoneObserver(client, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	client.Transport = second
	_, _ = observer(context.Background())
	if diff := cmp.Diff(true, firstRequests > 0); diff != "" {
		t.Errorf("copied transport use mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, secondRequests); diff != "" {
		t.Errorf("original transport use mismatch (-want +got):\n%s", diff)
	}
}
