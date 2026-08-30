package reconciliationcontrol_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/reconciliationcontrol"
)

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func TestControllerRequestInvalidTargetStartsNoAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int64
	attempt := reconciliationcontrol.Attempt[string](func(context.Context, reconciliationcontrol.TargetIdentity) (string, reconciliationcontrol.Directive, error) {
		calls.Add(1)
		return "unexpected", reconciliationcontrol.AwaitAnotherRequest(), nil
	})

	controller, err := reconciliationcontrol.Start(ctx, 1, attempt)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	gotErr := controller.Request(context.Background(), reconciliationcontrol.TargetIdentity{})
	if diff := cmp.Diff(true, errors.Is(gotErr, reconciliationcontrol.ErrInvalidTargetIdentity)); diff != "" {
		t.Errorf("Request() invalid identity mismatch (-want +got):\n%s", diff)
	}
	cancel()
	if got := controller.Wait(); !errors.Is(got, context.Canceled) {
		t.Errorf("Wait() error = %v, want context.Canceled", got)
	}
	if diff := cmp.Diff(int64(0), calls.Load()); diff != "" {
		t.Errorf("Attempt calls mismatch (-want +got):\n%s", diff)
	}
}

func TestStartRejectsInvalidConfiguration(t *testing.T) {
	validAttempt := reconciliationcontrol.Attempt[int](func(context.Context, reconciliationcontrol.TargetIdentity) (int, reconciliationcontrol.Directive, error) {
		return 0, reconciliationcontrol.AwaitAnotherRequest(), nil
	})
	doneContext, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name          string
		ctx           context.Context
		maxConcurrent int
		attempt       reconciliationcontrol.Attempt[int]
		want          error
	}{
		{name: "nil context", ctx: nil, maxConcurrent: 1, attempt: validAttempt, want: reconciliationcontrol.ErrInvalidContext},
		{name: "done context", ctx: doneContext, maxConcurrent: 1, attempt: validAttempt, want: context.Canceled},
		{name: "zero concurrency", ctx: context.Background(), maxConcurrent: 0, attempt: validAttempt, want: reconciliationcontrol.ErrInvalidConcurrency},
		{name: "negative concurrency", ctx: context.Background(), maxConcurrent: -1, attempt: validAttempt, want: reconciliationcontrol.ErrInvalidConcurrency},
		{name: "nil attempt", ctx: context.Background(), maxConcurrent: 1, attempt: nil, want: reconciliationcontrol.ErrInvalidAttempt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := reconciliationcontrol.Start(tt.ctx, tt.maxConcurrent, tt.attempt)
			if diff := cmp.Diff(true, errors.Is(err, tt.want)); diff != "" {
				t.Errorf("Start() error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff((*reconciliationcontrol.Controller[int])(nil), controller); diff != "" {
				t.Errorf("Start() Controller mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
