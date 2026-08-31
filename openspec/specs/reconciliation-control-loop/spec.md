# reconciliation-control-loop Specification

## Purpose
Provides a target-independent, level-based Controller lifecycle that serializes target Attempts, preserves target-owned successful values without deciding whether they are Reconciliation Results, follows only explicit reevaluation Directives, and retains no authoritative target or durable scheduling state.

## Conceptual Model

### Target Identity and Request

A Target Identity is a caller-established stable control identity for one subject to evaluate. It contains no observed target state and does not define the semantic target of a target-specific Reconciliation. A Request is a passive wake-up for exactly one Target Identity. Its occurrence, cause, payload, and count are not facts used by an Attempt.

### Attempt and Outcome

An Attempt is one target-specific runtime occurrence that acquires the current facts required by that target's contract. It may invoke zero or one semantic Reconciliation. A Completion associates the exact Target Identity, one opaque target-owned successful value, and exactly one Control Directive. An Attempt Failure associates the exact Target Identity with either Target Attempt Failed or Control Directive Rejected and contains neither a successful value nor a Control Directive. The control capability defines no universal Reconciliation input, Observation, Result, Failure, outcome classification, or shared interface.

### Control Directive

A Control Directive is exactly Await Another Request, Reevaluate Immediately, or Reevaluate After Delay with one positive finite delay. It is the only scheduling meaning the Controller accepts from a successful Attempt. Failure, cancellation, target value content, request cause, and prior outcomes never imply a Directive.

### Controller

A Controller is one caller-scoped lifecycle that owns disposable request eligibility, same-target exclusion, a finite concurrency bound across targets, delayed eligibility, bounded in-process Report publication, and orderly cancellation. At most one Attempt for a Target Identity is Active. Duplicate Pending Requests may coalesce, while a Request received during an Active Attempt preserves at least one later Attempt. The Controller starts no new Attempt while a Report is awaiting publication. Report publication commits its Completion or Attempt Failure and then applies a successful Control Directive. Cancellation may discard an unpublished Report and unapplied Directive, closes Reports after Active Attempts return, and exposes the caller context error as the lifecycle outcome. The Controller owns no authoritative target, Observation, Reconciliation, Result, semantic Result Destination, or durable scheduling state.

## Requirements

### Requirement: exact-target-request

The Controller MUST accept evaluation Requests only for one valid caller-established Target Identity and keep every resulting Attempt, Completion, and Failure bound to that exact identity.

- **入力と受理**: A request contains one valid stable Target Identity and no required event payload. Its kind and key are non-empty and contain neither only Unicode whitespace nor leading or trailing Unicode whitespace; embedded whitespace is allowed.
- **振る舞いの規則**: Accepted kind and key bytes are preserved exactly. Scheduling equality is byte-exact, case-sensitive equality of both fields with no Unicode normalization.
- **失敗の扱い**: The request operation synchronously rejects an invalid Target Identity and starts no attempt.

#### Scenario: RCL-ETR-1 Valid target is requested [happy]

- **GIVEN** a Host has one valid stable Target Identity
- **WHEN** it requests reconciliation
- **THEN** every attempt and reported outcome caused for that target contains that exact identity

#### Scenario: RCL-ETR-2 Target identity is invalid [error]

- **GIVEN** a Host has no valid Target Identity
- **WHEN** it requests reconciliation
- **THEN** the request is rejected and no attempt starts

#### Scenario: RCL-ETR-3 Target identity equality is exact [compatibility]

- **GIVEN** valid identities differ only by case, composed versus decomposed Unicode, or another byte distinction
- **WHEN** the Host requests them
- **THEN** the control loop treats them as distinct targets, while identities with byte-equal kind and key compare equal and embedded whitespace remains unchanged

### Requirement: level-based-attempt

Each Attempt MUST evaluate one target from facts acquired for that Attempt rather than from the Request occurrence or a previous outcome.

- **入力と受理**: The attempt receives the Target Identity and caller lifecycle required to acquire its target-specific current facts.
- **振る舞いの規則**: Request cause, count, metadata, payload, prior result, prior failure, and prior Control Directive are not supplied as authoritative decision inputs.
- **副作用**: The generic Controller does not acquire, interpret, or store target-specific Observations and does not decide whether the Attempt invokes semantic Reconciliation.

#### Scenario: RCL-LBA-1 Provider event wakes a target [happy]

- **GIVEN** a Provider event causes a Host to request one target
- **WHEN** the target's attempt evaluates it
- **THEN** the decision is based on current target-specific facts acquired for that attempt and not on the Provider event payload

#### Scenario: RCL-LBA-2 Prior result exists [compatibility]

- **GIVEN** a target has a prior successful result and a later request
- **WHEN** the later attempt evaluates the target
- **THEN** the prior result is not an authoritative input to that attempt

### Requirement: per-target-exclusion-and-coalescing

One caller-scoped control-loop instance MUST serialize Attempts for each Target Identity, preserve later eligibility for a request received during active work, and allow distinct targets to progress independently within one positive finite concurrency bound.

- **入力と受理**: The configured concurrency bound is a positive finite integer.
- **振る舞いの規則**: Duplicate requests may coalesce while a target is Pending. A request received while that target is Active preserves at least one later Attempt. Distinct targets may be Active concurrently.
- **排他・冪等**: At most one Attempt for a Target Identity is Active, and the total Active count never exceeds the configured bound.
- **失敗の扱い**: An invalid concurrency bound starts no control lifecycle.

#### Scenario: RCL-PEC-1 Duplicate pending requests may coalesce [idempotency]

- **GIVEN** one target is Pending but not Active
- **WHEN** the Host requests that target repeatedly
- **THEN** at least one serialized Attempt runs, no occurrence requires its own Attempt, and the loop may satisfy all Pending duplicates with one Attempt

#### Scenario: RCL-PEC-2 Request arrives during active attempt [concurrency]

- **GIVEN** one target has an Active Attempt
- **WHEN** the Host requests the same target before that Attempt returns
- **THEN** no second Attempt overlaps it and at least one later Attempt becomes eligible after it returns

#### Scenario: RCL-PEC-3 Distinct targets are requested [concurrency]

- **GIVEN** distinct valid targets and available concurrency capacity
- **WHEN** the Host requests them
- **THEN** their Attempts may run concurrently without exchanging target or result state

#### Scenario: RCL-PEC-4 Request and completion coincide [concurrency]

- **GIVEN** a target request arrives while its Active Attempt is returning
- **WHEN** the control loop resolves both occurrences
- **THEN** it leaves at least one later eligible Attempt and never runs two Attempts for that target concurrently

### Requirement: explicit-control-directive

The control loop MUST schedule internal reevaluation after a successful Attempt only according to that Attempt's one valid Control Directive.

- **入力と受理**: A successful completion contains exactly Await Another Request, Reevaluate Immediately, or Reevaluate After Delay with one positive finite delay.
- **振る舞いの規則**: Await creates no internal eligibility; Immediate creates eligibility without waiting for another request; Delay creates eligibility after its duration. A new request may make a delayed target eligible earlier.
- **排他・冪等**: Request eligibility and Directive eligibility coalesce without creating same-target overlap or duplicate obligations.
- **失敗の扱い**: A delayed Directive with a non-positive delay is rejected. If an otherwise successful target-specific Attempt provides no valid Directive, the Controller produces Control Directive Rejected, distinguishable from Target Attempt Failed, and schedules no Directive-based reevaluation.

#### Scenario: RCL-ECD-1 Attempt awaits another request [happy]

- **GIVEN** a successful Attempt selects Await Another Request
- **WHEN** the completion is accepted
- **THEN** no later Attempt is internally scheduled for that target

#### Scenario: RCL-ECD-2 Attempt requests immediate reevaluation [happy]

- **GIVEN** a successful Attempt selects Reevaluate Immediately
- **WHEN** the completion is accepted
- **THEN** the target becomes eligible for another serialized Attempt without an external request

#### Scenario: RCL-ECD-3 Attempt requests delayed reevaluation [happy]

- **GIVEN** a successful Attempt selects Reevaluate After Delay with a positive finite delay
- **WHEN** that delay expires without an earlier request
- **THEN** the target becomes eligible for another serialized Attempt

#### Scenario: RCL-ECD-4 Request precedes delayed eligibility [boundary]

- **GIVEN** one target is waiting for delayed reevaluation
- **WHEN** the Host requests it before the delay expires
- **THEN** it may become eligible immediately and the later delay does not create another pending obligation

#### Scenario: RCL-ECD-5 Non-positive delayed directive is rejected [error]

- **GIVEN** a caller supplies zero or negative duration for Reevaluate After Delay
- **WHEN** it submits the delayed Directive for use
- **THEN** the Directive is rejected and cannot direct an Attempt reevaluation

#### Scenario: RCL-ECD-6 Missing directive is rejected by control [error]

- **GIVEN** a target-specific Attempt reports success without a valid Directive
- **WHEN** the control loop processes the outcome
- **THEN** it reports one target-bound Control Directive Rejected failure, exposes no Completion, and schedules no Directive-based Attempt

#### Scenario: RCL-ECD-7 Directive is committed after Completion delivery [boundary]

- **GIVEN** a successful Attempt returns Reevaluate Immediately and its Completion Report is waiting for the consumer
- **WHEN** the consumer has not yet received that Report
- **THEN** the Directive creates no eligible Attempt until the Report is delivered

### Requirement: target-result-isolation

A successful Completion MUST preserve its target-owned value without requiring a shared result classification or claiming that semantic Reconciliation occurred.

- **振る舞いの規則**: The generic Controller associates the successful value with its Target Identity and Control Directive but does not inspect the value to determine scheduling or semantic status. The value may be a Reconciliation Result or another target-specific success, including a successful Observation for which no semantic Reconciliation was possible.
- **副作用**: The Controller does not convert target-specific evidence, proposals, conditions, effects, or non-Reconciliation successes into a universal result taxonomy and defines no universal Reconciliation input, Observation, Result, Failure, or outcome interface.

#### Scenario: RCL-TRI-1 Unrelated result types use control [compatibility]

- **GIVEN** two Controller instances use target-specific Attempt boundaries with unrelated successful-value meanings
- **WHEN** each successfully completes through its Controller
- **THEN** each Host consumer receives its target-specific result unchanged and scheduling follows only its Control Directive

#### Scenario: RCL-TRI-2 Successful Attempt performs no Reconciliation [boundary]

- **GIVEN** a target-specific Attempt establishes a successful Observation but cannot establish the inputs required for semantic Reconciliation
- **WHEN** the Controller publishes its Completion
- **THEN** it preserves the target-specific successful value and Directive without reporting that a Reconciliation Result exists

### Requirement: failure-does-not-retry

An Attempt Failure MUST remain target-bound, distinguish Target Attempt Failed from Control Directive Rejected, and MUST NOT create an implicit reevaluation.

- **振る舞いの規則**: Failure reports the exact valid Target Identity, one stable semantic failure kind, and the corresponding target-attempt or Directive-validation error without a successful result or Control Directive.
- **排他・冪等**: A request already received during the failed Active Attempt may cause later request-driven work; failure itself creates none.
- **失敗の扱い**: Failure for one target does not terminate or contaminate work for another target.

#### Scenario: RCL-FNR-1 Attempt fails without a pending request [error]

- **GIVEN** one target's Attempt fails and no later request is pending
- **WHEN** the failure is reported
- **THEN** no Attempt is automatically scheduled for that target

#### Scenario: RCL-FNR-2 Request was preserved before failure [idempotency]

- **GIVEN** a request arrived while an Attempt was Active
- **WHEN** that Attempt fails
- **THEN** a later Attempt may run because of the preserved request rather than because the failure was retried

#### Scenario: RCL-FNR-3 Another target continues [error]

- **GIVEN** distinct targets are being controlled
- **WHEN** one target's Attempt fails
- **THEN** the other target can continue and receives no state from that failure

#### Scenario: RCL-FNR-4 Invalid directive is distinguishable [error]

- **GIVEN** a target-specific Attempt boundary returns a successful value with an invalid Directive
- **WHEN** the control loop reports the Attempt Failure
- **THEN** the Host can distinguish Control Directive Rejected from Target Attempt Failed

#### Scenario: RCL-FNR-5 Target Attempt error outranks returned values [error]

- **GIVEN** a target-specific Attempt reports failure together with any nominal successful value and any valid or invalid Directive
- **WHEN** the control loop processes that outcome
- **THEN** it reports one Target Attempt Failed outcome preserving the exact target and error cause, exposes no Completion or Directive, and schedules no Directive-based Attempt

### Requirement: caller-lifecycle

The control loop MUST stop through the supplied caller lifecycle without establishing successful completion for unfinished work.

- **前提条件**: Each target-specific Attempt boundary observes the supplied cancellation and returns within its documented bound.
- **振る舞いの規則**: Cancellation stops request acceptance and new scheduling, discards Pending and Delayed eligibility, cancels Active Attempts, waits for them to return, closes the one stable Report stream, and makes the supplied caller context error available as the lifecycle outcome. After Active Attempts return, neither Report consumption nor Pending or Delayed work may add another wait condition.
- **Request context**: A request operation's context bounds only that submission. Once the operation returns success, later cancellation of its submission context does not cancel or become evidence for its Attempt. If both submission and Controller contexts have ended before acceptance, the Controller lifecycle outcome takes precedence.
- **出力配送**: While a prospective Report is awaiting publication, the Controller starts no new Attempt but continues accepting and coalescing Requests, permits already Active Attempts to return, and observes cancellation. While the lifecycle continues and the consumer receives Reports, every returned Attempt outcome is published exactly once without loss or duplication. Publication atomically commits the Completion or Failure and only then applies a successful Directive. This in-process publication is not delivery of a semantic Reconciliation Result to its Result Destination. Cancellation may discard the pending Report and unapplied Directive without publishing either outcome so that a stopped consumer cannot prevent lifecycle termination. No completion order is guaranteed across distinct concurrently Active targets.
- **失敗の扱い**: An Attempt that has not successfully completed before cancellation establishes no successful completion.

#### Scenario: RCL-CL-1 Caller cancels with active and pending work [concurrency]

- **GIVEN** the control loop has Active, Pending, or Delayed targets
- **WHEN** the caller cancels its lifecycle
- **THEN** no new Attempt starts, Active Attempts are cancelled and awaited, pending work is discarded, and unfinished Attempts produce no successful completion

#### Scenario: RCL-CL-2 Controller adds no shutdown wait after Attempts return [boundary]

- **GIVEN** every Active Attempt has returned after observing caller cancellation
- **WHEN** the control loop finishes stopping
- **THEN** it closes Reports and exposes the caller context error without waiting for Report consumption, Pending work, Delayed work, or another timer

#### Scenario: RCL-CL-3 Report consumer stops before cancellation [boundary]

- **GIVEN** a prospective Report is pending for a stopped consumer
- **WHEN** the caller cancels
- **THEN** the control loop may discard the undelivered Report without publishing its outcome, waits for every Active Attempt, closes the Report stream, and exposes the caller context error without waiting for Report consumption

#### Scenario: RCL-CL-4 Attempt boundary returns nominal success after cancellation [concurrency]

- **GIVEN** one Active target-specific Attempt boundary observes caller cancellation and then returns a nominal successful value and Directive
- **WHEN** the control loop receives that return after stopping began
- **THEN** it establishes no Completion Report or Directive-based eligibility, waits for the Attempt boundary return, closes Reports, and exposes the caller context error

#### Scenario: RCL-CL-5 Submission context ends after acceptance [boundary]

- **GIVEN** a request operation has returned success for one target
- **WHEN** only that operation's submission context is later cancelled
- **THEN** the accepted Attempt remains governed by the Controller lifecycle and receives neither cancellation nor evidence from the submission context

#### Scenario: RCL-CL-6 Submission and Controller contexts have ended [error]

- **GIVEN** both a request submission context and the Controller lifecycle context have ended before the request is accepted
- **WHEN** the Host submits the request
- **THEN** the operation returns the Controller lifecycle context error and starts no Attempt

#### Scenario: RCL-CL-7 Report delivery commits before cancellation [happy]

- **GIVEN** one prospective successful Report and caller cancellation have not yet committed
- **WHEN** Report delivery commits before cancellation
- **THEN** the one lifecycle Report stream publishes exactly one target-bound Completion, applies its Directive afterward, and never duplicates that outcome

#### Scenario: RCL-CL-8 Pending Report applies bounded backpressure [boundary]

- **GIVEN** one Report is pending delivery while requests arrive and other Active Attempts return
- **WHEN** the Report consumer remains blocked
- **THEN** no new Attempt starts, new Requests are accepted and may coalesce, already Active Attempt boundaries can return, and cancellation still closes Reports and completes Wait without consumer progress

#### Scenario: RCL-CL-9 Cancellation commits before Report delivery [concurrency]

- **GIVEN** one target Attempt has returned a prospective outcome that has not been delivered
- **WHEN** caller cancellation commits before Report delivery
- **THEN** the Controller discards that prospective outcome regardless of Attempt return time or consumer activity and publishes no Completion or Failure for it

#### Scenario: RCL-CL-10 Simultaneous delivery and cancellation have one winner [concurrency]

- **GIVEN** a prospective successful Report, a ready Report consumer, and caller cancellation become ready concurrently
- **WHEN** the public lifecycle resolves the race
- **THEN** exactly one outcome holds: the Report is published once and its Directive commits before stopping, or no Report is published and no Directive-based Attempt starts; Reports closes and Wait returns the caller context error in either case

#### Scenario: RCL-CL-11 Normal operation reports every returned outcome once [happy]

- **GIVEN** multiple target Attempts return outcomes, the lifecycle is not cancelled, and the consumer continues receiving Reports
- **WHEN** normal Report delivery proceeds
- **THEN** every returned outcome is published exactly once without loss or duplication, and the Controller promises no completion order between distinct concurrently Active targets

### Requirement: disposable-control-state

The control loop MUST retain no authoritative target, outcome, or durable scheduling state and MUST be able to reconcile after state loss from a new request and newly acquired facts.

- **振る舞いの規則**: Prior requests, completions, failures, pending eligibility, and delayed eligibility are not required restoration inputs.
- **副作用**: The control loop creates no authoritative repository, lease, leader record, or durable queue.

#### Scenario: RCL-DCS-1 Control state is discarded [compatibility]

- **GIVEN** all prior control-loop state has been lost
- **WHEN** a new control lifecycle receives a valid target request
- **THEN** it can start an Attempt that reacquires current facts without restoring a prior request or outcome
