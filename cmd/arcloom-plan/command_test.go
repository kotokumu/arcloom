package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/providers/codex/appserver"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

type commandFixture struct {
	owner           string
	repository      string
	milestone       int64
	title           string
	mu              sync.Mutex
	requests        []string
	turns           []codexappserver.ReadOnlyTurnRequest
	records         []json.RawMessage
	rootReads       int
	noCurrent       bool
	partial         bool
	httpFailure     bool
	deliveryFailure bool
	aiOutput        string
	aiError         error
	outputFailureAt int
	outputCalls     int
	evidence        string
	evidenceError   error
	clockCalls      int
	onHTTP          func(context.Context)
	onTurn          func(context.Context)
	onOutput        func(context.Context)
}

func (f *commandFixture) RoundTrip(request *http.Request) (*http.Response, error) {
	if f.onHTTP != nil {
		f.onHTTP(request.Context())
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if request.Method != http.MethodGet || request.URL.Host != "api.github.com" {
		return nil, errors.New("wrong GitHub transport")
	}
	f.requests = append(f.requests, request.URL.String())
	owner, repository, number, title := f.owner, f.repository, f.milestone, f.title
	if owner == "" {
		owner = "owner"
	}
	if repository == "" {
		repository = "repo"
	}
	if number == 0 {
		number = 7
	}
	if title == "" {
		title = "  Release 日本語  "
	}
	if !strings.HasPrefix(request.URL.Path, "/repos/"+owner+"/"+repository+"/") {
		return nil, errors.New("wrong repository")
	}
	if f.httpFailure || (f.deliveryFailure && f.rootReads > 0 && strings.Contains(request.URL.Path, "/milestones/")) {
		return nil, errors.New("secret transport diagnostic")
	}
	var body string
	if strings.Contains(request.URL.Path, "/milestones/") {
		if !strings.HasSuffix(request.URL.Path, "/milestones/"+strconv.FormatInt(number, 10)) {
			return nil, errors.New("wrong milestone")
		}
		f.rootReads++
		description := "## Goal\n\nDeliver the result\n\n## Acceptance Conditions\n\n### 1\n\nOperator accepts the result\n"
		if f.noCurrent {
			description = "missing narrative"
		}
		encoded, _ := json.Marshal(map[string]any{"number": number, "title": title, "description": description, "state": "open", "due_on": nil})
		body = string(encoded)
	} else {
		if request.URL.Query().Get("milestone") != strconv.FormatInt(number, 10) {
			return nil, errors.New("wrong milestone")
		}
		state := "open"
		if f.rootReads > 1 {
			state = "closed"
		}
		body = `[{"id":42,"title":"Task one","state":"` + state + `"}]`
		if f.partial {
			body = `[{"id":42,"title":"Task one","state":"open"},{"id":43}]`
		}
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
}
func (f *commandFixture) CompleteReadOnlyTurn(ctx context.Context, request codexappserver.ReadOnlyTurnRequest) (codexappserver.CompletedTurn, error) {
	if f.onTurn != nil {
		f.onTurn(ctx)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.turns = append(f.turns, request)
	if f.aiError != nil {
		return codexappserver.CompletedTurn{}, f.aiError
	}
	output := f.aiOutput
	if output == "" {
		output = `{"outcome":"retain","proposedPlan":null}`
	}
	return codexappserver.CompletedTurn{FinalOutput: output}, nil
}
func (f *commandFixture) dependencies() commandDependencies {
	return commandDependencies{
		httpClient: &http.Client{Transport: f},
		newCodexClient: func(path string, grace time.Duration) (codexappserver.Client, error) {
			if path != "/exact/codex" || grace != 2*time.Second {
				return nil, errors.New("wrong process configuration")
			}
			return f, nil
		},
		writeRecord: func(ctx context.Context, value any) error {
			if f.onOutput != nil {
				f.onOutput(ctx)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			f.outputCalls++
			if f.outputCalls == f.outputFailureAt {
				return errors.New("output unavailable")
			}
			encoded, err := json.Marshal(value)
			if err == nil {
				f.records = append(f.records, encoded)
			}
			return err
		},
		readEvidence: func(context.Context, string) (string, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.evidence, f.evidenceError
		},
		now: func() time.Time {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.clockCalls++
			return time.Date(2026, 9, 3, 0, 0, f.clockCalls, 0, time.UTC)
		},
		build: buildEvidence{GoVersion: "test-go", Revision: "test-revision", Version: "test-build"},
	}
}

func TestRunCommandValidatesBeforeExternalWork(t *testing.T) {
	type args struct {
		ctx  context.Context
		args []string
		deps commandDependencies
	}
	for _, flags := range [][]string{
		{}, {"--owner", "../wrong", "--repo", "repo", "--milestone", "7"},
		{"--owner", "owner", "--repo", "repo", "--milestone", "0"},
		{"--unknown", "value"},
		{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "relative", "--shutdown-grace", "0s", "--model", "model", "--effort", "high", "--workdir", "relative"},
	} {
		fixture := &commandFixture{}
		tests := []struct {
			name string
			args args
			want int
		}{{name: strings.Join(flags, " "), args: args{context.Background(), flags, fixture.dependencies()}, want: 1}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if diff := cmp.Diff(tt.want, runCommand(tt.args.ctx, tt.args.args, tt.args.deps)); diff != "" {
					t.Fatal(diff)
				}
				if len(fixture.requests) != 0 || len(fixture.turns) != 0 {
					t.Fatal("invalid configuration did external work")
				}
			})
		}
	}
}

func TestRunCommandFreshGitHubFactsExactAssessmentAndProvenance(t *testing.T) {
	fixture := &commandFixture{evidence: "Operator verified acceptance.\n日本語"}
	directory := t.TempDir()
	flags := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "selected-model", "--effort", "high", "--workdir", directory, "--delivery-evidence", "operator.txt"}
	if got := runCommand(context.Background(), flags, fixture.dependencies()); got != 0 {
		t.Fatalf("exit=%d records=%s", got, fixture.records)
	}
	if len(fixture.requests) != 4 || fixture.rootReads != 2 || len(fixture.turns) != 1 || len(fixture.records) != 2 {
		t.Fatalf("requests=%v turns=%d records=%s", fixture.requests, len(fixture.turns), fixture.records)
	}
	turn := fixture.turns[0]
	if turn.Model() != "selected-model" || turn.ReasoningEffort() != "high" || turn.WorkingDirectory() != directory {
		t.Fatal("Codex selection changed")
	}
	var material struct {
		CurrentPlan  planRecord `json:"currentPlan"`
		Observations string     `json:"observations"`
	}
	if err := json.Unmarshal([]byte(turn.Input()), &material); err != nil {
		t.Fatal(err)
	}
	if material.CurrentPlan.Name != "  Release 日本語  " || len(material.CurrentPlan.Tasks) != 1 {
		t.Fatal("Plan meaning changed")
	}
	var observation deliveryObservation
	if err := json.Unmarshal([]byte(material.Observations), &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Progress.Tasks[0].State != "closed" || observation.OperatorEvidence == nil || *observation.OperatorEvidence != fixture.evidence {
		t.Fatalf("stale/current evidence=%+v", observation)
	}
	var handoff assessmentRecord
	var processed processedRecord
	if err := json.Unmarshal(fixture.records[0], &handoff); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fixture.records[1], &processed); err != nil {
		t.Fatal(err)
	}
	if handoff.Type != "assessment" || processed.Type != "processed_report" || handoff.Target.Key != "owner/repo/milestones/7" || handoff.Target.Kind != "github-milestone" {
		t.Fatal("target/record association")
	}
	if handoff.Assessment.Outcome != "retain" || processed.Assessment.Outcome != "retain" || processed.Directive != "await_request" {
		t.Fatal("classification/directive")
	}
	if processed.Progress.Tasks[0].State != "open" || processed.CurrentPlan.Name != material.CurrentPlan.Name {
		t.Fatal("Snapshot replaced by later observation")
	}
	if diff := cmp.Diff(handoff.Assessment, *processed.Assessment); diff != "" {
		t.Fatal(diff)
	}
	provenance := processed.Provenance
	if provenance.Snapshot == nil || provenance.Delivery == nil || !provenance.Snapshot.CompletedAt.Before(provenance.Delivery.StartedAt) {
		t.Fatal("acquisition order")
	}
	if provenance.GitHubAPIVersion != githubplan.RESTAPIVersion || provenance.CodexSupportedVersion != codexappserver.SupportedCodexVersion || provenance.CodexObservedVersion != nil || provenance.Build.Revision != "test-revision" {
		t.Fatal("fabricated/missing provenance")
	}
	if !strings.Contains(provenance.TargetURL, "/owner/repo/milestone/7") {
		t.Fatal("native target reference")
	}
}

func TestRunCommandOutcomeAndFailureMatrix(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		fixture        *commandFixture
		want           int
		records        int
		turns          int
		classification string
		failure        string
		outcome        string
	}{
		{name: "retain", fixture: &commandFixture{}, want: 0, records: 2, turns: 1, classification: "assessed", outcome: "retain"},
		{name: "complete", fixture: &commandFixture{aiOutput: `{"outcome":"complete","proposedPlan":null}`}, want: 0, records: 2, turns: 1, classification: "assessed", outcome: "complete"},
		{name: "insufficient", fixture: &commandFixture{aiOutput: `{"outcome":"insufficient_information","proposedPlan":null}`}, want: 0, records: 2, turns: 1, classification: "assessed", outcome: "insufficient_information"},
		{name: "revise", fixture: &commandFixture{aiOutput: `{"outcome":"revise","proposedPlan":{"name":"New plan","goal":"Changed goal","acceptanceConditions":["Accepted"],"tasks":["Revised task"],"targetDate":"2026-12-31"}}`}, want: 0, records: 2, turns: 1, classification: "assessed", outcome: "revise"},
		{name: "no current", fixture: &commandFixture{noCurrent: true}, want: 2, records: 1, classification: "current_plan_not_established"},
		{name: "partial membership", fixture: &commandFixture{partial: true}, want: 2, records: 1, classification: "current_plan_not_established"},
		{name: "HTTP failure", fixture: &commandFixture{httpFailure: true}, want: 1, records: 1, classification: "attempt_failure", failure: "observation_unavailable"},
		{name: "delivery observation failure", fixture: &commandFixture{deliveryFailure: true}, want: 1, records: 1, classification: "attempt_failure", failure: "delivery_observation_unavailable"},
		{name: "AI failure", fixture: &commandFixture{aiError: errors.New("private SDK error")}, want: 1, records: 1, turns: 1, classification: "attempt_failure", failure: "ai_boundary_failure"},
		{name: "invalid AI", fixture: &commandFixture{aiOutput: "invalid"}, want: 1, records: 1, turns: 1, classification: "attempt_failure", failure: "ai_contract_failure"},
		{name: "recipient output failure", fixture: &commandFixture{outputFailureAt: 1}, want: 1, records: 0, turns: 1},
		{name: "processed output failure", fixture: &commandFixture{outputFailureAt: 2}, want: 1, records: 1, turns: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := scenario.fixture
			flags := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir()}
			if diff := cmp.Diff(scenario.want, runCommand(context.Background(), flags, fixture.dependencies())); diff != "" {
				t.Fatal(diff)
			}
			if len(fixture.records) != scenario.records || len(fixture.turns) != scenario.turns {
				t.Fatalf("records=%s turns=%d", fixture.records, len(fixture.turns))
			}
			for _, record := range fixture.records {
				if strings.Contains(string(record), "private SDK") || strings.Contains(string(record), "secret transport") {
					t.Fatal("provider error exposed")
				}
			}
			if scenario.classification != "" {
				var record processedRecord
				if err := json.Unmarshal(fixture.records[len(fixture.records)-1], &record); err != nil {
					t.Fatal(err)
				}
				if record.Classification != scenario.classification {
					t.Fatalf("classification=%s", record.Classification)
				}
				if diff := cmp.Diff(scenario.failure, record.Failure); diff != "" {
					t.Fatal(diff)
				}
				if scenario.classification != "assessed" && record.Assessment != nil {
					t.Fatal("fabricated assessment")
				}
				if scenario.classification == "assessed" {
					want := assessmentValue{Outcome: scenario.outcome, AssessedPlan: planRecord{Name: "  Release 日本語  ", Goal: "Deliver the result", AcceptanceConditions: []string{"Operator accepts the result"}, Tasks: []string{"Task one"}}}
					if scenario.outcome == "revise" {
						date := "2026-12-31"
						want.ProposedPlan = &planRecord{Name: "New plan", Goal: "Changed goal", AcceptanceConditions: []string{"Accepted"}, Tasks: []string{"Revised task"}, TargetDate: &date}
					}
					var handoff assessmentRecord
					if err := json.Unmarshal(fixture.records[0], &handoff); err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff(want, handoff.Assessment); diff != "" {
						t.Fatal(diff)
					}
					if diff := cmp.Diff(&want, record.Assessment); diff != "" {
						t.Fatal(diff)
					}
				}
			}
		})
	}
}

func TestRunCommandCancellationAtConcreteBoundaries(t *testing.T) {
	for _, stage := range []string{"HTTP", "Codex", "output"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fixture := &commandFixture{}
			entered := make(chan struct{}, 1)
			block := func(ctx context.Context) { entered <- struct{}{}; <-ctx.Done() }
			switch stage {
			case "HTTP":
				fixture.onHTTP = block
			case "Codex":
				fixture.onTurn = block
			case "output":
				fixture.onOutput = block
			}
			flags := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir()}
			done := make(chan int, 1)
			go func() { done <- runCommand(ctx, flags, fixture.dependencies()) }()
			<-entered
			cancel()
			if got := <-done; got != 130 {
				t.Fatalf("exit=%d", got)
			}
		})
	}
}

func TestRunCommandEvidenceIsCurrentAndFailuresAreVisible(t *testing.T) {
	fixture := &commandFixture{evidence: "first observation"}
	flags := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir(), "--delivery-evidence", "evidence.txt"}
	if got := runCommand(context.Background(), flags, fixture.dependencies()); got != 0 {
		t.Fatal(got)
	}
	fixture.evidence = "new acceptance evidence"
	if got := runCommand(context.Background(), flags, fixture.dependencies()); got != 0 {
		t.Fatal(got)
	}
	if strings.Contains(fixture.turns[1].Input(), "first observation") || !strings.Contains(fixture.turns[1].Input(), "new acceptance evidence") {
		t.Fatal("cached operator evidence")
	}
	fixture.evidenceError = errors.New("unreadable")
	if got := runCommand(context.Background(), flags, fixture.dependencies()); got != 1 {
		t.Fatal(got)
	}
	if len(fixture.turns) != 2 {
		t.Fatal("unavailable evidence reached Codex")
	}
}

func TestConcurrentCommandsKeepExactBindingsIsolated(t *testing.T) {
	for _, name := range []string{"repository", "repository-similar"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := &commandFixture{owner: "selected", repository: name, milestone: 17, title: name}
			flags := []string{"--owner", "selected", "--repo", name, "--milestone", "17", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir()}
			if got := runCommand(context.Background(), flags, fixture.dependencies()); got != 0 {
				t.Fatal(got)
			}
			var record processedRecord
			if err := json.Unmarshal(fixture.records[1], &record); err != nil {
				t.Fatal(err)
			}
			if record.Target.Key != "selected/"+name+"/milestones/17" || record.CurrentPlan.Name != name || record.Assessment.AssessedPlan.Name != name {
				t.Fatal("cross-target contamination")
			}
			for _, request := range fixture.requests {
				if !strings.HasPrefix(request, "https://api.github.com/repos/selected/"+name+"/") {
					t.Fatal("wrong target observed")
				}
			}
		})
	}
}
