package codexplancontrol

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

const contractHelperStartupTimeout = 5 * time.Second

func TestNewAssessor_boundsCancellationAndIsolatesConcurrentAssessments(t *testing.T) {
	helperDirectory := t.TempDir()
	helperPath := filepath.Join(helperDirectory, "codex")
	build := exec.Command("go", "build", "-o", helperPath, "./testdata/codexstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex contract helper: %v\n%s", err, output)
	}
	t.Setenv("PATH", helperDirectory)

	current := must(plan.New(
		"Cancellation Plan",
		must(plan.NewGoal("Stop one non-cooperative assessment")),
		[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Cancellation is bounded"))},
		[]plan.Task{must(plan.NewTask("Wait for cancellation"))},
		nil,
	))
	t.Run("non cooperative interaction stops within grace", func(t *testing.T) {
		grace := 50 * time.Millisecond
		capturePath := filepath.Join(t.TempDir(), "capture.json")
		t.Setenv("CODEX_STUB_MODE", "hang")
		t.Setenv("CODEX_STUB_CAPTURE", capturePath)
		configuration := NewConfiguration(
			must(NewModel("gpt-5.6-sol")),
			must(NewReasoningEffort("high")),
			must(NewWorkingDirectory(t.TempDir())),
			must(NewShutdownGrace(grace)),
		)
		assessor := must(NewAssessor(configuration, func(_ context.Context, observation string) (string, error) { return observation, nil }))
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan struct {
			assessment plancontrol.Assessment
			err        error
		}, 1)
		go func() {
			assessment, err := plancontrol.Assess(ctx, current, "hang", assessor)
			result <- struct {
				assessment plancontrol.Assessment
				err        error
			}{assessment: assessment, err: err}
		}()

		readyDeadline := time.After(contractHelperStartupTimeout)
		for {
			if _, err := os.Stat(capturePath); err == nil {
				break
			}
			select {
			case <-readyDeadline:
				t.Fatal("Codex helper did not start")
			default:
				time.Sleep(time.Millisecond)
			}
		}
		cancelledAt := time.Now()
		cancel()
		var completed struct {
			assessment plancontrol.Assessment
			err        error
		}
		select {
		case completed = <-result:
		case <-time.After(grace + 500*time.Millisecond):
			t.Fatal("non-cooperative interaction exceeded cancellation bound")
		}
		elapsed := time.Since(cancelledAt)
		if !errors.Is(completed.err, context.Canceled) {
			t.Errorf("Assess() error = %v, want context.Canceled", completed.err)
		}
		if completed.assessment.Outcome() != 0 || completed.assessment.AssessedPlan().IsValid() {
			t.Errorf("Assessment = %#v, want zero assessment", completed.assessment)
		}
		if elapsed > grace+500*time.Millisecond {
			t.Errorf("cancellation took %v, want no more than %v plus scheduling tolerance", elapsed, grace)
		}
	})

	t.Run("blocked request write stops within grace", func(t *testing.T) {
		grace := 50 * time.Millisecond
		readyDirectory := t.TempDir()
		t.Setenv("CODEX_STUB_MODE", "block_before_turn")
		t.Setenv("CODEX_STUB_READY_DIRECTORY", readyDirectory)
		t.Setenv("CODEX_STUB_CAPTURE", filepath.Join(t.TempDir(), "capture.json"))
		configuration := NewConfiguration(
			must(NewModel("gpt-5.6-sol")),
			must(NewReasoningEffort("high")),
			must(NewWorkingDirectory(t.TempDir())),
			must(NewShutdownGrace(grace)),
		)
		assessor := must(NewAssessor(configuration, func(_ context.Context, _ string) (string, error) {
			return strings.Repeat("x", 2<<20), nil
		}))
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := plancontrol.Assess(ctx, current, "blocked", assessor)
			result <- err
		}()

		readyDeadline := time.After(contractHelperStartupTimeout)
		for {
			if _, err := os.Stat(filepath.Join(readyDirectory, "blocked.ready")); err == nil {
				break
			}
			select {
			case <-readyDeadline:
				t.Fatal("Codex helper did not block before turn input")
			default:
				time.Sleep(time.Millisecond)
			}
		}
		time.Sleep(10 * time.Millisecond)
		cancelledAt := time.Now()
		cancel()
		select {
		case err := <-result:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Assess() error = %v, want context.Canceled", err)
			}
			if elapsed := time.Since(cancelledAt); elapsed > grace+500*time.Millisecond {
				t.Errorf("blocked-write cancellation took %v, want no more than %v plus scheduling tolerance", elapsed, grace)
			}
		case <-time.After(grace + 500*time.Millisecond):
			t.Fatal("blocked request write exceeded cancellation bound")
		}
	})

	t.Run("cancelled interaction does not affect concurrent success", func(t *testing.T) {
		readyDirectory := t.TempDir()
		t.Setenv("CODEX_STUB_MODE", "mixed")
		t.Setenv("CODEX_STUB_READY_DIRECTORY", readyDirectory)
		t.Setenv("CODEX_STUB_CAPTURE", filepath.Join(t.TempDir(), "capture.json"))
		configuration := NewConfiguration(
			must(NewModel("gpt-5.6-sol")),
			must(NewReasoningEffort("high")),
			must(NewWorkingDirectory(t.TempDir())),
			must(NewShutdownGrace(50*time.Millisecond)),
		)
		assessor := must(NewAssessor(configuration, func(_ context.Context, observation string) (string, error) { return observation, nil }))
		hangingPlan := must(plan.New(
			"Hanging Plan",
			must(plan.NewGoal("Remain isolated while cancelled")),
			[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("No result crosses calls"))},
			nil,
			nil,
		))
		successPlan := must(plan.New(
			"Successful Plan",
			must(plan.NewGoal("Complete independently")),
			[]plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Success is unaffected"))},
			[]plan.Task{must(plan.NewTask("Return complete"))},
			nil,
		))

		hangingContext, cancelHanging := context.WithCancel(context.Background())
		hangingResult := make(chan error, 1)
		go func() {
			_, err := plancontrol.Assess(hangingContext, hangingPlan, "hang", assessor)
			hangingResult <- err
		}()
		type successResult struct {
			assessment plancontrol.Assessment
			err        error
		}
		successfulResult := make(chan successResult, 1)
		go func() {
			assessment, err := plancontrol.Assess(context.Background(), successPlan, "success", assessor)
			successfulResult <- successResult{assessment: assessment, err: err}
		}()

		readyDeadline := time.After(contractHelperStartupTimeout)
		for {
			_, hangErr := os.Stat(filepath.Join(readyDirectory, "hang.ready"))
			_, successErr := os.Stat(filepath.Join(readyDirectory, "success.ready"))
			if hangErr == nil && successErr == nil {
				break
			}
			select {
			case <-readyDeadline:
				t.Fatal("concurrent helpers did not receive isolated material")
			default:
				time.Sleep(time.Millisecond)
			}
		}
		assertCapturedMaterial(t, filepath.Join(readyDirectory, "hang.ready"), hangingPlan, "hang")
		assertCapturedMaterial(t, filepath.Join(readyDirectory, "success.ready"), successPlan, "success")

		var completed successResult
		select {
		case completed = <-successfulResult:
		case <-time.After(2 * time.Second):
			t.Fatal("successful concurrent assessment did not complete")
		}
		if completed.err != nil {
			t.Fatalf("successful Assess() error = %v", completed.err)
		}
		if completed.assessment.Outcome() != plancontrol.Complete || !completed.assessment.AssessedPlan().Equal(successPlan) {
			t.Errorf("successful Assessment = %#v, want Complete for successful Plan", completed.assessment)
		}
		cancelHanging()
		select {
		case err := <-hangingResult:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("hanging Assess() error = %v, want context.Canceled", err)
			}
		case <-time.After(550 * time.Millisecond):
			t.Fatal("hanging concurrent assessment exceeded cancellation bound")
		}
	})
}

func TestNewAssessor_repeatedAssessmentsRetainNoPriorJudgment(t *testing.T) {
	helperDirectory := t.TempDir()
	helperPath := filepath.Join(helperDirectory, "codex")
	build := exec.Command("go", "build", "-o", helperPath, "./testdata/codexstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex contract helper: %v\n%s", err, output)
	}
	t.Setenv("PATH", helperDirectory)
	t.Setenv("CODEX_STUB_MODE", "correlated")
	t.Setenv("CODEX_STUB_CAPTURE", filepath.Join(t.TempDir(), "capture.json"))
	readyDirectory := t.TempDir()
	t.Setenv("CODEX_STUB_READY_DIRECTORY", readyDirectory)
	configuration := NewConfiguration(
		must(NewModel("gpt-5.6-sol")),
		must(NewReasoningEffort("high")),
		must(NewWorkingDirectory(t.TempDir())),
		must(NewShutdownGrace(100*time.Millisecond)),
	)
	assessor := must(NewAssessor(configuration, func(_ context.Context, observation string) (string, error) { return observation, nil }))
	firstPlan := must(plan.New("First Plan", must(plan.NewGoal("First goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("First accepted"))}, nil, nil))
	secondPlan := must(plan.New("Second Plan", must(plan.NewGoal("Second goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("Second accepted"))}, nil, nil))

	first, err := plancontrol.Assess(context.Background(), firstPlan, "first", assessor)
	if err != nil || first.Outcome() != plancontrol.Retain || !first.AssessedPlan().Equal(firstPlan) {
		t.Fatalf("first Assessment = %#v, error = %v", first, err)
	}
	assertCapturedMaterial(t, filepath.Join(readyDirectory, "first.ready"), firstPlan, "first")
	second, err := plancontrol.Assess(context.Background(), secondPlan, "second", assessor)
	if err != nil || second.Outcome() != plancontrol.Complete || !second.AssessedPlan().Equal(secondPlan) {
		t.Fatalf("second Assessment = %#v, error = %v", second, err)
	}
	assertCapturedMaterial(t, filepath.Join(readyDirectory, "second.ready"), secondPlan, "second")
}

func assertCapturedMaterial(t *testing.T, path string, current plan.Plan, observations string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read captured assessment material: %v", err)
	}
	want, err := encodeAssessmentMaterial(current, observations)
	if err != nil {
		t.Fatalf("encode expected assessment material: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("captured assessment material = %s, want %s", got, want)
	}
}
