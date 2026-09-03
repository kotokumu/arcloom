package controlruntime

import (
	"errors"
	"time"
)

var ErrInvalidDirective = errors.New("invalid control directive")

// DirectiveKind identifies target-independent reevaluation behavior.
type DirectiveKind uint8

const (
	AwaitRequest DirectiveKind = iota + 1
	ReevaluateImmediately
	ReevaluateAfterDelay
)

// Directive is a constructed target-independent scheduling decision. Its zero
// value and values not returned by this package's constructors are invalid.
type Directive struct {
	kind  DirectiveKind
	delay time.Duration
}

// AwaitAnotherRequest selects no internal reevaluation.
func AwaitAnotherRequest() Directive { return Directive{kind: AwaitRequest} }

// ImmediateReevaluation selects immediate internal reevaluation.
func ImmediateReevaluation() Directive { return Directive{kind: ReevaluateImmediately} }

// DelayedReevaluation selects reevaluation after one positive delay.
func DelayedReevaluation(delay time.Duration) (Directive, error) {
	if delay <= 0 {
		return Directive{}, ErrInvalidDirective
	}
	return Directive{kind: ReevaluateAfterDelay, delay: delay}, nil
}

// Kind returns the Directive kind, or zero for an invalid Directive.
func (d Directive) Kind() DirectiveKind { return d.kind }

// Delay returns the delay only for ReevaluateAfterDelay.
func (d Directive) Delay() (time.Duration, bool) {
	if d.kind != ReevaluateAfterDelay || d.delay <= 0 {
		return 0, false
	}
	return d.delay, true
}

func (d Directive) isValid() bool {
	switch d.kind {
	case AwaitRequest, ReevaluateImmediately:
		return d.delay == 0
	case ReevaluateAfterDelay:
		return d.delay > 0
	default:
		return false
	}
}
