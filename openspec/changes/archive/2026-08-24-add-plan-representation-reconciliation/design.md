# Plan Representation Reconciliation

## 0. Document Scope

| Information | Normative document |
|---|---|
| Product concept, scope, capabilities, and principles | `PRODUCT.md` |
| Component responsibilities, external authority, dependency rules, and Port ownership | `ARCHITECTURE.md` |
| Accepted Plan behavior | `openspec/specs/plan/spec.md` |
| Proposed observable behavior | `specs/plan-representation-reconciliation/spec.md` and `specs/plan/spec.md` in this Change |
| Structure, contracts, and verification design for this Change | This DesignDoc |
| Implementation workflow and review gates | `docs/DEVELOPMENT.md` |

## 1. Purpose / Out of Scope

### Purpose

The Plan Representation Controller determines whether one valid expected Plan corresponds to one current provider-independent observation of its external representation. It produces immutable evidence and a determination without acquiring authority over external facts.

### Out of Scope

- A GitHub, Linear, or other Planning Provider implementation
- Target discovery or target identity in the Controller contract
- Plan sufficiency evaluation
- External mutation, Change construction, Change Authorization, or retry orchestration
- Persistence, reconciliation history, authoritative mirrors, or caches required for correctness
- Provider request or response DTOs

## 2. Behavior Design

### 2.1 Functional Requirements

#### Target-bound observation

The Controller receives current facts through a consumer-owned Port whose implementation is configured for one external target.

| ID | Rule | Source |
|---|---|---|
| FR-1 | Controller construction rejects a missing Observer; a call on a zero Controller returns the same stable failure. | PRR-1, PRR-7 |
| FR-2 | A reconciliation call accepts one valid expected Plan and requests one new logical Observation. | PRR-1, PRR-2, PRR-8 |
| FR-3 | The invoking Host supplies an expected Plan and Observer derived from the same Change target; Provider target identity and configuration never enter the Port, Plan, Observation, evidence, or Result. | PRR-1, PRR-2 |

#### Provider-independent observation

Observation construction represents knowledge about Plan locations rather than Provider resources or errors.

| ID | Rule | Source |
|---|---|---|
| FR-4 | Root, scalar, target-date, and collection states admit only the combinations defined by PRR-2. | PRR-2 |
| FR-5 | Observation builders reuse Plan-owned scalar and collection-validity contracts, then represent their violation codes and collapse duplicate collection members to one valid member plus one violation. | PRR-2, PLN-3, PLN-5 |
| FR-6 | A zero or contradictory Observation crossing the Port is an Observer-contract failure, not reconciliation evidence. | PRR-2, PRR-7 |

#### Semantic correspondence and evidence

The Controller owns Plan-representation correspondence, location, covering, aggregation, and ordering rules.

| ID | Rule | Source |
|---|---|---|
| FR-7 | Known values use exact preserved comparison; complete collections use order-independent exact membership. | PRR-3 |
| FR-8 | Known mismatches become typed Differences with location-specific payloads. | PRR-4 |
| FR-9 | Unavailable facts remain separate from Differences and suppress only evidence they cover. | PRR-5 |
| FR-10 | Difference and unavailable collections are unique, immutable, and canonically ordered. | PRR-4, PRR-5, PRR-6 |

#### Evidence-derived result

The target-independent Reconciliation contract derives the determination from evidence cardinality; the Controller owns Plan-specific evidence.

| ID | Rule | Source |
|---|---|---|
| FR-11 | Empty evidence means `Satisfied`, Difference-only evidence means `NotSatisfied`, and any unavailable information means `Undecidable`. | PRR-6 |
| FR-12 | A Result cannot be constructed or mutated into a state inconsistent with its evidence. | PRR-6 |

#### Failure and lifecycle semantics

Call failures remain distinct from evidence about the external representation.

| ID | Rule | Source |
|---|---|---|
| FR-13 | Caller cancellation and deadlines take precedence over all call-time validation and Observer-contract failures before return. | PRR-7 |
| FR-14 | Invalid expected Plan and Observer-contract failures return stable Controller failure categories and no valid Result. | PRR-1, PRR-7 |
| FR-15 | Reconciliation is read-only, non-persistent, independent of prior runtime state, and safe for concurrent calls through the same Controller and Observer. | PRR-8 |

### 2.2 Non-functional Requirements

N/A. This Change introduces no measurable performance, capacity, availability, or security threshold. Cancellation, immutability, and statelessness are functional contract rules in section 2.1.

## 3. Structure Design

### 3.1 Conceptual Model

| Concept | Meaning | Identity | Business rule / invariant |
|---|---|---|---|
| Plan | Existing provider-independent planning intent in the expected role | Value equality is not defined; exact element values provide correspondence keys | Exactly one valid Plan supplies expected meaning. Plan owns element validity and preserved text. |
| External Plan Representation Target | One Provider-native target whose durable facts represent a Plan | Owned and interpreted only by the Planning Context | One Observer implementation remains bound to one target for its lifetime. Its identity never crosses the Port. |
| Plan Representation Observation | Immutable Disposable Projection of current external facts into Plan vocabulary | No durable identity | It represents exactly one logical observation and admits only the state combinations in PRR-2. It is never authoritative. |
| Plan Location | Provider-independent semantic position addressed by evidence | Location kind plus exact member text when the kind is a member | It is one of the closed locations in PRR-4. Covering locations suppress descendant evidence. |
| Semantic Correspondence | Rule relating expected Plan meaning to known observed meaning | None | It compares exact preserved values, ignores collection order, and never infers unavailable facts. |
| Known Difference | One mismatch established from known meaning | Category, location, and category-specific payload | Its category and payload agree; it contains no Provider identity or error. |
| Unavailable Information | One Plan location whose required fact cannot be established | Plan location | It is separate from Difference and never implies absence or equality. |
| Reconciliation Result | Immutable aggregate of complete evidence and its derived determination | No durable identity | Evidence alone determines `Satisfied`, `NotSatisfied`, or `Undecidable`; callers cannot select the determination. |

#### Concept relationships

| Source | Relationship | Target | Multiplicity and lifecycle |
|---|---|---|---|
| Planning Context | Authoritatively owns | External Plan Representation Target | One Context owns zero or more targets and their native lifecycle. |
| Observer implementation | Is configured for | External Plan Representation Target | The invoking Host selects the target from the applicable Change; the implementation validates and keeps exactly one immutable binding. |
| Plan Representation Observation | Projects facts of | External Plan Representation Target | Exactly one logical Observation per call; the Projection is disposable. |
| Semantic Correspondence | Relates | Plan and Plan Representation Observation | Exactly one valid expected Plan and one valid Observation. |
| Known Difference | Identifies | Plan Location | Exactly one location per Difference. |
| Unavailable Information | Identifies | Plan Location | Exactly one allowed root, scalar, or collection location. |
| Reconciliation Result | Contains | Known Difference and Unavailable Information | Zero or more unique items after covering and canonical ordering. |

#### Concept minimality

| Candidate | Decision | Reason |
|---|---|---|
| Expected Plan | Keep as a role of Plan | It adds no state or invariant beyond the existing Plan. |
| Reconciliation Attempt | Do not introduce | The call has no identity, durable state, or independent lifecycle. Call semantics belong to the Controller contract. |
| Determination | Keep as Result state | It is derived and cannot have an independent lifecycle or setter. |
| Evidence Set | Keep as Result state | Result owns completeness, immutability, and consistency with determination. |
| Observation Availability and Collection Completeness | Keep as Observation state | They are axes of knowledge, not independent state owners. |
| Provider error, resource, identifier, page, retry, or snapshot | Exclude | These belong to Provider integration and do not carry provider-independent reconciliation meaning. |
| Repository, cache, or history | Exclude | Correctness requires reconstruction from current external facts and no durable Arcloom state. |

### 3.2 Responsibility and Package Design

#### Responsibility assignment

| Responsibility / decision | Owner | Information and authority used | State / invariant affected | Change driver | Not owner / reason |
|---|---|---|---|---|---|
| Plan element validity, Plan-name validation, and collection uniqueness | Plan | Plan vocabulary and invariants | Valid Plan values and violation codes | Plan meaning changes | Provider and Controller consume Plan-owned validation results and do not duplicate Plan rules. |
| External target existence, native meaning, permissions, and lifecycle | Planning Context | Provider-native authoritative facts | Durable external facts | External system changes | Arcloom Packages are not authoritative stores. |
| Meaning and validity of the expected Plan / external target association | Change | One proposed Change and its Change target | The relationship is valid for that Change | Change semantics change | Host consumes the relationship; Provider does not know the expected Plan. |
| Supply of the associated expected Plan and target-bound Observer | Invoking Host | The association owned by Change | The Controller receives values concerning the same Change target | Host integration changes | Composition Root only wires Components; Host does not redefine association validity. |
| Immutable target binding, reads, pages, retries, and contradiction detection | Observer implementation in a Planning Provider Module | Host-supplied Provider configuration and native facts | One logical Observation of one bound target | Provider contract changes | Controller has no Provider identity or native consistency facts. |
| Observation state algebra, Plan-location covering, and boundary validity | Observation values and constructors owned by Plan Representation Controller | Plan vocabulary, Plan validation results, and affected Plan locations | Valid provider-independent Observation | Observation contract changes | Provider detects affected facts but does not redefine least-covering or validity rules. |
| Exact correspondence and Plan-specific evidence derivation | Semantic Correspondence rule owned by Plan Representation Controller | Expected Plan and valid Observation | Difference category, location, and payload | Plan-representation rules change | Controller coordination and Reconciliation Core do not own Plan comparison policy. |
| Evidence covering, aggregation, ordering, element validity, element immutability, and determination input | Plan Representation Result | Complete Plan-specific evidence | Result evidence meaning and valid immutable elements | Plan-representation result rules change | Caller and Controller coordination do not sort, deduplicate, or select evidence. |
| Target-independent Result collection snapshot and determination derivation | Reconciliation Core Result | Immutable target-specific Difference and unavailable values supplied together | Defensive collection copies and three-state evidence-to-determination truth table | Common reconciliation meaning changes | Core does not claim element immutability and does not inspect Plan locations, payloads, or violation vocabulary. |
| Observation invocation, call validation, context precedence, and read-only coordination | Plan Representation Controller | Context, expected Plan validity, and Observer contract | No observation on invalid input; no result on failure; isolated concurrent calls | Controller call contract changes | Semantic Correspondence and Result values retain their own rules; no Attempt concept is needed. |

#### Package responsibilities

| Package | Responsibility | Provides | Does not provide |
|---|---|---|---|
| `plan` | Implements Plan concepts and their invariants | Existing Plan values plus reusable Plan-name and collection validation | Observation, correspondence, evidence, or Provider adaptation |
| `reconciliation` | Implements target-independent Result meaning | Generic Result with immutable collection snapshots and evidence-to-determination derivation | Evidence element validity, Plan locations, Plan-specific observation, or orchestration |
| `planrepresentation` | Implements the Plan Representation Controller Component and its owned Port | Observation vocabulary/builders, Observer Port, Controller, Difference, Unavailable Information, and Result | Provider implementation, external identity, mutation, authorization, persistence, or Plan sufficiency |

#### Package boundaries

| Boundary (package) | Hidden detail | Dependency direction |
|---|---|---|
| `plan` | Text and date validation implementation | Depends only on the Go standard library. |
| `reconciliation` | Immutable generic Result representation and determination derivation | Depends only on the Go standard library. |
| `planrepresentation` | Observation representation, correspondence, covering, aggregation, canonical ordering, and Result construction | Depends inward on `plan` and `reconciliation`, plus the Go standard library. It does not depend on `githubplanning` or another Provider Package. |

#### SOLID risk assessment

| Principle | Risk | Mitigation |
|---|---|---|
| SRP | Controller could accumulate Provider reads or generic reconciliation policy. | Keep Provider coherence in Observer implementations and target-independent determination in `reconciliation`. |
| OCP | A generic Provider or generic Controller abstraction could be added before another real consumer exists. | Publish only the Plan Representation Observer Port and concrete Plan-specific contracts. |
| LSP | Observer implementations could return Provider errors or impossible Observation states. | Define one strict consumer-owned contract and convert violations to one stable Controller failure. |
| ISP | A Planning Provider interface could combine read, preview, and mutation operations. | The Observer Port contains only the read operation required by this Controller. |
| DIP | Provider DTOs and identifiers could leak into evidence. | Observation builders and evidence expose only Plan vocabulary and stable Plan violation codes. |

#### Independent evolution scenario impact

| Scenario / confidence | Primary decision owner | Expected propagation | Unexplained impact | Duplicated policy decision | Verdict |
|---|---|---|---|---|---|
| Add a Plan element / evidence-backed | Plan and Plan Representation Controller; Core only if common determination changes | Plan contract, Observation/evidence vocabulary, Controller tests, Provider adapters | None | None | Pass |
| Add a Planning Provider / evidence-backed | New Planning Provider Module | New Observer implementation and composition only | None | None | Pass |
| Provider not-found and access semantics differ / evidence-backed | Provider adapter for fact interpretation; Controller for availability meaning | Adapter and its contract tests | None | None | Pass |
| Multi-page collection changes / evidence-backed | Provider adapter for coherence; Controller for incomplete-collection evidence | Adapter plus Controller contract tests | None | None | Pass |
| Provider SDK changes / evidence-backed | Provider Module | Provider implementation only | None | None | Pass |
| Runtime becomes concurrent and long-lived / evidence-backed | Controller contract, Host composition, and Observer implementation | Concurrency-safe implementation and race tests | None | None | Pass; the Port requires isolated concurrent calls and no mutable target or prior Result is held. |
| Persistence is requested / speculative | Product and Architecture constraints must change first | Not an extension of this design | None | None | Risk recorded; no Port added. |

#### Procedural risk

- Rules at risk of being placed in a Host or Controller method: semantic correspondence, evidence ordering, coverage, and determination derivation.
- Behavior that remains with owned state: Observation values protect their state algebra; Semantic Correspondence derives Plan evidence; Plan Representation Result protects Plan-specific evidence; Core Result protects common immutability and determination.
- Premature abstractions avoided: generic Controller interface, Provider repository, reconciliation workflow, strategy, registry, and persistence Port.

### 3.3 Interface Design

#### Plan-owned reusable validation

```go
package plan

// ValidateName applies the same Plan-name rules and typed ValidationError
// contract used by New. It preserves Plan as the sole owner of these rules.
func ValidateName(name string) error

// ValidateAcceptanceConditions and ValidateTasks accept publicly constructible
// element values and their zero values. They apply the same zero-element and
// exact-preserved-text duplicate rules used by New, including typed errors and
// input indexes. Empty input is valid because aggregate completeness remains a
// separate Plan invariant. No precedence is promised for simultaneous
// independent violations.
func ValidateAcceptanceConditions(values []AcceptanceCondition) error
func ValidateTasks(values []Task) error
```

#### Target-independent Result

```go
package reconciliation

type Determination uint8

const (
	Satisfied Determination = iota + 1
	NotSatisfied
	Undecidable
)

// Result owns snapshot copies of one target-specific complete evidence
// collection. D and U must be immutable valid values owned by the caller's
// target-specific Result; Core does not make mutable element graphs immutable.
// Determination is derived internally: no evidence is Satisfied,
// Difference-only evidence is NotSatisfied, and any U is Undecidable.
// The zero Result is invalid and panic-free: Determination returns zero and
// both evidence accessors return nil. NewResult(nil, nil) is a distinct valid
// Satisfied Result.
type Result[D any, U any] struct { /* immutable collection snapshot */ }

func NewResult[D any, U any](differences []D, unavailable []U) Result[D, U]
func (r Result[D, U]) Determination() Determination
func (r Result[D, U]) Differences() []D
func (r Result[D, U]) UnavailableInformation() []U
```

#### Consumer-owned observation Port

```go
package planrepresentation

// Observer is a consumer-owned function Port. A non-nil value is configured
// by the Host for one immutable target binding. It is safe for concurrent calls.
// Each call isolates its reads and returns one logical Observation.
//
// If ctx is cancelled or expires, Observer returns ctx.Err(). Any Observation
// returned with an error is ignored. A non-context error, a context error that
// does not equal ctx.Err(), or an invalid Observation violates this Port.
// Provider-native errors are converted to availability before return.
type Observer func(ctx context.Context) (Observation, error)
```

#### Observation construction

```go
package planrepresentation

// ObservationValidationError identifies an impossible Observation state.
// Inspect it with errors.As. It never wraps a Provider error.
type ObservationValidationError struct { /* immutable */ }
func (e *ObservationValidationError) Error() string

// Root observations carry no target identity. Absent and unavailable roots
// cannot carry descendant state.
func AbsentObservation() Observation
func UnavailableObservation() Observation

// NewPresentObservation rejects zero or incompatible child observations.
func NewPresentObservation(
	name PlanNameObservation,
	goal GoalObservation,
	conditions AcceptanceConditionsObservation,
	tasks TasksObservation,
	targetDate TargetDateObservation,
) (Observation, error)

// LeastUnavailableLocation owns the Plan-location covering rule. An Observer
// supplies affected Plan locations discovered from Provider facts; it does not
// reimplement root/scalar/collection covering. Empty or invalid input returns
// ObservationValidationError. A member maps to its collection, one scalar or
// collection remains there, and locations spanning top-level branches map to
// root. The Observer applies the returned location through the corresponding
// root, scalar, or collection unavailable constructor.
func LeastUnavailableLocation(affected ...Location) (Location, error)
```

```go
package planrepresentation

// Classifiers classify raw values by Plan-owned validity rules.
// Invalid raw values become stable violations and are not retained as values.
func ClassifyPlanName(value string) PlanNameObservation
func UnavailablePlanName() PlanNameObservation
func ClassifyGoal(value string) GoalObservation
func UnavailableGoal() GoalObservation
func ClassifyTargetDate(value string) TargetDateObservation
func AbsentTargetDate() TargetDateObservation
func UnavailableTargetDate() TargetDateObservation

// Complete and incomplete constructors preserve the first exact valid member,
// use Plan-owned collection validation, aggregate violations by Plan violation
// code, and never normalize text. Nil and empty input are equivalent. Inputs
// are consumed during construction and are never retained by reference.
func CompleteAcceptanceConditions(values []string) AcceptanceConditionsObservation
func IncompleteAcceptanceConditions(values []string) AcceptanceConditionsObservation
func CompleteTasks(values []string) TasksObservation
func IncompleteTasks(values []string) TasksObservation
```

#### Controller and failures

```go
package planrepresentation

type FailureCode string

const (
	InvalidObserver       FailureCode = "invalid_observer"
	InvalidContext        FailureCode = "invalid_context"
	InvalidExpectedPlan   FailureCode = "invalid_expected_plan"
	ObservationContract  FailureCode = "observation_contract"
)

// FailureError is inspected with errors.As. It never wraps Provider errors.
type FailureError struct { /* immutable */ }
func (e *FailureError) Error() string
func (e *FailureError) Code() FailureCode

// Controller is immutable after construction and safe for concurrent use.
// Its Observer must satisfy the same concurrent-use contract. A zero Controller
// is unusable but panic-free: with non-nil ctx, Reconcile returns InvalidObserver.
type Controller struct { /* non-nil bound Observer only */ }

// NewController fails with InvalidObserver when observer is missing. Provider
// construction validates only structural configuration and binding form;
// target existence, access, and lifecycle remain Observe outcomes.
func NewController(observer Observer) (*Controller, error)

// Reconcile returns either one valid immutable Result or an error, never both.
// Nil context returns InvalidContext before every other call-time failure. For
// a non-nil context, ctx.Err() observed before return is returned directly and
// is inspected with errors.Is; it precedes InvalidObserver, InvalidExpectedPlan,
// and Port violations. InvalidObserver precedes InvalidExpectedPlan. An
// Observer-returned context error is valid only when equal to ctx.Err(); another
// context error is a Port violation. Error returns always include the zero
// Result, and a nil error always includes a valid Result.
func (c *Controller) Reconcile(ctx context.Context, expected plan.Plan) (Result, error)
```

#### Result and evidence

```go
package planrepresentation

type DifferenceCategory uint8
const (
	ExpectedAbsentCategory DifferenceCategory = iota + 1
	UnexpectedPresentCategory
	ValueDifferentCategory
	InvalidObservedCategory
)

type LocationKind uint8
const (
	PlanRootLocation LocationKind = iota + 1
	PlanNameLocation
	GoalLocation
	AcceptanceConditionCollectionLocation
	AcceptanceConditionMemberLocation
	TaskCollectionLocation
	TaskMemberLocation
	TargetDateLocation
)

type Location struct { /* immutable */ }
// The zero Location is invalid; Kind returns zero and Member returns false.
func (l Location) Kind() LocationKind
func (l Location) Member() (string, bool)

// Location constructors are also used by Observer implementations to report
// affected Plan meaning. Member constructors validate exact Plan text.
func PlanRoot() Location
func PlanName() Location
func Goal() Location
func AcceptanceConditions() Location
// Invalid member text returns *plan.ValidationError directly and is inspected
// with errors.As; no Provider error is wrapped.
func AcceptanceCondition(statement string) (Location, error)
func Tasks() Location
func Task(name string) (Location, error)
func TargetDate() Location

// Meaning is a closed immutable sum. Implementations are unexported; consumers
// type-switch on the exported variant interfaces.
type Meaning interface { isMeaning() }
type PlanMeaning interface {
	Meaning
	isPlanMeaning()
	Plan() plan.Plan
}
type TextMeaning interface {
	Meaning
	isTextMeaning()
	Text() string
}
type TargetDateMeaning interface {
	Meaning
	isTargetDateMeaning()
	TargetDate() plan.TargetDate
}

// Difference is a closed immutable sum. Each concrete type exposes only the
// payload allowed by its category.
type Difference interface {
	Category() DifferenceCategory
	Location() Location
	isDifference()
}

type ExpectedAbsentDifference interface {
	Difference
	isExpectedAbsentDifference()
	Expected() Meaning
}
type UnexpectedPresentDifference interface {
	Difference
	isUnexpectedPresentDifference()
	Observed() Meaning
}
type ValueDifferentDifference interface {
	Difference
	isValueDifferentDifference()
	Expected() Meaning
	Observed() Meaning
}
type InvalidObservedDifference interface {
	Difference
	isInvalidObservedDifference()
	Violation() plan.ViolationCode
}

type UnavailableInformation interface {
	Location() Location
	isUnavailableInformation()
}

// Result wraps the Core Result and is constructible only inside this Package.
// It supplies only non-nil closed immutable evidence values. Accessors return
// independent slice copies on every call. The zero Result is the error return:
// Determination returns zero and evidence accessors return nil without panic.
type Result struct { /* reconciliation.Result[Difference, UnavailableInformation] */ }
func (r Result) Determination() reconciliation.Determination
func (r Result) Differences() []Difference
func (r Result) UnavailableInformation() []UnavailableInformation
```

#### Example call site

```go
// The Host derives both values from the same applicable proposed Change.
expectedPlan := planFromChange(change)
observer, err := newGitHubPlanObserver(client, change.PlanTarget())
if err != nil { // invalid Provider configuration or target binding
	return err
}
controller, err := planrepresentation.NewController(observer)
if err != nil {
	return err
}

result, err := controller.Reconcile(ctx, expectedPlan)
if err != nil {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var failure *planrepresentation.FailureError
	if errors.As(err, &failure) {
		return reportReconciliationFailure(failure.Code())
	}
	return err
}
switch result.Determination() {
case reconciliation.Satisfied:
	// The complete current observation corresponds to the expected Plan.
case reconciliation.NotSatisfied:
	for _, difference := range result.Differences() {
		switch difference := difference.(type) {
		case planrepresentation.ExpectedAbsentDifference:
			reportExpectedAbsent(difference.Location(), difference.Expected())
		case planrepresentation.UnexpectedPresentDifference:
			reportUnexpectedPresent(difference.Location(), difference.Observed())
		case planrepresentation.ValueDifferentDifference:
			reportDifferent(difference.Location(), difference.Expected(), difference.Observed())
		case planrepresentation.InvalidObservedDifference:
			reportInvalid(difference.Location(), difference.Violation())
		}
	}
case reconciliation.Undecidable:
	// Known Differences remain available alongside unavailable locations.
	reportPartialEvidence(result.Differences(), result.UnavailableInformation())
}

func reportMeaning(meaning planrepresentation.Meaning) {
	switch meaning := meaning.(type) {
	case planrepresentation.PlanMeaning:
		reportPlan(meaning.Plan())
	case planrepresentation.TextMeaning:
		reportText(meaning.Text())
	case planrepresentation.TargetDateMeaning:
		reportTargetDate(meaning.TargetDate())
	default:
		panic("unreachable: planrepresentation.Meaning is a closed sum")
	}
}
```

#### Boundary decisions

| Boundary | Hidden detail | Reason |
|---|---|---|
| `Observer` | Provider target identity, SDK DTOs, reads, pagination, retries, and error interpretation | The Plan Representation Controller is the real consumer; external authority and Provider volatility must not alter its vocabulary. |
| Observation constructors | State encoding, use of Plan validation results, least-location covering, violation classification, and immutable copies | Provider adapters report native facts and affected Plan locations but do not reproduce the consumer contract. |
| Semantic Correspondence and Plan Representation Result | Exact comparison, evidence construction, covering, aggregation, and canonical ordering | These are Plan Representation policy and remain independent of call coordination. |
| `Controller` | Observer invocation, call validation, context precedence, and concurrent call isolation | The Host receives one stable protocol without acquiring Plan comparison policy. |
| `reconciliation.Result` | Defensive evidence-collection snapshots and target-independent three-state derivation | Every target-specific Reconciliation Module uses the same common collection and determination invariant without importing Plan types. Evidence element validity and immutability remain target-specific preconditions. |

#### Interface traceability

| Public contract | Requirement / constraint | Owner | Consumer | Why separate |
|---|---|---|---|---|
| `plan.ValidateName` | PLN-2, PLN-5; no duplicated Plan rule | Plan | Observation constructors | Plan-name validity is owned by Plan but required without aggregate construction. |
| Plan collection validators | PLN-3; no duplicated Plan rule | Plan | Observation constructors | Element and duplicate meaning are owned by Plan but must be applied incrementally to an imperfect Observation. |
| `reconciliation.Result` | PRR-6; Architecture Reconciliation Core | Reconciliation Core | Plan Representation Result | Common collection snapshots and determination must remain target-independent; Plan Result supplies immutable valid elements. |
| `Observer` | PRR-1, PRR-2, PRR-7; external authority | Plan Representation Controller | Controller | Isolates one external dependency and its consistency model. |
| Observation constructors | PRR-2; immutable valid Observation | Plan Representation Controller | Observer implementations | Prevents Provider-specific or impossible state from becoming the normal contract. |
| `NewController` / `Reconcile` | PRR-1, PRR-7, PRR-8 | Plan Representation Controller | Invoking Host | Protects Port presence, failure precedence, read-only, stateless, and concurrent call semantics. The Host retains the expected-Plan/target association precondition. |
| Difference variants and Plan Meaning variants | PRR-4 | Plan Representation Controller | Result consumer | Closed sums expose only category-valid payloads without infrastructure DTOs. |
| Result and evidence accessors | PRR-4, PRR-5, PRR-6 | Plan Representation Result over Reconciliation Core Result | Result consumer | Exposes Plan meaning while preserving Core-owned copies and derived determination. |

#### Interface risks

- Oversized interfaces: `Observer` is one consumer-required function contract.
- Primitive obsession: target identity is excluded; raw text is accepted only at Observation-construction boundaries and immediately classified into Plan vocabulary.
- Infrastructure leakage: no Provider identifier, DTO, resource type, or error appears in a signature.
- Boolean flag risks: complete and incomplete collections use distinct named constructors instead of a mode flag.

### 3.4 Database Design

N/A. This Change persists neither Observation nor Result and introduces no database or persistence Port.

## 4. Design Decisions

### 4.1 Keep Plan-specific evidence out of Reconciliation Core

- Adopted: Generic `reconciliation.Result[D,U]` owns immutable evidence copies and derives the target-independent determination. `planrepresentation` validates and supplies the complete Plan-specific evidence types.
- Rejected: A generic evidence hierarchy in Core. Core stores opaque type parameters and does not interpret Plan locations or violation categories.
- Rejected: A public function accepting evidence counts. It cannot protect correspondence between the actual evidence and determination.
- Rejected: Determination logic only in `planrepresentation`. The same three-state meaning would be duplicated by later Controllers.

### 4.2 Build valid Observation states instead of exporting DTO fields

- Adopted: Named constructors create immutable state variants and classify raw Provider-independent values through Plan-owned rules.
- Rejected: An exported struct with status flags and optional fields. It admits contradictory root, scalar, and collection states.
- Rejected: A Provider-native read model. It violates Port ownership and couples correspondence to one Provider.

### 4.3 Bind the external target outside the Port vocabulary

- Adopted: Change owns the meaning and validity of its target relationship. The invoking Host uses that relationship to supply the expected Plan and select the same external target. It constructs an Observer function whose Provider implementation validates and retains one immutable target binding. The Port receives no target identifier.
- Rejected: A target identifier parameter on `Observe`. It leaks Provider identity and permits one collaborator to mix targets.
- Rejected: Target identity in Observation or Result. Reconciliation evidence concerns Plan meaning, not external addressing.

### 4.4 Keep one cohesive Plan Representation Package

- Adopted: Observation validity, correspondence, evidence, and Result construction remain in `planrepresentation` because they change with the same Plan-representation rules and protect one Component boundary.
- Rejected: Packages named observer, comparator, evidence, sorter, and result. They map a procedure into boundaries and expose internal sequencing.
- Rejected: Place the Controller in `plan`. Plan does not own external representation correspondence.

### 4.5 Expose reused Plan validation from its owner

- Adopted: `plan.ValidateName` and standalone collection validators expose existing element and uniqueness rules and typed errors without constructing a Plan aggregate. Observation construction invokes the validators incrementally so duplicate classification does not redefine exact duplicate meaning.
- Rejected: Duplicate text checks in `planrepresentation` or each Provider. Rule changes would propagate inconsistently.
- Rejected: Construct a placeholder Plan only to validate its name. It couples name validation to unrelated Goal and acceptance-condition requirements.

### 4.6 Treat missing Observer as construction failure

- Adopted: `Observer` is a named function Port, so `NewController` can reject a nil Port without reflection or typed-nil interface ambiguity before a caller context or expected Plan exists.
- Rejected: A nil check during every call. It creates an artificial precedence conflict and permits an unusable Controller value.
- Rejected: A one-method Go interface. It adds typed-nil ambiguity without adding a second behavior or stateful substitution contract.

### 4.7 Require concurrent-use safety at the Port

- Adopted: Controller is immutable after construction and Observer implementations isolate per-call reads and support concurrent calls.
- Rejected: Caller-side serialization. It would make a long-lived Host coordinate a Provider-specific implementation constraint and weaken substitutability.

## 5. Impact, Migration, and Rollback

- Impact: Adds `reconciliation` and `planrepresentation`; adds backward-compatible validation functions to `plan`. Existing Plan and GitHub dry-run callers require no changes.
- Migration: No external facts or durable Arcloom state are migrated. Provider implementations can be added independently in later Changes.
- Rollback: Removing the new Packages and Plan validation functions restores the previous code surface. No external or stored state requires rollback.

## 6. Test Specification

### 6.1 Requirement Coverage

| Requirement / criterion | Observable behavior or verification | Method | Verification owner | Required evidence |
|---|---|---|---|---|
| PLN-3 | Standalone collection validation and Plan construction return equivalent typed zero-element/duplicate results; aggregate acceptance-condition completeness remains Plan-only | Automated behavior tests | `plan` | Literal `ViolationCode`, `ElementKind`, and index for shared cases; explicit divergent nil/empty acceptance-condition result |
| PLN-5 | Standalone name validation and Plan construction return equivalent typed name results | Automated behavior tests | `plan` | Literal valid and invalid boundary results through `errors.As` |
| PRR-1 input and binding | Constructor/zero Controller failures are automated; same-Change association is an external precondition | Behavior tests plus Host design/API review | `planrepresentation` for tests; future Host owner for association | Error priority and Observer call count; Host review cites Change-owned association |
| PRR-1 target identity exclusion | No Provider target type, identifier, or field crosses exported contracts | Exported API review using `go doc` | `planrepresentation` reviewer | Saved exported API output in PR evidence |
| PRR-2 state algebra | Every allowed state and representative forbidden combination is accepted or rejected through public constructors | Automated black-box constructor tests | `planrepresentation` | Tables in section 6.2 and literal typed errors |
| PRR-2 coherence localization | Least covering Plan location is deterministic | Automated public helper tests | `planrepresentation` | Table in section 6.3 |
| PRR-2 Provider-native projection | A native representation projects only Plan vocabulary without Provider identifiers, DTOs, or errors | Deferred contract tests for each concrete Provider, which is outside this Change | Future Planning Provider owner | Milestone, Issue, Project, or applicable native fixture mapped to literal Observation values |
| PRR-3 correspondence | Exact scalar equality and order-independent exact collection membership | Automated behavior tests | `planrepresentation` | Literal Differences for case, whitespace, line ending, Unicode, order, and rename cases |
| PRR-4 Difference evidence | Category payload, uniqueness, covering, aggregation, and full canonical order | Automated behavior tests | `planrepresentation` | Exact exported Difference sequence and typed payloads |
| PRR-5 unavailable evidence | Location, covering, partial preservation, and full canonical order | Automated behavior tests | `planrepresentation` | Exact exported unavailable and Difference sequences |
| PRR-6 Result | Evidence collections are snapshotted and determination is evidence-derived | Automated Core and Plan Result tests | `reconciliation` and `planrepresentation` | All evidence-presence combinations and mutation checks |
| PRR-7 Controller outcomes | Input/error priority, context lifecycle, invalid Observation, and Provider-detail exclusion | Automated Port-boundary tests | `planrepresentation` | `errors.Is`, `errors.As`, zero Result, and Observer call count |
| PRR-7 Provider translations | Access, rate-limit, service, not-found, and detected-concurrency meanings | Deferred contract tests for each concrete Provider, which is outside this Change | Future Planning Provider owner | Adapter tests showing unavailable versus authoritative absence without native errors crossing |
| PRR-8 read-only surface | Controller has no mutation or authorization dependency | Exported API and import review | `planrepresentation` reviewer | `go doc`, `go list -deps`, and source import review in PR evidence |
| PRR-8 stateless/concurrent behavior | Successive and concurrent calls depend only on their own logical Observation | Automated behavior and race tests | `planrepresentation` | Correlated-call assertions and `go test -race` success |

### 6.2 Observation State Algebra

Level: public-constructor tests assert only success or typed construction error. State meaning is verified by returning the constructed Observation through a fake Observer and asserting the public Controller Result; no Observation private state or accessor is required.

| State subject | Allowed Given | When | Then |
|---|---|---|---|
| Root | Present children | Construct, return through Observer, and reconcile | Result reflects the supplied child meanings |
| Root | Authoritative absence | Construct, return through Observer, and reconcile | One covering root absence Difference and no descendants |
| Root | Unavailable existence | Construct, return through Observer, and reconcile | One covering root unavailable item and no descendants |
| Root | Zero Observation returned by Observer | Reconcile | `ObservationContract` and zero Result |
| Name / Goal | Valid raw value | Classify, construct, and reconcile | Equality or literal `ValueDifferent` proves exact-byte preservation |
| Name / Goal | Invalid raw value | Classify, construct, and reconcile | Scalar `InvalidObservedDifference` with the Plan-owned violation code and no raw invalid payload |
| Name / Goal | Explicit unavailable | Construct and reconcile | Scalar unavailable and no scalar Difference |
| Name / Goal | Zero child | Construct present Observation | `ObservationValidationError` |
| Target date | Valid raw date | Classify, construct, and reconcile | Equality or literal `ValueDifferent` proves exact date preservation |
| Target date | Explicit absence | Construct and reconcile | Presence comparison yields the specified absence/presence Difference |
| Target date | Invalid raw date | Classify, construct, and reconcile | Target-date `InvalidObservedDifference` with `InvalidTargetDate` and no raw invalid payload |
| Target date | Explicit unavailable | Construct and reconcile | Target-date unavailable and no target-date Difference |
| Target date | Zero child | Construct present Observation | `ObservationValidationError` |
| Collection | Complete × no members/no violations | Construct and reconcile | Expected member absences and no empty-collection violation |
| Collection | Incomplete × no members/no violations | Construct and reconcile | Collection unavailable and no expected member absence |
| Collection | Complete/incomplete × valid members | Construct and reconcile | Result proves distinct exact membership and order-independent correspondence |
| Collection | Complete/incomplete × invalid members | Construct and reconcile | Collection `InvalidObservedDifference` values only; no raw invalid member payload |
| Collection | Duplicate valid member | Construct and reconcile | Correspondence uses one representative and Result includes one duplicate violation |
| Collection | Repeated identical invalid member | Construct and reconcile | Result includes one aggregated element-validity violation and no duplicate violation |
| Present aggregate | Any zero or incompatible child state | Construct | `ObservationValidationError`; no Observation |

No test forges private Observation fields. Constructor tests use only public inputs, and Controller contract tests use the public zero Observation as the externally returnable invalid value.

### 6.3 Least Unavailable Location

Level: public helper and Location-constructor unit tests.

| Given affected locations | When | Then |
|---|---|---|
| One scalar location | Compute least unavailable location | The same scalar |
| One collection location | Compute least unavailable location | The same collection |
| One acceptance-condition member | Compute least unavailable location | Acceptance-condition collection |
| One Task member | Compute least unavailable location | Task collection |
| Members from one acceptance-condition collection | Compute least unavailable location | Acceptance-condition collection |
| Members from one Task collection | Compute least unavailable location | Task collection |
| Two different top-level branches | Compute least unavailable location | Plan root |
| Plan root plus any descendant | Compute least unavailable location | Plan root |
| Empty input | Compute | `ObservationValidationError` |
| Zero or otherwise invalid Location | Compute | `ObservationValidationError` |
| Blank or invalid UTF-8 acceptance-condition member text | Construct member Location | Typed `plan.ValidationError` through `errors.As` |
| Blank, invalid UTF-8, or multiline Task member text | Construct member Location | Typed `plan.ValidationError` through `errors.As` |

### 6.4 Correspondence and Evidence

Level: Controller behavior unit tests through the public Observer Port and exported Result.

| Behavior | Given | When | Then |
|---|---|---|---|
| Equal complete representation | Equal scalars and complete collections in different order | Reconcile | `Satisfied` and empty evidence |
| Exact scalar mismatch | One name or Goal differs by case, whitespace, line ending, or Unicode sequence | Reconcile | One `ValueDifferentDifference` with literal expected and observed text |
| Invalid observed Plan name | Blank, invalid UTF-8, CR, LF, NEL, U+2028, or U+2029 raw name | Classify, construct, and reconcile against a valid expected Plan | Plan-name `InvalidObservedDifference` with literal `InvalidText` or `MultilineName` code |
| Invalid observed Goal | Blank or invalid UTF-8 raw Goal | Classify, construct, and reconcile against a valid expected Plan | Goal `InvalidObservedDifference` with literal `InvalidText` code |
| Exact acceptance-condition membership | A complete collection replaces one expected statement with text differing by case, whitespace, line ending, or Unicode sequence | Reconcile | Literal `ExpectedAbsentDifference` for the expected statement and `UnexpectedPresentDifference` for the observed statement |
| Exact Task membership | A complete collection replaces one expected Task name with text differing by case, whitespace, or Unicode sequence | Reconcile | Literal `ExpectedAbsentDifference` for the expected name and `UnexpectedPresentDifference` for the observed name |
| Expected member absence | Complete collection omits one expected member | Reconcile | Member `ExpectedAbsentDifference` |
| Incomplete omission | Incomplete collection omits one expected member | Reconcile | Collection unavailable; no absence for the omitted member |
| Incomplete known evidence | Incomplete collection includes one unexpected valid member and invalid members | Reconcile | Preserve unexpected and invalid Differences plus collection unavailable |
| Complete empty collection | Complete collection has no members | Reconcile | Expected member absences and no empty-collection invalid evidence |
| Target date presence | Expected present/absent crossed with observed present/absent | Reconcile | Equal, expected absent, or unexpected present as specified |
| Target date value | Both dates present and unequal | Reconcile | `ValueDifferentDifference` with both dates |
| Target date invalid/unavailable | Invalid or unavailable observed target date | Reconcile | `InvalidObservedDifference` or unavailable, never both at that location |
| Root absence covering | Authoritative root absence | Reconcile | Exactly one root `ExpectedAbsentDifference` with complete expected Plan; no descendants |
| Root unavailable covering | Root unavailable | Reconcile | Exactly one root unavailable item; no Differences or descendants |

#### Canonical Difference sequence

The literal fixture uses expected name `Release`, Goal `Ship`, acceptance conditions `A` and `e\u0301`, Tasks `Build` and `\u8a66\u9a13`, and target date `2026-09-01`. The present complete Observation uses name ` release `, Goal `Ship!`, acceptance-condition raw values `a`, `é`, blank, and duplicate `a`; Task raw values ` build `, `\u8a66\u9a13`, multiline `bad\n`, and duplicate ` build `; and an absent target date. The escape sequence denotes the exact original Unicode code points.

| Order | Location | Category | Payload |
|---|---|---|---|
| 1 | Plan name | `ValueDifferent` | Expected `Release`, observed ` release ` |
| 2 | Goal | `ValueDifferent` | Expected `Ship`, observed `Ship!` |
| 3 | Acceptance-condition collection | `InvalidObserved` | `DuplicateAcceptanceCondition` |
| 4 | Acceptance-condition collection | `InvalidObserved` | `InvalidText` |
| 5 | Acceptance-condition member `A` | `ExpectedAbsent` | Expected `A` |
| 6 | Acceptance-condition member `a` | `UnexpectedPresent` | Observed `a` |
| 7 | Acceptance-condition member `e\u0301` | `ExpectedAbsent` | Expected `e\u0301` |
| 8 | Acceptance-condition member `é` | `UnexpectedPresent` | Observed `é` |
| 9 | Task collection | `InvalidObserved` | `DuplicateTask` |
| 10 | Task collection | `InvalidObserved` | `MultilineName` |
| 11 | Task member ` build ` | `UnexpectedPresent` | Observed ` build ` |
| 12 | Task member `Build` | `ExpectedAbsent` | Expected `Build` |
| 13 | Target date | `ExpectedAbsent` | Expected `2026-09-01` |

Separate fixtures assert unavailable order as Plan name, Goal, acceptance-condition collection, Task collection, and target date; root covering; same-location violation-code order; same-code aggregation; and that incomplete membership preserves established unexpected/invalid Differences. No valid current state produces different Difference categories at one Location, so no private sorter test invents that state.

### 6.5 Result and Immutability

Level: `reconciliation` Result unit tests and `planrepresentation` Result/Controller behavior unit tests.

| Invariant | Given | When | Then |
|---|---|---|---|
| Core determination | Valid Core Results with empty, Difference-only, unavailable-only, and mixed evidence | Read determination | `Satisfied`, `NotSatisfied`, `Undecidable`, `Undecidable` |
| Core zero distinction | Zero Core Result and `NewResult(nil, nil)` | Read accessors | Zero determination/nil collections versus `Satisfied`/empty collections |
| Core input snapshot | Caller-owned Difference and unavailable slices | Mutate slices after `NewResult` | Core Result collections retain original elements; element deep copies are not promised |
| Core output snapshot | Result evidence accessors | Mutate each returned slice | Later accessor reads retain original elements and determination |
| Observation input isolation | Caller-owned raw collection slices | Mutate after collection construction | Reconciliation uses the original classified members |
| Plan Result output isolation | Difference and unavailable accessors | Mutate each returned slice independently | Later reads and concurrent callers remain unchanged |
| Root expected payload | Expected Plan with collection elements | Mutate every slice returned by payload Plan accessors | Later payload reads preserve the complete original Plan |
| Closed payload variants | Every Difference category and allowed location | Read exported variant | Only the category-specific payload and valid closed Meaning variant are exposed |

### 6.6 Error, Context, and Lifecycle

Level: Controller Port-boundary unit tests using one fake Observer function and synchronization channels where timing matters.

| Case | Given | When | Then |
|---|---|---|---|
| Missing Observer construction | Nil Observer | Construct Controller | `InvalidObserver` |
| Zero Controller | Non-nil live context | Reconcile | `InvalidObserver`, zero Result, no Observer call |
| Nil context precedence | Nil context plus zero Controller and invalid Plan | Reconcile | `InvalidContext`, zero Result, no Observer call |
| Cancelled context precedence | Already-cancelled context plus zero Controller and invalid Plan | Reconcile | `context.Canceled`, zero Result, no Observer call |
| Expired deadline precedence | Expired context plus invalid Plan | Reconcile | `context.DeadlineExceeded`, zero Result, no Observer call |
| Invalid expected Plan | Valid Controller and live context | Reconcile | `InvalidExpectedPlan`, zero Result, no Observer call |
| Cancellation before successful return | Observer reaches a channel barrier after producing a valid Observation; caller cancels before release | Return | Context error and zero Result |
| Cancellation before failure return | Observer reaches a barrier before returning a contract error; caller cancels before release | Return | Context error and zero Result |
| Matching Observer context error | Observer returns the supplied `ctx.Err()` | Reconcile | Applicable context error and zero Result |
| Unrelated Observer context error | Caller context remains live while Observer returns a context error | Reconcile | `ObservationContract` without wrapping that error |
| Non-context Observer error | Observer returns a Provider-like sentinel | Reconcile | `ObservationContract`; `errors.Is` does not reveal the sentinel |
| Observation plus error | Observer returns both | Reconcile | Observation is ignored; applicable error priority decides |
| Cancellation after return | Valid Result has returned | Cancel caller context | Result evidence and determination remain unchanged |

### 6.7 Current-State and Concurrency Tests

Level: Controller Port-boundary unit tests; concurrency cases also run under the Go race detector.

| Behavior | Given | When | Then |
|---|---|---|---|
| Current-fact reconstruction | Observer returns a different current Observation on two sequential calls | Reconcile twice | Each Result reflects only its call and no prior Result |
| Concurrent isolation | Each caller context carries a distinct test correlation value; a concurrency-safe fake uses a barrier and derives its Observation from that value | Reconcile through one Controller concurrently | Each Result matches its caller's expected Plan/Observation pair without asserting call order |
| Concurrent mutation isolation | Concurrent callers mutate their returned Result slices | Read all Results again | No caller affects another Result |
| Race freedom | The concurrent isolation suite | Run `go test -race` | No data race |

### 6.8 Plan Validation Equivalence

The same literal input tables SHALL exercise standalone validation and Plan construction for zero-element and duplicate rules. Assertions compare typed `ViolationCode`, `ElementKind`, and index through `errors.As`. Nil or empty acceptance conditions intentionally diverge: standalone collection validation succeeds, while Plan construction returns `MissingAcceptanceCondition` because aggregate completeness is not a collection-validation rule.

| Rule | Required cases |
|---|---|
| Plan name | Valid non-ASCII, preserved surrounding whitespace, blank, invalid UTF-8, CR, LF, NEL, U+2028, and U+2029 |
| Acceptance-condition collection | Nil, empty, valid non-empty, exact duplicate, similar-but-distinct Unicode, zero element, and simultaneous zero-element/duplicate violations |
| Task collection | Nil, empty, valid non-empty, exact duplicate, similar-but-distinct Unicode, zero element, and simultaneous zero-element/duplicate violations |

Level: `plan` public-contract unit tests. Invalid UTF-8 remains covered by existing acceptance-condition and Task constructor tests. Forbidden line separators apply only to Task names; acceptance-condition line endings are valid and their exact bytes remain covered by existing preservation tests. Standalone collection validators receive only publicly constructible values and zero values.

#### Observation classifier equivalence

The same literal raw inputs SHALL exercise the Plan-owned validator or value constructor and the corresponding Observation classifier. Valid cases are observed through equality or literal `ValueDifferent`; invalid cases are observed as scalar `InvalidObservedDifference`. Observation internals are never inspected.

| Classifier | Shared literal cases | Expected public Result evidence |
|---|---|---|
| Plan name | Valid non-ASCII, preserved surrounding whitespace, blank, invalid UTF-8, CR, LF, NEL, U+2028, U+2029 | Exact value correspondence, `InvalidText`, or `MultilineName` matching `plan.ValidateName` |
| Goal | Valid non-ASCII, preserved surrounding whitespace, multiline valid text, blank, invalid UTF-8 | Exact value correspondence or `InvalidText` matching `plan.NewGoal` |
| Target date | `0001-01-01`, `9999-12-31`, `2028-02-29`, `0000-01-01`, `2027-02-29`, non-zero-padded form, surrounding whitespace, timestamp | Exact date correspondence or `InvalidTargetDate` matching `plan.ParseTargetDate` |

### 6.9 Testability Feedback

- Public constructors and accessors are tested as black boxes; tests do not forge private fields or assert private helper calls.
- A fake `Observer` is the only substitute and tests the consumer-owned external boundary, not internal sequencing.
- Provider failure translation is not claimed by this Change; each future Provider must satisfy its deferred contract tests.
- Host association and exported API dependency checks use explicit review evidence because no Host or concrete Provider is implemented here.
- Implementation uses the repository's `go-test-authoring` workflow and starts each applicable Go test function from a `gotests` table scaffold.
