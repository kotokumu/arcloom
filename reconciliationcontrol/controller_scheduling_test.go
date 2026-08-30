package reconciliationcontrol_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/reconciliationcontrol"
)

func TestControllerSchedulingMixedRequestStress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	started := make(chan reconciliationcontrol.TargetIdentity, 4)
	var globalActive atomic.Int64
	var maximumObserved atomic.Int64
	var stateMu sync.Mutex
	activeByTarget := map[reconciliationcontrol.TargetIdentity]bool{}
	overlapped := false
	attempt := reconciliationcontrol.Attempt[string](func(ctx context.Context, target reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
		stateMu.Lock()
		if activeByTarget[target] {
			overlapped = true
		}
		activeByTarget[target] = true
		stateMu.Unlock()
		active := globalActive.Add(1)
		for {
			observed := maximumObserved.Load()
			if active <= observed || maximumObserved.CompareAndSwap(observed, active) {
				break
			}
		}
		select {
		case started <- target:
		default:
		}
		select {
		case <-release:
		case <-ctx.Done():
		}
		globalActive.Add(-1)
		stateMu.Lock()
		activeByTarget[target] = false
		stateMu.Unlock()
		return target.Key(), reconciliationcontrol.AwaitAnotherRequest(), nil
	})
	controller := must(reconciliationcontrol.Start(ctx, 4, attempt))
	targets := make([]reconciliationcontrol.TargetIdentity, 8)
	for index := range targets {
		targets[index] = must(reconciliationcontrol.NewTargetIdentity("stress", string(rune('a'+index))))
	}
	for index := range 4 {
		if err := controller.Request(context.Background(), targets[index]); err != nil {
			t.Fatalf("initial Request() error = %v", err)
		}
	}
	for range 4 {
		<-started
	}
	for index := range 120 {
		if err := controller.Request(context.Background(), targets[index%len(targets)]); err != nil {
			t.Fatalf("mixed Request() error = %v", err)
		}
	}
	close(release)
	for range 12 {
		select {
		case report := <-controller.Reports():
			result, _, completed := report.Completion()
			if !completed || result != report.Target().Key() {
				t.Errorf("cross-target result: target %q, result %q, completed %v", report.Target().Key(), result, completed)
			}
		case <-time.After(time.Second):
			t.Fatal("stress Report was lost")
		}
	}
	cancel()
	_ = controller.Wait()
	stateMu.Lock()
	defer stateMu.Unlock()
	if overlapped {
		t.Error("same target Attempts overlapped")
	}
	if got := maximumObserved.Load(); got > 4 {
		t.Errorf("maximum concurrent Attempts = %d, want <= 4", got)
	}
}

func TestControllerSchedulingDuplicatePendingRequestsCoalesce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})
	targetStarted := make(chan struct{}, 2)
	attempt := reconciliationcontrol.Attempt[string](func(ctx context.Context, target reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
		if target.Key() == "blocker" {
			close(blockerStarted)
			select {
			case <-releaseBlocker:
			case <-ctx.Done():
				return "", reconciliationcontrol.Directive{}, ctx.Err()
			}
		} else {
			targetStarted <- struct{}{}
		}
		return target.Key(), reconciliationcontrol.AwaitAnotherRequest(), nil
	})
	controller := must(reconciliationcontrol.Start(ctx, 1, attempt))
	blocker := must(reconciliationcontrol.NewTargetIdentity("test", "blocker"))
	target := must(reconciliationcontrol.NewTargetIdentity("test", "target"))
	if err := controller.Request(context.Background(), blocker); err != nil {
		t.Fatalf("blocker Request() error = %v", err)
	}
	<-blockerStarted
	for range 3 {
		if err := controller.Request(context.Background(), target); err != nil {
			t.Fatalf("duplicate Request() error = %v", err)
		}
	}
	close(releaseBlocker)
	<-controller.Reports()
	select {
	case <-targetStarted:
	case <-time.After(time.Second):
		t.Fatal("coalesced target Attempt did not start")
	}
	<-controller.Reports()
	cancel()
	_ = controller.Wait()
	if got := len(targetStarted); got != 0 {
		t.Errorf("duplicate Pending requests started %d extra Attempts", got)
	}
}

func TestControllerSchedulingCoincidentCompletionAndRequestPreservesLaterAttempt(t *testing.T) {
	for iteration := range 20 {
		ctx, cancel := context.WithCancel(context.Background())
		gate := make(chan struct{})
		started := make(chan struct{}, 2)
		var calls atomic.Int64
		attempt := reconciliationcontrol.Attempt[int64](func(ctx context.Context, _ reconciliationcontrol.TargetIdentity) (int64, reconciliationcontrol.Directive, error) {
			call := calls.Add(1)
			started <- struct{}{}
			if call == 1 {
				select {
				case <-gate:
				case <-ctx.Done():
					return 0, reconciliationcontrol.Directive{}, ctx.Err()
				}
			}
			return call, reconciliationcontrol.AwaitAnotherRequest(), nil
		})
		controller := must(reconciliationcontrol.Start(ctx, 1, attempt))
		target := must(reconciliationcontrol.NewTargetIdentity("test", "coincident"))
		if err := controller.Request(context.Background(), target); err != nil {
			t.Fatalf("iteration %d first Request() error = %v", iteration, err)
		}
		<-started
		requested := make(chan error, 1)
		go func() {
			<-gate
			requested <- controller.Request(context.Background(), target)
		}()
		close(gate)
		if err := <-requested; err != nil {
			t.Fatalf("iteration %d coincident Request() error = %v", iteration, err)
		}
		for range 2 {
			select {
			case <-controller.Reports():
			case <-time.After(time.Second):
				t.Fatalf("iteration %d expected two Reports", iteration)
			}
		}
		cancel()
		_ = controller.Wait()
		if got := calls.Load(); got != 2 {
			t.Errorf("iteration %d Attempt calls = %d, want 2", iteration, got)
		}
	}
}

func TestControllerSchedulingDistinctTargetsProgressWithinBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan reconciliationcontrol.TargetIdentity, 2)
	release := make(chan struct{})
	attempt := reconciliationcontrol.Attempt[string](func(ctx context.Context, target reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
		started <- target
		select {
		case <-release:
			return target.Key(), reconciliationcontrol.AwaitAnotherRequest(), nil
		case <-ctx.Done():
			return "", reconciliationcontrol.Directive{}, ctx.Err()
		}
	})
	controller := must(reconciliationcontrol.Start(ctx, 2, attempt))
	first := must(reconciliationcontrol.NewTargetIdentity("test", "first"))
	second := must(reconciliationcontrol.NewTargetIdentity("test", "second"))
	if err := controller.Request(context.Background(), first); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	if err := controller.Request(context.Background(), second); err != nil {
		t.Fatalf("second Request() error = %v", err)
	}

	gotKeys := map[string]bool{}
	for range 2 {
		select {
		case target := <-started:
			gotKeys[target.Key()] = true
		case <-time.After(time.Second):
			t.Fatal("distinct target did not start within the concurrency bound")
		}
	}
	if diff := cmp.Diff(map[string]bool{"first": true, "second": true}, gotKeys); diff != "" {
		t.Errorf("started targets mismatch (-want +got):\n%s", diff)
	}
	close(release)
	for range 2 {
		select {
		case <-controller.Reports():
		case <-time.After(time.Second):
			t.Fatal("Completion Report was not published")
		}
	}
	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}

func TestControllerSchedulingActiveRequestIsAcceptedAndRunsLater(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan int, 2)
	releaseFirst := make(chan struct{})
	call := 0
	attempt := reconciliationcontrol.Attempt[int](func(ctx context.Context, target reconciliationcontrol.TargetIdentity) (int, reconciliationcontrol.Directive, error) {
		call++
		current := call
		started <- current
		if current == 1 {
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				return 0, reconciliationcontrol.Directive{}, ctx.Err()
			}
		}
		return current, reconciliationcontrol.AwaitAnotherRequest(), nil
	})
	controller := must(reconciliationcontrol.Start(ctx, 1, attempt))
	target := must(reconciliationcontrol.NewTargetIdentity("test", "same"))
	if err := controller.Request(context.Background(), target); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first Attempt did not start")
	}

	accepted := make(chan error, 1)
	go func() { accepted <- controller.Request(context.Background(), target) }()
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("second Request() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Request during Active Attempt was not accepted")
	}
	close(releaseFirst)
	select {
	case <-controller.Reports():
	case <-time.After(time.Second):
		t.Fatal("first Completion Report was not published")
	}
	select {
	case got := <-started:
		if diff := cmp.Diff(2, got); diff != "" {
			t.Errorf("later Attempt number mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(time.Second):
		t.Fatal("preserved later Attempt did not start")
	}
	select {
	case <-controller.Reports():
	case <-time.After(time.Second):
		t.Fatal("later Completion Report was not published")
	}
	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}
