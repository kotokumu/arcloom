// Package planreconciliation composes one read-only Plan evaluation from
// current target-bound observations and Plan Control.
package planreconciliation

import (
	"context"
	"errors"

	"github.com/kotokumu/arcloom/plansnapshot"
	"github.com/kotokumu/arcloom/reconciliationcontrol"
)

var ErrInvalidPlanTarget = errors.New("invalid plan target")

// DeliveryObserver obtains the observation vocabulary used by Plan Control
// after a fresh Snapshot establishes a current Plan. It must return within its
// documented bound after ctx ends. A context error wins; another error is hidden
// behind DeliveryObservationUnavailable by the Plan Attempt.
type DeliveryObserver[O any] func(context.Context) (O, error)

// PlanTarget binds one exact identity to its Plan observation boundaries.
type PlanTarget[O any] struct {
	identity reconciliationcontrol.TargetIdentity
	observer plansnapshot.Observer
	delivery DeliveryObserver[O]
}

// NewPlanTarget constructs one self-identifying immutable Plan binding.
func NewPlanTarget[O any](identity reconciliationcontrol.TargetIdentity, observer plansnapshot.Observer, delivery DeliveryObserver[O]) (PlanTarget[O], error) {
	if identity.Kind() == "" || identity.Key() == "" || observer == nil || delivery == nil {
		return PlanTarget[O]{}, ErrInvalidPlanTarget
	}
	return PlanTarget[O]{identity: identity, observer: observer, delivery: delivery}, nil
}

// TargetResolver performs local, Provider-I/O-free binding resolution for one
// exact requested identity. It must observe ctx and may be called concurrently
// for distinct identities. A context error wins; another error or an invalid or
// mismatched binding is hidden behind TargetBindingUnavailable.
type TargetResolver[O any] func(context.Context, reconciliationcontrol.TargetIdentity) (PlanTarget[O], error)

func (t PlanTarget[O]) isValidFor(identity reconciliationcontrol.TargetIdentity) bool {
	return t.identity == identity && t.observer != nil && t.delivery != nil
}
