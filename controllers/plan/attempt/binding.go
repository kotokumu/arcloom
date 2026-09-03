package planattempt

import (
	"context"
	"errors"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan/control"
)

var ErrInvalidPlanAttemptBinding = errors.New("invalid Plan Attempt Binding")

// PlanAttemptBinding preserves one immutable identity/executable association.
// Its zero value is invalid. Construction performs no observation or assessment.
type PlanAttemptBinding struct {
	identity controlruntime.TargetIdentity
	attempt  controlruntime.Attempt[AttemptResult]
}

// NewPlanAttemptBinding accepts only a valid target and non-nil assessor.
// Rejection returns the zero binding and ErrInvalidPlanAttemptBinding.
func NewPlanAttemptBinding[O any](target PlanTarget[O], assessor plancontrol.Assessor[O]) (PlanAttemptBinding, error) {
	if target.identity.Kind() == "" || target.identity.Key() == "" || !target.isValidFor(target.identity) || assessor == nil {
		return PlanAttemptBinding{}, ErrInvalidPlanAttemptBinding
	}
	attempt, err := NewAttempt(target.identity.Kind(), func(_ context.Context, requested controlruntime.TargetIdentity) (PlanTarget[O], error) {
		if !target.isValidFor(requested) {
			return PlanTarget[O]{}, ErrInvalidPlanAttemptBinding
		}
		return target, nil
	}, assessor)
	if err != nil {
		return PlanAttemptBinding{}, ErrInvalidPlanAttemptBinding
	}
	return PlanAttemptBinding{identity: target.identity, attempt: attempt}, nil
}

func (b PlanAttemptBinding) Identity() controlruntime.TargetIdentity        { return b.identity }
func (b PlanAttemptBinding) Attempt() controlruntime.Attempt[AttemptResult] { return b.attempt }
