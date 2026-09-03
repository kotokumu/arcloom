package planattempt

import (
	"context"
	"errors"
	"fmt"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

var (
	ErrInvalidTargetKind     = errors.New("invalid target kind")
	ErrInvalidTargetResolver = errors.New("invalid target resolver")
	ErrInvalidAssessor       = errors.New("invalid plan control assessor")
)

// FailureCode identifies a stable Plan Attempt boundary failure.
type FailureCode string

const (
	TargetBindingUnavailable       FailureCode = "target_binding_unavailable"
	DeliveryObservationUnavailable FailureCode = "delivery_observation_unavailable"
)

// FailureError hides Resolver and Delivery Observer implementation errors.
type FailureError struct {
	code FailureCode
}

func (f *FailureError) Error() string {
	if f == nil || f.code == "" {
		return "plan reconciliation failed"
	}
	return fmt.Sprintf("plan reconciliation failed: %s", f.code)
}

func (f *FailureError) Code() FailureCode {
	if f == nil {
		return ""
	}
	return f.code
}

// AttemptResultKind identifies which successful Plan evaluation occurred.
type AttemptResultKind uint8

const (
	CurrentPlanNotEstablished AttemptResultKind = iota + 1
	CurrentPlanAssessed
)

// AttemptResult preserves one successful Snapshot and its explicit branch.
type AttemptResult struct {
	kind       AttemptResultKind
	snapshot   plansnapshot.Snapshot
	assessment plancontrol.Assessment
}

func (r AttemptResult) Kind() AttemptResultKind { return r.kind }

func (r AttemptResult) Snapshot() plansnapshot.Snapshot { return r.snapshot }

func (r AttemptResult) Assessment() (plancontrol.Assessment, bool) {
	return r.assessment, r.kind == CurrentPlanAssessed
}

// NewAttempt constructs the Plan Module's direct generic Attempt implementation.
// One returned Attempt may be called concurrently for distinct identities, so
// the supplied Resolver and Assessor and resolved observation boundaries must
// support that use; a Controller excludes concurrent calls for an equal identity.
// Each invocation resolves the exact binding, obtains Snapshot, and, only when a
// current Plan exists, obtains Delivery observations and calls Plan Control in
// that order. Context errors observed before or after a boundary win unchanged;
// every failure returns zero AttemptResult and Directive. Success is read-only,
// performs no Authorization or external mutation, and awaits another Request.
func NewAttempt[O any](targetKind string, resolver TargetResolver[O], assessor plancontrol.Assessor[O]) (controlruntime.Attempt[AttemptResult], error) {
	validatedKind, err := controlruntime.NewTargetIdentity(targetKind, "validation")
	if err != nil || validatedKind.Kind() != targetKind {
		return nil, ErrInvalidTargetKind
	}
	if resolver == nil {
		return nil, ErrInvalidTargetResolver
	}
	if assessor == nil {
		return nil, ErrInvalidAssessor
	}

	return func(ctx context.Context, requested controlruntime.TargetIdentity) (AttemptResult, controlruntime.Directive, error) {
		if ctx == nil {
			return AttemptResult{}, controlruntime.Directive{}, controlruntime.ErrInvalidContext
		}
		if err := ctx.Err(); err != nil {
			return AttemptResult{}, controlruntime.Directive{}, err
		}
		if requested.Kind() != targetKind {
			return AttemptResult{}, controlruntime.Directive{}, &FailureError{code: TargetBindingUnavailable}
		}

		target, resolveErr := resolver(ctx, requested)
		if err := ctx.Err(); err != nil {
			return AttemptResult{}, controlruntime.Directive{}, err
		}
		if resolveErr != nil || !target.isValidFor(requested) {
			return AttemptResult{}, controlruntime.Directive{}, &FailureError{code: TargetBindingUnavailable}
		}

		snapshot, observeErr := plansnapshot.Observe(ctx, target.observer)
		if err := ctx.Err(); err != nil {
			return AttemptResult{}, controlruntime.Directive{}, err
		}
		if observeErr != nil {
			return AttemptResult{}, controlruntime.Directive{}, observeErr
		}
		current, hasCurrent := snapshot.CurrentPlan()
		if !hasCurrent {
			return AttemptResult{kind: CurrentPlanNotEstablished, snapshot: snapshot}, controlruntime.AwaitAnotherRequest(), nil
		}

		observations, deliveryErr := target.delivery(ctx)
		if err := ctx.Err(); err != nil {
			return AttemptResult{}, controlruntime.Directive{}, err
		}
		if deliveryErr != nil {
			return AttemptResult{}, controlruntime.Directive{}, &FailureError{code: DeliveryObservationUnavailable}
		}

		assessment, assessErr := plancontrol.Assess(ctx, current, observations, assessor)
		if err := ctx.Err(); err != nil {
			return AttemptResult{}, controlruntime.Directive{}, err
		}
		if assessErr != nil {
			return AttemptResult{}, controlruntime.Directive{}, assessErr
		}
		return AttemptResult{kind: CurrentPlanAssessed, snapshot: snapshot, assessment: assessment}, controlruntime.AwaitAnotherRequest(), nil
	}, nil
}
