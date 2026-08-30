## Why

Arcloom provides target-specific Observation, Reconciliation, assessment, Authorization, and application-request capabilities, but it does not provide a reusable way to drive them from current external state until another Observation is required. Each Host must currently invent trigger handling, same-target concurrency, reevaluation, cancellation, and the boundary between later wake-ups and separate external interactions. This prevents Plan control from operating as a dependable Feedback Loop and would duplicate the same lifecycle policy for later targets. The design must also preserve Reconciliation as the target-specific judgment that relates expected and observed meaning; generic control must not absorb that responsibility or impose Plan Representation's evidence-derived Result on unrelated Reconciliations.

## Intended Outcomes

- An Arcloom Host can run target-specific reconciliation through one target-independent, level-based control contract.
- A trigger requests evaluation of current state without becoming evidence or prescribing the decision.
- Each target-specific Reconciliation retains ownership of its expected and observed meaning, judgment authority, Result, and Failure contract.
- Generic control owns execution lifecycle and preserves target Results without interpreting them.
- Plan Representation owns its Evidence, Reconciliation Determination, and Result without imposing them on other reconciliation targets.
- Plan control becomes the first standard use of the generic control contract. It resolves each requested target binding, begins with a fresh Plan Snapshot, and performs Plan Reconciliation only after that Snapshot establishes a valid current Plan and a later Delivery Observation is available.

## Success Criteria

- SC-1: Concurrent requests for one target never execute more than one reconciliation attempt for that target at the same time.
- SC-2: At least one later attempt occurs when a request arrives while the same target is already being reconciled, while duplicate pending requests may be coalesced.
- SC-3: Every attempt reacquires the current inputs required by its target contract rather than treating the triggering event or a prior result as decision evidence.
- SC-4: An attempt is repeated without a new external request only when its successful return carries an explicit Control Directive for reevaluation; a target result, failure, or uncertain external receipt never causes an implicit retry.
- SC-5: At least two target-specific reconciliation contracts can use the control capability without adopting one shared domain-result taxonomy.
- SC-6: A Plan attempt with a valid current Plan establishes an assessment from that exact Plan and later Delivery Observations; a successful Snapshot with no current Plan instead reports that no current Plan was established and does not invoke Plan Control.
- SC-7: When every target-attempt boundary honors its documented cancellation contract, cancelling the controller terminates active attempts and exposes the caller lifecycle outcome even when Report consumption has stopped; an unfinished attempt establishes no successful result.
- SC-8: Plan Representation continues to derive the exact `Satisfied`, `NotSatisfied`, or `Undecidable` determination required by its complete Evidence while an unrelated Reconciliation can expose a result with no such classification.
- SC-9: After a separate external application interaction, an explicit new request performs fresh observation without receiving application receipt, event payload, or the preceding Result as target state.

## Scope

### In Scope

- Target-independent control of reconciliation requests for stable target identities.
- Level-based invocation, same-target exclusion, request coalescing, explicit reevaluation, and caller-owned cancellation.
- Separation between target-specific reconciliation results and target-independent control directives.
- Plan reconciliation as the first standard consumer of the generic control capability.
- Plan progression rules that distinguish successful current-Plan observation from an actual Plan Reconciliation assessment, plus reentry after a separately invoked application request.
- Architecture updates for the ownership and dependency boundaries of the control capability and Plan-specific composition.
- Modeling Reconciliation as target-specific judgment rather than a cross-target Result implementation, and relocating the existing evidence-derived Result implementation to Plan Representation, its only production consumer.

### Out of Scope

- A fixed Delivery or Development Improvement workflow.
- A universal taxonomy for target-specific reconciliation results, conditions, or external effects.
- A universal Reconciliation input, Observation, Result, Failure, or outcome interface.
- A compatibility facade or deprecated evidence Result package that preserves the unsupported cross-target abstraction.
- A durable queue, leader election, distributed coordination, or an Arcloom-owned source of truth.
- Provider-specific watch, polling, persistence, or mutation protocols.
- Automatic Task execution or an Agent runtime.
- Automatic retransmission after an external request may have been received.
- Automatic authorization or application of a target-specific Adjustment from inside the control loop.
- Replacement of existing observation, Plan Control, Authorization, or application-request semantics.

## Capabilities

### New Capabilities

- `reconciliation-control-loop`: Lets an Arcloom Host request level-based target attempts while preserving per-target exclusion, request coalescing, explicit reevaluation, Report publication, and cancellation without defining target-specific Reconciliation meaning.
- `plan-reconciliation-loop`: Lets an Arcloom Host repeatedly observe one bound Plan target and, when a valid current Plan is established, reconcile it through Plan Control while leaving Authorization and external application as explicit separate interactions.

### Modified Capabilities

None. Existing `plan-control`, `github-plan-snapshot`, `plan-application-request`, and `authorization` guarantees are composed without changing their accepted meanings.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Reconciliation | Target-specific Reconciliation capability | Preserve as one read-only judgment that relates expected and observed meaning for one semantic target and establishes one target-specific Result. Generic control invokes but does not redefine it. |
| Target Identity | `reconciliation-control-loop` | Add a stable control identity used only to isolate and correlate requests, attempts, and Reports; it is not the semantic target or observed state. |
| Reconciliation Request | `reconciliation-control-loop` | Add a request to evaluate one target's current level; it is not observation evidence. |
| Reconciliation Attempt | `reconciliation-control-loop` | Add one caller-lifecycle-bound invocation occurrence for one target; it is not a separate semantic owner. |
| Control Directive | `reconciliation-control-loop` | Add target-independent instruction to await another request or reevaluate, separate from domain results. |
| Plan Target Binding | `plan-reconciliation-loop` | Add a self-identifying validated association with its Plan Snapshot Observer and Delivery Observer; the Plan Control Assessor remains a Plan Control dependency. |
| Plan Attempt Result | `plan-reconciliation-loop` | Add an explicit Plan-specific outcome distinguishing a successful Snapshot that establishes no current Plan from a current Plan reconciled to one exact Assessment. |
| Plan Representation Evidence, Reconciliation Determination, and Result | `plan-representation-reconciliation` | Preserve their accepted behavior and make their existing owner responsible for their implementation contract. |
| Plan Revision and Request Receipt Evidence | `plan-application-request` | Reference without changing ownership or meaning. |

## Decisions Required

None. Product and Architecture principles determine that Reconciliation relates expected and observed meaning without external mutation, current external facts remain authoritative, triggers are not evidence, external application remains a separate authorized interaction, request receipt is not target state, and target-specific Result meanings must not be generalized.

## Impact

- Arcloom Hosts gain a reusable control-loop contract and a standard Plan reconciliation composition.
- `ARCHITECTURE.md` changes because a new control responsibility and a Plan-specific composition boundary are introduced.
- Existing Plan Representation behavior remains compatible, but callers importing the standalone evidence-derived Result implementation must migrate to the Plan Representation contract.
- No durable data migration or Provider contract change is required.
