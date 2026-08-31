## Why

Arcloom can control one Plan Attempt through public contracts, but it does not provide an operable Plan Feedback Loop that receives external wake-ups and delivers Plan-specific Results to the next decision or work owner. Consumers must assemble that behavior themselves, and the repository has no deterministic environment that proves repeated fresh observation and externally applied progress converge a Plan to completion.

Milestone #2 requires an accepted operating contract before a reference Host, local integration environment, and real GitHub/Codex proof can be implemented without moving Reconciliation, Control, Result Destination, or External Actor responsibilities into a Composition Root.

## Intended Outcomes

- An operator can run a caller-scoped Plan Feedback Loop through explicit target wake-ups.
- Every later evaluation reacquires current Plan and delivery facts through the existing Plan Attempt contract.
- A Plan-specific Result reaches its configured Result Destination only after the corresponding Controller Report is published.
- Observation-only success, Attempt Failure, destination failure, and caller termination remain distinguishable.
- The same operating contract supports deterministic local verification and real GitHub/Codex operation.

## Success Criteria

- SC-1: An initial Request and at least two later ordinary Requests can be accepted for one target without passing trigger payloads or prior outcomes as current facts.
- SC-2: Every Current Plan Assessed completion delivers its exact Plan Control Assessment to one configured Plan-specific Result Destination after Report publication and no more than once for that qualifying Report.
- SC-3: Current Plan Not Established and Attempt Failure produce no semantic Result delivery.
- SC-4: Result Destination failure is observable and creates no implicit retry, Authorization, application, Task execution, or external mutation.
- SC-5: Caller cancellation stops intake and reaches bounded shutdown after active boundaries return.
- SC-6: A deterministic local verification completes multiple fresh-observation cycles without network access, credentials, manual changes, or wall-clock sleeps and ends with a Complete Assessment.
- SC-7: One real GitHub Milestone Plan can use the same contract with GitHub observation and Codex Plan Control.

## Scope

### In Scope

- Caller-scoped operation of repeated identity-only Plan Requests.
- Consumption of target-bound Controller Reports.
- Delivery of Current Plan Assessed Results to a Plan-specific Result Destination.
- Observable treatment of non-Reconciliation success, Attempt Failure, destination failure, and caller termination.
- Deterministic multi-cycle verification and real GitHub/Codex verification of the accepted behavior.

### Out of Scope

- Authorization, Plan application, Task execution, Delivery Acceptance, or external mutation.
- Durable queues, persistent Controller state, distributed exclusion, automatic retries, polling, or inferred scheduling.
- A universal Reconciliation, Observation, Result, Failure, outcome, Result Destination, or workflow contract shared across targets.
- Replacement of the Codex app-server SDK's process lifecycle or JSON-RPC responsibilities.
- Final Reconciliation remodelling, refactoring, or module and interface finalization owned by Milestone #3.

## Capabilities

### New Capabilities

- `plan-feedback-loop-operation`: Accepts explicit Plan target wake-ups, consumes Plan Attempt Reports, and delivers exact assessed Plan Results to their configured destination while preserving existing Reconciliation, Control, Observation, and External Actor boundaries.

### Modified Capabilities

None.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Plan Feedback Loop Operation | `plan-feedback-loop-operation` | Add the caller-scoped relationship among Request, Plan Attempt Report, and Plan Result Destination delivery. |
| Plan Result Destination | `plan-feedback-loop-operation` | Add the target-specific destination of the exact Plan Control Assessment contained by Current Plan Assessed without exposing the enclosing Snapshot or creating a shared Result abstraction. |
| Request | `reconciliation-control-loop` | Reference its existing identity-only wake-up meaning unchanged. |
| Plan Attempt Result | `plan-reconciliation-loop` | Reference its existing Current Plan Not Established and Current Plan Assessed classifications unchanged. |
| Report | `reconciliation-control-loop` | Reference its existing in-process publication meaning unchanged and keep it distinct from semantic Result delivery. |

## Decisions Required

- Decide which Plan Attempt Result branches constitute a deliverable semantic Result.
- Decide the delivery cardinality and observable behavior when the Plan Result Destination rejects or cannot establish delivery.
- Decide the caller-scoped relationship between repeated Request intake, Report consumption, Result delivery, and termination without prescribing durable scheduling.

## Impact

Operators and Host integrations gain one accepted way to operate Plan Feedback Loops and observe delivery failures. Existing Reconciliation Controller, Plan Attempt, GitHub observation, Codex Plan Control, and Codex app-server contracts remain authoritative for their current responsibilities. No external data migration is introduced. PRODUCT.md is not changed because the existing Delivery Reconciliation, Observation, Plan, and Plan Control capabilities already own the product scope. ARCHITECTURE.md changes only if modeling discovers a responsibility or dependency rule not already represented by the Host, Composition Root, and Result Destination boundaries.
