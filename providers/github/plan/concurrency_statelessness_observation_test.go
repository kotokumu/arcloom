package githubplan_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type correlationContextKey struct{}

type concurrentObservation struct {
	label  string
	result planrepresentation.Result
	err    error
}

func TestObserverConcurrentCallsKeepFactsAndProviderFailuresIsolated(t *testing.T) {
	labels := []string{"alpha", "beta", "error"}
	transport := &correlationTransport{
		labels:         map[string]bool{"alpha": true, "beta": true, "error": true},
		failure:        "error",
		rootRelease:    make(chan struct{}),
		requestByLabel: make(map[string]int, 3),
	}
	observer := must(githubplan.NewIssueObserver(
		&http.Client{Transport: transport},
		must(githubplan.NewRepository("owner", "repo")),
		must(githubplan.NewResourceNumber(42)),
	))
	controller := must(planrepresentation.NewController(observer))
	results := make(chan concurrentObservation, len(labels))
	var calls sync.WaitGroup
	for _, label := range labels {
		label := label
		calls.Add(1)
		go func() {
			defer calls.Done()
			ctx := context.WithValue(context.Background(), correlationContextKey{}, label)
			expected := must(plan.New(
				"Plan-"+label,
				must(plan.NewGoal("Goal-"+label)),
				[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A-" + label))},
				[]plan.Task{must(plan.NewTask("Task-" + label + "-1")), must(plan.NewTask("Task-" + label + "-2"))},
				nil,
			))
			result, err := controller.Reconcile(ctx, expected)
			results <- concurrentObservation{label: label, result: result, err: err}
		}()
	}
	calls.Wait()
	close(results)

	seen := make(map[string]bool, len(labels))
	for outcome := range results {
		seen[outcome.label] = true
		if outcome.label == "error" {
			if outcome.err != nil {
				t.Errorf("error call returned Provider error = %v", outcome.err)
			}
			unavailable := outcome.result.UnavailableInformation()
			if len(unavailable) != 1 || unavailable[0].Location().Kind() != planrepresentation.TaskCollectionLocation {
				t.Errorf("error call unavailable information = %v, want Task collection only", unavailable)
			}
			if diff := cmp.Diff(planrepresentation.Undecidable, outcome.result.Determination()); diff != "" {
				t.Errorf("error call determination mismatch (-want +got):\n%s", diff)
			}
			continue
		}
		if outcome.err != nil {
			t.Errorf("%s call returned error = %v", outcome.label, outcome.err)
		}
		if diff := cmp.Diff(planrepresentation.Satisfied, outcome.result.Determination()); diff != "" {
			t.Errorf("%s determination mismatch (-want +got):\n%s", outcome.label, diff)
		}
		if diff := cmp.Diff(0, len(outcome.result.Differences())); diff != "" {
			t.Errorf("%s differences mismatch (-want +got):\n%s", outcome.label, diff)
		}
		if diff := cmp.Diff(0, len(outcome.result.UnavailableInformation())); diff != "" {
			t.Errorf("%s unavailable information mismatch (-want +got):\n%s", outcome.label, diff)
		}
	}
	if diff := cmp.Diff(map[string]bool{"alpha": true, "beta": true, "error": true}, seen); diff != "" {
		t.Errorf("concurrent call labels mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(len(labels)*3, transport.requestCount()); diff != "" {
		t.Errorf("concurrent request count mismatch (-want +got):\n%s", diff)
	}
	for _, label := range labels {
		if diff := cmp.Diff(3, transport.requestCountFor(label)); diff != "" {
			t.Errorf("%s request count mismatch (-want +got):\n%s", label, diff)
		}
	}
}

func TestObserverRepeatedCallsReconstructChangedCurrentFactsWithoutCache(t *testing.T) {
	transport := &changingFactsTransport{}
	observer := must(githubplan.NewMilestoneObserver(
		&http.Client{Transport: transport},
		must(githubplan.NewRepository("owner", "repo")),
		must(githubplan.NewResourceNumber(42)),
	))
	controller := must(planrepresentation.NewController(observer))
	for generation := 1; generation <= 2; generation++ {
		label := fmt.Sprintf("generation-%d", generation)
		expected := must(plan.New(
			"Plan-"+label,
			must(plan.NewGoal("Goal-"+label)),
			[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A-" + label))},
			[]plan.Task{must(plan.NewTask("Task-" + label))},
			nil,
		))
		result, err := controller.Reconcile(context.Background(), expected)
		if err != nil {
			t.Fatalf("generation %d Reconcile() error = %v", generation, err)
		}
		if diff := cmp.Diff(planrepresentation.Satisfied, result.Determination()); diff != "" {
			t.Errorf("generation %d determination mismatch (-want +got):\n%s", generation, diff)
		}
		if diff := cmp.Diff(0, len(result.Differences())); diff != "" {
			t.Errorf("generation %d differences mismatch (-want +got):\n%s", generation, diff)
		}
		if diff := cmp.Diff(0, len(result.UnavailableInformation())); diff != "" {
			t.Errorf("generation %d unavailable information mismatch (-want +got):\n%s", generation, diff)
		}
	}
	if diff := cmp.Diff(2, transport.rootReads); diff != "" {
		t.Errorf("root reread count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(2, transport.collectionReads); diff != "" {
		t.Errorf("collection reread count mismatch (-want +got):\n%s", diff)
	}
}

func TestObserverCanReconstructAfterRuntimeStateIsDiscarded(t *testing.T) {
	firstTransport := &changingFactsTransport{}
	firstObserver := must(githubplan.NewMilestoneObserver(&http.Client{Transport: firstTransport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	firstController := must(planrepresentation.NewController(firstObserver))
	firstExpected := must(plan.New("Plan-generation-1", must(plan.NewGoal("Goal-generation-1")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A-generation-1"))}, []plan.Task{must(plan.NewTask("Task-generation-1"))}, nil))
	firstResult, err := firstController.Reconcile(context.Background(), firstExpected)
	if err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(planrepresentation.Satisfied, firstResult.Determination()); diff != "" {
		t.Fatalf("first determination mismatch (-want +got):\n%s", diff)
	}

	secondTransport := &fixedFactsTransport{label: "generation-2"}
	secondObserver := must(githubplan.NewMilestoneObserver(&http.Client{Transport: secondTransport}, must(githubplan.NewRepository("owner", "repo")), must(githubplan.NewResourceNumber(42))))
	secondController := must(planrepresentation.NewController(secondObserver))
	secondExpected := must(plan.New("Plan-generation-2", must(plan.NewGoal("Goal-generation-2")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("A-generation-2"))}, []plan.Task{must(plan.NewTask("Task-generation-2"))}, nil))
	secondResult, err := secondController.Reconcile(context.Background(), secondExpected)
	if err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}
	if diff := cmp.Diff(planrepresentation.Satisfied, secondResult.Determination()); diff != "" {
		t.Errorf("second determination mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, len(secondResult.Differences())); diff != "" {
		t.Errorf("second differences mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, len(secondResult.UnavailableInformation())); diff != "" {
		t.Errorf("second unavailable information mismatch (-want +got):\n%s", diff)
	}
}

type correlationTransport struct {
	mu             sync.Mutex
	labels         map[string]bool
	failure        string
	rootCount      int
	rootRelease    chan struct{}
	rootReleased   bool
	requests       []string
	requestByLabel map[string]int
}

func (t *correlationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	label, ok := request.Context().Value(correlationContextKey{}).(string)
	if !ok || !t.labels[label] {
		return nil, errors.New("missing test correlation")
	}
	path := request.URL.Path
	page := request.URL.Query().Get("page")
	t.mu.Lock()
	t.requests = append(t.requests, label+" "+path+"?page="+page)
	t.requestByLabel[label]++
	root := page == "" && !strings.Contains(path, "/sub_issues") && !strings.HasSuffix(path, "/issues")
	if root {
		t.rootCount++
		if t.rootCount == len(t.labels) && !t.rootReleased {
			close(t.rootRelease)
			t.rootReleased = true
		}
	}
	release := t.rootRelease
	t.mu.Unlock()
	if root {
		<-release
	}
	if label == t.failure && page == "2" {
		return nil, errors.New("provider detail for " + label)
	}
	var body string
	headers := make(http.Header)
	switch page {
	case "":
		content := `{"goal":"Goal-` + label + `","acceptance_conditions":["A-` + label + `"],"target_date":null}`
		encoded := must(json.Marshal("<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(content)) + "\n-->\n\n## Human narrative\nThis suffix is not machine data.\n"))
		title := must(json.Marshal("Plan-" + label))
		body = `{"number":42,"title":` + string(title) + `,"body":` + string(encoded) + `}`
	case "1":
		body = fmt.Sprintf(`[{"id":100,"number":100,"node_id":"node-100","title":%q}]`, "Task-"+label+"-1")
		target := request.URL.Scheme + "://" + request.URL.Host + request.URL.Path + "?page=2&per_page=100"
		headers = http.Header{"Link": []string{`<` + target + `>; rel="next"`}}
	default:
		body = fmt.Sprintf(`[{"id":101,"number":101,"node_id":"node-101","title":%q}]`, "Task-"+label+"-2")
	}
	if strings.Contains(request.URL.Path, "/milestones/") {
		content := `{"goal":"Goal-` + label + `","acceptance_conditions":["A-` + label + `"]}`
		encoded := must(json.Marshal("<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(content)) + "\n-->\n\n## Human narrative\nThis suffix is not machine data.\n"))
		title := must(json.Marshal("Plan-" + label))
		body = `{"number":42,"title":` + string(title) + `,"description":` + string(encoded) + `}`
	}
	return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (t *correlationTransport) requestCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.requests)
}

func (t *correlationTransport) requestCountFor(label string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requestByLabel[label]
}

type changingFactsTransport struct {
	mu              sync.Mutex
	rootReads       int
	collectionReads int
	label           string
}

func (t *changingFactsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if request.URL.Query().Get("page") == "" && !strings.Contains(request.URL.Path, "/issues") {
		t.rootReads++
		t.label = fmt.Sprintf("generation-%d", t.rootReads)
	}
	label := t.label
	if strings.Contains(request.URL.Path, "/issues") {
		t.collectionReads++
	}
	t.mu.Unlock()
	if strings.Contains(request.URL.Path, "/milestones/") {
		content := `{"goal":"Goal-` + label + `","acceptance_conditions":["A-` + label + `"]}`
		encoded := must(json.Marshal("<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(content)) + "\n-->\n\n## Human narrative\nThis suffix is not machine data.\n"))
		title := must(json.Marshal("Plan-" + label))
		body := `{"number":42,"title":` + string(title) + `,"description":` + string(encoded) + `}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	body := fmt.Sprintf(`[{"id":100,"number":100,"node_id":"node-100","title":%q}]`, "Task-"+label)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}

type fixedFactsTransport struct {
	label string
}

func (t *fixedFactsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if strings.Contains(request.URL.Path, "/milestones/") {
		content := `{"goal":"Goal-` + t.label + `","acceptance_conditions":["A-` + t.label + `"]}`
		encoded := must(json.Marshal("<!-- arcloom-plan:v1\n" + base64.RawURLEncoding.EncodeToString([]byte(content)) + "\n-->\n\n## Human narrative\nThis suffix is not machine data.\n"))
		title := must(json.Marshal("Plan-" + t.label))
		body := `{"number":42,"title":` + string(title) + `,"description":` + string(encoded) + `}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	body := fmt.Sprintf(`[{"id":100,"number":100,"node_id":"node-100","title":%q}]`, "Task-"+t.label)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
}
