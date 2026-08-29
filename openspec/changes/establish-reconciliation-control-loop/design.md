# Reconciliation Control Loop Design

## 0. Document Scope

| Information | Governing document |
|---|---|
| Product value, Feedback Loop scope, and external authority principles | [PRODUCT.md](https://github.com/kotokumu/arcloom/blob/main/PRODUCT.md) |
| Component ownership, dependency direction, Port ownership, and external Context boundaries | [ARCHITECTURE.md](https://github.com/kotokumu/arcloom/blob/main/ARCHITECTURE.md) |
| Capability boundary, concepts, relationships, and Requirement candidates | `model.md` in this Change |
| Observable control and Plan reconciliation guarantees | Delta specs in this Change |
| Package boundaries, public contracts, tests, and implementation decisions for this Change | This DesignDoc |
| Implementation order and verification progress | `tasks.md` in this Change |

---

## 1. Purpose / Non-Goals

### Purpose

- Add one target-independent owner for level-based request scheduling, same-target exclusion, explicit reevaluation, result reporting, and caller cancellation.
- Add one read-only Plan-specific attempt that composes fresh Plan Snapshot observation and Plan Control assessment as the first standard Consumer.
- Keep target results, observation meaning, Authorization, application, external Actor behavior, and event acquisition outside generic control.

### Non-Goals

- A fixed Delivery or Development Improvement workflow.
- Durable queues, distributed exclusion, leader election, leases, or persisted loop history.
- A universal Observation, Condition, Outcome, Adjustment, or external-effect interface.
- Automatic Plan Authorization, application, retry, or receipt-driven scheduling.
- Provider watches, polling, webhooks, or Agent and Task execution.

### Risk Assessment

| Risk level | Trigger | Construction gate |
|---|---|---|
| High | New public control contracts and changed cross-Component Architecture dependencies | Complete conceptual model, independent evolution scenarios, responsibility and Architecture gates, interface and test design, detailed TDD plan, Architecture review, and human approval before implementation. |

---

## 2. Behavior Design

### 2.1 Functional Requirements

The delta specs are normative. This table assigns their guarantees to decision owners without redefining them.

| Requirement | Function | Decision owner |
|---|---|---|
| `reconciliation-control-loop/exact-target-request` | Stable request and outcome correlation | Reconciliation Target Identity and Reconciliation Controller |
| `reconciliation-control-loop/level-based-attempt` | Current-fact evaluation without event payload authority | Target-specific Reconciler contract |
| `reconciliation-control-loop/per-target-exclusion-and-coalescing` | Pending, active, concurrency-bound, and coalescing rules | Reconciliation Controller |
| `reconciliation-control-loop/explicit-control-directive` | Await, immediate, and delayed scheduling meaning | Control Directive and Reconciliation Controller |
| `reconciliation-control-loop/target-result-isolation` | Opaque target-owned result transport | Target-specific Reconciler and Completion |
| `reconciliation-control-loop/failure-does-not-retry` | Target-bound failure without inferred retry | Reconciliation Controller |
| `reconciliation-control-loop/caller-lifecycle` | Stop, cancel, wait, and close | Reconciliation Controller and caller lifecycle |
| `reconciliation-control-loop/disposable-control-state` | Recovery without authoritative internal state | Reconciliation Controller |
| `plan-reconciliation-loop/fresh-plan-observation` | One fresh snapshot and successful no-current-Plan result | Plan Reconciliation Attempt and Plan Snapshot Observation |
| `plan-reconciliation-loop/current-plan-assessment` | Assessment of the exact current Plan | Plan Reconciliation Attempt and Plan Controller |
| `plan-reconciliation-loop/read-only-plan-attempt` | Await-only result and external-effect exclusion | Plan Reconciliation Attempt |
| `plan-reconciliation-loop/ordinary-plan-reentry` | Identity-only explicit reentry after any event | Reconciliation Request and Plan Reconciliation Attempt |

### 2.2 Non-Functional Requirements

| Non-functional requirement | Threshold | Measurement |
|---|---|---|
| Concurrent isolation | At least 100 mixed-target requests produce no same-target overlap, lost active-period request, cross-target result contamination, or race detector finding. | Deterministic concurrency tests plus `go test -race ./...`. |
| Cancellation progress | After every Active Reconciler returns from cancellation, the Report stream closes and `Wait` returns without another dependency on Report consumption, Pending or Delayed work, or timers. | Deterministic barriers and wait groups; a generous one-second timeout is only a deadlock guard, not a latency assertion. |
| Goroutine lifecycle | After each cancellation test, every Reconciler goroutine started by the Controller has returned before output closes. | Test-owned wait groups and leak-sensitive completion assertions. |

---

## 3. Structure Design

### 3.1 Conceptual Model

| Concept | Meaning | Identity | Invariant enforced by design |
|---|---|---|---|
| Reconciliation Target Identity | Stable correlation key, not target state | Non-empty target kind and key | Request, Attempt, Completion, and Failure retain the exact value. |
| Reconciliation Request | Occurrence that makes one target eligible | Its Target Identity | No cause or payload crosses the Reconciler Port. |
| Reconciliation Control Loop | Caller-scoped owner of disposable scheduling and Report-delivery state | One started Controller instance | Same-target exclusion, concurrency bound, explicit scheduling, and orderly cancellation hold together. |
| Reconciliation Attempt | One current-fact target evaluation | One Target Identity and caller lifecycle | Success has one result and Directive; failure has neither. |
| Target Attempt Result | Target-owned result role | Target-specific | Generic control never interprets it. |
| Control Directive | Target-independent next-evaluation instruction | Await, Immediate, or positive finite Delay | Only a successful Directive creates internal eligibility. |
| Successful Attempt Completion | Target/result/Directive association | One returned Attempt | Its three values cannot be split or substituted. |
| Attempt Failure | Valid target/semantic failure-kind/error association | One returned Attempt | Target Attempt Failed and Control Directive Rejected are distinguishable and create no internal eligibility. |
| Plan Target Binding | Self-identifying association with target observation boundaries | Exact Target Identity | Resolved identity equals requested identity; Snapshot and Delivery Observers are valid together. |
| Plan Reconciliation Attempt | Ordered fresh Snapshot, later Delivery Observation, and optional Assessment composition | One invocation of one resolved Plan Target Binding | It performs no Authorization or application. |
| Plan Reconciliation Result | Fresh Snapshot associated with zero or one Assessment | One successful Plan Attempt | Assessment exists only for the exact current Plan in that Snapshot. |

### 3.2 Responsibility Assignment

| Responsibility / decision | Owner | Information and authority used | State / invariant affected | Change driver | Not owner / reason |
|---|---|---|---|---|---|
| Validate and compare control identities | Reconciliation Target Identity | Caller-supplied target kind and key | Stable same-target correlation | Control identity rules | Provider and Plan do not own generic scheduling identity. |
| Accept, coalesce, and schedule requests | Reconciliation Controller | Target identity, per-target state, concurrency count, current time | Exclusion, pending preservation, active bound | Scheduling and lifecycle rules | Composition Root only wires dependencies. |
| Select target-specific result and Directive | Target-specific Reconciler | Current target facts and target rules | Successful Completion contract | Target requirements | Controller cannot interpret target results. |
| Deliver success or typed failure and commit Directive | Reconciliation Controller | Returned target-attempt outcome, Directive validity, exact target, and Report consumer acceptance | Success/failure separation; one bounded Pending Delivery state | Control protocol | Target Attempt does not own output multiplexing, protocol validation, or delivery commitment across targets. |
| Stop and await active work | Reconciliation Controller | Caller context, active Attempt set, and pending Report delivery | No start after stop; no unfinished completion; no consumer-induced shutdown block | Caller lifecycle | Target Reconciler owns only its own cancellation response. |
| Resolve a Plan Target Binding | Plan Reconciliation Attempt Module | Requested identity and caller-supplied resolver | Exact requested/resolved identity and target Observer association | Plan integration boundary | Composition Root supplies the resolver but does not perform lookup or branch on its result. |
| Establish current Plan Snapshot | Plan Snapshot Observation Module | Planning Context observation Port and Plan invariants | Snapshot coherence | Planning representation and Provider facts | Plan Reconciliation does not reinterpret Snapshot validity. |
| Establish Plan Control Assessment | Plan Controller | Exact current Plan, Delivery Observations, external AI response | Assessment invariants | Plan assessment meaning | Plan Reconciliation preserves rather than re-evaluates it. |
| Compose Snapshot, later Delivery Observations, and optional Assessment | Plan Reconciliation Attempt Module | Resolved self-identifying binding, configured Assessor, and existing Snapshot and Plan Control contracts | Acquisition order, exact observation/Assessment association, stable runtime boundary failures, and read-only boundary | Plan loop composition | Host must not own lookup, ordering, failure branches, or construct a partial result. |
| Submit a later request | Host | External event or interaction known to the Host | Request occurrence only | Integration choice | Neither Controller nor Plan module interprets event or receipt causes. |
| Authorize and apply a Proposed Plan | Existing Authorization and Plan Application Request Modules plus external Actor | Exact Revision, current policy/evidence, Actor-native state | Existing application invariants | Authorization and external mutation | Generic Controller and Plan Reconciliation explicitly exclude it. |

### 3.3 Independent Evolution Scenario Impact

| Scenario / confidence | Primary decision owner | Expected propagation | Unexplained impact | Duplicated policy | Verdict |
|---|---|---|---|---|---|
| New non-Plan target and result type / Committed | Target-specific Reconciler | New Reconciler, result, composition, and behavior tests; generic contracts unchanged | None | No | Pass |
| Target result classifications change / Evidence-backed plausible | Target-specific Reconciler | Target result and tests only | None | No | Pass |
| Provider watch, polling, manual, or application event source changes / Committed | Host and Provider boundary | Request producer changes; request remains identity-only | None | No | Pass |
| Duplicate burst and completion race / Committed | Reconciliation Controller | Scheduler state and concurrency tests | None | No | Pass |
| New failure categories / Committed | Target-specific Reconciler | Target error and tests; Controller still reports without retry | None | No | Pass |
| Internal scheduling state loss / Committed | Reconciliation Controller and Host | New Controller plus external request; no restoration contract | None | No | Pass |
| Distributed exclusion becomes required / Committed non-goal | External deployment owner | Remains outside this in-process contract | None inside accepted scope | No | Pass |
| Planning Provider changes / Evidence-backed plausible | Planning Provider Module | Provider adapter and observation tests | None | No | Pass |
| Delivery Observation vocabulary expands / Evidence-backed plausible | Plan Control caller and Assessor | Typed observation value and assessment adapter | None | No | Pass |
| Plan assessment classifications change / Evidence-backed plausible | Plan Controller | Plan Result preservation tests; generic Controller unchanged | None | No | Pass |
| Plan application result precedes another request / Committed | Host | Host may submit ordinary request; no receipt crosses either Port | None | No | Pass |
| Repeated Immediate directives consume unbounded work / Speculative | No accepted owner | Record reconsideration trigger only | Rate behavior remains deliberately unspecified | No | Risk only |

### 3.4 Architecture Boundary Plan

| Boundary candidate | Consumer / evidence | State, data, or policy owner | Constraint protected | Dependency direction | Simpler existing alternative | Decision |
|---|---|---|---|---|---|---|
| Reconciliation Controller Component | Hosts and target-specific Reconcilers; RCL requirements | Owns only caller-scoped scheduling state | Same-target exclusion, concurrency, delayed eligibility, cancellation | Target-specific Control Modules depend inward on its Reconciler and Directive contracts; Composition Root constructs it | Composition Root cannot own repeated lifecycle or policy | Accept |
| Plan Reconciliation Attempt Module | Plan loop caller; PRL requirements | Owns Plan Target Binding resolution, ordered Snapshot/Delivery Observation/Assessment composition, and read-only rule | Prevents Host orchestration, partial result, and application leakage | Depends on Reconciliation Controller contract, Plan Snapshot Observation, and Plan Controller | Host lookup and branching leave Plan composition decision ownerless | Accept |
| Generic Observation Component | No consumer needs shared observation semantics | N/A | None | N/A | Existing target-owned observation contracts | Reject |
| Generic external-effect or Adjustment application | No generic safety or idempotency contract exists | Target-specific Application Request and Actor | Avoids unsafe retry and Authorization leakage | Remains outside Controller | Existing explicit application capabilities | Reject |
| Public timer/clock Port | Production consumer needs only elapsed delay | Reconciliation Controller | No external boundary or substitution need | N/A | Private time function used by package tests | Reject |
| Durable queue/repository | Explicitly outside Product scope for this Change | External Host/deployment | Restart delivery | N/A | New external request after restart | Reject |

| Scenario / confidence | Primary boundary or owner | Expected propagation | Unexplained impact | Verdict |
|---|---|---|---|---|
| New target integration / Committed | New target-specific Control Module | Its result, Reconciler implementation, Composition Root, and tests | None | Pass |
| Provider replacement / Evidence-backed plausible | Target-specific Provider Module | Provider Port implementation and contract tests | None | Pass |
| Deployment-form replacement / Committed | Composition Root and Host | Wiring and request source only | None | Pass |
| Durable scheduling requirement / Committed non-goal | External Host/deployment | No propagation into Controller until a future product decision | None | Pass |
| Distributed same-target ownership / Committed non-goal | External Host/deployment | No propagation into in-process exclusion | None | Pass |

### 3.5 Package Design

| Package | Responsibility | Publishes | Hides | Permitted dependencies |
|---|---|---|---|---|
| `reconciliationcontrol` | Target Identity, Directive, semantic Completion/Failure reporting, one bounded Pending Delivery state, and one caller-scoped Controller lifecycle | TargetIdentity, Directive, FailureKind, Failure, Report, Reconciler, Controller, Start | Per-target state machine, ready selection, delay ordering, delivery commitment, goroutines, channels, cancellation coordination | Standard library only |
| `planreconciliation` | Validate and resolve a self-identifying Plan Target Binding and perform ordered read-only Snapshot, Delivery Observation, and Plan Control composition | PlanTarget, TargetResolver, DeliveryObserver, setup error sentinels, FailureCode, FailureError, Result, NewReconciler | Binding identity validation and resolution, acquisition order, no-current-Plan branch, and stable runtime boundary failure normalization | `reconciliationcontrol`, `plansnapshot`, `plancontrol`, standard library |
| `plansnapshot` | Existing Snapshot observation meaning | Existing Observer, Observe, Snapshot | Provider-bound observation validation | Existing dependencies unchanged |
| `plancontrol` | Existing Plan Control assessment meaning | Existing Assessor, Assess, Assessment | AI response validation | Existing dependencies unchanged |

| Source package | Target package | Public contract | Dependency reason | Details that must not cross |
|---|---|---|---|---|
| `planreconciliation` | `reconciliationcontrol` | Directive | A Plan success selects Await Another Request | Scheduling internals and target queues do not enter Plan logic. |
| `planreconciliation` | `plansnapshot` | Observer, Observe, Snapshot | Every resolved Plan Attempt begins with exactly one fresh Snapshot | Provider targets, HTTP DTOs, and Provider errors do not cross. |
| `planreconciliation` | `plancontrol` | Assessor, Assess, Assessment | A valid current Plan receives current Delivery Observations acquired afterward and one existing assessment | AI SDK, Provider response, and process details do not cross. |
| Composition Root | `planreconciliation` | TargetResolver, Assessor, and NewReconciler | Supplies self-identifying target bindings and one Plan Control Assessor, then wires the returned Reconciler | Binding lookup, identity validation, acquisition order, error branching, and scheduling state do not enter the Root. |
| Composition Root | `reconciliationcontrol` | Start and Controller | Starts one selected target-specific lifecycle and connects request/report consumers | Business branching and scheduling policy do not enter the Root. |

### 3.6 Interface Design

The following Go contracts are design output and are not yet implemented source code.

#### Reconciliation Target Identity

```go
package reconciliationcontrol

// TargetIdentity is a stable caller-established control key, not target state.
// NewTargetIdentity rejects empty, whitespace-only, or leading/trailing Unicode
// whitespace in kind or key. It otherwise preserves bytes exactly and performs
// no case folding or Unicode normalization.
type TargetIdentity struct { /* immutable kind and key */ }

func NewTargetIdentity(kind, key string) (TargetIdentity, error)
func (t TargetIdentity) Kind() string
func (t TargetIdentity) Key() string
```

#### Control Directive

```go
package reconciliationcontrol

type DirectiveKind uint8

const (
    AwaitRequest DirectiveKind = iota + 1
    ReevaluateImmediately
    ReevaluateAfterDelay
)

// Directive is valid only when constructed below. Delay is present only for
// ReevaluateAfterDelay and must be positive and finite.
type Directive struct { /* kind and optional delay */ }

func AwaitAnotherRequest() Directive
func ImmediateReevaluation() Directive
func DelayedReevaluation(delay time.Duration) (Directive, error)
func (d Directive) Kind() DirectiveKind
func (d Directive) Delay() (time.Duration, bool)
```

#### Target-specific Reconciler

```go
package reconciliationcontrol

// Reconciler reacquires every target-specific fact required for one decision.
// It returns one result and valid Directive with nil error, or an error; result
// and Directive are ignored on error. A non-nil error always becomes
// TargetAttemptFailed, even when accompanied by a valid or invalid Directive.
// It must stop waiting and return after ctx.Done. A nominal success returned
// after Controller stopping begins establishes no Completion. The generic
// Controller never inspects R.
type Reconciler[R any] func(
    ctx context.Context,
    target TargetIdentity,
) (R, Directive, error)
```

#### Attempt Report

```go
package reconciliationcontrol

// Report is exactly one successful Completion or one Failure for its Target.
// Completion presence is independent from whether the R zero value is valid.
type Report[R any] struct { /* immutable target and outcome */ }

func (r Report[R]) Target() TargetIdentity
func (r Report[R]) Completion() (result R, directive Directive, ok bool)
func (r Report[R]) Failure() (failure Failure, ok bool)
```

#### Attempt Failure

```go
package reconciliationcontrol

type FailureKind uint8

const (
    TargetAttemptFailed FailureKind = iota + 1
    ControlDirectiveRejected
)

// Failure is a valid target-bound attempt or Directive protocol failure.
// It contains no result or Directive. Unwrap returns the exact target-attempt
// error for TargetAttemptFailed or the stable Directive validation cause for
// ControlDirectiveRejected, so errors.Is preserves the documented cause.
type Failure struct { /* immutable kind and cause */ }

func (f Failure) Kind() FailureKind
func (f Failure) Error() string
func (f Failure) Unwrap() error
```

#### Reconciliation Controller

```go
package reconciliationcontrol

var (
    ErrInvalidContext        = errors.New("invalid control context")
    ErrInvalidConcurrency    = errors.New("invalid concurrency bound")
    ErrInvalidReconciler     = errors.New("invalid reconciler")
    ErrInvalidTargetIdentity = errors.New("invalid target identity")
    ErrControllerNotStarted  = errors.New("controller not started")
)

// Controller is exactly one started caller-scoped control lifecycle. It is
// safe for concurrent Request, Reports, and Wait calls and must not be copied.
// Its zero value is unusable.
type Controller[R any] struct { /* one private lifecycle */ }

// Start validates all inputs and starts exactly one lifecycle. maxConcurrent
// must be positive, reconciler non-nil, and ctx non-nil and not already done.
func Start[R any](
    ctx context.Context,
    maxConcurrent int,
    reconciler Reconciler[R],
) (*Controller[R], error)

// Request synchronously validates target and submits one identity-only wake-up.
// Its ctx bounds submission only and never enters the Reconciler as evidence or
// cancellation. Once Request returns nil, later cancellation of this ctx does
// not affect the accepted Attempt.
// It returns ErrInvalidTargetIdentity for a zero/invalid target, a supplied
// submission context error if that context wins, or the Controller lifecycle
// context error after stopping. If both contexts are already done before
// acceptance, the Controller lifecycle error wins. A nil ctx returns
// ErrInvalidContext. A nil return means the scheduler accepted the request.
func (c *Controller[R]) Request(
    ctx context.Context,
    target TargetIdentity,
) error

// Reports returns the one Report stream for this lifecycle; every concurrent or
// repeated call returns that same channel. The zero Controller returns nil.
// Each accepted Attempt outcome becomes one prospective Report and is never
// duplicated. Attempt return time and consumer activity do not commit delivery.
// If Report delivery wins its race with lifecycle cancellation, it publishes
// the Completion or Failure exactly once; if cancellation wins, the prospective
// Report is discarded. Without lifecycle cancellation and while the consumer
// continues receiving, every returned Attempt outcome is published exactly once;
// no order is promised across distinct concurrently Active targets. The Controller
// retains at most one undelivered Report and starts
// no new Attempt until delivery, while still accepting/coalescing Requests,
// receiving Active returns, and observing cancellation. A successful Directive
// creates eligibility only after its Completion Report is delivered. On lifecycle
// cancellation, the prospective Report and unapplied Directive may be discarded
// without publishing an outcome so consumption cannot block termination. The channel closes after all Active
// Reconcilers return.
func (c *Controller[R]) Reports() <-chan Report[R]

// Wait returns only after the Report stream closes and all Active Reconcilers
// return. It then returns the original Controller lifecycle context error.
// Concurrent or repeated calls return the same outcome. A zero Controller
// returns ErrControllerNotStarted.
func (c *Controller[R]) Wait() error
```

#### Plan Target Binding

```go
package planreconciliation

// DeliveryObserver obtains the caller-owned observation vocabulary only after
// a fresh Snapshot has established a valid current Plan. A context error seen
// before return wins; any other error is normalized by the Plan Reconciler.
type DeliveryObserver[O any] func(context.Context) (O, error)

// PlanTarget is an immutable self-identifying binding of the target-specific
// observation boundaries required by one Plan Attempt. Its zero value is invalid.
type PlanTarget[O any] struct { /* TargetIdentity, Observer, DeliveryObserver */ }

// NewPlanTarget returns ErrInvalidPlanTarget unless target is valid and both
// observation boundaries are non-nil.
func NewPlanTarget[O any](
    target reconciliationcontrol.TargetIdentity,
    observer plansnapshot.Observer,
    delivery DeliveryObserver[O],
) (PlanTarget[O], error)

// TargetResolver maps one requested identity to its exact immutable binding.
// It is a local composition boundary, performs no Provider I/O, observes ctx,
// and returns within its documented cancellation bound after ctx.Done.
type TargetResolver[O any] func(
    context.Context,
    reconciliationcontrol.TargetIdentity,
) (PlanTarget[O], error)
```

#### Plan Reconciliation Failure

```go
package planreconciliation

var (
    ErrInvalidPlanTarget     = errors.New("invalid plan target")
    ErrInvalidTargetKind     = errors.New("invalid target kind")
    ErrInvalidTargetResolver = errors.New("invalid target resolver")
    ErrInvalidAssessor       = errors.New("invalid plan control assessor")
)

type FailureCode string

const (
    TargetBindingUnavailable       FailureCode = "target_binding_unavailable"
    DeliveryObservationUnavailable FailureCode = "delivery_observation_unavailable"
)

// FailureError is a stable provider-independent runtime Plan Attempt failure.
// Resolver and Delivery Observer errors do not unwrap through it. Configuration
// errors are returned synchronously by NewPlanTarget or NewReconciler instead.
// Existing plansnapshot and plancontrol errors remain owned and returned by
// those APIs. A caller context error observed before or after any boundary call
// wins unchanged; every Attempt error return has zero Result/Directive.
type FailureError struct { /* code */ }

func (f *FailureError) Error() string
func (f *FailureError) Code() FailureCode
```

#### Plan Reconciliation Result and Reconciler

```go
package planreconciliation

// Result preserves one successful fresh Snapshot and an optional Assessment.
// Assessment is present exactly when the Snapshot exposed a current Plan and
// Plan Control succeeded for that exact Plan. The zero Result is invalid.
type Result struct { /* immutable snapshot and optional assessment */ }

func (r Result) Snapshot() plansnapshot.Snapshot
func (r Result) Assessment() (plancontrol.Assessment, bool)

// NewReconciler returns the Plan Module's direct implementation of the generic
// Reconciler Port. targetKind follows TargetIdentity's kind validation
// and resolver and assessor must be non-nil. Invalid configuration returns a
// setup sentinel before any Reconciler exists. Each invocation checks caller cancellation,
// accepts only that exact kind without invoking the resolver otherwise,
// resolves one valid binding, verifies its identity exactly equals the request,
// invokes
// plansnapshot.Observe exactly once, and only for a current Plan invokes the
// DeliveryObserver once followed by plancontrol.Assess once. Context errors
// observed by each existing boundary retain their current precedence.
// Every success returns AwaitAnotherRequest. Read-only prohibits Plan or Actor
// mutation, Authorization, and application; observation and AI assessment may
// retain their existing Provider I/O, cost, and telemetry effects.
func NewReconciler[O any](
    targetKind string,
    resolver TargetResolver[O],
    assessor plancontrol.Assessor[O],
) (reconciliationcontrol.Reconciler[Result], error)
```

#### Example Call Sites

The following examples are design-only and are not executed source.

```go
resolver := func(ctx context.Context, requested reconciliationcontrol.TargetIdentity) (
    planreconciliation.PlanTarget[DeliveryObservations],
    error,
) {
    if err := ctx.Err(); err != nil {
        return planreconciliation.PlanTarget[DeliveryObservations]{}, err
    }
    binding, ok := configuredPlanTargets[requested]
    if !ok { return planreconciliation.PlanTarget[DeliveryObservations]{}, bindingUnavailable }
    return planreconciliation.NewPlanTarget(
        requested,
        binding.ObservePlan,
        binding.ObserveDelivery,
    )
}

reconciler, err := planreconciliation.NewReconciler("plan", resolver, planAssessor)
if err != nil { return err }
controller, err := reconciliationcontrol.Start(ctx, 4, reconciler)
if err != nil { return err }
target, err := reconciliationcontrol.NewTargetIdentity("plan", externalTargetKey)
if err != nil { return err }
if err := controller.Request(ctx, target); err != nil { return err }
for report := range controller.Reports() { consume(report) }
if err := controller.Wait(); !errors.Is(err, ctx.Err()) { return err }
```

```go
// A later external interaction is not converted inside either package.
// The Host may explicitly submit the same identity for any reason.
if err := controller.Request(ctx, target); err != nil { return err }
```

#### Boundary Decisions

| Boundary | Hidden detail | Reason |
|---|---|---|
| `Reconciler[R]` | Observation and target-specific decision implementation | Controller has real Consumers with unrelated results and must not import their Components. |
| `planreconciliation.TargetResolver[O]` | Identity-to-Plan-boundary lookup | Plan Module owns validation and failure normalization while the Composition Root only supplies bindings. |
| `plansnapshot.Observer` | Planning Provider access and target binding | Existing Snapshot consumer contract owns this external boundary. |
| `planreconciliation.DeliveryObserver[O]` | Current caller-selected Delivery Observation acquisition | Plan Module owns invocation order but not observation vocabulary or Provider implementation. |
| `plancontrol.Assessor[O]` | AI Provider interaction | Existing Plan Control consumer contract owns judgment acquisition. |
| Private time source | Timer implementation | Only package tests need deterministic substitution; no production Consumer requires a public Port. |

#### Interface Traceability

| Public contract | Requirement / invariant | Owner | Consumer | Why separate |
|---|---|---|---|---|
| TargetIdentity | exact-target-request; stable equality | Reconciliation Controller Component | Host and Reconciler | Plain strings cannot preserve validity and semantic pairing. |
| Directive | explicit-control-directive | Reconciliation Controller Component | Reconciler and Controller | Target result and error must not carry implicit scheduling meaning. |
| Reconciler[R] | level-based-attempt; target-result-isolation | Reconciliation Controller Component | Controller | Isolates target Components without a universal result interface. |
| Failure and Report[R] | typed failure or success and target association | Reconciliation Controller Component | Host | A result-only channel loses failure kind and Directive correlation. |
| Controller[R] and Start | concurrency, coalescing, report backpressure, lifecycle outcome, disposable state | Reconciliation Controller Component | Host | A pure function or reusable Run configuration cannot protect one lifecycle's guarantees. |
| PlanTarget and TargetResolver[O] | self-identifying binding, exact requested/resolved equality, synchronous setup validation, and stable runtime boundary failure | Plan Reconciliation Attempt Module | Composition Root supplies; returned Reconciler consumes | Host-side correspondence checks and lookup branching would leak Plan attempt orchestration. |
| DeliveryObserver[O] | post-Snapshot current observation acquisition | Plan Reconciliation Attempt Module | Returned Plan Reconciler | Pre-acquired observations cannot prove the designed order. |
| planreconciliation.Result | fresh Snapshot/Assessment association | Plan Reconciliation Attempt Module | Plan loop Host | Snapshot or Assessment alone cannot represent both successful branches. |
| planreconciliation.NewReconciler | PRL requirements | Plan Reconciliation Attempt Module | Reconciliation Controller | Keeps binding, ordered acquisition, and Plan branching out of Host and existing single-responsibility Components. |

#### Interface Risks

- Oversized interfaces: none; Reconciler is one consumer-owned function contract.
- Primitive obsession: target kind and key are validated once into TargetIdentity.
- Infrastructure leakage: Provider DTOs, event payloads, application results, AI SDK types, and timers do not cross.
- Boolean flags: none; Directive uses explicit constructors and kinds.
- Backpressure: slow Report consumption may delay accepted work, but lifecycle cancellation discards undelivered Reports as needed, waits only for Active Reconciler cancellation bounds, closes Reports, and makes the caller context error available through Wait.
- Lifecycle scope: one started Controller is exactly one lifecycle; callers start another Controller rather than overlapping Runs on shared mutable state.

### 3.7 Test Specification

#### Requirement Coverage

| Requirement | Observable behavior | Automated verification | Owner | Evidence |
|---|---|---|---|---|
| exact-target-request | Valid correlation and invalid rejection | Public black-box tests | `reconciliationcontrol` | Passing test names and output |
| level-based-attempt | Reconciler receives identity only and reacquires test state | State-of-world behavior tests | `reconciliationcontrol` | Passing tests |
| per-target-exclusion-and-coalescing | No overlap, bounded concurrency, no lost active-period request | Deterministic channel barriers plus race detector | `reconciliationcontrol` | Passing race suite |
| explicit-control-directive | Await, Immediate, Delay, early request, invalid delay | Deterministic scheduler tests with private fake time | `reconciliationcontrol` | Passing tests |
| target-result-isolation | Two unrelated result types remain unchanged | Generic black-box tests | `reconciliationcontrol` | Passing tests |
| failure-does-not-retry | Target Attempt Failed or Control Directive Rejected alone creates no new call; preserved request still does | Barrier-based behavior tests | `reconciliationcontrol` | Passing tests |
| caller-lifecycle | No post-cancel starts; stopped Report consumer cannot block Active-call wait, channel close, or lifecycle outcome | Cancellation, backpressure, and wait-group tests | `reconciliationcontrol` | Passing tests and timing evidence |
| disposable-control-state | New Controller acts from new request only | Restart-style behavior test | `reconciliationcontrol` | Passing test |
| fresh-plan-observation | Binding resolution and exactly one fresh Snapshot first; no-current branch skips later boundaries | Table-driven tests | `planreconciliation` | Passing tests |
| current-plan-assessment | Delivery Observation follows Snapshot; exact Plan/observations and preserved Assessment | Ordered table-driven tests with Assessor boundary | `planreconciliation` | Passing tests |
| read-only-plan-attempt | Every assessment Await; no application dependency exists | Black-box tests and dependency check | `planreconciliation` | Passing tests and build check |
| ordinary-plan-reentry | Prior receipt/event cannot enter input; second call observes fresh | Two-attempt composition test | `planreconciliation` | Passing test |
| Real Plan proof | Generic assessment, separate application, explicit request, later Snapshot | Opt-in reproducible verification | Host/proof owner | Recorded disposable proof |

#### Scenario Traceability

The listed test names are the required observable black-box names or table subtest names. Renaming requires updating this table so no scenario loses evidence.

| Scenario ID | Exact automated test / case or verification procedure | Owner | Evidence |
|---|---|---|---|
| RCL-ETR-1 | `TestControllerRequest/valid_target_is_preserved` | `reconciliationcontrol` | Test output |
| RCL-ETR-2 | `TestControllerRequest/invalid_target_is_rejected_synchronously` | `reconciliationcontrol` | Test output |
| RCL-ETR-3 | `TestTargetIdentity/equality_is_byte_exact_case_sensitive_and_not_normalized` | `reconciliationcontrol` | Public value and Controller test output |
| RCL-LBA-1 | `TestControllerReconciler/provider_event_payload_is_not_input` | `reconciliationcontrol` | Test output |
| RCL-LBA-2 | `TestControllerReconciler/prior_result_is_not_input` | `reconciliationcontrol` | Test output |
| RCL-PEC-1 | `TestControllerScheduling/pending_duplicates_do_not_require_one_attempt_each` | `reconciliationcontrol` | Test and race output |
| RCL-PEC-2 | `TestControllerScheduling/active_period_request_is_preserved` | `reconciliationcontrol` | Test and race output |
| RCL-PEC-3 | `TestControllerScheduling/distinct_targets_progress_within_bound` | `reconciliationcontrol` | Test and race output |
| RCL-PEC-4 | `TestControllerScheduling/coincident_request_and_completion_leave_later_work` | `reconciliationcontrol` | Repeated test and race output |
| RCL-ECD-1 | `TestControllerDirective/await_requires_request` | `reconciliationcontrol` | Test output |
| RCL-ECD-2 | `TestControllerDirective/immediate_makes_target_eligible` | `reconciliationcontrol` | Test output |
| RCL-ECD-3 | `TestControllerDirective/delay_expiry_makes_target_eligible` | `reconciliationcontrol` | Test output |
| RCL-ECD-4 | `TestControllerDirective/early_request_invalidates_old_delay` | `reconciliationcontrol` | Test output |
| RCL-ECD-5 | `TestDelayedReevaluation/non_positive_delay_is_rejected` | `reconciliationcontrol` | Public constructor test output |
| RCL-ECD-6 | `TestControllerDirective/zero_directive_reports_directive_failure` | `reconciliationcontrol` | Public Controller test output |
| RCL-ECD-7 | `TestControllerDirective/directive_commits_only_after_completion_delivery` | `reconciliationcontrol` | Barrier-based test output |
| RCL-TRI-1 | `TestControllerResultIsolation/unrelated_typed_controllers_preserve_results` | `reconciliationcontrol` | Compile and test output |
| RCL-FNR-1 | `TestControllerFailure/reconciler_failure_without_request_does_not_repeat` | `reconciliationcontrol` | Test output |
| RCL-FNR-2 | `TestControllerFailure/preserved_request_survives_failure` | `reconciliationcontrol` | Test and race output |
| RCL-FNR-3 | `TestControllerFailure/other_target_continues` | `reconciliationcontrol` | Test and race output |
| RCL-FNR-4 | `TestControllerFailure/directive_failure_is_distinguishable` | `reconciliationcontrol` | Test output |
| RCL-FNR-5 | `TestControllerFailure/target_attempt_error_outranks_returned_values` | `reconciliationcontrol` | Test output and `errors.Is` assertion |
| RCL-CL-1 | `TestControllerCancellation/active_and_pending_work_stops` | `reconciliationcontrol` | Test and race output |
| RCL-CL-2 | `TestControllerCancellation/no_wait_dependency_after_active_return` | `reconciliationcontrol` | Barrier ordering; timeout only guards deadlock |
| RCL-CL-3 | `TestControllerCancellation/stopped_report_consumer_cannot_block` | `reconciliationcontrol` | Barrier ordering and race output |
| RCL-CL-4 | `TestControllerCancellation/nominal_success_after_stop_is_not_completion` | `reconciliationcontrol` | Test output |
| RCL-CL-5 | `TestControllerRequest/submission_context_after_acceptance_does_not_cancel_attempt` | `reconciliationcontrol` | Test and race output |
| RCL-CL-6 | `TestControllerRequest/controller_context_wins_when_both_done` | `reconciliationcontrol` | Test output |
| RCL-CL-7 | `TestControllerLifecycle/report_delivery_wins_and_publishes_once` | `reconciliationcontrol` | Public black-box test output |
| RCL-CL-8 | `TestControllerLifecycle/pending_report_applies_bounded_global_backpressure` | `reconciliationcontrol` | Barrier and race output |
| RCL-CL-9 | `TestControllerLifecycle/cancellation_wins_and_discards_prospective_outcome` | `reconciliationcontrol` | Barrier and race output |
| RCL-CL-10 | `TestControllerLifecycle/simultaneous_delivery_and_cancellation_commit_exactly_one_outcome` | `reconciliationcontrol` | Repeated public API race test and race-detector output |
| RCL-CL-11 | `TestControllerLifecycle/normal_operation_reports_every_returned_outcome_once` | `reconciliationcontrol` | Mixed-target public API and race-detector output |
| RCL-DCS-1 | `TestControllerLifecycle/new_controller_requires_only_new_request_and_facts` | `reconciliationcontrol` | Test output |
| PRL-FPO-1 | `TestPlanReconciler/current_plan_snapshot_precedes_later_boundaries` | `planreconciliation` | Test output |
| PRL-FPO-0 | `TestPlanReconciliationConfiguration/invalid_setup_is_rejected_before_reconciler_exists` | `planreconciliation` | Public constructor test output |
| PRL-FPO-2 | `TestPlanReconciler/no_current_plan_skips_delivery_and_assessment` | `planreconciliation` | Test output |
| PRL-FPO-3 | `TestPlanReconciler/snapshot_failure_skips_later_boundaries` | `planreconciliation` | Test output |
| PRL-FPO-4 | `TestPlanReconciler/unavailable_invalid_or_mismatched_binding_has_stable_failure` | `planreconciliation` | Test output and non-unwrapping assertion |
| PRL-FPO-5 | `TestPlanReconciler/two_identities_use_only_their_exact_bindings` | `planreconciliation` | Test output |
| PRL-FPO-6 | `TestPlanReconciler/wrong_kind_skips_resolver` | `planreconciliation` | Test output |
| PRL-FPO-7 | `TestPlanReconciler/context_error_wins_at_each_boundary` | `planreconciliation` | Table-test output |
| PRL-CPA-1 | `TestPlanReconciler/current_plan_uses_later_delivery_observations` | `planreconciliation` | Test output |
| PRL-CPA-2 | `TestPlanReconciler/plan_control_failure_is_preserved` | `planreconciliation` | Test output |
| PRL-CPA-3 | `TestPlanReconciler/delivery_failure_is_stable_and_skips_assessor` | `planreconciliation` | Test output and non-unwrapping assertion |
| PRL-RPA-1 | `TestPlanReconciler/revise_preserves_proposal_without_application` | `planreconciliation` | Test and dependency output |
| PRL-RPA-2 | `TestPlanReconciler/complete_awaits_without_acceptance_or_mutation` | `planreconciliation` | Test and dependency output |
| PRL-OPR-1 | `TestPlanReconciler/reentry_is_explicit_and_fresh` | `planreconciliation` | Integration test output |
| PRL-OPR-2 | `TestPlanReconciler/external_event_without_request_starts_nothing` | `planreconciliation` | Integration test output |

#### Proposal Success-Criterion Traceability

| Success criterion | Exact evidence | Owner |
|---|---|---|
| SC-1 | RCL-PEC-2, RCL-PEC-3, RCL-PEC-4 tests plus `go test -race ./...` | `reconciliationcontrol` |
| SC-2 | RCL-PEC-1 and RCL-PEC-2 tests | `reconciliationcontrol` |
| SC-3 | RCL-LBA-1, RCL-LBA-2, PRL-FPO-1, and PRL-FPO-5 tests | Generic and Plan packages |
| SC-4 | RCL-ECD-1 through RCL-ECD-7, RCL-FNR-1, and PRL-OPR-2 tests | Generic and Plan packages |
| SC-5 | RCL-TRI-1 compile and black-box test | `reconciliationcontrol` |
| SC-6 | PRL-OPR-1 integration test and the opt-in real Plan verification procedure in task 5.3 | `planreconciliation` and proof owner |
| SC-7 | RCL-CL-1 through RCL-CL-11 tests | `reconciliationcontrol` |

#### Behavior Tests

| Behavior | Given | When | Then | Level |
|---|---|---|---|---|
| Pending duplicates | Same valid target requested repeatedly while no slot is available | Slot becomes available | At least one serialized Attempt runs; no occurrence requires its own Attempt and exact coalescing count is not asserted | Unit |
| Preserve request during active | Attempt blocked by test barrier | Same target requested | No overlap and one later Attempt | Unit/race |
| Bound distinct targets | More distinct targets than capacity | Requests arrive | Active count never exceeds bound and all targets stay isolated | Unit/race |
| Resolve completion/request race | Completion and request barriers release together | Scheduler handles both | At least one later Attempt and no overlap | Unit/race |
| Await | Success returns Await | No later request | No additional Attempt | Unit |
| Immediate | Success returns Immediate | Attempt completes | One later Attempt becomes eligible | Unit |
| Delay | Success returns valid Delay | Fake time advances | Later Attempt starts only after eligibility | Unit |
| Early external request | Target is delayed | Request arrives | Target starts earlier and old delay creates no duplicate | Unit |
| Failure precedence | Target Attempt returns error with arbitrary result and Directive | Controller processes it | One Target Attempt Failed outcome with exact cause, no Completion/Directive, and no later call | Unit |
| Invalid Directive | Delayed constructor receives non-positive duration, or Target Attempt returns a result and zero Directive | Construct/process | Constructor rejects the delay; Controller reports distinguishable Control Directive Rejected for zero Directive and no later call occurs | Unit |
| Cancellation | Active Reconcilers and an undelivered Report exist | Caller cancels while consumer is stopped | Active calls return, undelivered Report may drop, Reports closes, and Wait returns caller context error | Unit/race |
| Nominal return after cancellation | Active Reconciler waits for `ctx.Done` then returns result/Directive | Caller cancels | No Completion/Directive is accepted; Reports closes after return and Wait returns caller context error | Unit/race |
| Submission context | Request is blocked, accepted, or races a stopped Controller | Submission or lifecycle context ends | Only pre-acceptance submission is bounded; accepted Attempt uses Controller context; Controller outcome wins when both ended | Unit/race |
| Complete public lifecycle | Started Controller and prospective Report | Delivery or cancellation commits first under test barrier | Delivery winner publishes once; cancellation winner discards; Reports closes and Wait returns caller context error | Black-box/race |
| Pending Report backpressure | Report consumer blocks while requests and Active returns occur | Public operations continue | No new Attempt starts, Request returns accepted/coalesced, existing Reconciler returns, and cancellation closes Reports/Wait without consumer progress | Black-box/race |
| Delivery/cancellation race | Consumer readiness and cancellation become concurrent | Public lifecycle resolves them | Exactly one: one Report is published and Directive commits, or no Report is published and no Directive-based Attempt starts; lifecycle terminates in both | Repeated black-box/race |
| Normal multi-target delivery | Multiple concurrent targets return while lifecycle and consumer continue | Reports are received | Every outcome arrives once without loss/duplication; no cross-target completion order is asserted | Black-box/race |
| Plan without current Plan | Resolved fresh Snapshot has coherent progress only | Plan Reconciler runs | Delivery Observer and Assessor are not called; Result has Snapshot, no Assessment, Await | Unit |
| Plan assessment | Resolved fresh Snapshot has current Plan | Delivery Observer and Assessor return | Call order is Snapshot then Delivery then Assessment; exact Assessment is preserved and Await returned | Unit |
| Plan reentry | First Plan call completes and external interaction occurs | Caller explicitly requests same target | Second call starts from fresh Snapshot without event/receipt input | Integration |

#### Invariant Tests

| Invariant | Example | Expected result |
|---|---|---|
| One Active Attempt per Target | 100 requests for one target across goroutines | Maximum observed active count for that target is one. |
| Global concurrency bound | 100 distinct targets with bound four | Maximum total active count is four. |
| Completion validity | Reconciler returns result with zero Directive | Failure Report; no reevaluation. |
| Controller lifecycle scope | Host needs a second independent lifecycle | Host calls Start again | A distinct Controller owns the distinct bound and same-target exclusion scope. |
| Report stream identity | Reports is called repeatedly and concurrently | Controller is active or zero | Active Controller returns the same channel; zero Controller returns nil. |
| Plan result association | Assessor captures current Plan | Result Assessment refers to the same Plan from Snapshot. |

#### Error and Edge Tests

| Case | Given | When | Then |
|---|---|---|---|
| Invalid Controller configuration | Nil/done context, zero bound, or nil Reconciler | Start | Stable construction or supplied context error; no lifecycle. |
| Zero Controller | Zero Controller receives Request or Wait | Invoke | Stable not-started error; no goroutine exists. |
| Invalid Target Identity | Request receives zero identity | Submit | Synchronous `ErrInvalidTargetIdentity`; no Report and no Reconciler call. |
| Invalid Delay | Zero or negative duration | Construct | Public constructor rejects it; no Directive or delayed eligibility exists. |
| Stopped Controller | Request occurs after lifecycle cancellation | Submit | Original Controller context error; no attempt starts. |
| Request context | Nil/already-done context, blocked submission timeout, accepted request then submission cancel, or both contexts done | Submit/cancel | Stable precedence and no submission context reaches an accepted Reconciler. |
| Plan binding failure | Wrong kind, resolver error, or invalid PlanTarget | Reconciler runs | Stable target-binding failure; no Observer, Delivery Observer, or Assessor call. |
| Plan observation failure | Observer returns provider-independent failure | Reconciler runs | Zero Result/Directive plus existing Snapshot failure; later boundaries are not called. |
| Plan delivery-observation failure | Delivery Observer fails after valid current Plan | Reconciler runs | Zero Result/Directive plus stable Plan boundary failure; Assessor is not called. |
| Plan assessment failure | Assessor returns failure | Reconciler runs | Zero Result/Directive plus existing Plan Control failure. |

#### Testability Feedback

- Interface concern resolved: one stable Report stream provides normal backpressure and exact-once happy-path delivery, while cancellation may discard undelivered Reports and `Wait` exposes the original lifecycle outcome independently of consumer progress.
- Responsibility concern: exposing a public Clock would create a test-only abstraction; private scheduler time injection is sufficient.
- Coupling concern: Plan application types are absent from `planreconciliation`, allowing a dependency test to enforce the safety boundary.
- Ordering concern resolved: the Plan Module returns the generic Reconciler directly and owns binding resolution plus Snapshot-before-Delivery-before-Assessment invocation.

### 3.8 Detailed Design and TDD Plan

| Implementation unit | Implements concept / responsibility / contract | Dependencies | Behavior / test | Migration or rollback impact |
|---|---|---|---|---|
| `reconciliationcontrol` identity and Directive values | Target Identity and Control Directive invariants | Standard library | Constructor and zero/invalid tests | Additive; remove package on rollback |
| `reconciliationcontrol` Report and Reconciler contracts | Success/failure association and opaque result boundary | Standard library | Result-type and invalid-outcome tests | Additive |
| `reconciliationcontrol` Controller scheduler | Control Loop lifecycle, request operation, Report delivery, Wait outcome, and per-target state | Standard library | Coalescing, races, delay, typed failure, backpressure, cancellation, restart tests | Additive |
| `planreconciliation` PlanTarget, Result, and NewReconciler | Plan Target Binding, Attempt, and Result | Existing public contracts | Resolution, ordered Snapshot/Delivery/Assessment, read-only, reentry tests | Additive |
| Architecture and dependency checks | Component/dependency decisions | Repository docs/build checks | Prohibited-dependency scan and conformance review | Revert canonical doc changes |
| Real Plan verification | SC-6 and proof behavior | Existing public Provider/Actor contracts | Opt-in exact evidence sequence | External state remains Actor-owned |

| Implementation unit | Simplest viable representation | State / identity / lifecycle / boundary need | Rejected simpler alternative | Verdict |
|---|---|---|---|---|
| TargetIdentity | Immutable value | Stable correlation and validation | Two raw strings spread through APIs | Pass |
| Directive | Immutable discriminated value | Invalid/positive-delay invariant | Boolean requeue plus optional duration | Pass |
| Reconciler[R] | Named function type | Consumer-owned target boundary with multiple concrete Consumers | Universal result interface or broad Service | Pass |
| Report[R] | Immutable discriminated value | Exact target/success/failure association | Separate uncorrelated result and error channels | Pass |
| Failure | Immutable discriminated value | Reconciler/protocol distinction without scheduling inference | Raw unclassified error | Pass |
| Controller[R] | One started stateful lifecycle | Multi-request lifecycle, concurrency, backpressure, and terminal outcome invariants | Reusable Run configuration or stateless function per request | Pass |
| PlanTarget[O] | Immutable validated boundary association | Identity-bound ordered collaboration | Host map lookup inside an adapter | Pass |
| Plan Result | Immutable value | Snapshot/Assessment association | Assessment alone | Pass |
| Plan NewReconciler | Factory returning the generic Port implementation | Cohesive binding and ordered read-only composition | Host adapter or Plan Controller expansion | Pass |

| Behavior / criterion | Construction mode | Red test or verification | Smallest implementation | Refactor target / evidence |
|---|---|---|---|---|
| Target and Directive validity | TDD | Constructor tables fail against scaffold | Values and constructors | Remove duplicated validation |
| Result isolation | TDD | Two unrelated generic result tests fail | Report and Reconciler types | Keep generic layer result-opaque |
| Per-target scheduling | TDD | Same-target barrier test fails | Minimal keyed state scheduler | Centralize state transitions, not helpers |
| Active-period request preservation | TDD | Completion/request race test fails | Pending marker under scheduler ownership | Remove channel-order assumptions |
| Immediate and delayed directives | TDD | Directive behavior tests fail | Ready queue plus one private delay mechanism | Consolidate timer invalidation |
| Failure and cancellation | TDD | Typed no-retry, stopped-consumer, and bounded-stop tests fail | Explicit failure branch, interruptible Report delivery, and active wait | Preserve failure/request/directive distinction |
| Plan binding, no-current, and assessment branches | TDD | Resolution and ordered Plan table tests fail | PlanTarget, Result, and NewReconciler | Keep existing Snapshot and Assessment owners unchanged |
| Application exclusion | Verification | Dependency/conformance check | No application import or call | Evidence in design review |
| Real Plan sequence | Non-automated verification | Recorded input/evidence method | Opt-in Host composition | Disposable proof document |

---

## 4. Design Decisions

### 4.1 Generic control owns scheduling but not target meaning

- Adopted: `reconciliationcontrol` receives identity-only requests and target-specific Reconciler outcomes.
- Rejected: A universal Reconciliation result, Condition model, Observation hierarchy, or Adjustment interface.
- Reason: The Controller needs only target identity and Directive to protect its invariants. Target result classifications have independent owners and change drivers.

### 4.2 A request is a wake-up, not an event record

- Adopted: The concurrency-safe Request operation accepts only TargetIdentity and synchronously rejects an invalid identity.
- Rejected: Event type, cause, payload, receipt, prior result, or generation embedded in Request.
- Reason: Each Attempt must derive a level-based decision from current facts. Preserving event data would invite edge-triggered behavior and cross-Context authority leakage.

### 4.3 Directive is separate from result and error

- Adopted: Every success has one explicit Await, Immediate, or Delay Directive; failure has none.
- Rejected: `error` retryability, boolean `requeue`, or target result inspection.
- Reason: Explicit scheduling prevents implicit retries and keeps target outcome evolution out of the generic Component.

### 4.4 One scheduler owns all per-Controller mutable state

- Adopted: One scheduler goroutine owns target states, the concurrency count, ready eligibility, delay ordering, and at most one Pending Delivery Report; Attempt goroutines return immutable outcomes through an internal channel bounded by the concurrency limit.
- Rejected: One worker, timer goroutine, or mutex-protected state machine per target.
- Reason: Central ownership makes completion/request races deterministic and avoids hidden state across ad hoc goroutines. Active Attempt count remains bounded.

### 4.5 Internal state is disposable

- Adopted: Requests, pending markers, delays, and pending Report delivery exist only for one started Controller lifecycle.
- Rejected: Repository, checkpoint, lease, leader, or restoration API.
- Reason: Arcloom owns no authoritative control state. External event sources or Hosts request current evaluation again after restart.

### 4.6 Plan reconciliation is read-only composition

- Adopted: `planreconciliation` resolves a Plan Target Binding, performs exactly one fresh Snapshot observation, and only for a valid current Plan obtains Delivery Observations and an optional Plan Control assessment, then always Await.
- Rejected: Plan application, receipt interpretation, or automatic wake-up from any application result.
- Reason: Existing application safety is at-most-once only within one invocation. An automatic repeated Attempt could resend a prior uncertain Revision without cross-invocation idempotency. Explicit application remains separately authorized and externally owned.

### 4.7 Plan composition has its own owner

- Adopted: A Plan Reconciliation Attempt Module synchronously validates target kind, Resolver, and Assessor setup; resolves a self-identifying binding; verifies requested/resolved identity equality; owns ordered Snapshot/Delivery Observation/Assessment calls and stable runtime composition failures; and returns the generic Reconciler implementation directly.
- Rejected: Host lookup and adapter branching, expanding Plan Controller to observe, or making Snapshot Observation assess a Plan.
- Reason: The exact target binding, invocation order, and association between one fresh Snapshot and optional Assessment are Plan-specific invariants. Existing Components intentionally own only one part, while the Composition Root only supplies dependencies.

### 4.8 Go generics, a lifecycle object, and channels express the control contract

- Adopted: A generic result type, named Reconciler function, one started Controller with Request/Reports/Wait operations, and a typed Report channel.
- Rejected: `any` result with type assertions, a broad Service interface, reflection, or Python-style callback router.
- Reason: Go generics preserve unrelated result types at compile time. The lifecycle object makes request rejection and terminal context outcome explicit; the Report channel retains typed stream delivery while cancellation can interrupt backpressure.

### 4.9 Report delivery is the Directive commit point

- Adopted: The Controller retains at most one Pending Delivery Report and starts no new Attempt while it is pending. Delivery atomically publishes the Completion or Failure and only then applies a successful Directive. The Controller still accepts and coalesces requests, receives Active returns into a concurrency-bounded internal channel, and observes cancellation. Cancellation may discard the prospective Report and unapplied Directive without publishing an outcome; the Controller then awaits Active Reconcilers, closes Reports, and returns the caller context error from Wait.
- Rejected: Unbounded result buffering or allowing a stopped Report consumer to block caller cancellation.
- Reason: One explicit commit point makes slow-consumer behavior deterministic and bounds in-memory delivery state. Arcloom owns no durable outcome queue, so undelivered in-memory Reports and Directives cannot outrank termination.

### 4.10 Kubernetes is a behavioral reference, not a storage template

- Adopted: level-based target requests, current-state evaluation, per-target serialization, coalescing, explicit reevaluation, and state-of-world tests.
- Rejected: copying API Server resources, `resourceVersion`, `spec/status`, finalizers, leader election, or automatic error backoff without an Arcloom requirement.
- Reason: Arcloom does not own the external source of truth or a cluster control plane. Authorization and external application remain stronger separate boundaries.

---

## 5. Impact, Migration, and Rollback

### Impact

- Adds `reconciliationcontrol` and `planreconciliation` Packages.
- Updates `ARCHITECTURE.md` with the Reconciliation Controller Component, Plan Reconciliation Attempt Module, Port ownership, and permitted dependencies.
- Existing `reconciliation`, `plansnapshot`, `plancontrol`, `authorization`, and `planapplication` public contracts remain compatible.
- No Provider contract, database, durable data, or external target migration exists.

### Migration

- Hosts opt into the Controller and submit target identities from their existing event sources.
- A Plan Host supplies a Target Resolver of self-identifying validated Snapshot and Delivery Observation bindings plus one Plan Control Assessor; the Plan Module owns correspondence checks and invocation order, and the Host explicitly submits later requests.
- Existing one-shot callers continue to use their current APIs.

### Rollback

- Remove the two additive Packages and Architecture entries.
- Existing one-shot observation, reconciliation, assessment, Authorization, and application behavior remains available.
- No external state rollback is performed because this Change adds no automatic mutation.

### Open Questions

None.
