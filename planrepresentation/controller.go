package planrepresentation

import (
	"context"
	"errors"
	"fmt"

	"github.com/kotokumu/arcloom/plan"
)

// Observer is the consumer-owned read Port for one immutable external target
// binding. The binding is selected and structurally validated by the Host or
// Provider implementation and remains fixed for the Observer lifetime; target
// identity never crosses this Port. Each call isolates its reads and returns
// one logical Observation, and the Observer implementation must support safe
// concurrent calls without sharing per-call state. Provider-specific errors
// are converted to Observation availability before they cross this boundary.
// If ctx is cancelled or its deadline expires, the Observer returns ctx.Err().
// If an Observer returns both an Observation and an error, Reconcile ignores
// the Observation. A non-context error, or a context error not equal to the
// supplied context's current Err, is an observation-contract failure; no
// Provider error is exposed.
type Observer func(ctx context.Context) (Observation, error)

// FailureCode identifies a stable Controller call or observation-boundary
// failure.
type FailureCode string

const (
	InvalidObserver     FailureCode = "invalid_observer"
	InvalidContext      FailureCode = "invalid_context"
	InvalidExpectedPlan FailureCode = "invalid_expected_plan"
	ObservationContract FailureCode = "observation_contract"
)

// FailureError is a stable provider-independent Controller failure.
type FailureError struct {
	code FailureCode
}

func (e *FailureError) Error() string {
	if e == nil {
		return "plan representation reconciliation failure"
	}
	return fmt.Sprintf("plan representation reconciliation failed: %s", e.code)
}

// Code returns the stable failure category.
func (e *FailureError) Code() FailureCode {
	if e == nil {
		return ""
	}
	return e.code
}

// Controller coordinates one Observer and Plan-specific reconciliation. It is
// immutable after construction and safe for concurrent calls when its Observer
// satisfies the Observer concurrent-use contract. A zero Controller is
// unusable but remains panic-free.
type Controller struct {
	observer Observer
}

// NewController constructs a Controller with a required Observer Port. A nil
// Observer returns a FailureError whose Code is InvalidObserver.
func NewController(observer Observer) (*Controller, error) {
	if observer == nil {
		return nil, &FailureError{code: InvalidObserver}
	}
	return &Controller{observer: observer}, nil
}

// Reconcile observes and compares one valid expected Plan. It returns either
// one valid immutable Result or an error with the zero Result, never both. A
// nil context returns InvalidContext before every other call-time outcome. For
// a non-nil context, its current Err is returned directly before invalid
// Observer state, invalid expected Plan, or Observer-contract failures;
// InvalidObserver precedes InvalidExpectedPlan. An Observer-returned context
// error is accepted only when it equals the context's current Err; an
// unrelated context error is an ObservationContract failure. An Observation
// returned with any Observer error is ignored. Cancellation after a valid
// Result is returned does not alter that Result.
func (c *Controller) Reconcile(ctx context.Context, expected plan.Plan) (Result, error) {
	if ctx == nil {
		return Result{}, &FailureError{code: InvalidContext}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if c == nil || c.observer == nil {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		return Result{}, &FailureError{code: InvalidObserver}
	}
	if !expected.IsValid() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		return Result{}, &FailureError{code: InvalidExpectedPlan}
	}

	observation, observerErr := c.observer(ctx)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if observerErr != nil {
		if isContextError(observerErr) {
			if current := ctx.Err(); current != nil && observerErr == current {
				return Result{}, current
			}
		}
		return Result{}, &FailureError{code: ObservationContract}
	}
	if !observation.valid() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		return Result{}, &FailureError{code: ObservationContract}
	}
	result := reconcileObservation(expected, observation)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
