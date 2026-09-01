// Package planapplication owns one invocation-local request to an external
// Actor for an exact authorized Plan revision.
package planapplication

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kotokumu/arcloom/arcloom/authorization"
	"github.com/kotokumu/arcloom/controllers/plan"
)

var (
	errInvalidContext         = errors.New("invalid application context")
	errInvalidTargetReference = errors.New("invalid target reference")
	errInvalidRevision        = errors.New("invalid revision")
	errEqualRevision          = errors.New("equal revision")
)

// TargetReference is a stable immutable value, not target state.
type TargetReference struct {
	context  string
	identity string
	valid    bool
}

func NewTargetReference(contextValue, identity string) (TargetReference, error) {
	if strings.TrimSpace(contextValue) == "" || strings.TrimSpace(identity) == "" {
		return TargetReference{}, errInvalidTargetReference
	}
	return TargetReference{context: contextValue, identity: identity, valid: true}, nil
}

func (t TargetReference) Context() string  { return t.context }
func (t TargetReference) Identity() string { return t.identity }

// Revision preserves one target reference and two valid, meaningfully
// different Plans. The caller owns the precondition that the target and
// current Plan came from one fresh observation.
type Revision struct {
	target   TargetReference
	current  plan.Plan
	proposed plan.Plan
	valid    bool
}

func NewRevision(target TargetReference, current, proposed plan.Plan) (Revision, error) {
	if !target.valid {
		return Revision{}, errInvalidRevision
	}
	if !current.IsValid() || !proposed.IsValid() {
		return Revision{}, errInvalidRevision
	}
	if current.Equal(proposed) {
		return Revision{}, errEqualRevision
	}
	return Revision{target: target, current: current, proposed: proposed, valid: true}, nil
}

func (r Revision) Target() TargetReference { return r.target }
func (r Revision) Current() plan.Plan      { return r.current }
func (r Revision) Proposed() plan.Plan     { return r.proposed }

func (r Revision) Equal(other Revision) bool {
	if !r.valid || !other.valid {
		return false
	}
	return r.target == other.target &&
		r.current.Equal(other.current) &&
		r.proposed.Equal(other.proposed)
}

// Request is the exact immutable revision supplied to an Actor.
type Request struct {
	revision Revision
	valid    bool
}

func (r Request) Revision() Revision { return r.revision }

type ReceiptKind uint8

const (
	ReceiptAcknowledged ReceiptKind = iota + 1
	ReceiptRefused
	KnownNotReceived
	ReceiptUncertain
)

// ReceiptEvidence describes only the Actor interaction. A native reference
// is optional and is never interpreted as target state. The zero value is
// ReceiptUncertain. Established evidence wins over cancellation uncertainty;
// a possible transmission without stronger evidence remains uncertain.
type ReceiptEvidence struct {
	kind      ReceiptKind
	reference string
}

func AcknowledgedReceipt(reference string) ReceiptEvidence {
	return ReceiptEvidence{kind: ReceiptAcknowledged, reference: reference}
}

func RefusedReceipt(reference string) ReceiptEvidence {
	return ReceiptEvidence{kind: ReceiptRefused, reference: reference}
}

func NotReceivedReceipt() ReceiptEvidence {
	return ReceiptEvidence{kind: KnownNotReceived}
}

func UncertainReceipt() ReceiptEvidence {
	return ReceiptEvidence{kind: ReceiptUncertain}
}

func (e ReceiptEvidence) Kind() ReceiptKind {
	switch e.kind {
	case ReceiptAcknowledged, ReceiptRefused, KnownNotReceived, ReceiptUncertain:
		return e.kind
	default:
		return ReceiptUncertain
	}
}

func (e ReceiptEvidence) Reference() (string, bool) {
	if (e.kind == ReceiptAcknowledged || e.kind == ReceiptRefused) && e.reference != "" {
		return e.reference, true
	}
	return "", false
}

func (e ReceiptEvidence) established() bool {
	return e.Kind() != ReceiptUncertain
}

// Actor owns action-time interpretation, conflict handling, and mutation.
// RequestApplication does not transmit when cancellation is already observed.
// If cancellation is observed before transmission, the Actor reports
// KnownNotReceived; if transmission may have occurred without stronger
// evidence, it reports ReceiptUncertain. An established receipt is preserved
// even when cancellation is observed afterward.
type Actor func(context.Context, Request) ReceiptEvidence

// Result exposes either a non-Authorized authorization decision or receipt
// evidence. It never represents target state.
type Result struct {
	decision    authorization.Decision
	hasDecision bool
	receipt     ReceiptEvidence
	hasReceipt  bool
	valid       bool
}

func (r Result) AuthorizationDecision() (authorization.Decision, bool) {
	if !r.valid || !r.hasDecision {
		return authorization.Undecidable, false
	}
	return r.decision, true
}

func (r Result) Receipt() (ReceiptEvidence, bool) {
	if !r.valid || !r.hasReceipt {
		return ReceiptEvidence{}, false
	}
	return r.receipt, true
}

type FailureCode string

const (
	InvalidRevision FailureCode = "invalid_revision"
	InvalidActor    FailureCode = "invalid_actor"
)

type Failure struct {
	code FailureCode
}

func (f Failure) Error() string {
	if f.code == "" {
		return "invalid application failure"
	}
	return fmt.Sprintf("application failure: %s", f.code)
}

func (f Failure) Code() FailureCode { return f.code }

func decisionResult(decision authorization.Decision) Result {
	if decision != authorization.Denied && decision != authorization.Undecidable {
		decision = authorization.Undecidable
	}
	return Result{decision: decision, hasDecision: true, valid: true}
}

func receiptResult(receipt ReceiptEvidence) Result {
	if !receipt.established() {
		receipt = UncertainReceipt()
	}
	return Result{receipt: receipt, hasReceipt: true, valid: true}
}

// RequestApplication evaluates the exact Revision for the current invocation
// and invokes the Actor at most once only for an Authorized evaluation.
func RequestApplication(
	ctx context.Context,
	revision Revision,
	policy authorization.Policy[Revision],
	actor Actor,
) (Result, error) {
	if ctx == nil {
		return Result{}, errInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if !revision.valid {
		return Result{}, Failure{code: InvalidRevision}
	}
	if actor == nil {
		return Result{}, Failure{code: InvalidActor}
	}

	evaluation, err := policy.Evaluate(ctx, revision)
	if err != nil {
		return Result{}, err
	}
	decision := evaluation.Decision()
	if decision != authorization.Authorized {
		return decisionResult(decision), nil
	}
	if !evaluation.Subject().Equal(revision) {
		return decisionResult(authorization.Undecidable), nil
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	evidence := actor(ctx, Request{revision: revision, valid: true})
	if evidence.established() {
		return receiptResult(evidence), nil
	}
	return receiptResult(UncertainReceipt()), nil
}
