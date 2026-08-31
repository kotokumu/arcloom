## Context

The repository already provides the four behavior owners needed inside one Plan cycle:

- `reconciliationcontrol.Controller[planreconciliation.AttemptResult]` owns identity-only Request scheduling, target exclusion, Report publication, and caller cancellation.
- `planreconciliation.NewAttempt` owns target resolution, fresh Snapshot observation, later delivery observation, Plan Control assessment, and Plan Attempt Result classification.
- `githubplan.NewMilestoneSnapshotObserver` implements fresh GitHub Plan observation.
- `codexplancontrol.NewAssessor` uses `codexappserver.Client`, whose SDK owns process lifecycle, JSON-RPC, correlation, and bounded shutdown.

There is no executable Composition Root, no repository-owned Host that consumes the Controller Report stream, and no Plan-specific Result Destination boundary. The existing integration test proves only two directly wired Controller cycles; it does not exercise a runnable Host, destination failure, or deterministic convergence through at least three fresh cycles.

This is a high-risk design because it adds a caller-scoped runtime owner across existing lifecycle boundaries. `cmd/arcloom-plan` and `internal/planhost` together form one reference Arcloom Host application outside the Arcloom Runtime Component graph defined by `ARCHITECTURE.md`; they do not add a Runtime Component. The design keeps this application boundary internal and provisional. Milestone #3, not this change, decides the final Module and public Package structure after operational evidence exists.

## Goals / Non-Goals

### Goals

- Add one repository-owned reference Host that can accept repeated explicit triggers for one exact Plan target.
- Consume the existing Controller Report stream once and route only Current Plan Assessed to one Plan-specific destination.
- Preserve Controller scheduling, Plan Attempt semantics, GitHub fact authority, Codex SDK lifecycle ownership, and External Actor mutation authority.
- Provide a deterministic local integration environment that proves at least three fresh cycles converge to Complete after explicit external progress.
- Provide an executable Composition Root for real GitHub observation and Codex assessment through the same Host contract.

### Non-Goals

- Finalizing the Reconciliation Module taxonomy, public Host API, or Controller-development Interfaces.
- Adding a universal Result, Result Destination, Observation, Failure, workflow, or trigger abstraction.
- Moving process, stdio, JSON-RPC, correlation, or shutdown into the Host.
- Authorization, Plan application, Task execution, GitHub mutation, automatic polling, retries, durable queues, or persisted loop history.
- Making deterministic test collaborators production abstractions.

## Conceptual-Model-to-Implementation Mapping

| Specification concept / Requirement | Owning component | Physical representation | Notes |
|---|---|---|---|
| Plan Feedback Loop Operation | Reference Host application | A caller-scoped `Host` in `internal/planhost` | It is application code outside the Arcloom Runtime Component graph; internal placement avoids claiming the final Module boundary before Milestone #3. |
| Explicit Plan Trigger / `explicit-plan-trigger` | Reference Plan Host | `Host.Trigger(context.Context) error` | The method has no payload or target parameter; it can submit only the configured identity. |
| Deliverable Plan Result / `assessed-plan-result-delivery` | Plan Control plus Plan Feedback Loop Operation | Host classifies `AttemptResult.Kind()`, then extracts the existing exact `plancontrol.Assessment` | Plan Control owns Assessment meaning; no shared Result type or Snapshot exposure is introduced. |
| Plan Result Destination | Plan Feedback Loop Operation | Plan-specific function type receiving exact Target Identity and exact `plancontrol.Assessment` | It is not an application or External Actor Port. |
| Plan Result Delivery | Reference Plan Host | One synchronous destination invocation after the corresponding Controller Report is received; the processed Report becomes externally readable only after success | Delivery state remains invocation-local. |
| Exact Plan operation configuration | Plan Attempt Module plus Reference Host | One immutable `planreconciliation.PlanAttemptBinding` contains the exact Target Identity and Attempt constructed from the same `PlanTarget`; destination and caller lifecycle are supplied separately | The Host cannot receive an independently paired identity and Attempt and does not construct the Attempt. |
| Caller-scoped operation lifecycle | Reference Plan Host | Host-owned child context, Controller, report pump, terminal outcome, and `Wait` | All state is disposable. |
| Deterministic local verification | External-package integration tests | Test-local controlled Planning Context, External Actor operations, assessor, trigger timing, and recording destination | These are specific fixtures in `_test.go`, not production Interfaces or a test-helper Package. |
| Real GitHub/Codex operation | Reference Host application | A command-local GitHub binding adapter creates the Plan Target from existing provider-independent Snapshot values; the Composition Root wires it with the Codex assessor, Host, and terminal destination | GitHub HTTP and representation mapping remain in `githubplan`; the adapter only selects caller-owned `plansnapshot.ProgressEvidence`. |

## Responsibility Assignment

| Responsibility / decision | Owner | Information and authority used | State / invariant preserved | Candidate excluded and reason |
|---|---|---|---|---|
| Accept or reject a trigger and schedule Attempts | Reconciliation Controller | Exact Target Identity, lifecycle, eligibility, concurrency, Directive | Existing exclusion, coalescing, Report publication, and cancellation invariants | Reference Host does not add a scheduler, queue, timer, or retry policy. |
| Acquire and classify one current Plan evaluation | Plan Attempt | Exact binding, fresh Snapshot, post-Snapshot observations, assessor | Current Plan Not Established differs from Current Plan Assessed and Failure | Host does not inspect external facts or perform Plan judgment. |
| Decide whether a published Plan Attempt Result is deliverable | Plan Feedback Loop Operation | Existing Attempt Result kind from the exact published Report | Only Current Plan Assessed is semantically delivered | Controller remains result-opaque; Destination does not classify. |
| Deliver one assessed Plan Result | Plan Result Destination | Exact target and exact Plan Control Assessment extracted from Current Plan Assessed | One invocation per qualifying Report at most; no Authorization or mutation authority | External Actor is not invoked and Plan application is not smuggled into delivery. |
| Stop after delivery failure | Reference Plan Host | Destination return, caller lifecycle, Controller lifecycle | Failure is distinct, intake stops, no retry is created | Controller Failure and Directive do not represent destination failure. |
| Change authoritative Task progress between cycles | External Actor acting on Planning Context | Provider-native facts and permissions | Arcloom remains read-only and external facts remain authoritative | Host and local destination cannot mutate production facts. |
| Own GitHub HTTP and representation mapping | `githubplan` | GitHub target and REST contract | Fresh provider facts do not leak as DTOs | Host only wires the observer. |
| Own Codex process and JSON-RPC lifecycle | `codexappserver` | Executable, protocol, process authority, shutdown bound | Isolated read-only Turn and bounded cancellation | `codexplancontrol` and Host receive only the SDK Client contract. |
| Translate assessment material and Codex output | `codexplancontrol` | Exact Plan, caller-owned observations, completed Turn | Provider output cannot weaken Plan Control result semantics | SDK remains independent of Plan vocabulary. |
| Adapt concrete GitHub observations into caller-owned progress vocabulary | Reference Host application's GitHub binding adapter | Existing `githubplan` Snapshot Observer and provider-independent `plansnapshot.ProgressEvidence` | Every call is fresh and no GitHub DTO crosses the adapter | Composition Root only calls the adapter constructor; `githubplan` retains HTTP and representation mapping. |
| Parse deployment configuration and instantiate concrete dependencies | `cmd/arcloom-plan` Composition Root | Operator-supplied target, credentials, Codex configuration, output selection | No domain or scheduling decision is duplicated | Composition Root does not classify Results, map GitHub facts, construct JSON-RPC, or implement lifecycle rules. |

## Package and Dependency Design

| Package | Responsibilities implemented | Contracts published | Implementation hidden | Dependencies permitted |
|---|---|---|---|---|
| `internal/planhost` | Provisional external reference Host lifecycle, sole Controller Report consumption, exact Assessment routing, processed-Report observation, and destination-failure termination | Internal `Host`, `Start`, `Trigger`, `Reports`, `Wait`, `DeliveryError`, and Plan-specific destination function type | Controller construction, bounded processed-Report buffer, terminal race resolution, cancellation coordination | `context`, `errors`, `sync`, `plancontrol`, `planreconciliation`, `reconciliationcontrol` |
| `planreconciliation` | Existing Plan Target and Attempt ownership plus exact single-target executable binding | Add immutable `PlanAttemptBinding` and its constructor/accessors | Target, resolver closure, identity/Attempt association, and validity remain private | Existing dependencies only |
| `cmd/arcloom-plan` | Executable Reference Host application and Composition Root | Command invocation only; no reusable product contract | Flag/environment parsing, command-local GitHub binding adapter, credential transport, terminal representation, OS cancellation | `internal/planhost` and existing concrete Provider packages |
| `internal/planhost` external tests | Deterministic local feedback-loop verification | No production contract | Controlled authoritative facts, explicit Actor changes, assessment calculation, destination recording | Public contracts of packages under test |

| Source | Target | Contract used | Why | Details that must not cross |
|---|---|---|---|---|
| `internal/planhost` | `reconciliationcontrol` | `Controller`, `TargetIdentity`, `Report` | Delegate all scheduling and Report publication | Host destination state never enters Controller Result or Directive. |
| `internal/planhost` | `planreconciliation` | `PlanAttemptBinding`, `AttemptResult` | Start control from one inseparable exact identity/Attempt association and classify only its public branch | Plan Target, Snapshot, assessor, and provider details are not interpreted by Host. |
| `internal/planhost` | `plancontrol` | `Assessment` | Deliver the exact semantic Plan Control Result without the enclosing Attempt/Snapshot | Codex types never enter Host. |
| `cmd/arcloom-plan` | `githubplan` | Milestone target and Snapshot Observer | Bind the real Planning Context | GitHub DTOs and errors remain inside `githubplan`. |
| `cmd/arcloom-plan` | `codexplancontrol` / `codexappserver` | Assessor construction and SDK Client | Bind real Plan Control without owning Codex lifecycle | JSON-RPC, process, and protocol values remain in the SDK. |

No package named `helper`, `util`, `shared`, `common`, `manager`, or `processor` is introduced. The internal Host is named for its exact integration responsibility and is intentionally not the final public module boundary.

## Interface Design

The following signatures are design targets, not a generic SDK contract.

```go
package planreconciliation

var ErrInvalidPlanAttemptBinding = errors.New("invalid Plan Attempt Binding")

// PlanAttemptBinding is an immutable exact association between one valid Plan
// Target Identity and the configured Attempt constructed from that same target.
// Its zero value is invalid.
type PlanAttemptBinding struct { /* private identity and attempt */ }

// NewPlanAttemptBinding validates target and assessor, then constructs the
// exact single-target resolver and Attempt without external I/O. Every invalid
// input or internal construction invariant failure returns the zero Binding and
// ErrInvalidPlanAttemptBinding. The returned Attempt admits only Identity().
func NewPlanAttemptBinding[O any](
    PlanTarget[O],
    plancontrol.Assessor[O],
) (PlanAttemptBinding, error)

func (b PlanAttemptBinding) Identity() reconciliationcontrol.TargetIdentity
func (b PlanAttemptBinding) Attempt() reconciliationcontrol.Attempt[AttemptResult]
```

```go
package planhost

var (
    ErrHostNotStarted = errors.New("plan host not started")
    ErrInvalidPlanResultDestination = errors.New("invalid Plan Result Destination")
)

// DeliveryError identifies a Plan Result Destination failure. Unwrap returns
// the caller-owned destination cause. It never represents Controller Failure
// or caller lifecycle cancellation.
type DeliveryError struct { /* private immutable cause */ }
func (e *DeliveryError) Error() string
func (e *DeliveryError) Unwrap() error

// PlanResultDestination receives only the exact valid Assessment extracted
// from CurrentPlanAssessed. Nil means this invocation completed its own
// destination processing; it does not establish Authorization, application,
// target mutation, or any other externally authoritative effect.
type PlanResultDestination func(
    context.Context,
    reconciliationcontrol.TargetIdentity,
    plancontrol.Assessment,
) error

// Start validates one exact Plan Attempt Binding, destination, and caller
// lifecycle, then starts one caller-scoped Host.
func Start(
    context.Context,
    planreconciliation.PlanAttemptBinding,
    PlanResultDestination,
) (*Host, error)

// Trigger submits an ordinary identity-only Request for the configured target.
func (h *Host) Trigger(context.Context) error

// Reports exposes exact target-bound Reports after their operation handling is
// complete. For CurrentPlanAssessed, receipt therefore proves that its one
// destination invocation returned nil. A failed or cancelled delivery exposes
// no processed Report for that Controller Report and terminates the stream.
func (h *Host) Reports() <-chan reconciliationcontrol.Report[planreconciliation.AttemptResult]

// Wait returns the committed caller lifecycle error or a distinct Plan Result
// delivery failure after Controller and delivery work have ended.
func (h *Host) Wait() error
```

| Contract | Consumer | Owner | Inputs / preconditions | Output / postconditions | Error / side effects | Constraint protected |
|---|---|---|---|---|---|---|
| `PlanAttemptBinding` | Reference Host Composition Root and Host | Plan Attempt Module | Validate Plan Target first, then assessor | Valid input returns an immutable exact Target Identity plus Attempt association; its Attempt admits only that identity | Zero/invalid target, nil assessor, or internal construction invariant failure returns zero Binding and `ErrInvalidPlanAttemptBinding`. Construction performs no observation or assessment; a different requested identity fails as Target Binding Unavailable before observation. | Prevents independent identity/Attempt pairing and keeps single-target construction out of Host. |
| `Host.Start` | Reference Host Composition Root | Reference Plan Host | Non-nil active context, valid nonzero `PlanAttemptBinding`, non-nil destination | One started Host and stable Report stream | Validation order is context presence, caller lifecycle, binding, destination. Nil context returns `reconciliationcontrol.ErrInvalidContext`; ended context returns its exact error; invalid binding returns `planreconciliation.ErrInvalidPlanAttemptBinding`; nil destination returns `ErrInvalidPlanResultDestination`. Every failure starts no goroutine, Controller, stream, observation, assessment, or delivery. | All-or-nothing caller-scoped startup and exact target association. |
| `Host.Trigger` | Explicit Trigger Source | Reference Plan Host | Running Host and active submission context | Existing Controller acceptance result | Zero Host returns `ErrHostNotStarted`; submission cancellation returns its context error unless a Host terminal outcome already committed | Trigger cause cannot become an Attempt fact. |
| `Host.Reports` | Reference command and tests | Reference Plan Host | Started Host | Stable stream of exact successfully handled Reports in Controller order | Zero Host returns nil; a one-entry internal buffer prevents the current destination from depending on reader speed; cancellation may discard an unprocessed Report | Makes one-cycle completion observable without competing for Controller's stream. |
| `PlanResultDestination` | Reference Plan Host | Plan Feedback Loop Operation | Exact target, exact valid Assessment, active lifecycle | Nil means destination invocation processing returned successfully | Non-nil while caller lifecycle remains active becomes `DeliveryError`; possible destination-owned effects remain unspecified | Separates semantic delivery from Report publication and external application. |
| `Host.Wait` | Host operator | Reference Plan Host | Started Host | Stable terminal caller-lifecycle or `DeliveryError` outcome for repeated and concurrent calls | Zero Host returns `ErrHostNotStarted`; no retry or persistence | Makes delivery failure and cancellation unambiguous. |

### Error and terminal precedence

| Competing condition | Committed terminal meaning |
|---|---|
| Caller lifecycle ends before destination success or failure is accepted | Caller lifecycle error; destination return cannot replace it. |
| Destination returns non-nil while caller lifecycle is active | `DeliveryError` wrapping that exact cause, even when the cause itself matches a context sentinel. |
| Destination returns nil while caller lifecycle is active | Delivery succeeds; the handled Report becomes available and the Host remains Running. |
| Delivery failure has committed before caller cancellation | The same `DeliveryError` remains the stable Host terminal outcome. |
| Host has stopped and `Trigger` is called | Return the stable Host terminal outcome; start no Request. |

The zero Host is unusable: `Trigger` and `Wait` return `ErrHostNotStarted`, and `Reports` returns nil. `Trigger`, `Reports`, and repeated or concurrent `Wait` calls are safe for concurrent use.

### Representative consumer flows

```go
binding, err := planreconciliation.NewPlanAttemptBinding(target, assessor)
if err != nil { return err }
host, err := planhost.Start(lifecycle, binding, destination)
if err != nil { return err } // Includes exact ended-lifecycle/configuration errors.

// One cycle: receiving an assessed Report means destination processing returned nil.
if err := host.Trigger(submission); err != nil { return err }
report, ok := <-host.Reports()
if !ok {
    waitErr := host.Wait()
    var deliveryErr *planhost.DeliveryError
    if errors.As(waitErr, &deliveryErr) { return deliveryErr }
    return waitErr // Caller lifecycle result.
}
_ = report // Inspect Completion or Failure, then end the one-shot lifecycle.

// Repeated cycles: change external facts only through the External Actor,
// trigger again, and wait for the next successfully handled exact Report.

// Failure/cancellation: a closed Reports stream without the expected handled
// Report is resolved through Wait as DeliveryError or the caller context error.
```

## Decisions

### Decision: Keep the reference Host internal until Milestone #3

- **Choice**: Implement the accepted operation under `internal/planhost` and expose only the executable command as a deployment entry point.
- **Rationale**: #2 needs running evidence, while #3 owns remodelling, refactoring, and final module/interface boundaries. An internal package permits coherent tests without prematurely making a provisional Host API a compatibility promise.
- **Alternatives**: A public `planfeedback` package would freeze a boundary before the evidence review. Keeping all logic in `main` would make the Composition Root own behavior and obstruct focused tests.
- **Consequences**: #3 may move or replace the internal package. #2 tests must protect behavior rather than private structure.

### Decision: Reuse the generic Controller unchanged

- **Choice**: Start it with concurrency one and the Plan Attempt produced from the single configured Plan Target.
- **Rationale**: Exact requests, coalescing, exclusion, publication, Directives, and cancellation already have accepted semantics and tests.
- **Alternatives**: A Plan-specific scheduler duplicates control decisions. Adding Result delivery to `reconciliationcontrol` makes target-independent control interpret target semantics.
- **Consequences**: Trigger occurrences remain level-based and may coalesce. The Host does not promise one Attempt per trigger.

### Decision: Expose exact Reports only after operation handling completes

- **Choice**: The Host is the Controller stream's sole consumer. For Current Plan Assessed it invokes the destination after the Controller publication commits and exposes the exact Report through a one-entry processed-Report buffer only after destination success. Other Reports are exposed after classification. Delivery failure or cancellation exposes no successfully handled Report for that Controller Report.
- **Rationale**: Receiving a Host Report is a deterministic one-cycle completion signal, no competing consumer can steal a Controller Report, and a slow observer cannot block the destination for the current Report.
- **Alternatives**: Publishing before destination creates a race for one-shot consumers. Two consumers on the Controller channel race. A generic event/outcome wrapper creates an unnecessary cross-stage model.
- **Consequences**: An unread processed Report can eventually backpressure a later Report, but it never blocks delivery for the Report already being handled; cancellation must still select over publication and avoid waiting for the observer.

### Decision: Fail-stop on Plan Result Destination failure

- **Choice**: Record a Plan-specific delivery failure, cancel the child Controller lifecycle, reject later triggers, await active boundaries, and return that failure from `Wait`.
- **Rationale**: Continuing silently would lose feedback while appearing healthy. Fail-stop makes failure observable without durable replay or automatic retry.
- **Alternatives**: Automatic retry is explicitly out of scope and cannot establish exactly-once external effect. Continuing after failure requires another observable failure stream and a dropped-result policy not supported by current requirements.
- **Consequences**: Recovery starts a new disposable Host and performs a fresh ordinary Request. The prior Result is never replayed; the destination may have acted before returning an error, so no external-effect claim is made.

### Decision: Keep deterministic collaborators local to integration tests

- **Choice**: Model authoritative Plan facts, External Actor progress, assessment calculation, trigger timing, and destination recording as specifically named test fixtures in the external integration test file.
- **Rationale**: The environment needs shared invariants but is not a production capability or general simulation framework.
- **Alternatives**: A production fake/simulation package adds abstractions used only by tests. Unrelated mocks reproduce implementation call order and make refactoring harder.
- **Consequences**: Tests can inspect fresh observation inputs and delivery records while remaining coupled only to public behavior. If multiple future capabilities need the same environment, #3 can reconsider a bounded test-support package from evidence.

### Decision: Make the real executable a thin one-cycle composition

- **Choice**: A command-local GitHub binding adapter builds one Milestone Plan target and a fresh post-Snapshot observer that exposes only provider-independent Progress Evidence. `cmd/arcloom-plan` wires that target with one Codex assessor through the Go SDK, the configured Attempt, reference Host, and terminal Plan Result Destination, then submits one explicit trigger per invocation.
- **Rationale**: One-shot invocation is a concrete explicit Request source, works with disposable state, and permits repeated real cycles after external progress without inventing polling or a command protocol.
- **Alternatives**: A daemon, stdin event protocol, GitHub webhook server, or scheduler adds deployment policy and lifecycle scope not needed for #2.
- **Consequences**: The real proof invokes the command initially and at least twice later. Repeated invocations use the same Host contract but never reuse prior Result state.

## Behavioral Test Specification

| Requirement | Given | When | Then | Test level |
|---|---|---|---|---|
| `plan-feedback-loop-operation/exact-plan-operation-configuration` | Invalid binding inputs; or nil/already-ended Host context, zero Binding, or nil destination | Binding construction or Start is requested | Constructor returns its stable zero/error contract; Host startup fails with its specified error before observation or Controller work | Unit / external-package |
| `plan-feedback-loop-operation/explicit-plan-trigger` | A running Host and controlled target | Initial and later triggers are submitted | Attempts receive only the configured identity; duplicates obey existing Controller behavior | Integration |
| `plan-feedback-loop-operation/assessed-plan-result-delivery` | Assessed, no-current, and Attempt Failure Reports | Host consumes each | Only exact Assessment reaches destination; exact Report is exposed after handling; at most one invocation per qualifying Report | Unit and integration |
| `plan-feedback-loop-operation/delivery-failure-and-fresh-reentry` | Destination fails after one assessed Report | Delivery returns failure, then a new Host is started | First Host stops without retry; new Host observes fresh facts and does not replay | Integration |
| `plan-feedback-loop-operation/caller-scoped-operation-lifecycle` | Idle, active observation, blocked Report forwarding, and in-flight delivery states | Caller cancels | Intake ends; active boundary returns; streams close; `Wait` preserves cancellation | Unit with channels; race test |
| Multi-cycle acceptance | One valid Plan with three externally controlled progress states | Initial trigger, External Actor progress, later trigger, further progress, third trigger | Observation count is three, assessment input reflects each fresh state, destination receives three exact assessed results, final outcome is Complete | Deterministic integration |
| Real acceptance | GitHub Milestone #2 and configured Codex SDK | Command is run initially and after at least two external progress changes | Same contracts yield fresh Reports and destination observations, with evidence recorded outside product state | Manual reproducible verification |

### Deterministic convergence matrix

| Cycle | Source-owned Planning revision before trigger | External Actor operation since prior cycle | Fact-derived expected Assessment | Expected processed Report and delivery |
|---|---|---|---|---|
| Initial | S0: all three Tasks Open | None | Retain | Exact S0 Assessment delivered once; exact handled Report exposed. |
| First later | S1: first Task Closed, two Open | Actor closes only the first Task | Retain | Exact S1 Assessment delivered once; no S0 material enters assessment. |
| Second later | S2: all three Tasks Closed | Actor closes only the remaining Tasks | Complete | Exact S2 Assessment delivered once; final handled Report is Complete. |

The simulated Planning Context records every revision writer. The integration test asserts that the only S0→S1 and S1→S2 writes are the two named External Actor operations and that Host startup, triggers, observation, assessment, destination success, destination failure, and cancellation produce no Planning revision.

### Error, boundary, and concurrency verification

| Contract edge | Deterministic control | Required assertion |
|---|---|---|
| Cancellation versus destination | Channels establish cancel-first, success-first, and failure-first commit orders separately | Cancel-first exposes no processed Report and `Wait` returns the lifecycle error; success-first exposes the processed Report, then cancellation makes `Wait` return the lifecycle error; failure-first exposes no processed Report and preserves `DeliveryError`. |
| Same semantic Assessment in distinct Reports | Two fresh Attempts deliberately produce equal Assessments | Each qualifying Report invokes the destination once, for two total invocations; cardinality is per Report, not per value. |
| Current Plan Not Established recovery | First observation has coherent progress without a current Plan; Actor establishes a valid current Plan; later trigger occurs in the same Running Host | First Report causes no delivery; the later Attempt sees only the new revision and delivers its Assessment. |
| Attempt Failure recovery | First boundary fails; source recovers; later trigger occurs in the same Running Host | Failure creates no delivery or retry; later Attempt performs fresh observation and can succeed. |
| Trigger during In Flight delivery | Destination is channel-blocked while another trigger is accepted | Trigger carries no payload; if the earlier delivery succeeds, later work follows Controller semantics; if it fails first, Host cancellation establishes no later handled success. |
| Public ordering | Destination entry and processed `Host.Reports` receipt are channel-gated | Destination cannot enter before Controller publication; assessed Host Report cannot be received before destination returns nil. No private call sequence is asserted. |
| Delivery failure after possible destination effect | Destination records an invocation then returns an error while the caller lifecycle remains active | One invocation, no retry, no Planning Context revision by Host, no processed Report, and stable `DeliveryError`. |

### Real verification matrix

| Item | Required record |
|---|---|
| Owner and target | Operator identity, exact native GitHub repository/Milestone reference, and execution timestamp for each run. |
| Three execution points | Initial run, first later run after a named provider-native Task change, and second later run after another named Task change. |
| Preconditions | Provider-native Task names/states before each run and the configured Codex SDK version/model boundary, without recording secrets. |
| Observable output | Exact Report classification, exact Plan Control Assessment classification, destination completion, command exit outcome, and native references used as evidence. |
| Acceptance | Fresh observation evidence differs by the named external changes; no Codex prose match is required; the final contract classification is Complete. |
| Storage meaning | The record is disposable operator-owned verification, contains no `arcloom-plan:v1` or other required Arcloom metadata, and is not authoritative Planning Context. |

### Requirement and invariant coverage

| Subject | Owning verification |
|---|---|
| Exact single-target configuration and zero work on rejection | `PlanAttemptBinding` constructor/exact-identity tests, Host startup unit tests, and command configuration tests |
| Identity-only repeated triggers, coalescing, and no prior-cycle facts | Trigger concurrency tests plus S0/S1/S2 integration |
| Per-Report assessed-only delivery and exact Assessment | Branch table tests, equal-Assessment distinct-Report test, and convergence integration |
| No Authorization, application, Task execution, or Host/destination mutation | Planning revision-writer history plus dependency/code review |
| Delivery error distinction, no retry, and uncertain destination effect | Delivery failure tests and stable `DeliveryError` assertions |
| Caller cancellation, terminal precedence, and disposable runtime | Deterministic winner tests, repeated cancellation, concurrent repeated `Wait`, post-stop `Trigger`, stopped-consumer test, and fresh restart test |
| Real GitHub/Codex operation through owned boundaries | Real verification matrix and command contract tests |

The convergence assessor is derived from the exact current Plan and current simulated progress; it is not scripted by invocation number. Each observation records a distinct revision so a cached Snapshot fails the test. Channels and explicit Actor methods coordinate semantic commit boundaries; no wall-clock sleep is used. `go test -race ./...` is an additional data-race detector, not proof of terminal-winner, exclusion, or delivery cardinality semantics.

Implementation follows Red-Green-Refactor in this order: invalid startup, assessed-only delivery, report ordering, delivery fail-stop, cancellation races, multi-cycle convergence, then real composition. Tests assert public behavior and do not require private function call order.

## Independent Evolution Scenario Review

| Change scenario | Primary owner | Expected propagation | Design verdict |
|---|---|---|---|
| Trigger source changes from command invocation to webhook or another process | Host integration | New source calls `Trigger`; Attempt input remains identity-only | Pass; no Trigger Source Interface is added prematurely. |
| Duplicate or active-time triggers increase | Reconciliation Controller | Existing exclusion and coalescing tests plus Host integration test | Pass; Host adds no queue or counter. |
| Plan becomes temporarily not established and later recovers | Plan Attempt and Planning Context | Reports show no-current; later trigger observes fresh facts; no destination delivery for the earlier report | Pass. |
| Destination implementation changes or temporarily fails | Plan-specific destination and Host lifecycle | Destination implementation changes; failure stops Host; new Host observes fresh state | Pass; no generic destination or replay queue. |
| GitHub observation is partial or unavailable | GitHub Snapshot and Plan Attempt | Existing Snapshot/Attempt failure classification; Host routing unchanged | Pass. |
| Codex protocol, executable, or process behavior changes | Codex app-server SDK | SDK implementation and contract tests only unless completed-Turn semantics change | Pass; Host never sees JSON-RPC or process state. |
| Deployment changes from one-shot command to a long-running service | Host integration | A new source/lifecycle composition reuses operation behavior; no current service abstraction is added | Evidence-backed risk only; #3 evaluates after #2 proof. |
| Multiple Plan targets are needed in one Host | Host integration and Controller | Current reference Host remains one target; a multi-target operation requires new requirements | Deferred; current Controller capability is not used to justify a speculative Host API. |

## Risks / Trade-offs

- The internal Host is a provisional procedural coordination point → keep it internal, assign every decision above, and require #3 modelling before promotion or splitting.
- Processed Report observation adds bounded backpressure only after the current destination outcome → use a one-entry buffer, let cancellation select over later publication, and test a stopped consumer.
- A destination might perform an external effect and still return failure → expose failure without claiming non-delivery and never retry automatically.
- A trigger accepted while a prior destination call is active may make later Controller work eligible → explicit sources should trigger from external occurrences, while Controller remains the only scheduling owner; deterministic convergence sequences external progress after delivery. #3 reviews real evidence before changing this rule.
- The real command needs credentials and Codex installation → configuration remains operator-owned, secrets are never persisted or printed, and deterministic acceptance remains credentialless.
- A one-shot command does not itself prove a daemon lifecycle → the reusable internal Host integration test proves repeated triggers in one lifecycle; the command proves concrete real bindings.
- The GitHub post-Snapshot observation vocabulary may evolve → a command-local binding adapter selects only existing provider-independent Progress Evidence; `githubplan` retains HTTP and representation mapping, and no Provider DTO enters Plan Control or the Host.

## Migration / Rollback

There is no persisted Product state or external data migration. Rollback removes the internal Host and command while leaving existing Controller, Plan Attempt, GitHub, Codex adapter, and SDK behavior intact. The additive `PlanAttemptBinding` can be removed before any public release if #3 selects another exact single-target construction contract.

## Open Questions

None. Command flag names and terminal formatting may be chosen during implementation because they do not change the accepted behavior, architecture boundary, or task order.

## Approval Gate

The design risk is High. Construction MUST NOT begin until a human approves this DesignDoc, including the provisional internal Host boundary, fail-stop delivery policy, one-shot real Composition Root, and deterministic integration strategy. Milestone #3 remains responsible for final module and interface approval after #2 evidence exists.
