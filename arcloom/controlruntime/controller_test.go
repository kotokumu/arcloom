package controlruntime_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/controlruntime"
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
	attempt := controlruntime.Attempt[string](func(context.Context, controlruntime.TargetIdentity) (string, controlruntime.Directive, error) {
		calls.Add(1)
		return "unexpected", controlruntime.AwaitAnotherRequest(), nil
	})

	controller, err := controlruntime.Start(ctx, 1, attempt)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	gotErr := controller.Request(context.Background(), controlruntime.TargetIdentity{})
	if diff := cmp.Diff(true, errors.Is(gotErr, controlruntime.ErrInvalidTargetIdentity)); diff != "" {
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
	validAttempt := controlruntime.Attempt[int](func(context.Context, controlruntime.TargetIdentity) (int, controlruntime.Directive, error) {
		return 0, controlruntime.AwaitAnotherRequest(), nil
	})
	doneContext, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name          string
		ctx           context.Context
		maxConcurrent int
		attempt       controlruntime.Attempt[int]
		want          error
	}{
		{name: "nil context", ctx: nil, maxConcurrent: 1, attempt: validAttempt, want: controlruntime.ErrInvalidContext},
		{name: "done context", ctx: doneContext, maxConcurrent: 1, attempt: validAttempt, want: context.Canceled},
		{name: "zero concurrency", ctx: context.Background(), maxConcurrent: 0, attempt: validAttempt, want: controlruntime.ErrInvalidConcurrency},
		{name: "negative concurrency", ctx: context.Background(), maxConcurrent: -1, attempt: validAttempt, want: controlruntime.ErrInvalidConcurrency},
		{name: "nil attempt", ctx: context.Background(), maxConcurrent: 1, attempt: nil, want: controlruntime.ErrInvalidAttempt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, err := controlruntime.Start(tt.ctx, tt.maxConcurrent, tt.attempt)
			if diff := cmp.Diff(true, errors.Is(err, tt.want)); diff != "" {
				t.Errorf("Start() error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff((*controlruntime.Controller[int])(nil), controller); diff != "" {
				t.Errorf("Start() Controller mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
