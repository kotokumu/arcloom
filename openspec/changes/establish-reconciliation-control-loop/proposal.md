## Why

Arcloom provides target-specific observation, reconciliation, assessment, authorization, and application-request capabilities, but it does not provide a reusable way to drive those capabilities from current external state until another observation is required. Each Host must currently invent trigger handling, same-target concurrency, reevaluation, cancellation, and the boundary between later wake-ups and separate external interactions. This prevents Plan control from operating as a dependable Feedback Loop and would duplicate the same lifecycle policy for later reconciliation targets.

## Intended Outcomes

- An Arcloom Host can run target-specific reconciliation through one target-independent, level-based control contract.
- A trigger requests evaluation of current state without becoming evidence or prescribing the decision.
- Target-specific outcomes and proposed external effects remain owned by their target capability.
- Plan reconciliation becomes the first standard use of the generic control contract, resolves each requested target binding inside the Plan Module, and establishes an assessment only after a fresh current Plan observation and later Delivery Observation.

## Success Criteria

- SC-1: Concurrent requests for one target never execute more than one reconciliation attempt for that target at the same time.
- SC-2: At least one later attempt occurs when a request arrives while the same target is already being reconciled, while duplicate pending requests may be coalesced.
- SC-3: Every attempt derives its decision from current inputs acquired for that attempt rather than from the triggering event or a prior result.
- SC-4: An attempt is repeated without a new external request only when its successful return carries an explicit Control Directive for reevaluation; a target result, failure, or uncertain external receipt never causes an implicit retry.
- SC-5: At least two target-specific reconciliation contracts can use the control capability without adopting one shared domain-result taxonomy.
- SC-6: A Plan target can be evaluated through fresh observation and Plan Control assessment; after a separate external application interaction, an explicit new reconciliation request performs fresh re-observation without receiving application receipt as target state.
- SC-7: When every target-attempt boundary honors its documented cancellation contract, cancelling the controller terminates active attempts and exposes the caller lifecycle outcome even when Report consumption has stopped; an unfinished attempt establishes no successful result.

## Scope

### In Scope

- Target-independent control of reconciliation requests for stable target identities.
- Level-based invocation, same-target exclusion, request coalescing, explicit reevaluation, and caller-owned cancellation.
- Separation between target-specific reconciliation results and target-independent control directives.
- Plan reconciliation as the first standard consumer of the generic control capability.
- Plan progression rules for current observation and assessment, plus reentry after a separately invoked application request.
- Architecture updates for the ownership and dependency boundaries of the control capability and Plan-specific composition.

### Out of Scope

- A fixed Delivery or Development Improvement workflow.
- A universal taxonomy for target-specific reconciliation results, conditions, or external effects.
- A durable queue, leader election, distributed coordination, or an Arcloom-owned source of truth.
- Provider-specific watch, polling, persistence, or mutation protocols.
- Automatic Task execution or an Agent runtime.
- Automatic retransmission after an external request may have been received.
- Automatic authorization or application of a target-specific Adjustment from inside the control loop.
- Replacement of existing observation, Plan Control, Authorization, or application-request semantics.

## Capabilities

### New Capabilities

- `reconciliation-control-loop`: Lets an Arcloom Host request level-based reconciliation for independently identified targets while preserving per-target exclusion, request coalescing, explicit reevaluation, and cancellation semantics without defining target-specific outcomes.
- `plan-reconciliation-loop`: Lets an Arcloom Host repeatedly establish Plan Control assessments from fresh observations while leaving authorization and external application as explicit separate interactions.

### Modified Capabilities

None. Existing `plan-control`, `github-plan-snapshot`, `plan-application-request`, and `authorization` guarantees are composed without changing their accepted meanings.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Reconciliation Target | `reconciliation-control-loop` | Add target-independent identity used only to isolate and correlate control requests. |
| Reconciliation Request | `reconciliation-control-loop` | Add a request to evaluate one target's current level; it is not observation evidence. |
| Reconciliation Attempt | `reconciliation-control-loop` | Add one caller-lifecycle-bound evaluation for one target. |
| Control Directive | `reconciliation-control-loop` | Add target-independent instruction to await another request or reevaluate, separate from domain results. |
| Plan Target Binding | `plan-reconciliation-loop` | Add a self-identifying validated association with its Plan Snapshot Observer and Delivery Observer; the Plan Control Assessor remains a Reconciler-level dependency. |
| Plan Reconciliation Attempt Result | `plan-reconciliation-loop` | Add the Plan-specific outcome of ordered current Snapshot observation, later Delivery Observation, and optional Plan Control assessment. |
| Plan Snapshot | `plan-control` | Reference without changing ownership or meaning. |
| Plan Revision and Request Receipt Evidence | `plan-application-request` | Reference without changing ownership or meaning. |

## Decisions Required

None. Product and Architecture principles already determine that current external facts remain authoritative, triggers are not evidence, external application remains a separate authorized interaction, request receipt is not target state, and target-specific result meanings must not be generalized.

## Impact

- Arcloom Hosts gain a reusable control-loop contract and a standard Plan reconciliation composition.
- `ARCHITECTURE.md` changes because a new control responsibility and a Plan-specific composition boundary are introduced.
- Existing public contracts remain behaviorally compatible; the new capabilities consume them according to their current guarantees.
- No durable data migration or Provider contract change is required.
