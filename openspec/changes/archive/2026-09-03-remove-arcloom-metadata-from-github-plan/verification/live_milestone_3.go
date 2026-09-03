//go:build ignore

// This opt-in operator proof is excluded from builds and deterministic tests.
// Run from the repository root; the GitHub CLI supplies a credential in memory.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	plansnapshot "github.com/kotokumu/arcloom/controllers/plan/snapshot"
	githubplan "github.com/kotokumu/arcloom/providers/github/plan"
)

const rootPath = "/repos/kotokumu/arcloom/milestones/3"
const tasksPath = "/repos/kotokumu/arcloom/issues"
const frozenName = "Arcloom Reconciliation Model and Module Finalization"
const frozenGoal = "Use the deterministic and real Plan Feedback Loop evidence from Milestone #2 to remodel and refactor Reconciliation, then finalize only the modules and interfaces required by verified behavior."

var frozenConditions = []string{
	"Milestone #2 provides a runnable reference Host, deterministic local test environment, multi-cycle convergence test, and real GitHub Plan evidence.",
	"The responsibilities of Reconciliation, Control, Feedback Loop composition, Result Destination, External Actor, and Observation are re-audited against that evidence.",
	"The accepted conceptual model, specifications, and Architecture contain no unresolved responsibility or boundary decision required by implementation.",
	"The implementation is refactored to the accepted model while the deterministic and real-loop verification remains green.",
	"Only evidence-backed module responsibilities, boundaries, and consumer-owned interfaces remain.",
}

var frozenTasks = []string{
	"Re-audit Reconciliation, Control, Feedback Loop, Result Destination, External Actor, and Observation responsibilities.",
	"Finalize the modules and interfaces required for Controller development.",
	"Remodel Reconciliation from verified Feedback Loop evidence.",
	"Refactor Reconciliation to the accepted model without weakening loop behavior.",
}

type responseEvidence struct {
	URL, SHA256, RequestedVersion, SelectedVersion, ServerDate, CapturedAt string
	Status                                                                 int
	body                                                                   []byte
}

type captureTransport struct {
	token     string
	responses []responseEvidence
}

func (t *captureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "api.github.com" {
		return nil, errors.New("proof permits only GitHub HTTPS reads")
	}
	if request.URL.Path != rootPath && (request.URL.Path != tasksPath || request.URL.Query().Get("milestone") != "3") {
		return nil, errors.New("proof target differs from Milestone 3")
	}
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", "Bearer "+t.token)
	request.Header.Set("Cache-Control", "no-cache")
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		return nil, errors.New("GitHub transport unavailable")
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.New("GitHub response could not be captured")
	}
	t.responses = append(t.responses, responseEvidence{
		URL: request.URL.String(), SHA256: fmt.Sprintf("%x", sha256.Sum256(body)),
		RequestedVersion: request.Header.Get("X-GitHub-Api-Version"),
		SelectedVersion:  response.Header.Get("X-GitHub-Api-Version-Selected"),
		ServerDate:       response.Header.Get("Date"), CapturedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Status: response.StatusCode, body: body,
	})
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

type rootFacts struct {
	Number                    int
	Title, Description, State string
	DueOn                     *string `json:"due_on"`
}

type taskFacts struct {
	ID           int64
	Number       int
	Title, State string
	PullRequest  json.RawMessage `json:"pull_request"`
}

type taskProgress struct{ Name, State string }

type report struct {
	ImplementationCommit, Runtime, StartedAt, CompletedAt, APIVersion string
	Responses                                                         []responseEvidence
	HasCurrent, HasProgress, MembershipComplete, MarkerAbsent         bool
	Name, Goal                                                        string
	AcceptanceConditions, Tasks                                       []string
	TargetDate                                                        *string
	OverallState                                                      string
	TaskProgress                                                      []taskProgress
	NativeTasks                                                       []taskFacts
	NativeMatchesFixture, SnapshotMatchesNative                       bool
	Verdict                                                           string
}

func stateText(state plansnapshot.ProgressState) string {
	switch state {
	case plansnapshot.Open:
		return "open"
	case plansnapshot.Closed:
		return "closed"
	default:
		return "unknown"
	}
}

func run() error {
	live := flag.Bool("live", false, "opt in to authenticated read-only Milestone 3 proof")
	commit := flag.String("implementation-commit", "", "final product implementation commit")
	flag.Parse()
	if !*live || *commit == "" {
		return errors.New("explicit --live and --implementation-commit are required")
	}
	resolved, err := exec.Command("git", "rev-parse", "--verify", *commit+"^{commit}").Output()
	if err != nil {
		return errors.New("implementation commit does not resolve")
	}
	*commit = strings.TrimSpace(string(resolved))
	// Prevent evidence for changed product code being attributed to an older SHA.
	if err := exec.Command("git", "diff", "--exit-code", *commit, "--", "providers", "controllers", "go.mod", "go.sum").Run(); err != nil {
		return errors.New("product files differ from implementation commit")
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--", "providers", "controllers").Output()
	if err != nil || len(bytes.TrimSpace(untracked)) != 0 {
		return errors.New("untracked product files prevent commit-bound proof")
	}
	token, err := exec.Command("gh", "auth", "token", "--hostname", "github.com").Output()
	if err != nil || len(bytes.TrimSpace(token)) == 0 {
		return errors.New("authenticated GitHub CLI access is required")
	}
	transport := &captureTransport{token: strings.TrimSpace(string(token))}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	repository, err := githubplan.NewRepository("kotokumu", "arcloom")
	if err != nil {
		return err
	}
	number, err := githubplan.NewResourceNumber(3)
	if err != nil {
		return err
	}
	target, err := githubplan.NewMilestoneTarget(repository, number)
	if err != nil {
		return err
	}
	observer, err := githubplan.NewMilestoneSnapshotObserver(client, target)
	if err != nil {
		return err
	}
	result := report{ImplementationCommit: *commit, Runtime: runtime.Version(), APIVersion: githubplan.RESTAPIVersion, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	snapshot, observeErr := plansnapshot.Observe(ctx, observer)
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	result.Responses = transport.responses
	var root rootFacts
	for _, response := range transport.responses {
		if response.Status != http.StatusOK {
			return errors.New("live proof deferred: GitHub did not return HTTP 200")
		}
		if strings.HasSuffix(response.URL, rootPath) {
			if err := json.Unmarshal(response.body, &root); err != nil {
				return errors.New("invalid root evidence")
			}
		} else {
			var tasks []taskFacts
			if err := json.Unmarshal(response.body, &tasks); err != nil {
				return errors.New("invalid task evidence")
			}
			for _, task := range tasks {
				if len(task.PullRequest) == 0 {
					result.NativeTasks = append(result.NativeTasks, task)
				}
			}
		}
	}
	sort.Slice(result.NativeTasks, func(i, j int) bool { return result.NativeTasks[i].ID < result.NativeTasks[j].ID })
	current, hasCurrent := snapshot.CurrentPlan()
	progress, hasProgress := snapshot.Progress()
	result.HasCurrent, result.HasProgress = hasCurrent, hasProgress
	result.Name, result.Goal = current.Name(), current.Goal().Text()
	for _, condition := range current.AcceptanceConditions() {
		result.AcceptanceConditions = append(result.AcceptanceConditions, condition.Statement())
	}
	for _, task := range current.Tasks() {
		result.Tasks = append(result.Tasks, task.Name())
	}
	if date, present := current.TargetDate(); present {
		value := date.String()
		result.TargetDate = &value
	}
	result.MembershipComplete, result.OverallState = progress.MembershipComplete(), stateText(progress.OverallState())
	for _, task := range progress.Tasks() {
		result.TaskProgress = append(result.TaskProgress, taskProgress{task.Name(), stateText(task.State())})
	}
	expectedDescription := "## Goal\n\n" + frozenGoal + "\n\n## Acceptance Conditions"
	for i, condition := range frozenConditions {
		expectedDescription += fmt.Sprintf("\n\n### %d\n\n%s", i+1, condition)
	}
	expectedDescription += "\n"
	result.MarkerAbsent = root.Number == 3 && !strings.Contains(root.Description, "arcloom-plan:v1")
	result.NativeMatchesFixture = root.Number == 3 && root.Title == frozenName && root.Description == expectedDescription && root.DueOn == nil && root.State == "open" && len(result.NativeTasks) == 4
	numbers := []int{47, 48, 57, 58}
	var nativeProgress []taskProgress
	for i, task := range result.NativeTasks {
		nativeProgress = append(nativeProgress, taskProgress{task.Title, task.State})
		if i >= len(numbers) || task.Number != numbers[i] || task.Title != frozenTasks[i] || task.State != "open" {
			result.NativeMatchesFixture = false
		}
	}
	result.SnapshotMatchesNative = observeErr == nil && hasCurrent && hasProgress && result.MembershipComplete && result.NativeMatchesFixture && result.Name == root.Title && result.Goal == frozenGoal && reflect.DeepEqual(result.AcceptanceConditions, frozenConditions) && reflect.DeepEqual(result.Tasks, frozenTasks) && result.TargetDate == nil && result.OverallState == root.State && reflect.DeepEqual(result.TaskProgress, nativeProgress)
	result.Verdict = "DEFERRED: native drift or unavailable evidence"
	if result.NativeMatchesFixture {
		result.Verdict = "FAIL: public Snapshot differs from native facts"
		if result.SnapshotMatchesNative && result.MarkerAbsent {
			result.Verdict = "PASS: technical proof; human approval pending"
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return err
	}
	if !result.SnapshotMatchesNative || !result.MarkerAbsent {
		return errors.New("live proof did not pass")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
