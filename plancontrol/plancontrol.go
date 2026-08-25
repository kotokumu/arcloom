// Package plancontrol establishes a Plan-specific assessment from a current
// Plan, caller-owned observations, and an external AI judgment.
package plancontrol

import (
	"context"
	"errors"
	"fmt"

	"github.com/kotokumu/arcloom/plan"
)

// Outcome names a Plan-specific AI judgment. Complete, Retain, Revise, and
// InsufficientInformation are intentionally Plan Control's own outcomes; they
// are not a universal Reconciliation result taxonomy.
type Outcome uint8

const (
	Complete Outcome = iota + 1
	Retain
	Revise
	InsufficientInformation
)

// AssessorResponse is an untrusted passive boundary artifact. Claims may be
// empty, duplicated, contradictory, or unknown, and ProposedPlans may have
// invalid cardinality or contain a zero Plan. Assess validates those states
// before constructing an Assessment. The producer must not mutate either slice
// while Assess is reading the response; Assess snapshots the selected Plan
// values into its immutable result before returning and does not retain the
// response slices.
type AssessorResponse struct {
	Claims        []Outcome
	ProposedPlans []plan.Plan
}

// ErrUntranslatableAIResponse identifies an AI response that was received but
// could not be decoded far enough to represent its claims and proposal
// cardinality as an AssessorResponse. When returned by an Assessor, Assess
// maps this sentinel, including wrapped or joined forms, to AIContractFailure
// without exposing the underlying error.
var ErrUntranslatableAIResponse = errors.New("untranslatable AI response")

// Assessor is the consumer-owned port for one external AI judgment. It receives
// a valid current Plan and caller-selected observation vocabulary O. The
// Assessor must not mutate state reachable from observations and the caller
// must not mutate that reachable state until the call returns unless it has
// independent synchronization. After ctx.Done, an implementation must stop
// waiting for its Provider and return. Provider transmission, retention,
// token/cost, telemetry, and other operational effects remain the concrete
// Provider contract; Plan Control does not authorize or apply a Plan, execute
// Tasks, decide Delivery Acceptance, or persist authoritative state.
type Assessor[O any] func(context.Context, plan.Plan, O) (AssessorResponse, error)

// Assessment is an immutable Plan-specific AI judgment associated with the
// exact current Plan supplied to Assess. Its zero value is not a successful
// result. A successful Revise Assessment contains exactly one valid proposed
// Plan; other outcomes contain no proposed Plan.
type Assessment struct {
	outcome      Outcome
	assessedPlan plan.Plan
	proposedPlan plan.Plan
}

// Outcome reports the AI judgment established by this Assessment. The zero
// value means that no valid Assessment was established.
func (a Assessment) Outcome() Outcome { return a.outcome }

// AssessedPlan returns the exact current Plan associated with this Assessment.
// The zero Assessment returns the zero Plan.
func (a Assessment) AssessedPlan() plan.Plan { return a.assessedPlan }

// ProposedPlan returns the proposed Plan and whether one is present. Presence
// is true only for a valid Revise Assessment; all other outcomes, including
// the zero Assessment, return false and the zero Plan.
func (a Assessment) ProposedPlan() (plan.Plan, bool) {
	return a.proposedPlan, a.outcome == Revise
}

// FailureCode identifies one of the five stable reasons why no valid
// Assessment was established. A supplied context cancellation or deadline is
// returned as the caller-owned context error instead of a FailureCode.
type FailureCode string

const (
	InvalidContext     FailureCode = "invalid_context"
	InvalidCurrentPlan FailureCode = "invalid_current_plan"
	InvalidAssessor    FailureCode = "invalid_assessor"
	AIContractFailure  FailureCode = "ai_contract_failure"
	AIBoundaryFailure  FailureCode = "ai_boundary_failure"
)

// FailureError is provider-independent and does not unwrap or expose Provider
// errors. It is returned only when no valid Assessment exists; a valid
// InsufficientInformation outcome is not a FailureError. Its zero value has an
// empty Code and the stable Error string "plan control failed".
type FailureError struct {
	code FailureCode
}

func (f *FailureError) Error() string {
	if f.code == "" {
		return "plan control failed"
	}
	return fmt.Sprintf("plan control failed: %s", f.code)
}

// Code returns the stable FailureCode. The zero FailureError returns an empty
// code.
func (f *FailureError) Code() FailureCode {
	return f.code
}

// Assess establishes one immutable Assessment with a nil error, or returns the
// public zero Assessment with one stable FailureError or the supplied context
// error. The caller supplies current as a snapshot established from
// authoritative external facts; Plan Control validates only its structural
// validity and does not establish that authority, freshness, application
// safety, or semantic completion. It validates context, current Plan, and
// Assessor before one Assessor invocation. A supplied context error observed
// before return wins over the Assessor response or error and is returned
// unchanged as caller-owned request termination, not as a FailureError. A
// successful response is validated for one supported outcome and, for Revise,
// one valid Plan unequal to current. Assess may incur the configured
// Assessor's external interaction effects, but it does not authorize or apply
// a Plan, execute Tasks, decide Delivery Acceptance, acquire observations,
// persist state, or retain an authoritative runtime decision.
func Assess[O any](ctx context.Context, current plan.Plan, observations O, assessor Assessor[O]) (Assessment, error) {
	if ctx == nil {
		return Assessment{}, &FailureError{code: InvalidContext}
	}
	if err := ctx.Err(); err != nil {
		return Assessment{}, err
	}
	if !current.IsValid() {
		return Assessment{}, &FailureError{code: InvalidCurrentPlan}
	}
	if assessor == nil {
		return Assessment{}, &FailureError{code: InvalidAssessor}
	}

	response, assessorErr := assessor(ctx, current, observations)
	if err := ctx.Err(); err != nil {
		return Assessment{}, err
	}

	var result Assessment
	if assessorErr != nil {
		if errors.Is(assessorErr, ErrUntranslatableAIResponse) {
			result = Assessment{}
			assessorErr = &FailureError{code: AIContractFailure}
		} else {
			result = Assessment{}
			assessorErr = &FailureError{code: AIBoundaryFailure}
		}
		if err := ctx.Err(); err != nil {
			return Assessment{}, err
		}
		return result, assessorErr
	}

	result, assessorErr = assessmentFromResponse(current, response)
	if err := ctx.Err(); err != nil {
		return Assessment{}, err
	}
	return result, assessorErr
}

func assessmentFromResponse(current plan.Plan, response AssessorResponse) (Assessment, error) {
	if len(response.Claims) != 1 {
		return Assessment{}, &FailureError{code: AIContractFailure}
	}
	outcome := response.Claims[0]
	if outcome < Complete || outcome > InsufficientInformation {
		return Assessment{}, &FailureError{code: AIContractFailure}
	}
	if outcome != Revise {
		if len(response.ProposedPlans) != 0 {
			return Assessment{}, &FailureError{code: AIContractFailure}
		}
		return Assessment{outcome: outcome, assessedPlan: current}, nil
	}
	if len(response.ProposedPlans) != 1 {
		return Assessment{}, &FailureError{code: AIContractFailure}
	}
	proposed := response.ProposedPlans[0]
	if !proposed.IsValid() || plansEqual(current, proposed) {
		return Assessment{}, &FailureError{code: AIContractFailure}
	}
	return Assessment{outcome: Revise, assessedPlan: current, proposedPlan: proposed}, nil
}

func plansEqual(left, right plan.Plan) bool {
	if left.IsValid() != right.IsValid() || left.Name() != right.Name() || left.Goal().Text() != right.Goal().Text() {
		return false
	}
	leftConditions, rightConditions := left.AcceptanceConditions(), right.AcceptanceConditions()
	if len(leftConditions) != len(rightConditions) {
		return false
	}
	for i := range leftConditions {
		if leftConditions[i].Statement() != rightConditions[i].Statement() {
			return false
		}
	}
	leftTasks, rightTasks := left.Tasks(), right.Tasks()
	if len(leftTasks) != len(rightTasks) {
		return false
	}
	for i := range leftTasks {
		if leftTasks[i].Name() != rightTasks[i].Name() {
			return false
		}
	}
	leftDate, leftHasDate := left.TargetDate()
	rightDate, rightHasDate := right.TargetDate()
	return leftHasDate == rightHasDate && (!leftHasDate || leftDate.String() == rightDate.String())
}
