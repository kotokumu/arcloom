## MODIFIED Requirements

### Requirement: exact-target-request

The Controller MUST accept evaluation Requests only for one valid caller-established Target Identity and keep every resulting Attempt, Completion, and Attempt Failure bound to that exact identity.

- **入力と受理**:

  | Rule | Kind or key condition | Acceptance |
  |---|---|---|
  | Valid identity | Both are non-empty, neither is only Unicode whitespace, and neither has leading or trailing Unicode whitespace | Accept; embedded whitespace is allowed. |
  | Empty identity part | Either is empty | Reject synchronously. |
  | Whitespace-only identity part | Either consists only of Unicode whitespace | Reject synchronously. |
  | Boundary whitespace | Either has leading or trailing Unicode whitespace | Reject synchronously. |

- **振る舞いの規則**: Accepted kind and key bytes are preserved exactly. Equality is byte-exact and case-sensitive for both fields, with no Unicode normalization.
- **不変条件**: Every Request, Attempt, Completion, and Attempt Failure associated with one accepted target retains the exact accepted Target Identity.
- **失敗の扱い**: Rejection starts no Attempt and creates no Report.

#### Scenario: RCL-ETR-1 Valid target is requested [happy]

- **GIVEN** a Host has one valid stable Target Identity
- **WHEN** it requests reconciliation
- **THEN** every Attempt and reported outcome for that target contains that exact identity

#### Scenario: RCL-ETR-2 Target identity is invalid [error]

- **GIVEN** a Host supplies an empty, whitespace-only, or boundary-whitespace identity part
- **WHEN** it requests reconciliation
- **THEN** the request is rejected synchronously and no Attempt starts

#### Scenario: RCL-ETR-3 Target identity equality is exact [compatibility]

- **GIVEN** valid identities differ only by case, composed versus decomposed Unicode, or another byte distinction
- **WHEN** the Host requests them
- **THEN** the Controller treats them as distinct targets, while byte-equal kind and key values compare equal and embedded whitespace remains unchanged

### Requirement: level-based-attempt

Each Attempt MUST evaluate one target from facts acquired for that Attempt rather than from the Request occurrence or a previous outcome.

- **入力と受理**: The Attempt receives only the exact Target Identity and caller lifecycle required to acquire its target-specific current facts.
- **振る舞いの規則**: Request cause, count, metadata, payload, prior result, prior failure, and prior Control Directive are not authoritative decision inputs. The target-specific Attempt decides which current facts to acquire and whether semantic Reconciliation can occur.
- **不変条件**: The generic Controller neither acquires nor interprets target-specific Observations and never stores a prior target outcome as the authority for a later Attempt.
- **副作用**: Wake-up causes remain with the Host and do not cross the Attempt boundary.

#### Scenario: RCL-LBA-1 Provider event wakes a target [happy]

- **GIVEN** a Provider event causes a Host to request one target
- **WHEN** the target-specific Attempt evaluates it
- **THEN** the decision uses current target-specific facts acquired for that Attempt and not the Provider event payload or a prior outcome

#### Scenario: RCL-LBA-2 Prior result exists [compatibility]

- **GIVEN** a target has a prior successful result and a later Request
- **WHEN** the later Attempt evaluates the target
- **THEN** the prior result is not an authoritative input to that Attempt

### Requirement: per-target-exclusion-and-coalescing

One caller-scoped Controller MUST serialize Attempts for each Target Identity, preserve later eligibility for a Request received during active work, and allow distinct targets to progress independently within one positive finite concurrency bound.

- **入力と受理**: The configured concurrency bound is a positive finite integer. An invalid bound starts no Controller lifecycle.
- **振る舞いの規則**:

  | Current target state | Trigger | Guard | Next target state | Observable control result |
  |---|---|---|---|---|
  | Inactive | Valid Request | Controller is Running | Pending | The target becomes eligible. |
  | Pending | Another valid Request | Controller is Running | Pending | Requests may coalesce; no occurrence requires its own Attempt. |
  | Pending | Scheduler starts work | Capacity is available and no Report is pending | Active | Exactly one Attempt starts. |
  | Active | Valid Request for the same target | Controller is Running | Active with Pending Reentry | No overlapping Attempt starts; at least one later Attempt is preserved. |
  | Active with Pending Reentry | Another valid Request | Controller is Running | Active with Pending Reentry | Further Requests may coalesce. |
  | Active with Pending Reentry | Active Attempt returns | Controller remains Running and its Report is published | Pending | At least one later Attempt remains eligible. |
  | Active with Pending Reentry | Caller cancellation commits | Any publication state | Inactive within Stopping | Preserved eligibility is discarded and no later Attempt starts. |

- **不変条件**:
  - At most one Attempt for a Target Identity is Active.
  - The total Active Attempt count never exceeds the configured bound.
  - State and outcomes for distinct Target Identities are never exchanged.
- **排他・冪等**: Coincident Request and Attempt return preserve at least one later eligibility without same-target overlap. Duplicate Pending Requests may be satisfied by one Attempt.
- **失敗の扱い**: Invalid Target Identity rejection follows [[reconciliation-control-loop/exact-target-request]]; invalid concurrency configuration creates no lifecycle.

#### Scenario: RCL-PEC-1 Duplicate pending requests may coalesce [idempotency]

- **GIVEN** one target is Pending but not Active
- **WHEN** the Host requests that target repeatedly
- **THEN** at least one serialized Attempt runs and the Controller may satisfy all Pending duplicates with that one Attempt

#### Scenario: RCL-PEC-2 Request arrives during active attempt [concurrency]

- **GIVEN** one target has an Active Attempt
- **WHEN** the Host requests the same target before that Attempt returns
- **THEN** no second Attempt overlaps it and at least one later Attempt remains eligible after its outcome is resolved

#### Scenario: RCL-PEC-3 Distinct targets are requested [concurrency]

- **GIVEN** distinct valid targets and available concurrency capacity
- **WHEN** the Host requests them
- **THEN** their Attempts may run concurrently without exceeding the bound or exchanging target or result state

#### Scenario: RCL-PEC-4 Request and completion coincide [concurrency]

- **GIVEN** a target Request arrives while its Active Attempt is returning
- **WHEN** the Controller resolves both occurrences
- **THEN** it preserves at least one later eligible Attempt and never runs two Attempts for that target concurrently

### Requirement: explicit-control-directive

The Controller MUST create internal reevaluation eligibility after a successful Attempt only according to that Attempt's one valid Control Directive and only after its Completion is published.

- **入力と受理**:

  | Partition | Directive condition | Acceptance or result |
  |---|---|---|
  | Await | Await Another Request | Valid; no delay value exists. |
  | Immediate | Reevaluate Immediately | Valid; no delay value exists. |
  | Positive finite delay | Reevaluate After Delay with duration greater than zero and finite | Valid. |
  | Non-positive delay | Reevaluate After Delay with zero or negative duration | Rejected; no Directive exists. |
  | Missing or invalid Directive | An otherwise successful Attempt provides no valid Directive | Controller rejects it as Control Directive Rejected. |

- **振る舞いの規則**:

  | Rule | Completion publication | Directive or event | Eligibility result | Side effect |
  |---|---|---|---|---|
  | Await | Committed | Await Another Request | Inactive unless a Request is already preserved | No internal eligibility. |
  | Immediate | Committed | Reevaluate Immediately | Pending | Another serialized Attempt may start. |
  | Delay | Committed | Reevaluate After Delay | Delayed, then Pending after the duration | A timer owns only disposable eligibility. |
  | Early Request | Committed | Valid Request before a delay expires | Pending immediately | The later delay creates no duplicate obligation. |
  | Publication pending | Not committed | Any valid Directive | Unchanged | The Directive is not applied. |

- **不変条件**: Request and Directive eligibility coalesce without same-target overlap or duplicate obligations. Target result content, failure, cancellation, and request cause never imply a Directive.
- **失敗の扱い**: Control Directive Rejected is distinguishable from Target Attempt Failed, exposes no Completion or Directive, and creates no Directive-based eligibility.

#### Scenario: RCL-ECD-1 Attempt awaits another request [happy]

- **GIVEN** a successful Attempt selects Await Another Request
- **WHEN** its Completion is published
- **THEN** no later Attempt is internally scheduled for that target

#### Scenario: RCL-ECD-2 Attempt requests immediate reevaluation [happy]

- **GIVEN** a successful Attempt selects Reevaluate Immediately
- **WHEN** its Completion is published
- **THEN** the target becomes Pending for another serialized Attempt without an external Request

#### Scenario: RCL-ECD-3 Attempt requests delayed reevaluation [happy]

- **GIVEN** a successful Attempt selects Reevaluate After Delay with a positive finite duration
- **WHEN** its Completion is published and that duration expires without an earlier Request
- **THEN** the target becomes Pending for another serialized Attempt

#### Scenario: RCL-ECD-4 Request precedes delayed eligibility [boundary]

- **GIVEN** one target is Delayed after a published positive Directive
- **WHEN** the Host requests it before the delay expires
- **THEN** it becomes eligible earlier and the stale delay creates no second obligation

#### Scenario: RCL-ECD-5 Non-positive delayed directive is rejected [error]

- **GIVEN** a caller supplies zero or negative duration for Reevaluate After Delay
- **WHEN** it submits the delayed Directive for use
- **THEN** the Directive is rejected and cannot direct an Attempt reevaluation

#### Scenario: RCL-ECD-6 Missing directive is rejected by control [error]

- **GIVEN** a target-specific Attempt reports success without a valid Directive
- **WHEN** the Controller processes the outcome
- **THEN** it reports Control Directive Rejected, exposes no Completion, and creates no Directive-based eligibility

#### Scenario: RCL-ECD-7 Directive is committed after Completion delivery [boundary]

- **GIVEN** a successful Attempt returns Reevaluate Immediately and its Completion Report is waiting for the consumer
- **WHEN** the consumer has not received that Report
- **THEN** the Directive creates no eligible Attempt until Report delivery commits

### Requirement: target-result-isolation

A successful Completion MUST preserve its target-owned value without requiring a shared result classification or claiming that semantic Reconciliation occurred.

- **振る舞いの規則**: The Controller associates the successful value with its exact Target Identity and Control Directive but does not inspect the value to determine scheduling or semantic status. The value may be a Reconciliation Result or another target-specific success, including a successful Observation for which semantic Reconciliation was not possible.
- **不変条件**: Scheduling depends only on the valid Control Directive; result meaning remains owned by the target-specific capability.
- **副作用**: Generic control defines no universal Reconciliation input, Observation, Result, Failure, outcome taxonomy, evidence, proposal, condition, or effect interface.

#### Scenario: RCL-TRI-1 Unrelated result types use control [compatibility]

- **GIVEN** two Controllers use target-specific Attempt boundaries with unrelated successful-value meanings
- **WHEN** each successfully publishes a Completion
- **THEN** each Host receives its target-specific value unchanged and scheduling follows only its Control Directive

#### Scenario: RCL-TRI-2 Successful Attempt performs no Reconciliation [boundary]

- **GIVEN** a target-specific Attempt establishes a successful Observation but lacks the inputs required for semantic Reconciliation
- **WHEN** the Controller publishes its Completion
- **THEN** it preserves the successful value and Directive without reporting that a Reconciliation Result exists

### Requirement: failure-does-not-retry

An Attempt Failure MUST remain target-bound, distinguish Target Attempt Failed from Control Directive Rejected, and MUST NOT create implicit reevaluation eligibility.

- **振る舞いの規則**:

  | Rule | Attempt return | Reported outcome | Eligibility result | Other targets |
  |---|---|---|---|---|
  | Target failure | Attempt reports failure together with any nominal value and any Directive | Target Attempt Failed with exact target and preserved error cause; no result or Directive | None from failure | Unchanged |
  | Directive failure | Attempt otherwise reports success without a valid Directive | Control Directive Rejected with exact target; no Completion or Directive | None from failure | Unchanged |
  | Preserved Request | Either failure while a same-target Request is already preserved | The same typed failure | Later Request-driven work remains eligible after publication | Unchanged |
  | No preserved Request | Either failure with no later Request | The same typed failure | Target becomes Inactive | Unchanged |

- **不変条件**: Failure kind, exact Target Identity, and corresponding error remain associated; Failure never contains a successful result or Control Directive.
- **排他・冪等**: A Request received during the failed Active Attempt may cause later work, but the Failure itself creates none.
- **失敗の扱い**: Failure for one target neither terminates nor contaminates work for another target.

#### Scenario: RCL-FNR-1 Attempt fails without a pending request [error]

- **GIVEN** one target's Attempt reports failure and no later Request is preserved
- **WHEN** Target Attempt Failed is published
- **THEN** no Attempt is automatically scheduled for that target and no returned value or Directive is exposed

#### Scenario: RCL-FNR-2 Request was preserved before failure [idempotency]

- **GIVEN** a Request arrived while an Attempt was Active
- **WHEN** that Attempt fails
- **THEN** a later Attempt may run because of the preserved Request rather than because the Failure was retried

#### Scenario: RCL-FNR-3 Another target continues [error]

- **GIVEN** distinct targets are being controlled
- **WHEN** one target's Attempt fails
- **THEN** the other target can continue and receives no state from that Failure

#### Scenario: RCL-FNR-4 Invalid directive is distinguishable [error]

- **GIVEN** a target-specific Attempt reports success with an invalid Directive
- **WHEN** the Controller reports the Attempt Failure
- **THEN** the Host observes Control Directive Rejected rather than Target Attempt Failed

#### Scenario: RCL-FNR-5 Target Attempt error outranks returned values [error]

- **GIVEN** a target-specific Attempt reports failure together with any nominal successful value and any valid or invalid Directive
- **WHEN** the Controller processes that outcome
- **THEN** it publishes Target Attempt Failed with the exact target and error cause, exposes no Completion or Directive, and creates no Directive-based eligibility

### Requirement: caller-lifecycle

The Controller MUST publish returned outcomes and stop through the supplied caller lifecycle without allowing Report backpressure, pending eligibility, timers, or unfinished work to establish post-cancellation success or prevent bounded shutdown after Active Attempts return.

- **前提条件**: Each target-specific Attempt observes the supplied cancellation and returns within its documented bound.
- **入力と受理**:

  | Controller lifecycle | Request submission lifecycle | Acceptance result |
  |---|---|---|
  | Running | Active until acceptance | Request may be accepted. |
  | Running | Ends before acceptance | Submission returns that lifecycle outcome and no Request is accepted. |
  | Running after acceptance | Ends later | The accepted work remains governed only by the Controller lifecycle. |
  | Ended before acceptance | Active or ended | Submission returns the Controller lifecycle outcome; it takes precedence when both lifecycles ended. |

- **振る舞いの規則**:

  | Current lifecycle state | Trigger | Guard | Next state | Output or side effects |
  |---|---|---|---|---|
  | Running | An Attempt returns | No Report is pending | Running with Report Pending | Retain one prospective Completion or Attempt Failure; start no new Attempt. |
  | Running with Report Pending | Consumer receives Report | Cancellation has not committed | Running | Publish exactly once; a successful Directive commits afterward. |
  | Running | Caller cancellation | Any target state | Stopping | Reject Requests, start no Attempt, discard Pending and Delayed eligibility, cancel Active Attempts, and discard any unpublished Report and unapplied Directive. |
  | Running with Report Pending | Caller cancellation | Report delivery has not committed | Stopping | Publish no prospective outcome and apply no Directive. |
  | Stopping | Active Attempt returns | Other Active Attempts remain | Stopping | Establish no post-stop Completion; continue waiting only for Active returns. |
  | Stopping | Last Active Attempt returns | None remain | Stopped | Close the stable Report stream and make the caller lifecycle outcome available to Wait. |

  | Race rule | Publication state when cancellation competes | Observable winner | Directive result | Lifecycle result |
  |---|---|---|---|---|
  | Delivery first | Report delivery commits first | Publish that one outcome exactly once | Apply only its valid Directive after publication | Then stop with the caller lifecycle outcome. |
  | Cancellation first | Cancellation commits first | Publish no prospective outcome | Apply no Directive | Stop with the caller lifecycle outcome. |
  | Simultaneous readiness | Neither is ordered beforehand | Exactly one of the preceding rows wins | Consistent with the winning row | Reports closes and Wait returns the caller lifecycle outcome. |

- **不変条件**:
  - Reports returns one stable stream for the lifecycle.
  - At most one unpublished Report is retained; no new Attempt starts while it awaits publication.
  - Report backpressure does not prevent Request acceptance and coalescing, already Active Attempt returns, or cancellation progress.
  - During normal operation with a receiving consumer, every returned outcome is published exactly once without loss or duplication; no order is promised across distinct concurrently Active targets.
  - Cancellation never waits for Report consumption, Pending or Delayed work, or another timer after all Active Attempts return.
  - An Attempt return observed after cancellation commits establishes no Completion or Directive eligibility, even if it contains nominal success.
- **副作用**: Report publication is in-process control output and is not delivery of a semantic Reconciliation Result to its Result Destination.
- **失敗の扱い**: An Attempt that has not successfully completed and published before cancellation establishes no successful Completion. Wait exposes the caller lifecycle outcome.

#### Scenario: RCL-CL-11 Normal operation reports every returned outcome once [happy]

- **GIVEN** multiple target Attempts return outcomes, the lifecycle continues, and the consumer receives Reports
- **WHEN** normal publication proceeds
- **THEN** every returned outcome is published exactly once without loss or duplication and no cross-target completion order is promised

#### Scenario: RCL-CL-1 Caller cancels with active and pending work [concurrency]

- **GIVEN** the Controller has Active, Pending, or Delayed targets
- **WHEN** the caller cancels its lifecycle
- **THEN** no new Attempt starts, Active Attempts are cancelled and awaited, pending eligibility is discarded, Reports closes after Active returns, and unfinished Attempts establish no Completion

#### Scenario: RCL-CL-2 Controller adds no shutdown wait after Attempts return [boundary]

- **GIVEN** every Active Attempt has returned after observing caller cancellation
- **WHEN** the Controller finishes stopping
- **THEN** it closes Reports and exposes the caller context error without waiting for Report consumption, Pending work, Delayed work, or another timer

#### Scenario: RCL-CL-3 Report consumer stops before cancellation [boundary]

- **GIVEN** a prospective Report is pending for a stopped consumer
- **WHEN** the caller cancels
- **THEN** the Controller may discard the unpublished Report, waits only for Active Attempts, closes Reports, and returns without consumer progress

#### Scenario: RCL-CL-4 Attempt boundary returns nominal success after cancellation [concurrency]

- **GIVEN** one Active Attempt observes caller cancellation and then returns a nominal successful value and Directive
- **WHEN** the Controller receives that return after stopping began
- **THEN** it establishes no Completion or Directive-based eligibility, closes Reports after all Active returns, and exposes the caller context error

#### Scenario: RCL-CL-5 Submission context ends after acceptance [boundary]

- **GIVEN** a Request operation has returned success for one target
- **WHEN** only that operation's submission context is later cancelled
- **THEN** the accepted work remains governed by the Controller lifecycle and receives neither cancellation nor evidence from the submission context

#### Scenario: RCL-CL-6 Submission and Controller contexts have ended [error]

- **GIVEN** both the submission context and Controller lifecycle context ended before Request acceptance
- **WHEN** the Host submits the Request
- **THEN** the operation returns the Controller lifecycle error and starts no Attempt

#### Scenario: RCL-CL-7 Report delivery commits before cancellation [happy]

- **GIVEN** one prospective successful Report and caller cancellation have not committed
- **WHEN** Report delivery commits before cancellation
- **THEN** Reports publishes that Completion exactly once, its Directive applies afterward, and the lifecycle then stops

#### Scenario: RCL-CL-8 Pending Report applies bounded backpressure [boundary]

- **GIVEN** one Report is pending while Requests arrive and other Active Attempts return
- **WHEN** the Report consumer remains blocked
- **THEN** no new Attempt starts, Requests can coalesce, Active Attempts can return, and cancellation still completes without consumer progress

#### Scenario: RCL-CL-9 Cancellation commits before Report delivery [concurrency]

- **GIVEN** one prospective Report has not been delivered
- **WHEN** caller cancellation commits first
- **THEN** the Controller discards that outcome, publishes no Completion or Attempt Failure, and applies no Directive

#### Scenario: RCL-CL-10 Simultaneous delivery and cancellation have one winner [concurrency]

- **GIVEN** a prospective successful Report, a ready consumer, and caller cancellation become ready concurrently
- **WHEN** the Controller resolves the race
- **THEN** exactly one outcome holds: the Report publishes once and its Directive commits before stopping, or no Report publishes and no Directive-based Attempt becomes eligible

### Requirement: disposable-control-state

The Controller MUST retain no authoritative target, outcome, or durable scheduling state and MUST be able to evaluate a target after control-state loss from a new Request and newly acquired facts.

- **振る舞いの規則**: A new lifecycle needs no restoration of prior Requests, Completions, Attempt Failures, Pending eligibility, or Delayed eligibility.
- **不変条件**: All mutable scheduling and unpublished Report state belongs to one started Controller lifecycle and is disposable.
- **副作用**: Generic control creates no authoritative repository, checkpoint, lease, leader record, or durable queue.

#### Scenario: RCL-DCS-1 Control state is discarded [compatibility]

- **GIVEN** all prior Controller state has been lost
- **WHEN** a new lifecycle receives a valid Target Identity Request
- **THEN** it can start an Attempt that reacquires current facts without restoring a prior request or outcome
