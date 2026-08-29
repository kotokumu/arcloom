# plan-application-request Specification

## Purpose
Requests external application of an exact authorized Plan Revision while preserving uncertainty and leaving target state and lifecycle authoritative in the Planning Context.

## Conceptual Model

### External Plan Target Reference and Plan Revision

An External Plan Target Reference is a stable provider-independent identifier for one externally owned Plan representation; it is not target state. A Plan Revision binds one such reference, one caller-established current Plan, and one valid meaningfully unequal proposed Plan. Arcloom preserves this association without independently asserting its provenance.

### Application Request and Result

An Application Request is a passive instruction concerning one exact Plan Revision. A Plan Application Result is either AuthorizationDenied, AuthorizationUndecidable, or Request Receipt Evidence. Request Receipt Evidence is exactly ReceiptAcknowledged, ReceiptRefused, KnownNotReceived, or ReceiptUncertain and describes only whether the exact request was received. No application result establishes mutation, completion, or current external Plan state.

## Requirements

### Requirement: exact-plan-revision

A Plan application request MUST concern one exact valid Plan Revision.

- **入力と受理**: A Revision contains one External Plan Target Reference, one valid caller-established current Plan, and one valid meaningfully unequal proposed Plan.
- **振る舞いの規則**: The Application Request preserves those exact values.
- **失敗の扱い**: An invalid or meaningfully unchanged Revision produces no external application request.
- **参照**: [related] `openspec/specs/plan/spec.md`

#### Scenario: Revision is valid [happy]

- **GIVEN** one valid target reference and valid meaningfully different current and proposed Plans
- **WHEN** the caller establishes a Plan Revision
- **THEN** it represents that exact revision for that exact target

#### Scenario: Revision is invalid or unchanged [error]

- **GIVEN** either Plan is invalid or the current and proposed Plans are meaningfully equal
- **WHEN** the caller requests external application
- **THEN** no Application Request is sent

### Requirement: caller-established-target-association

The capability MUST preserve the caller-established association between the External Plan Target Reference and current Plan without claiming independent verification.

- **前提条件**: The caller owns the precondition that the target and current Plan concern the same fresh external observation.
- **振る舞いの規則**: The same target/current association remains part of the Revision throughout one consideration.

#### Scenario: Caller supplies an association [happy]

- **GIVEN** a caller has established a valid target/current Plan association
- **WHEN** its valid Revision is considered for external application
- **THEN** that same association is preserved without an independent verification claim

### Requirement: invalid-application-input

Invalid application input MUST prevent Actor contact and produce the applicable stable failure meaning.

- **入力と受理**: A valid Revision, valid Authorization Policy, and Actor are required.
- **失敗の扱い**: An invalid Revision or missing Actor produces a stable invalid-input failure. An invalid Authorization Policy produces AuthorizationUndecidable.
- **副作用**: Invalid input sends no Application Request.

#### Scenario: Revision or Actor is invalid [error]

- **GIVEN** an invalid Revision or no Actor
- **WHEN** the caller requests external application
- **THEN** the applicable stable invalid-input failure is reported and no Actor is contacted

#### Scenario: Authorization Policy is invalid [error]

- **GIVEN** a valid Revision and Actor but an invalid Authorization Policy
- **WHEN** the caller requests external application
- **THEN** the result is AuthorizationUndecidable and no Actor is contacted

### Requirement: current-authorization-required

An Actor MUST receive an Application Request only when the exact Revision is Authorized during the current invocation.

- **前提条件**: Authorization evaluates the exact Revision as its Authorization Subject.
- **振る舞いの規則**: Current Authorized permits one possible transmission; current Denied produces AuthorizationDenied; current Undecidable produces AuthorizationUndecidable.
- **副作用**: Denied or Undecidable sends no Application Request.
- **参照**: [[authorization/exact-authorization-subject]]; [[authorization/authorization-decision-semantics]]

#### Scenario: Revision is Authorized [happy]

- **GIVEN** the exact Revision is Authorized in the current invocation
- **WHEN** the caller requests external application
- **THEN** that exact Revision may be sent to the Actor

#### Scenario: Revision is Denied [permission]

- **GIVEN** the exact Revision is Denied in the current invocation
- **WHEN** the caller requests external application
- **THEN** the result is AuthorizationDenied and no request is sent to the Actor

#### Scenario: Revision is Undecidable [permission]

- **GIVEN** the exact Revision is Undecidable in the current invocation
- **WHEN** the caller requests external application
- **THEN** the result is AuthorizationUndecidable and no request is sent to the Actor

#### Scenario: A prior authorization exists [permission]

- **GIVEN** the exact Revision was Authorized earlier but is not Authorized in the current invocation
- **WHEN** the caller requests external application
- **THEN** no request is sent to the Actor

### Requirement: at-most-one-transmission

One application-request invocation MUST transmit to the Actor at most once and MUST NOT retry after receipt may have occurred.

- **排他・冪等**: Once transmission may have begun, the same invocation cannot become sendable again.

#### Scenario: Actor receipt is uncertain [idempotency]

- **GIVEN** an Authorized Application Request may have reached the Actor
- **WHEN** communication fails before receipt can be established
- **THEN** the invocation does not retransmit that request

### Requirement: request-interaction-result

A Plan Application Result MUST describe only authorization or request receipt and keep all defined outcomes distinct.

- **振る舞いの規則**: The result is exactly AuthorizationDenied, AuthorizationUndecidable, ReceiptAcknowledged, ReceiptRefused, KnownNotReceived, or ReceiptUncertain.
- **副作用**: No result claims that the external Plan changed.

#### Scenario: Actor explicitly acknowledges [happy]

- **GIVEN** the Actor explicitly acknowledges receipt of the exact Application Request
- **WHEN** the capability establishes the result
- **THEN** it is ReceiptAcknowledged without claiming that the external Plan changed

#### Scenario: Actor explicitly refuses [happy]

- **GIVEN** the Actor explicitly refuses the exact Application Request
- **WHEN** the capability establishes the result
- **THEN** it is ReceiptRefused

#### Scenario: Receipt cannot be determined [error]

- **GIVEN** the capability cannot establish whether the Actor received the request
- **WHEN** it establishes the result
- **THEN** the result is ReceiptUncertain

#### Scenario: Request was definitely not received [happy]

- **GIVEN** the capability establishes that the Actor did not receive the request
- **WHEN** it establishes the result
- **THEN** the result is KnownNotReceived

### Requirement: cancellation-preserves-uncertainty

Cancellation MUST prevent transmission when observed before sending and preserve the strongest established receipt meaning after transmission may have begun.

- **振る舞いの規則**: Established ReceiptAcknowledged, ReceiptRefused, or KnownNotReceived remains the result. Without stronger evidence after possible receipt, the result is ReceiptUncertain.
- **排他・冪等**: Cancellation never causes retry.
- **失敗の扱い**: Cancellation before Actor invocation returns the caller's cancellation outcome and sends no request. An Actor that observes cancellation before transmission yields KnownNotReceived.

#### Scenario: Cancelled before transmission [error]

- **GIVEN** an application-request invocation has not begun transmission
- **WHEN** cancellation is observed
- **THEN** no request is sent to the Actor

#### Scenario: Cancelled after possible receipt [boundary]

- **GIVEN** the Actor might have received the request and no explicit acknowledgment or refusal exists
- **WHEN** cancellation is observed
- **THEN** the result is ReceiptUncertain

#### Scenario: Receipt evidence is established before cancellation [boundary]

- **GIVEN** ReceiptAcknowledged, ReceiptRefused, or KnownNotReceived is established
- **WHEN** cancellation is observed afterward
- **THEN** the result preserves that exact evidence and no retry occurs

#### Scenario: Cancellation wins before Actor invocation [error]

- **GIVEN** cancellation has been observed before Actor invocation begins
- **WHEN** the application-request invocation resolves
- **THEN** the caller's cancellation outcome is returned and no request is sent

#### Scenario: Actor observes cancellation before transmission [boundary]

- **GIVEN** the Actor has been invoked but has not begun transmission
- **WHEN** the Actor observes cancellation
- **THEN** it does not transmit and the result is KnownNotReceived

### Requirement: external-state-requires-fresh-observation

No Plan Application Result MUST establish the current external Plan; only a later current observation may establish it.

- **振る舞いの規則**: Receipt evidence concerns the request interaction only and cannot establish mutation, completion, or causality.
- **参照**: [[github-plan-snapshot/current-authoritative-snapshot]]

#### Scenario: Application was acknowledged [happy]

- **GIVEN** an Application Request resulted in ReceiptAcknowledged
- **WHEN** the caller needs the current external Plan
- **THEN** the caller uses a fresh authoritative observation to establish it

### Requirement: stateless-provider-independent-meaning

The application-request capability MUST retain no authoritative application state and express no provider-specific mutation semantics.

- **振る舞いの規則**: A later invocation receives no prior Result as authoritative input.
- **副作用**: The capability owns neither external target state nor a durable application lifecycle.

#### Scenario: A later invocation begins [happy]

- **GIVEN** one or more prior Plan Application Results exist
- **WHEN** another application request is considered
- **THEN** no prior Result is treated as authoritative input for that request
