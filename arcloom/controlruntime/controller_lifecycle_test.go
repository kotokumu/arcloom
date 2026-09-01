package controlruntime_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
)

func TestControllerZeroValueAndStableLifecycleViews(t *testing.T) {
	var zero controlruntime.Controller[int]
	if zero.Reports() != nil {
		t.Error("zero Controller Reports() is non-nil")
	}
	if err := zero.Wait(); !errors.Is(err, controlruntime.ErrControllerNotStarted) {
		t.Errorf("zero Controller Wait() error = %v", err)
	}
	if err := zero.Request(context.Background(), controlruntime.TargetIdentity{}); !errors.Is(err, controlruntime.ErrControllerNotStarted) {
		t.Errorf("zero Controller Request() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	controller := must(controlruntime.Start(ctx, 1, func(context.Context, controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
		return 1, controlruntime.AwaitAnotherRequest(), nil
	}))
	firstReports := controller.Reports()
	secondReports := controller.Reports()
	if firstReports != secondReports {
		t.Error("Reports() did not return one stable stream")
	}
	reportStreams := make(chan (<-chan controlruntime.Report[int]), 16)
	var reportCalls sync.WaitGroup
	for range 16 {
		reportCalls.Add(1)
		go func() {
			defer reportCalls.Done()
			reportStreams <- controller.Reports()
		}()
	}
	reportCalls.Wait()
	close(reportStreams)
	for stream := range reportStreams {
		if stream != firstReports {
			t.Error("concurrent Reports() returned another stream")
		}
	}
	cancel()
	var waits sync.WaitGroup
	errorsSeen := make(chan error, 4)
	for range 4 {
		waits.Add(1)
		go func() {
			defer waits.Done()
			errorsSeen <- controller.Wait()
		}()
	}
	waits.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("concurrent Wait() error = %v", err)
		}
	}
	if _, ok := <-controller.Reports(); ok {
		t.Error("Reports remained open after Wait")
	}
}

func TestControllerRequestContextBoundsOnlySubmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attemptStarted := make(chan context.Context, 1)
	release := make(chan struct{})
	controller := must(controlruntime.Start(ctx, 1, func(attemptCtx context.Context, _ controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
		attemptStarted <- attemptCtx
		select {
		case <-release:
			return 1, controlruntime.AwaitAnotherRequest(), nil
		case <-attemptCtx.Done():
			return 0, controlruntime.Directive{}, attemptCtx.Err()
		}
	}))
	target := must(controlruntime.NewTargetIdentity("test", "submission-context"))

	//nolint:staticcheck // The public contract explicitly rejects a nil context.
	if err := controller.Request(nil, target); !errors.Is(err, controlruntime.ErrInvalidContext) {
		t.Errorf("Request(nil) error = %v", err)
	}
	doneSubmission, stopSubmission := context.WithCancel(context.Background())
	stopSubmission()
	if err := controller.Request(doneSubmission, target); !errors.Is(err, context.Canceled) {
		t.Errorf("Request(done context) error = %v", err)
	}

	submission, cancelSubmission := context.WithCancel(context.Background())
	if err := controller.Request(submission, target); err != nil {
		t.Fatalf("accepted Request() error = %v", err)
	}
	attemptCtx := <-attemptStarted
	cancelSubmission()
	if err := attemptCtx.Err(); err != nil {
		t.Errorf("Attempt context was cancelled by submission context: %v", err)
	}
	close(release)
	<-controller.Reports()
	cancel()
	_ = controller.Wait()
}

func TestControllerCancellationDiscardsProspectiveReportAndStopsBackpressure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	returned := make(chan struct{})
	var calls atomic.Int64
	controller := must(controlruntime.Start(ctx, 1, func(context.Context, controlruntime.TargetIdentity) (int64, controlruntime.Directive, error) {
		call := calls.Add(1)
		if call == 1 {
			close(returned)
		}
		return call, controlruntime.ImmediateReevaluation(), nil
	}))
	first := must(controlruntime.NewTargetIdentity("test", "first"))
	second := must(controlruntime.NewTargetIdentity("test", "second"))
	if err := controller.Request(context.Background(), first); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	<-returned
	if err := controller.Request(context.Background(), second); err != nil {
		t.Fatalf("Request while Report blocked error = %v", err)
	}
	cancel()
	done := make(chan error, 1)
	go func() { done <- controller.Wait() }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Wait() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Wait blocked on an unconsumed Report")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("Attempts started during Report backpressure = %d, want 1", got)
	}
	if _, ok := <-controller.Reports(); ok {
		t.Error("a prospective Report was published after cancellation")
	}
}

func TestControllerCancellationWaitsForActiveAttemptAndPublishesNoPostStopSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	allowReturn := make(chan struct{})
	controller := must(controlruntime.Start(ctx, 1, func(attemptCtx context.Context, _ controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
		close(started)
		<-attemptCtx.Done()
		<-allowReturn
		return 99, controlruntime.AwaitAnotherRequest(), nil
	}))
	target := must(controlruntime.NewTargetIdentity("test", "active"))
	if err := controller.Request(context.Background(), target); err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	<-started
	cancel()
	waiting := make(chan error, 1)
	waitEntered := make(chan struct{})
	go func() {
		close(waitEntered)
		waiting <- controller.Wait()
	}()
	<-waitEntered
	close(allowReturn)
	select {
	case err := <-waiting:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Wait() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after Active Attempt")
	}
	if _, ok := <-controller.Reports(); ok {
		t.Error("nominal post-cancellation success was published")
	}
}

func TestControllerReadyConsumerReceivesNoSuccessReturnedAfterCancellation(t *testing.T) {
	for iteration := range 100 {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		var calls atomic.Int64
		controller := must(controlruntime.Start(ctx, 1, func(attemptCtx context.Context, _ controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
			calls.Add(1)
			close(started)
			<-attemptCtx.Done()
			return 99, controlruntime.ImmediateReevaluation(), nil
		}))
		target := must(controlruntime.NewTargetIdentity("test", "post-cancel-success"))
		if err := controller.Request(context.Background(), target); err != nil {
			t.Fatalf("iteration %d Request() error = %v", iteration, err)
		}
		<-started
		consumerReady := make(chan struct{})
		received := make(chan bool, 1)
		go func() {
			close(consumerReady)
			_, ok := <-controller.Reports()
			received <- ok
		}()
		<-consumerReady
		cancel()
		if err := controller.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("iteration %d Wait() error = %v", iteration, err)
		}
		select {
		case ok := <-received:
			if ok {
				t.Errorf("iteration %d published a post-cancellation Completion", iteration)
			}
		case <-time.After(time.Second):
			t.Fatalf("iteration %d ready Report consumer did not resolve", iteration)
		}
		if got := calls.Load(); got != 1 {
			t.Errorf("iteration %d Attempt calls = %d, want 1", iteration, got)
		}
	}
}

func TestControllerBackpressureAllowsActiveReturnsButStartsNoNewAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	releases := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
	returned := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
	started := make(chan string, 4)
	controller := must(controlruntime.Start(ctx, 3, func(attemptCtx context.Context, target controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
		started <- target.Key()
		if release, ok := releases[target.Key()]; ok {
			select {
			case <-release:
			case <-attemptCtx.Done():
			}
			close(returned[target.Key()])
		}
		return target.Key(), controlruntime.AwaitAnotherRequest(), nil
	}))
	for _, key := range []string{"a", "b", "c"} {
		if err := controller.Request(context.Background(), must(controlruntime.NewTargetIdentity("test", key))); err != nil {
			t.Fatalf("Request(%q) error = %v", key, err)
		}
	}
	for range 3 {
		<-started
	}
	close(releases["a"])
	<-returned["a"]
	if err := controller.Request(context.Background(), must(controlruntime.NewTargetIdentity("test", "d"))); err != nil {
		t.Fatalf("Request(d) during backpressure error = %v", err)
	}
	close(releases["b"])
	close(releases["c"])
	<-returned["b"]
	<-returned["c"]
	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v", err)
	}
	if got := len(started); got != 0 {
		t.Errorf("Attempts started during Report backpressure = %d, want 0", got)
	}
	if _, ok := <-controller.Reports(); ok {
		t.Error("Reports remained open after stopped-consumer cancellation")
	}
}

func TestControllerNormalMixedTargetOutcomesArePublishedExactlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	controller := must(controlruntime.Start(ctx, 3, func(_ context.Context, target controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
		if target.Key() == "failure" {
			return "", controlruntime.Directive{}, errors.New("target failed")
		}
		return target.Key(), controlruntime.AwaitAnotherRequest(), nil
	}))
	want := map[string]bool{"first": true, "failure": true, "third": true}
	for key := range want {
		target := must(controlruntime.NewTargetIdentity("test", key))
		if err := controller.Request(context.Background(), target); err != nil {
			t.Fatalf("Request(%q) error = %v", key, err)
		}
	}
	got := map[string]int{}
	for range len(want) {
		select {
		case report := <-controller.Reports():
			got[report.Target().Key()]++
		case <-time.After(time.Second):
			t.Fatal("mixed target Report was lost")
		}
	}
	cancel()
	_ = controller.Wait()
	for key := range want {
		if got[key] != 1 {
			t.Errorf("Report count for %q = %d, want 1", key, got[key])
		}
	}
}

func TestControllerSimultaneousPublicationAndCancellationHasOneWinner(t *testing.T) {
	for iteration := range 50 {
		ctx, cancel := context.WithCancel(context.Background())
		gate := make(chan struct{})
		started := make(chan struct{}, 2)
		var calls atomic.Int64
		controller := must(controlruntime.Start(ctx, 1, func(context.Context, controlruntime.TargetIdentity) (int64, controlruntime.Directive, error) {
			call := calls.Add(1)
			started <- struct{}{}
			if call == 1 {
				<-gate
			}
			return call, controlruntime.ImmediateReevaluation(), nil
		}))
		target := must(controlruntime.NewTargetIdentity("test", "publication-cancellation"))
		if err := controller.Request(context.Background(), target); err != nil {
			t.Fatalf("iteration %d Request() error = %v", iteration, err)
		}
		<-started
		type receivedReport struct {
			report controlruntime.Report[int64]
			ok     bool
		}
		received := make(chan receivedReport, 1)
		go func() {
			report, ok := <-controller.Reports()
			received <- receivedReport{report: report, ok: ok}
		}()
		cancelReady := make(chan struct{})
		go func() {
			<-gate
			close(cancelReady)
			cancel()
		}()
		close(gate)
		<-cancelReady
		if err := controller.Wait(); !errors.Is(err, context.Canceled) {
			t.Errorf("iteration %d Wait() error = %v", iteration, err)
		}
		select {
		case outcome := <-received:
			if outcome.ok {
				_, directive, completed := outcome.report.Completion()
				if !completed || directive.Kind() != controlruntime.ReevaluateImmediately {
					t.Errorf("iteration %d published non-completion outcome", iteration)
				}
			} else if got := calls.Load(); got != 1 {
				t.Errorf("iteration %d no Report but Directive-based calls = %d", iteration, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("iteration %d Report consumer did not resolve", iteration)
		}
	}
}

func TestControllerLifecycleErrorPrecedesEndedSubmissionContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	controller := must(controlruntime.Start(ctx, 1, func(context.Context, controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
		return 0, controlruntime.AwaitAnotherRequest(), nil
	}))
	cancel()
	_ = controller.Wait()
	submission, stopSubmission := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer stopSubmission()
	target := must(controlruntime.NewTargetIdentity("test", "precedence"))
	if err := controller.Request(submission, target); !errors.Is(err, context.Canceled) {
		t.Errorf("Request() error = %v, want Controller context.Canceled before submission DeadlineExceeded", err)
	}
}

func TestControllerRestartUsesOnlyNewRequestsAndCurrentFacts(t *testing.T) {
	target := must(controlruntime.NewTargetIdentity("test", "restart"))
	currentFact := "first"

	firstCtx, stopFirst := context.WithCancel(context.Background())
	first := must(controlruntime.Start(firstCtx, 1, func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
		return currentFact, must(controlruntime.DelayedReevaluation(time.Hour)), nil
	}))
	if err := first.Request(context.Background(), target); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	firstReport := <-first.Reports()
	firstResult, _, _ := firstReport.Completion()
	if firstResult != "first" {
		t.Errorf("first result = %q", firstResult)
	}
	stopFirst()
	_ = first.Wait()

	currentFact = "second"
	secondCtx, stopSecond := context.WithCancel(context.Background())
	var secondCalls atomic.Int64
	second := must(controlruntime.Start(secondCtx, 1, func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
		secondCalls.Add(1)
		return currentFact, controlruntime.AwaitAnotherRequest(), nil
	}))
	if got := secondCalls.Load(); got != 0 {
		t.Errorf("new Controller restored old eligibility; calls = %d", got)
	}
	if err := second.Request(context.Background(), target); err != nil {
		t.Fatalf("second Request() error = %v", err)
	}
	secondReport := <-second.Reports()
	secondResult, _, _ := secondReport.Completion()
	if secondResult != "second" {
		t.Errorf("second result = %q, want fresh fact", secondResult)
	}
	stopSecond()
	_ = second.Wait()
}
