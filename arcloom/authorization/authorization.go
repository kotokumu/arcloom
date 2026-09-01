// Package authorization evaluates generic, consumer-owned authorization
// subjects without retaining authoritative authorization state.
package authorization

import (
	"context"
	"errors"
)

// RuleConclusion is the result of one rule's interpretation of an exact
// subject.
type RuleConclusion uint8

const (
	Permit RuleConclusion = iota + 1
	Deny
	Unknown
)

// Decision is the aggregate result of a Policy evaluation.
type Decision uint8

const (
	Authorized Decision = iota + 1
	Denied
	Undecidable
)

var (
	errInvalidContext = errors.New("invalid authorization context")
	errInvalidPolicy  = errors.New("invalid authorization policy")
	errInvalidRule    = errors.New("invalid authorization rule")
)

// Rule evaluates one authorization interpretation for an exact subject.
//
// S and all state reachable from S are consumer-owned and must remain
// semantically immutable for the complete lifetime of any returned
// Evaluation. Rule implementations must not mutate S or reachable state and
// must be safe for concurrent Policy evaluations.
type Rule[S any] func(context.Context, S) (RuleConclusion, error)

// Policy is an immutable non-empty set of authorization rules.
type Policy[S any] struct {
	rules []Rule[S]
	valid bool
}

// NewPolicy constructs an immutable policy. The supplied rule slice is not
// retained.
func NewPolicy[S any](rules ...Rule[S]) (Policy[S], error) {
	if len(rules) == 0 {
		return Policy[S]{}, errInvalidPolicy
	}
	for _, rule := range rules {
		if rule == nil {
			return Policy[S]{}, errInvalidRule
		}
	}
	return Policy[S]{rules: append([]Rule[S](nil), rules...), valid: true}, nil
}

// Evaluation binds one exact subject to one aggregate Decision. The subject
// is retained as supplied; the consumer owns the semantic immutability
// precondition for S and all state reachable from it.
type Evaluation[S any] struct {
	subject  S
	decision Decision
	valid    bool
}

func (e Evaluation[S]) Subject() S { return e.subject }

func (e Evaluation[S]) Decision() Decision {
	if !e.valid {
		return Undecidable
	}
	return e.decision
}

// Evaluate applies the policy to the exact supplied subject. A zero or
// invalid Policy fails closed with an Undecidable Evaluation. Cancellation
// before completion returns the caller's context error and no Evaluation.
// Evaluate performs no deep-copy or provider-specific interpretation of S;
// the consumer-owned semantic immutability precondition applies for the
// Evaluation lifetime.
func (p Policy[S]) Evaluate(ctx context.Context, subject S) (Evaluation[S], error) {
	if ctx == nil {
		return Evaluation[S]{}, errInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return Evaluation[S]{}, err
	}
	if !p.valid || len(p.rules) == 0 {
		return Evaluation[S]{subject: subject, decision: Undecidable, valid: true}, nil
	}

	allPermit := true
	for _, rule := range p.rules {
		if err := ctx.Err(); err != nil {
			return Evaluation[S]{}, err
		}
		conclusion, err := rule(ctx, subject)
		if contextErr := ctx.Err(); contextErr != nil {
			return Evaluation[S]{}, contextErr
		}
		if err != nil {
			allPermit = false
			continue
		}
		switch conclusion {
		case Deny:
			return Evaluation[S]{subject: subject, decision: Denied, valid: true}, nil
		case Permit:
		case Unknown:
			allPermit = false
		default:
			allPermit = false
		}
	}

	decision := Undecidable
	if allPermit {
		decision = Authorized
	}
	return Evaluation[S]{subject: subject, decision: decision, valid: true}, nil
}
