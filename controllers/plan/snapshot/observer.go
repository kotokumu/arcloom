package plansnapshot

import (
	"context"
	"errors"
)

var (
	errInvalidContext        = errors.New("invalid observation context")
	errInvalidObserver       = errors.New("invalid observer")
	errInvalidObserverResult = errors.New("invalid observer result")
)

// ObservationFailureCode identifies a stable failure to establish a coherent
// current observation.
type ObservationFailureCode uint8

const ObservationUnavailable ObservationFailureCode = 1

// ObservationFailure means a current observation could not establish a
// coherent Snapshot. It does not claim that the target is absent.
type ObservationFailure struct {
	code ObservationFailureCode
}

func UnavailableObservationFailure() ObservationFailure {
	return ObservationFailure{code: ObservationUnavailable}
}

func (f ObservationFailure) Error() string {
	if f.code == ObservationUnavailable {
		return "observation unavailable"
	}
	return "invalid observation failure"
}

func (f ObservationFailure) Code() ObservationFailureCode { return f.code }

// Observer is the consumer-owned boundary for one immutable external target.
type Observer func(context.Context) (Snapshot, error)

// Observe validates the invocation and Observer result, preserving caller
// context errors while rejecting Provider-specific or malformed boundary
// results.
func Observe(ctx context.Context, observer Observer) (Snapshot, error) {
	if ctx == nil {
		return Snapshot{}, errInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if observer == nil {
		return Snapshot{}, errInvalidObserver
	}
	snapshot, err := observer(ctx)
	if contextErr := ctx.Err(); contextErr != nil {
		return Snapshot{}, contextErr
	}
	if err != nil {
		if failure, ok := validObservationFailure(err); ok {
			if snapshot.valid {
				return Snapshot{}, errInvalidObserverResult
			}
			return Snapshot{}, failure
		}
		return Snapshot{}, errInvalidObserverResult
	}
	if !snapshot.valid {
		return Snapshot{}, errInvalidObserverResult
	}
	return snapshot, nil
}

func validObservationFailure(err error) (ObservationFailure, bool) {
	if failure, ok := err.(ObservationFailure); ok && failure.code != 0 {
		return failure, true
	}
	if failure, ok := err.(*ObservationFailure); ok && failure != nil && failure.code != 0 {
		return *failure, true
	}
	return ObservationFailure{}, false
}
