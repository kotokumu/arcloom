## Purpose

Operates one caller-scoped Plan feedback loop from explicit target wake-ups through published Controller Reports to target-specific delivery of assessed Plan Results, without taking ownership of external facts, scheduling, application, or Task execution.

## ADDED Requirements

### Requirement: exact-plan-operation-configuration

A Plan Feedback Loop Operation MUST start only from one complete configuration and MUST use the exact Request identity established from that configuration's one self-identifying Plan target.

- **Input and Acceptance**:

  | Rule | Configuration condition | Acceptance or result |
  |---|---|---|
  | Complete exact-target configuration | One valid self-identifying Plan target has established one inseparable exact Target Identity and target-bound Plan Attempt association, and the active caller lifecycle and Plan Result Destination are present | Start one Configured operation for that identity, then enter Running. |
  | Missing relationship | Any required relationship is absent | Reject before Running and accept no trigger. |
  | Invalid Plan Attempt binding | The exact Target Identity and target-bound Attempt association was not established by the Plan Attempt capability | Reject before Running and accept no trigger. |
  | Ended caller lifecycle | The caller lifecycle has already ended | Preserve that lifecycle outcome and do not start the operation. |

- **Invariants**: A started operation has exactly one Target Identity, one target-bound Plan Attempt, one sole Report consumer, and one Plan Result Destination for its caller lifecycle.
- **Side Effects**: Validation performs no Plan observation, assessment, Authorization, application, Task execution, or external mutation.
- **Failure Handling**: Rejection establishes no Running operation, Request, Attempt, Report, or Plan Result Delivery.
- **References**: [[reconciliation-control-loop/exact-target-request]]; [[reconciliation-control-loop/caller-lifecycle]]; [[plan-reconciliation-loop/fresh-plan-observation]]

#### Scenario: Complete exact-target operation starts [happy]

- **GIVEN** a Host has supplied every required relationship for one exact valid Plan Target Identity
- **WHEN** the Host starts the Plan Feedback Loop Operation
- **THEN** one Running operation is established for that exact target

#### Scenario: Target-bound configuration is invalid [error]

- **GIVEN** the Host lacks a valid Plan Attempt binding that inseparably associates one exact Target Identity and target-bound Attempt
- **WHEN** the Host starts the Plan Feedback Loop Operation
- **THEN** startup is rejected before any trigger is accepted or external observation begins

### Requirement: explicit-plan-trigger

A Running Plan Feedback Loop Operation MUST reduce each accepted Explicit Plan Trigger to an ordinary Controller Request containing only its configured Target Identity.

- **Input and Acceptance**:

  | Operation state | Trigger submission state | Acceptance result |
  |---|---|---|
  | Running | Active until acceptance | The trigger may be accepted and one ordinary Request submission is attempted. |
  | Running | Ends before acceptance | The trigger is not accepted and submission exposes its lifecycle outcome. |
  | Stopping or Stopped | Active or ended | The trigger is rejected with the operation lifecycle outcome. |

- **Behavioral Rules**: Trigger cause, event type, payload, occurrence count, prior Report, prior Plan Attempt Result, prior delivery outcome, and prior failure do not enter the Request or Plan Attempt as current facts.
- **Invariants**: Every accepted trigger concerns the operation's one exact Target Identity. Controller scheduling, same-target exclusion, and coalescing remain governed by `reconciliation-control-loop`.
- **Concurrency and Idempotency**: Accepted trigger occurrences are not promised a one-to-one Attempt count. A trigger accepted during active work preserves no stronger eligibility than the Controller's existing Request contract.
- **Failure Handling**: A rejected trigger starts no new Attempt and creates no Report or delivery.
- **References**: [[reconciliation-control-loop/exact-target-request]]; [[reconciliation-control-loop/level-based-attempt]]; [[reconciliation-control-loop/per-target-exclusion-and-coalescing]]; [[plan-reconciliation-loop/ordinary-plan-reentry]]

#### Scenario: Later external event wakes the Plan [happy]

- **GIVEN** a Running operation whose external Planning Context changed after an earlier cycle
- **WHEN** its Trigger Source supplies another Explicit Plan Trigger
- **THEN** the operation submits only the exact Target Identity and the resulting Attempt reacquires current facts

#### Scenario: Duplicate triggers arrive during active work [idempotency]

- **GIVEN** the configured target has an Active Attempt
- **WHEN** the Trigger Source supplies the same trigger occurrence more than once
- **THEN** no trigger payload becomes an Attempt fact and the Controller's exclusion and coalescing rules determine later eligibility

### Requirement: assessed-plan-result-delivery

A Plan Feedback Loop Operation MUST invoke its Plan Result Destination only for a published Completion containing Current Plan Assessed and MUST offer that branch's exact Plan Control Assessment no more than once for that Report.

- **Behavioral Rules**:

  | Report classification | Publication state | Delivery output | Side Effects |
  |---|---|---|---|
  | Completion containing Current Plan Assessed | Publication committed and caller lifecycle remains active | Invoke the configured destination once with the exact target association and exact Plan Control Assessment contained by that branch | Establish one In Flight Plan Result Delivery. |
  | Completion containing Current Plan Assessed | Publication committed but cancellation has committed before delivery starts | No destination invocation | Follow caller termination without replay. |
  | Completion containing Current Plan Assessed | Publication not committed | No destination invocation | Preserve the prospective Report only under the Controller contract. |
  | Completion containing Current Plan Not Established | Publication committed | No destination invocation | Preserve observation-only success as the published Report. |
  | Attempt Failure | Publication committed | No destination invocation | Preserve the operational failure as the published Report. |

- **Invariants**:
  - Report publication is not Plan Result Delivery, and delivery never begins before publication commits.
  - Plan-specific delivery eligibility uses the existing Plan Attempt classification. Host transport does not interpret Assessment contents or decide the subsequent Plan action.
  - Plan Result Destination denotes subsequent Plan consideration; the concrete delivery mechanism neither owns that meaning nor makes the downstream decision.
  - The operation neither reconstructs nor reinterprets the Plan Control Assessment and does not expose the enclosing Snapshot to the destination.
  - At most one destination invocation is made for each qualifying published Report.
- **Side Effects**: Delivery grants no Authorization, Plan application, Delivery Acceptance, Task execution, External Actor contact, or target mutation authority.
- **Concurrency and Idempotency**: The operation is the sole consumer of its Controller Report stream; competing consumers cannot consume a Report instead of the operation or cause duplicate delivery.
- **References**: [[reconciliation-control-loop/caller-lifecycle]]; [[reconciliation-control-loop/target-result-isolation]]; [[plan-reconciliation-loop/current-plan-assessment]]; [[plan-reconciliation-loop/read-only-plan-attempt]]

#### Scenario: Assessed Plan Result is delivered [happy]

- **GIVEN** the Controller has published a Completion containing Current Plan Assessed to its operation
- **WHEN** the operation classifies the published Report
- **THEN** the configured destination receives that branch's exact Plan Control Assessment once and only after publication

#### Scenario: Current Plan is not established [boundary]

- **GIVEN** the Controller has published a Completion containing Current Plan Not Established
- **WHEN** the operation classifies the published Report
- **THEN** it makes no Plan Result Destination invocation

#### Scenario: Plan Attempt fails [error]

- **GIVEN** the Controller has published an Attempt Failure
- **WHEN** the operation classifies the published Report
- **THEN** it makes no Plan Result Destination invocation and does not turn the failure into a semantic Result

### Requirement: processed-plan-report-publication

A Plan Feedback Loop Operation MUST expose exact processed Controller Reports in Controller order during normal operation, and an exposed assessed Report MUST imply that its one destination invocation returned success.

- **Behavioral Rules**:

  | Current publication state | Trigger or event | Guard | Next state | Output or Side Effects |
  |---|---|---|---|---|
  | Pending | Publish the handled Report | Caller lifecycle is active; assessed delivery has succeeded, or the Report requires no semantic delivery | Published | Expose the exact Controller Report once. |
  | Pending | Caller cancellation or delivery failure commits | Publication has not committed | Discarded | Expose no processed Report for this pending occurrence; do not replay any Assessment. |
  | Published | Caller cancellation commits | Publication committed first | Published | Already-published buffered Reports remain readable after the stream closes. |

- **Invariants**:
  - Destination success and processed Report publication are separate commits. A successful delivery may have its pending processed Report discarded; missing processed evidence does not prove that no destination effect occurred.
  - A failed or cancelled delivery produces no successfully processed Report for that occurrence.
  - No-current and Attempt Failure remain exact processed Reports without semantic delivery.
  - Processed evidence is bounded, invocation-local, and disposable; no durable delivery ledger or exactly-once external effect is promised.
- **Side Effects**: Discarding pending evidence does not undo a delivery effect, mutate external Plan facts, or authorize replay.
- **Concurrency and Idempotency**: The operation does not wait for an evidence reader to resume after cancellation. Publication versus cancellation has one winner; already-published evidence is not published again.
- **Failure Handling**: Stream closure without the expected processed Report is resolved through the operation's terminal lifecycle or delivery-failure outcome, not inferred as successful handling.
- **References**: [[plan-feedback-loop-operation/assessed-plan-result-delivery]]; [[plan-feedback-loop-operation/caller-scoped-operation-lifecycle]]; [[reconciliation-control-loop/caller-lifecycle]]

#### Scenario: Assessed processed Report is observed [happy]

- **GIVEN** a published assessed Controller Report whose destination invocation has returned success under an active caller lifecycle
- **WHEN** the evidence consumer receives its processed Report
- **THEN** it receives that exact Report once after destination success

#### Scenario: Cancellation follows delivery while evidence publication is blocked [concurrency]

- **GIVEN** earlier published evidence fills the bounded buffer, a later Assessment delivery succeeds, and that later processed Report remains pending
- **WHEN** the caller cancels without the evidence reader resuming
- **THEN** the pending Report is discarded, the earlier published evidence remains readable, and termination neither waits for the reader nor undoes or replays the later delivery

### Requirement: delivery-failure-and-fresh-reentry

A failed Plan Result Delivery MUST stop the operation with a distinct observable delivery failure; the operation itself MUST NOT retry the delivery, apply the Result, or mutate external state, and any later operation MUST evaluate fresh current facts through an ordinary Request.

- **Behavioral Rules**:

  | Current delivery state | Trigger or event | Guard | Next delivery state | Output or Side Effects |
  |---|---|---|---|---|
  | In Flight | Destination reports success | Caller cancellation has not committed | Delivered | The operation remains Running. |
  | In Flight | Destination reports failure | Caller cancellation has not committed | Delivery Failed | Stop trigger intake, stop the operation, and expose the destination failure distinctly from Attempt Failure. |
  | In Flight | Caller cancellation commits | Success has not committed | Cancelled | Follow the caller-scoped operation lifecycle; do not replace cancellation with delivery success. |

- **Invariants**: Delivery Failed and Cancelled create no implicit Request, Control Directive, retry, Authorization, Plan application, Task execution, or external mutation.
- **Side Effects**: A reported delivery failure makes no claim about whether the external destination performed an effect before reporting failure.
- **Concurrency and Idempotency**: Delivery Failed has no transition back to In Flight. A newly started operation may accept a later trigger, but it does not redeliver the earlier Result and its Attempt receives no earlier Report, Result, or delivery outcome as facts.
- **Failure Handling**: Destination failure remains distinguishable from Target Attempt Failed, Control Directive Rejected, and caller cancellation.
- **References**: [[reconciliation-control-loop/failure-does-not-retry]]; [[plan-reconciliation-loop/fresh-plan-observation]]; [[plan-reconciliation-loop/ordinary-plan-reentry]]

#### Scenario: Destination rejects an assessed result [error]

- **GIVEN** one published Current Plan Assessed result has one In Flight delivery
- **WHEN** the Plan Result Destination reports failure
- **THEN** the operation stops with that delivery failure and neither retries the delivery nor changes the external Plan

#### Scenario: Operation restarts after delivery failure [happy]

- **GIVEN** an earlier operation stopped because delivery failed and external facts may since have changed
- **WHEN** a Host starts a new operation and supplies an Explicit Plan Trigger
- **THEN** the new Attempt observes current facts and does not redeliver or use the earlier result as a fact

### Requirement: caller-scoped-operation-lifecycle

A Plan Feedback Loop Operation MUST stop trigger intake and terminate through its Caller Lifecycle without allowing pending triggers, unpublished Reports, or unfinished delivery to establish post-cancellation success.

- **Preconditions**: The target-specific Attempt and Plan Result Destination observe the supplied cancellation and return within their accepted finite bounds.
- **Behavioral Rules**:

  | Current state | Trigger or event | Guard | Next state | Output or Side Effects |
  |---|---|---|---|---|
  | Configured | Host starts operation | Configuration is accepted | Running | Begin trigger intake and Controller Report consumption. |
  | Running | Caller cancellation commits | Any delivery or processed-publication state | Stopping | Reject new triggers, cancel active Controller and delivery work, establish no new delivery, and discard unpublished processed evidence without undoing an established delivery. |
  | Running | Delivery failure commits | Caller cancellation has not committed | Stopping | Reject new triggers and preserve the distinct delivery failure. |
  | Stopping | Active Controller and delivery boundaries return | No active boundary remains | Stopped | End Report consumption and expose the committed lifecycle or delivery-failure outcome. |
  | Stopped | Any trigger | Always | Stopped | Reject the trigger and start no work. |

- **Invariants**: Pending trigger occurrences, Controller eligibility, and operation delivery state are disposable and establish no authoritative target or durable scheduling state.
- **Concurrency and Idempotency**: If cancellation and destination success compete, the event whose success or cancellation commit occurs first determines the delivery outcome. Repeated cancellation does not create another terminal outcome.
- **Failure Handling**: After active boundaries return, Report backpressure, pending triggers, and disposable operation state cannot prevent Stopped. The finite termination guarantee relies on the active boundaries honoring the stated cancellation precondition; their unfinished work is never reported as successful termination.
- **References**: [[reconciliation-control-loop/caller-lifecycle]]; [[codex-plan-control-assessment/bounded-assessment-cancellation]]

#### Scenario: Caller cancels during delivery [error]

- **GIVEN** a Running operation has an In Flight Plan Result Delivery
- **WHEN** the caller cancels before delivery success commits
- **THEN** no delivery success is established and the operation reaches Stopped after active boundaries return

#### Scenario: Caller cancels while idle [happy]

- **GIVEN** a Running operation has no active Attempt or delivery
- **WHEN** the caller cancels it
- **THEN** trigger intake ends and the operation reaches Stopped without waiting for a new trigger
