package reconciliationcontrol_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/reconciliationcontrol"
)

func TestControllerAttemptReacquiresCurrentFacts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var current atomic.Int64
	current.Store(1)
	attempt := reconciliationcontrol.Attempt[int64](func(context.Context, reconciliationcontrol.TargetIdentity) (int64, reconciliationcontrol.Directive, error) {
		return current.Load(), reconciliationcontrol.AwaitAnotherRequest(), nil
	})
	controller := must(reconciliationcontrol.Start(ctx, 1, attempt))
	target := must(reconciliationcontrol.NewTargetIdentity("test", "level"))

	if err := controller.Request(context.Background(), target); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	select {
	case report := <-controller.Reports():
		got, _, ok := report.Completion()
		if diff := cmp.Diff(int64(1), got); diff != "" {
			t.Errorf("first current fact mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(true, ok); diff != "" {
			t.Errorf("first Completion presence mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(time.Second):
		t.Fatal("first Report was not published")
	}

	current.Store(2)
	if err := controller.Request(context.Background(), target); err != nil {
		t.Fatalf("second Request() error = %v", err)
	}
	select {
	case report := <-controller.Reports():
		got, _, ok := report.Completion()
		if diff := cmp.Diff(int64(2), got); diff != "" {
			t.Errorf("second current fact mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(true, ok); diff != "" {
			t.Errorf("second Completion presence mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(time.Second):
		t.Fatal("second Report was not published")
	}

	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}
