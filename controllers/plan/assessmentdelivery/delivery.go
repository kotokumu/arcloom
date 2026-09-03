// Package assessmentdelivery returns an existing published Plan Assessment for
// subsequent Plan consideration. It does not make or apply that decision.
package assessmentdelivery

import (
	"context"
	"errors"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
)

var (
	ErrInvalidRecipient = errors.New("invalid Plan Assessment recipient")
	ErrInvalidReport    = errors.New("invalid published Plan Report")
)

// Recipient implements one concrete handoff without receiving a Snapshot or
// authority to act. It must return within its agreed bound after ctx ends.
// Success means only that this invocation completed, not that a Plan changed.
type Recipient func(context.Context, controlruntime.TargetIdentity, plancontrol.Assessment) error

// DeliveryError preserves a recipient's failure, including uncertain effects.
// It is distinct from an Attempt Failure and never implies safe retry.
type DeliveryError struct{ cause error }

func (e *DeliveryError) Error() string { return "Plan Assessment delivery failed" }
func (e *DeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Deliver handles a Report received from the bound Plan Controller. It invokes
// recipient at most once, only for an existing Assessment, preserving identity
// and value. Nil context, ended context, nil recipient, then invalid Report are
// checked in that order. No-current and Failure Reports need no handoff.
// Caller cancellation observed before or after handoff wins over its return;
// other handoff errors become DeliveryError. There is no retry or observation.
func Deliver(ctx context.Context, report controlruntime.Report[planattempt.AttemptResult], recipient Recipient) error {
	if ctx == nil {
		return controlruntime.ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if recipient == nil {
		return ErrInvalidRecipient
	}
	if report.Target().Kind() == "" || report.Target().Key() == "" {
		return ErrInvalidReport
	}
	if _, ok := report.Failure(); ok {
		return nil
	}
	result, _, ok := report.Completion()
	if !ok {
		return ErrInvalidReport
	}
	assessment, assessed := result.Assessment()
	if !assessed {
		if result.Kind() == planattempt.CurrentPlanNotEstablished {
			return nil
		}
		return ErrInvalidReport
	}
	err := recipient(ctx, report.Target(), assessment)
	if ended := ctx.Err(); ended != nil {
		return ended
	}
	if err != nil {
		return &DeliveryError{cause: err}
	}
	return nil
}
