# Plan Control

## 0. Document Scope

| Information | Normative document |
|---|---|
| Product concept, scope, capabilities, and principles | `PRODUCT.md` |
| Component responsibilities, external authority, dependency rules, and Port ownership | `ARCHITECTURE.md` |
| Accepted Plan behavior | `openspec/specs/plan/spec.md` |
| Proposed observable Plan Control behavior | `specs/plan-control/spec.md` in this Change |
| Structure, contracts, and verification design for this Change | This DesignDoc |
| Implementation workflow and review gates | `docs/DEVELOPMENT.md` |

---

## 1. Purpose / Out of Scope

### Purpose

Plan Control establishes one Plan-specific assessment from one current Plan, caller-supplied observations, and an external AI judgment. It preserves the AI judgment while enforcing the provider-independent Assessment and Failure contracts without owning authoritative state.

### Out of Scope

- Plan creation or removal
- Observation acquisition or a product-wide Observation schema
- A concrete AI Agent Provider implementation, prompt, model, or protocol
- External Plan application, freshness validation, retry orchestration, or repeated-loop lifecycle
- Authorization, Change, Delivery Acceptance, or Task execution
- Persistence, history, cache, resume state, or an authoritative Plan mirror
- A common result or Module interface for other Reconciliation Modules

### Risk and Frozen Evidence

Risk is **High** because this Change introduces a public Go contract and changes an architecture boundary. Construction requires an explicit DesignDoc, independent design reviews, and human approval.

Evidence Packet `PC-2026-08-25-v1.1` is frozen for this design. Version 1.1 narrows the Authorization evidence to the only fact required here: Plan Control has no Authorization dependency. This clarification is immaterial to the model and boundary decisions because no accepted Plan Control responsibility consumes Authorization.

| Fact | Source | Relevance |
|---|---|---|
| Plan Control behavior and result distinctions | `specs/plan-control/spec.md` | Defines the accepted capability contract. |
| Provider-independent Plan meaning and validity | `openspec/specs/plan/spec.md`, `plan/plan.go` | Plan remains the owner of Plan structure. |
| Evidence-derived target-independent Result | `reconciliation/result.go` | Its semantics differ from an AI-owned Plan assessment. |
| Read-only external representation reconciliation | `planrepresentation` | It does not control Plan content or completion. |
| Statelessness and external authority | `PRODUCT.md`, `ARCHITECTURE.md`, repository instructions | Prohibits an authoritative Plan, observation, or decision store. |
| Change is experimental; Plan Control does not depend on Authorization | Accepted product-design decisions | Neither is a prerequisite or dependency of Plan Control. Broader Authorization design remains outside this Change. |
| One real Plan must later traverse observation, AI judgment, possible application, re-observation, and completion | Current development Goal | Defines the surrounding goal without adding those responsibilities to this Change. |

---

## 2. Behavior Design

### 2.1 Functional Requirements

#### Plan subject

Plan Control accepts one current valid Plan. It does not establish the Plan from an external Provider.

| ID | Rule | Source |
|---|---|---|
| FR-1 | Missing or invalid current Plan produces a stable input failure without calling the external AI boundary. | PLC-1 |
| FR-2 | One successful assessment remains associated with the exact current Plan supplied for that call. | PLC-1, PLC-6 |

#### AI-owned assessment

The external AI owns the semantic judgment. Plan Control owns the accepted provider-independent form of that judgment.

| ID | Rule | Source |
|---|---|---|
| FR-3 | A successful result contains exactly one of `Complete`, `Retain`, `Revise`, or `InsufficientInformation`. | PLC-2, PLC-3 |
| FR-4 | `Complete` records the AI judgment about Goal and acceptance-condition achievement and is distinct from Delivery Acceptance. | PLC-2 |
| FR-5 | Plan Control does not independently reinterpret a structurally valid AI judgment. | PLC-2, PLC-5 |
| FR-6 | A `Revise` assessment contains exactly one valid Plan that differs from the current Plan in at least one preserved Plan element value. | PLC-3, PLC-5 |

#### Caller-owned observations

Plan Control forwards observations to the external AI boundary without defining their schema or meaning.

| ID | Rule | Source |
|---|---|---|
| FR-7 | The caller-selected observation value reaches the external AI boundary without semantic interpretation by Plan Control. | PLC-4 |
| FR-8 | Adding an observation category does not require a product-wide Observation type or a change to Assessment semantics. | PLC-4 |

#### Assessment and failure distinction

A valid insufficient-information judgment is not a failed AI interaction or invalid AI response.

| ID | Rule | Source |
|---|---|---|
| FR-9 | Empty, unknown, contradictory, or otherwise invalid successful AI responses produce an AI-contract failure and no Assessment. | PLC-5 |
| FR-10 | AI unavailability, Provider-side timeout while the supplied context remains active, or another Provider failure produces an AI-boundary failure and no Assessment. | PLC-6 |
| FR-11 | Cancellation or deadline expiry of the supplied context before the final context sample returns that exact context error and no Assessment. | PLC-6 |
| FR-12 | A valid `InsufficientInformation` response produces an Assessment rather than an error. | PLC-3, PLC-6 |

#### Lifecycle and effects

| ID | Rule | Source |
|---|---|---|
| FR-13 | A call does not authorize, apply, persist, or execute its Assessment. | PLC-7 |
| FR-14 | A call retains no authoritative input, Assessment, Failure, or AI session state after return. | PLC-8 |
| FR-15 | Concurrent calls share no per-call state through Plan Control; the supplied external AI boundary remains responsible for its own concurrent-use contract. | PLC-8 |

### 2.2 Non-functional Requirements

N/A. This Change defines no measurable performance, capacity, availability, or security threshold. Immutability, cancellation, concurrency isolation, and statelessness are functional contract rules.

---

## 3. Structure Design

### 3.1 Conceptual Model

| Concept | Meaning | Identity | Business rule / invariant |
|---|---|---|---|
| Plan | Provider-independent planning intent in the subject or proposed-revision role | Immutable value composed of its preserved Plan elements | Every Plan crossing Plan Control is structurally valid. A revision creates another Plan and does not mutate the current Plan. |
| Plan Control Assessment | One valid Plan-specific semantic judgment made by the external AI | The assessed Plan value and exactly one outcome | It has exactly one supported outcome. `Revise` relates the assessed Plan to exactly one valid unequal Plan. Other outcomes contain no revised Plan. |
| Plan Control Failure | The reason that no valid Assessment was established | One stable failure category for the call | It never coexists with a valid Assessment. Insufficient information is not a Failure. |

Observation material and the raw AI response are passive boundary artifacts. Current requirements give neither artifact independent identity, lifecycle, behavior, or Arcloom-owned invariants.

#### Concept relationships

| Source | Relationship | Target | Multiplicity and lifecycle |
|---|---|---|---|
| Plan Control call | Assesses | Current Plan | Exactly one valid immutable Plan supplied by the caller. |
| External AI | Authors semantic judgment represented by | Plan Control Assessment | Exactly one valid judgment per successful call. |
| Plan Control Assessment | Concerns | Current Plan | Exactly one assessed Plan retained in the immutable Assessment. |
| `Revise` Assessment | Proposes | Plan | Exactly one valid Plan unequal to the assessed Plan; no external lifecycle is created. |
| Plan Control call | Produces | Assessment or Failure | Exactly one valid Assessment or one error, never both. |
| Plan Control call | Uses transiently | Observation material | Caller owns its meaning and lifecycle; Plan Control does not retain it. |

#### Concept minimality

| Candidate | Decision | Reason |
|---|---|---|
| Plan Control | Do not admit as a Concept | It is a Component responsibility. Plan, Assessment, and Failure retain all independent meanings and invariants without a Plan Control entity. |
| Observation | Do not admit as a universal Concept | No stable schema, identity, lifecycle, or invariant is established across callers. |
| Proposed Revised Plan | Keep as a role of Plan | It adds no meaning beyond a valid Plan related by a `Revise` Assessment. |
| Current Plan Snapshot | Keep as a role of Plan | Plan is already immutable; this scope defines no separate version or freshness identity. |
| AI Interaction or raw AI response | Keep as boundary behavior or passive data | They name an external protocol and delivery artifact rather than problem-area state. |
| `reconciliation.Result` | Do not merge with Assessment | It derives a three-state determination from evidence. Assessment represents an external AI's Plan-specific judgment and can contain a revised Plan. |
| Change | Exclude | The accepted behavior needs no Change identity or invariant. |
| Authorization | Exclude | Plan Control neither authorizes nor applies an Assessment. |

### 3.2 Responsibility and Package Design

#### Responsibility assignment

| Responsibility / decision | Owner | Information and authority used | State / invariant affected | Change driver | Not owner / reason |
|---|---|---|---|---|---|
| Plan structure and validity | Plan | Plan elements and provider-independent rules | Valid immutable Plan | Plan vocabulary changes | Plan Control consumes and does not duplicate Plan rules. |
| Completion, retention, revision, or insufficiency judgment | External AI | Current Plan and supplied observations | Semantic content of the judgment | AI decision behavior and completion semantics | Arcloom preserves rather than replaces the judgment. |
| Assessment outcome exclusivity and subject association | Plan Control Assessment | Supported outcomes and current Plan | One valid immutable Assessment | Plan Control result semantics change | A Provider response does not own Arcloom result invariants. |
| Revised-Plan cardinality and inequality | Plan Control Assessment | Current and proposed Plans | One valid unequal proposed Plan | Revision rules change | Plan owns individual Plan validity, not the relationship. |
| Stable reason that no Assessment exists | Plan Control Failure | Input, AI response, boundary, and cancellation facts | Assessment/Failure distinction | Failure contract changes | Insufficient information remains an Assessment. |
| Establish one Assessment or Failure | Plan Controller | Capability preconditions, Concepts, and external AI boundary | Stateless result/error exclusivity | Plan Control contract changes | Plan does not own control of planning intent. |
| Forward observation material for AI interpretation | Plan Controller | Caller-supplied value | No owned observation state | Observation sources change | No universal Observation Concept is justified. |
| AI Provider adaptation | Concrete AI Agent Provider Module | Provider protocol and consumer-owned Port | Provider-specific transient data | Provider, model, or protocol changes | Plan Control must not depend on an SDK or Provider DTO. |
| Authoritative Plan and observation facts | External Contexts | Provider-native authority and lifecycle | Durable external state | External system changes | Arcloom owns no authoritative store. |
| Apply a revised Plan or repeat the loop | Separate future capability or Host | Current external state and Provider mutation semantics | Plan mutation or loop lifecycle | Integration requirements change | This Plan Control call may invoke an external AI but does not mutate a Plan or own the repeated loop. |

#### Package responsibilities

| Package | Responsibility | Provides | Does not provide |
|---|---|---|---|
| `plan` | Implements Plan meaning and structural invariants | Existing immutable Plan values and validation | Assessment, AI adaptation, external application, or control lifecycle |
| `plancontrol` | Implements the Plan Controller, Assessment, Failure, and the consumer-owned AI Port | Stateless assessment operation and immutable provider-independent result/error contracts | Observation acquisition, AI Provider implementation, `reconciliation.Result`, Change, Authorization, application, persistence, or loop lifecycle |

#### Package boundaries

| Boundary (package) | Hidden detail | Dependency direction |
|---|---|---|
| `plan` | Text, collection, and date validation implementation | Depends only on the Go standard library. |
| `plancontrol` | Assessment construction, response validation, exact Plan comparison, failure mapping, and AI Port contract | Depends inward on `plan` and the Go standard library. It does not depend on `reconciliation`, `planrepresentation`, a Provider Package, Change, Authorization, persistence, or application. |

#### Architecture boundary gate

| Boundary candidate | Consumer / evidence | State, data, or policy owner | Constraint protected | Dependency direction | Simpler existing alternative | Decision |
|---|---|---|---|---|---|---|
| Plan Controller | Host or later enclosing loop; PLC-1–8 | Assessment and Failure own result meaning; Plan owns structure | Separates Plan control from representation reconciliation and external effects | Caller → `plancontrol` → `plan` | Reuse `planrepresentation` | Accept; representation consistency has different meaning and authority. |
| External AI Assessor Port | Plan Control; PLC-3–6 | External AI owns semantic judgment; Plan Control owns accepted form | Hides AI Provider, model, protocol, transport, and native response | `plancontrol` owns Port; Provider implementation depends on it | Direct SDK call | Accept; a real volatile external dependency exists. |
| Caller-selected observation type | Plan Control and AI Assessor; PLC-4 | Caller and external sources own meaning | Avoids a fixed or untyped Observation model | Generic type flows through the consumer Port | `any`, bytes, JSON, or a universal interface | Accept generic `O any`. |
| `plan` | Plan Control Assessment | Plan owns structural invariants | Prevents duplicated Plan DTO and validation | `plancontrol` → `plan` | Copy Plan fields | Reuse. |
| Persistence, application, Change, Authorization | No current consumer | External or independent owners | No current constraint needs these boundaries | No dependency | Placeholder Ports | Reject from this Change. |

#### Independent evolution scenario impact

| Scenario / confidence | Primary owner | Expected propagation | Unexplained impact | Verdict |
|---|---|---|---|---|
| AI control replaces sufficiency-only evaluation / Committed | Plan Controller and Assessment | Architecture, contracts, tests | None | Pass |
| Observation categories expand / Committed | Caller-owned observation value and AI adapter | Caller type and adapter only | None in Assessment | Pass |
| Repeated observation and control surrounds this scope / Committed | Future Host or enclosing capability | Composition and end-to-end tests | None in this stateless call | Pass |
| Provider cannot safely apply a revision / Committed | Future application boundary | Application outcome only | None in Plan Control | Pass |
| Other Reconciliation results evolve / Committed | Their respective owners | Their contracts only | None | Pass |
| Completion semantics or Assessment repertoire changes / Evidence-backed plausible | Assessment and external AI contract | Assessment contract, AI adapter, callers, tests | Expected public-contract propagation | Pass |
| Revision cardinality or validity changes / Evidence-backed plausible | Assessment; Plan for Plan structure | Relationship rules and tests; Plan only when its structure changes | None | Pass |
| Plan vocabulary changes / Evidence-backed plausible | Plan | Plan consumers and mappings | Explainable type propagation | Pass |
| Longitudinal evidence becomes relevant / Evidence-backed plausible | External observation authority and caller | Caller-selected observation value | No store required here | Pass |
| Planning or AI Provider changes / Evidence-backed plausible | Corresponding Provider boundary | Adapter and composition | None in Concepts | Pass |
| AI operational policy changes / Evidence-backed plausible | AI adapter and Failure mapping | Adapter behavior and boundary tests | None in Assessment | Pass |
| Freshness becomes an application concern / Evidence-backed plausible | Future application boundary | Re-observation or version validation | None in current scope | Pass |
| Change or Authorization evolves / Evidence-backed plausible | Independent owner | None | None | Pass |
| Provider-native Plan representation evolves / Observed | Plan Representation Provider boundary | Provider mapping and its tests | None in Plan Control | Pass |

Speculative multi-Plan control, shared decision authority, Arcloom-owned semantic verification, and Arcloom-owned durable state do not introduce extension points in this design.

### 3.3 Interface Design

The following Go declarations are proposed signatures and are not existing implementation.

```go
package plancontrol

import (
	"context"
	"errors"

	"github.com/kotokumu/arcloom/plan"
)

// Outcome names the initially supported Plan-specific AI judgments.
type Outcome uint8

const (
	Complete Outcome = iota + 1
	Retain
	Revise
	InsufficientInformation
)

// AssessorResponse is an untrusted passive boundary artifact. Claims and
// ProposedPlans intentionally admit unknown claims, invalid Plan zero values,
// and invalid cardinalities so Assess can own provider-independent validation.
type AssessorResponse struct {
	Claims        []Outcome
	ProposedPlans []plan.Plan
}

// ErrUntranslatableAIResponse means an AI response was received but could not
// be decoded far enough to represent its claims and proposal cardinality in
// AssessorResponse.
var ErrUntranslatableAIResponse = errors.New("untranslatable AI response")

// Assessor is the consumer-owned Port for one external AI judgment. O remains
// caller-owned observation vocabulary. Until the call returns, callers do not
// mutate state reachable from O unless that state provides its own
// synchronization; implementations never mutate that state. After ctx.Done(),
// implementations stop waiting for the Provider and return. Transmission,
// retention, cost, telemetry, and other operational effects of the configured
// external AI remain part of that implementation's Provider contract.
type Assessor[O any] func(
	ctx context.Context,
	current plan.Plan,
	observations O,
) (AssessorResponse, error)

// Assess establishes exactly one Assessment or returns its zero value and an
// error. After input validation it invokes assessor exactly once. The call may
// incur the Assessor's documented operational effects, but Assess does not
// authorize or apply a Plan, perform Tasks, or persist authoritative state.
func Assess[O any](
	ctx context.Context,
	current plan.Plan,
	observations O,
	assessor Assessor[O],
) (Assessment, error)
```

```go
package plancontrol

import "github.com/kotokumu/arcloom/plan"

// Assessment is an immutable Plan-specific AI judgment associated with the
// exact Plan supplied to Assess. Its zero value is not a successful result.
type Assessment struct {
	// unexported
}

func (Assessment) Outcome() Outcome
func (Assessment) AssessedPlan() plan.Plan

// ProposedPlan returns a valid unapplied Plan and true exactly for Revise.
func (Assessment) ProposedPlan() (plan.Plan, bool)
```

```go
package plancontrol

// FailureCode identifies why no valid Assessment was established.
type FailureCode string

const (
	InvalidContext     FailureCode = "invalid_context"
	InvalidCurrentPlan FailureCode = "invalid_current_plan"
	InvalidAssessor    FailureCode = "invalid_assessor"
	AIContractFailure  FailureCode = "ai_contract_failure"
	AIBoundaryFailure  FailureCode = "ai_boundary_failure"
)

// FailureError is provider-independent and does not unwrap Provider errors.
// Cancellation is returned as the supplied context error instead.
type FailureError struct {
	// unexported
}

func (*FailureError) Error() string
func (*FailureError) Code() FailureCode
```

#### Contract semantics

| Contract | Validated inputs and caller obligations | Postconditions | Error and side-effect contract |
|---|---|---|---|
| `Assess[O]` | Non-nil context, valid current Plan, non-nil Assessor | Returns one immutable Assessment associated with current, or zero Assessment plus error; invokes Assessor exactly once after validation | No persistence, Authorization, Plan application, or Task execution; the Assessor interaction may transmit data, consume tokens, incur cost, emit telemetry, or create Provider-side state |
| `Assessor[O]` | Receives a valid current Plan; until return, caller does not mutate state reachable from `O` unless that state synchronizes access | Returns one passive provider-independent response artifact; after `ctx.Done()`, stops waiting for the Provider and returns | Never mutates state reachable from `O`; matching context error means cancellation; wrapped `ErrUntranslatableAIResponse` means contract failure; other errors become boundary failure; Provider transmission and retention follow the concrete Provider contract |
| `AssessorResponse` | Provider syntax was decoded far enough to identify claims and proposal cardinality; claims may be unknown, Plan values may be invalid zero values, and cardinalities may be invalid | `Assess` reads it only after the Assessor relinquishes mutation of the returned slices | Exactly one known claim; `Revise` requires one valid unequal Plan; other outcomes require no Plan |
| `Assessment` | Constructed only after response validation | Outcome, subject Plan, and optional revised Plan remain mutually consistent | Zero value never crosses as success |
| `FailureError` | No valid Assessment exists | `errors.As` exposes one stable category | Provider errors and payloads do not unwrap or cross the boundary |

#### Deterministic evaluation and error precedence

`Assess` has the following total ordering. A caller can therefore predict the result when multiple invalid or failure conditions coexist.

1. A nil context returns `InvalidContext`.
2. For a non-nil context, an already-cancelled or expired supplied context returns its exact `ctx.Err()`. This precedes current-Plan and Assessor validation.
3. An invalid current Plan returns `InvalidCurrentPlan`.
4. A nil Assessor returns `InvalidAssessor`.
5. The Assessor is invoked exactly once.
6. Immediately after the Assessor returns, a non-nil `ctx.Err()` from the supplied context is returned exactly and both response and Assessor error are ignored. This includes cancellation or expiry while the Assessor was running.
7. If the supplied context remains active and the Assessor returned an error, the response is ignored. An error for which `errors.Is(err, ErrUntranslatableAIResponse)` is true becomes an `AIContractFailure` candidate; every other error, including an unrelated `context.Canceled` or `context.DeadlineExceeded`, becomes an `AIBoundaryFailure` candidate. If a joined error contains the untranslatable sentinel, the AI-contract category wins.
8. If the Assessor returned no error, `Assess` validates the response in this order: exactly one claim; supported Outcome; Outcome-specific proposal cardinality; proposed Plan validity; exact inequality from the current Plan. A violation produces an `AIContractFailure` candidate; otherwise validation produces an Assessment candidate.
9. Immediately before returning the candidate from step 7 or 8, `Assess` samples the supplied `ctx.Err()` a final time. If non-nil, that exact context error replaces an Assessor error, contract failure, or Assessment established after step 6. If nil, the candidate is returned. Cancellation after this final sample does not retroactively change the returned value.

The external AI call is an observable boundary interaction: it can transmit the Plan and observations, consume tokens, incur cost, emit telemetry, and create state governed by the concrete Provider contract. Plan Control's no-side-effect rule is narrower: it does not authorize or apply a Plan, perform a Task, decide Delivery Acceptance, or persist authoritative Arcloom state.

#### Example call site

The following is illustrative, uncompiled usage of the proposed contract.

```go
type DeliveryObservations struct {
	Schedule ScheduleFacts
	Progress ProgressFacts
	Quality  QualityFacts
}

assessment, err := plancontrol.Assess(
	ctx,
	current,
	observations,
	assessWithExternalAgent,
)
if err != nil {
	var failure *plancontrol.FailureError
	if errors.As(err, &failure) {
		return reportFailure(failure.Code())
	}
	return err
}

switch assessment.Outcome() {
case plancontrol.Complete:
	return reportComplete(assessment.AssessedPlan())
case plancontrol.Retain:
	return continueWith(assessment.AssessedPlan())
case plancontrol.Revise:
	proposed, ok := assessment.ProposedPlan()
	if !ok {
		return reportInvalidAssessment()
	}
	return presentRevision(assessment.AssessedPlan(), proposed)
case plancontrol.InsufficientInformation:
	return requestMoreInformation(assessment.AssessedPlan())
default:
	return reportUnsupportedAssessmentOutcome(assessment.Outcome())
}
```

#### Interface traceability

| Public contract | Requirement / invariant / external dependency | Owner | Consumer | Why separate |
|---|---|---|---|---|
| `Assess[O]` | PLC-1–8; one stateless Plan control decision | Plan Controller | Host or enclosing capability | Implements the Controller responsibility without requiring a stateful Go object or domain entity. |
| `Assessor[O]` | PLC-3–6; external AI authority and volatility | Plan Controller | Plan Control | Direct SDK dependency would leak Provider details. |
| Generic `O` | PLC-4; no fixed observation set | Caller and observation authorities | Plan Control and Assessor | `any` loses type agreement; fixed interfaces invent semantics. |
| `AssessorResponse` | PLC-5 invalid-response cases | Plan Control boundary | Assessor implementation and Plan Control | An already-valid Assessment would prevent Plan Control from enforcing response rules. |
| `Assessment` and `Outcome` | PLC-2–5; Assessment invariants | Plan Control Assessment | Caller | Flags, Provider DTOs, and generic results permit invalid or misleading states. |
| `FailureError` and `FailureCode` | PLC-1, PLC-5, PLC-6 | Plan Control Failure | Caller | Raw errors lose stable distinctions and leak Provider details. |
| Existing `plan.Plan` | PLC-1–5 and Plan requirements | Plan | Plan Control | A duplicate DTO would duplicate vocabulary and validation. |

#### Interface risks

- `O any` preserves type agreement but cannot enforce immutability. Until the Assessor returns, the caller must not mutate maps, slices, pointers, or other state reachable from `O` unless that state provides synchronization. `Assess` does not mutate or retain `O`, and the Assessor must never mutate reachable state. External transmission and retention remain explicit Provider-contract concerns.
- `AssessorResponse` intentionally admits invalid combinations. It remains a transient boundary artifact and never crosses a successful result boundary.
- A Provider adapter returns `ErrUntranslatableAIResponse` only when it cannot decode a received response far enough to identify its claims and proposal cardinality. Unknown decoded Outcome values, invalid zero Plan values, and invalid cardinalities are represented in `AssessorResponse` and rejected by `Assess` as AI-contract failures.
- `Assessor[O]` must not grow observation acquisition, application, Authorization, conversation management, model enumeration, or telemetry methods.
- `ProposedPlan() (plan.Plan, bool)` is an optional-value accessor derived from immutable Outcome, not a behavioral flag.
- Context errors unrelated to the supplied context are boundary failures rather than cancellation.

### 3.4 Database Design

N/A. This Change introduces no database, schema, persistence Port, durable business state, or authoritative state.

---

## 4. Design Decisions

### 4.1 Treat Plan Control as a Component responsibility, not a Concept

- Adopted: Plan, Plan Control Assessment, and Plan Control Failure own independent meaning and invariants. The Plan Controller establishes their contract.
- Rejected: A Plan Control aggregate or entity. Removing it loses no identity, lifecycle, authority, or state meaning and avoids modeling a procedure as a Concept.

### 4.2 Keep Plan-specific Assessment out of Reconciliation Core

- Adopted: `plancontrol` owns AI-authored Plan outcomes and has no dependency on `reconciliation.Result`.
- Rejected: Mapping complete, retain, revise, and insufficient information to satisfied, not-satisfied, and undecidable. The mapping loses AI authority, subject association, and the revised Plan.

### 4.3 Preserve caller-owned observation meaning with a generic parameter

- Adopted: `Assess[O]` and `Assessor[O]` pass one caller-selected observation type without interpreting it.
- Rejected: `any`, bytes, or JSON. They remove compile-time agreement or fix a transport representation.
- Rejected: A universal Observation interface or hierarchy. Current evidence provides no stable shared schema or invariant.

### 4.4 Use a stateless function instead of a Controller object

- Adopted: A package-level generic function receives its external AI Port per call.
- Rejected: A Controller that only stores an Assessor. No state, identity, lifecycle, or mutable invariant justifies the object.

### 4.5 Validate a passive response artifact before constructing Assessment

- Adopted: `AssessorResponse` intentionally represents zero, multiple, unknown, or contradictory claims, invalid zero Plan values, and proposal cardinalities. A response that cannot be decoded far enough to form this artifact is reported with `ErrUntranslatableAIResponse`. Only a validated immutable Assessment crosses the success boundary.
- Rejected: Returning an already-valid Assessment from the Assessor. It would move Plan-specific response validation into each Provider adapter.
- Rejected: Provider-native raw responses. They leak infrastructure vocabulary and cannot form a stable Port.

### 4.6 Keep revised-Plan comparison private

- Adopted: `plancontrol` compares exact preserved Plan element values through existing Plan accessors.
- Rejected: A public `Plan.Equal` contract. Plan Control is the only proven consumer, and no general equality semantics are accepted.

### 4.7 Keep Change, Authorization, application, and persistence absent

- Adopted: Assessment is an effect-free result. Later consumers may independently authorize or apply it.
- Rejected: Placeholder Ports or dependencies for future loop stages. They have no current consumer and would couple independent concepts to Plan Control.

---

## 5. Impact, Migration, and Rollback

- Impact: Updates `PRODUCT.md` and `ARCHITECTURE.md` to distinguish Plan Control from Plan Representation Reconciliation and generic Reconciliation Core semantics. A later implementation adds one provider-independent `plancontrol` Package.
- Migration: No existing public contract, external fact, Provider resource, or durable state is migrated. Existing Plan, Plan Representation, and GitHub Plan APIs remain unchanged.
- Rollback: Remove the new Package and revert the documentation changes. No stored or external state requires rollback.

---

## 6. Test Specification

### 6.1 Accepted-scenario traceability

Every accepted specification scenario has direct evidence. Rows marked static verify absence of a dependency or constraint that cannot be proven by one runtime example.

| Requirement / scenario | Test setup and stimulus | Expected evidence | Method |
|---|---|---|---|
| PLC-1 / Existing Plan is controlled | Supply a valid Plan, typed observations, and a Retain response | Assessor receives that exact Plan and observation value; Assessment refers to it | Public black-box test |
| PLC-1 / Plan does not exist | Supply the zero Plan | `InvalidCurrentPlan`, zero Assessment, zero Assessor calls | Public black-box test |
| PLC-2 / Goal and conditions achieved | Assessor returns Complete | Complete Assessment for the supplied Plan | Public black-box test |
| PLC-2 / All Tasks complete without outcome evidence | Observations say every Task is complete; Assessor returns InsufficientInformation | InsufficientInformation is preserved; no local completion inference | Public black-box test |
| PLC-2 / Obsolete Task remains incomplete | Observations contain one incomplete obsolete Task; Assessor returns Complete | Complete is preserved; no local Task gate | Public black-box test |
| PLC-3 / Plan remains suitable | Assessor returns Retain | Retain Assessment and no proposed Plan | Public black-box test |
| PLC-3 / Plan adjustment is required | Assessor returns Revise with one valid unequal Plan | Revise Assessment contains that Plan | Public black-box test |
| PLC-3 / AI cannot decide | Assessor returns InsufficientInformation | Valid Assessment, nil error, no proposed Plan | Public black-box test |
| PLC-3 / Another Module returns another result | Inspect exported contracts and dependencies | No common Module result or `reconciliation.Result` dependency constrains the other Module | Static API and architecture test |
| PLC-4 / Plan is delayed | Supply schedule and progress observations; return a valid revision | Exact typed observations reach Assessor; revision is preserved | Public black-box test |
| PLC-4 / Plan scope is insufficient | Supply Goal, acceptance-condition, and Task coverage observations; return a valid revision | Exact typed observations reach Assessor; revision is preserved | Public black-box test |
| PLC-4 / Task amount is unsuitable | Supply Task granularity and amount observations; return a valid revision | Exact typed observations reach Assessor; revision is preserved | Public black-box test |
| PLC-5 / Proposed Plan equals snapshot | Return Revise with a separately constructed exact-equivalent Plan | `AIContractFailure` and zero Assessment | Public black-box test |
| PLC-5 / Proposed Plan is invalid | Return Revise with an invalid zero Plan value | `AIContractFailure` and zero Assessment | Public black-box test |
| PLC-5 / Contradictory outcomes | Return multiple or duplicate claims, including Complete plus Revise | `AIContractFailure` and zero Assessment | Public black-box test |
| PLC-5 / Formally valid complete | Return Complete despite observations chosen to appear incomplete | Complete is preserved without semantic reinterpretation | Public black-box test |
| PLC-6 / AI is unavailable | Assessor returns unavailability, Provider-side timeout, and another Provider error in separate cases while supplied context is active | Each produces `AIBoundaryFailure`, zero Assessment, and no Provider-error unwrap | Public black-box test |
| PLC-6 / Request is cancelled | Cancel the supplied context before or during the Assessor call | Exact supplied context error and zero Assessment | Public black-box test |
| PLC-6 / AI reports insufficient information | Assessor successfully returns InsufficientInformation | Valid Assessment rather than Failure | Public black-box test |
| PLC-7 / Valid adjustment is proposed | Return one valid revision | Public API exposes only Assessment; no mutation or Authorization Port is invoked or imported | Public test plus static API/dependency review |
| PLC-8 / Runtime state is discarded | Perform a call, discard all returned runtime values, and repeat the same inputs and response | Equivalent observable Assessment is established again | Public black-box test |

### 6.2 Assessment-response matrix

| Claims | Proposed Plans | Expected result |
|---|---|---|
| `Complete` | none | Complete Assessment |
| `Retain` | none | Retain Assessment |
| `InsufficientInformation` | none | Insufficient-information Assessment |
| `Revise` | one valid unequal Plan | Revise Assessment with that Plan |
| none | any | `AIContractFailure` |
| one unknown Outcome value | any | `AIContractFailure` |
| duplicate or multiple claims | any | `AIContractFailure` |
| non-Revise claim | one or more | `AIContractFailure` |
| `Revise` | none or multiple | `AIContractFailure` |
| `Revise` | one invalid zero Plan | `AIContractFailure` |
| `Revise` | one exact-equivalent Plan | `AIContractFailure` |

### 6.3 Input, error, and cancellation precedence matrix

| Case | Given | Expected result and call count |
|---|---|---|
| Nil context | Nil context, any other inputs | `InvalidContext`; zero Assessor calls |
| Pre-cancelled valid input | Supplied context already cancelled | Exact `context.Canceled`; zero calls |
| Pre-expired valid input | Supplied deadline already exceeded | Exact `context.DeadlineExceeded`; zero calls |
| Pre-cancel plus invalid Plan | Cancelled context and zero Plan | Supplied context error wins; zero calls |
| Pre-cancel plus nil Assessor | Cancelled context and nil Assessor | Supplied context error wins; zero calls |
| Invalid Plan plus nil Assessor | Active context, zero Plan, nil Assessor | `InvalidCurrentPlan` wins; zero calls |
| Nil Assessor | Active context and valid Plan | `InvalidAssessor`; zero calls |
| Cancel while blocked, then valid response | Assessor blocks; supplied context is cancelled; Assessor returns a valid response | Exact supplied context error; response ignored; one call |
| Cancel while blocked, then Provider error | Assessor blocks; supplied context is cancelled; Assessor returns ordinary error | Exact supplied context error; Assessor error ignored; one call |
| Deadline while blocked | Deadline expires before Assessor returns | Exact `context.DeadlineExceeded`; one call |
| Matching context error | Assessor returns supplied `ctx.Err()` | Exact supplied context error after the post-call check |
| Unrelated context error | Supplied context remains active; Assessor returns `context.Canceled` or `context.DeadlineExceeded` | `AIBoundaryFailure` |
| Provider-side timeout | Supplied context remains active; Assessor returns its Provider timeout error | `AIBoundaryFailure` |
| Untranslatable response | Assessor returns wrapped `ErrUntranslatableAIResponse` | `AIContractFailure` |
| Joined untranslatable and Provider errors | Active context; Assessor returns a joined error containing the sentinel | `AIContractFailure` wins |
| Cancellation plus untranslatable response | Supplied context becomes cancelled; Assessor returns the sentinel | Exact supplied context error wins |
| Response plus error | Assessor returns both | Error classification wins; response ignored |
| Ordinary Provider failure | Active context; Assessor returns ordinary error | `AIBoundaryFailure`; Provider error does not unwrap |
| Cancellation after success | Cancel supplied context only after `Assess` returns a valid Assessment | Returned Assessment remains valid |

Every failure row asserts the public zero-Assessment observations: `Outcome() == 0`, `AssessedPlan().IsValid() == false`, and `ProposedPlan()` returns `false`. Typed failures use `errors.As` and exact `FailureCode`; supplied cancellation uses `errors.Is` against the supplied context error. Tests do not inspect private fields or assert private validation order when all paths yield the same public AI-contract category.

### 6.4 Plan inequality and immutable-result matrix

Exact Plan inequality preserves both value and collection order. Starting from one current Plan, construct one valid proposed Plan for each row.

| Difference | Expected result |
|---|---|
| Name value | Accepted revision |
| Goal text | Accepted revision |
| One acceptance-condition value | Accepted revision |
| Acceptance-condition order only | Accepted revision |
| One acceptance condition added | Accepted revision |
| One acceptance condition removed while at least one remains | Accepted revision |
| One Task value | Accepted revision |
| Task order only | Accepted revision |
| Task count from zero to one | Accepted revision |
| Task count from one to zero | Accepted revision |
| Task count from one to two | Accepted revision |
| Target-date value | Accepted revision |
| Target date absent to present | Accepted revision |
| Target date present to absent | Accepted revision |
| Separately constructed but exact-equivalent Plan | `AIContractFailure` |

Success tests assert only the public `Outcome`, `AssessedPlan`, and `ProposedPlan` accessors. After a successful return, mutating the Assessor's original response slices must not change the Assessment. Concurrent mutation before the Assessor returns violates the Assessor contract and is not a supported test condition.

### 6.5 Lifecycle, concurrency, and boundary verification

| Concern | Verification |
|---|---|
| Assessment/Failure exclusivity | Every success has nil error; every error has zero Assessment |
| Call isolation | Successive calls with correlated but different Plans, observations, and responses retain only their own values |
| Concurrent isolation | Correlated concurrent calls produce their own Assessments under `go test -race` |
| Stateless repetition | Repeating established inputs after discarding prior values establishes an equivalent result |
| No control-plane side effects | Exported API and imports contain no Plan application, Authorization, Change, persistence, Task execution, or Delivery Acceptance Port |
| No universal Module result | `plancontrol` has no `reconciliation` dependency and exposes no contract required by another Module |

The Reviewer owns both static checks. Required evidence is saved in the PR review: `go doc` output for the exported `plancontrol` API, `go list -deps` output showing no prohibited dependency, and source-import review confirming no application, Authorization, Change, persistence, Task execution, Delivery Acceptance, or universal Module-result contract.

### 6.6 Testability feedback

- Public behavior is testable with one `Assessor` function substitute; private helper tests and private call-order assertions are prohibited.
- Invalid claims, cardinalities, and zero Plan values remain testable through the passive response artifact without exposing them as successful Assessment state.
- Each future AI Agent Provider implementation owns Port conformance tests for payload translation, `ErrUntranslatableAIResponse` production, non-mutation of state reachable from observations, supplied-context propagation and return after cancellation, and Provider-error classification. Plan Control tests only the public mapping visible at its boundary.
- Go tests use the repository `go-test-authoring` workflow and begin from a `gotests` table scaffold.

---

## 7. Detailed Design and TDD Plan

| Implementation unit | Implements | Dependencies | Behavior / test | Migration or rollback impact |
|---|---|---|---|---|
| `plancontrol` Package | Plan Controller boundary | `plan`, standard library | Dependency and exported API review | Additive; removable |
| `Assess[O]` | Stateless capability contract | Context, Plan, Assessor, Assessment, Failure | Input, response, context, concurrency tests | None |
| `Assessor[O]` | External AI Port | Context, Plan, caller type, response artifact | Port-boundary contract tests | Provider implementations remain replaceable |
| `AssessorResponse` | Passive invalid-state-capable boundary artifact | Outcome, Plan | Public response matrix and post-return slice-mutation test | No persisted representation |
| `Assessment` and `Outcome` | Assessment Concept and invariants | Plan | Outcome, subject, revision, immutability tests | None |
| `FailureError` and `FailureCode` | Failure Concept and stable public meaning | Standard error conventions | `errors.As`, no unwrap, category tests | None |
| Exact Plan comparison inside `Assess` | Revision relationship invariant | Public Plan accessors | Public revision tests covering one-field changes, collection order, and exact equivalence | Must follow future Plan vocabulary changes |

| Implementation unit | Simplest viable representation | State / identity / lifecycle / boundary need | Rejected simpler alternative | Verdict |
|---|---|---|---|---|
| Plan Control operation | Generic package function | Stateless; no identity or lifecycle | Controller storing one Assessor | Function |
| External AI boundary | Named generic function type | Real external dependency and substitution need | Direct Provider SDK call | Port accepted |
| Observation input | Generic parameter | Caller-owned type agreement | Untyped value or universal interface | Generic parameter |
| Assessment | Immutable value with private state | Owns exclusivity and Plan relationship invariants | Tuple, flags, or generic Result | Value |
| Assessor response | Passive struct | Must represent invalid boundary states for validation | Already-valid Assessment | Boundary artifact |
| Failure | Typed error and code | Stable public distinction | Raw or string-matched error | Typed error |

Construction follows one Red-Green-Refactor cycle per behavior in sections 6.2 through 6.5. No implementation starts before independent architecture, interface, and test reviews pass and human approval is recorded.
