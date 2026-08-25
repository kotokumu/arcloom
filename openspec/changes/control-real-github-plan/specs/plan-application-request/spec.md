## Purpose

Request external application of an exact authorized Plan revision while preserving uncertainty. The external Actor owns action-time interpretation and mutation, while the Planning Context remains authoritative for target state and lifecycle.

## ADDED Requirements

### Requirement: PAR-1 Exact Plan revision

A Plan application request SHALL identify one target, its caller-established current Plan, and one proposed Plan.

#### Scenario: Revision is valid

- **WHEN** the current and proposed Plans are valid and meaningfully different
- **THEN** the request represents that exact revision for the exact target

#### Scenario: Revision is invalid or unchanged

- **WHEN** either Plan is invalid or the Plans are meaningfully equal
- **THEN** no external application is requested

### Requirement: PAR-2 Caller-established target association

The capability SHALL preserve the association supplied by the caller between the target and the current Plan without asserting that it independently verified the association.

#### Scenario: Caller supplies an association

- **WHEN** a valid application request is considered
- **THEN** the same target and current Plan association is used throughout that consideration

### Requirement: PAR-2A Invalid application input

An invalid Revision or missing Actor SHALL produce a stable invalid-input failure and SHALL NOT contact an Actor. An invalid Authorization Policy SHALL produce AuthorizationUndecidable and SHALL NOT contact an Actor.

#### Scenario: Revision or Actor is invalid

- **WHEN** the Revision is invalid or no Actor is supplied
- **THEN** the capability reports the applicable stable invalid-input failure and sends no request

#### Scenario: Authorization Policy is invalid

- **WHEN** the Authorization Policy is invalid
- **THEN** the result is AuthorizationUndecidable and sends no request

### Requirement: PAR-3 Current authorization is required

An external Actor SHALL receive a Plan application request only when the exact revision is Authorized during the current invocation.

#### Scenario: Revision is Authorized

- **WHEN** the current authorization decision is Authorized
- **THEN** the exact revision may be sent to the Actor

#### Scenario: Revision is Denied

- **WHEN** the current authorization decision is Denied
- **THEN** the result is AuthorizationDenied and no request is sent to the Actor

#### Scenario: Revision is Undecidable

- **WHEN** the current authorization decision is Undecidable
- **THEN** the result is AuthorizationUndecidable and no request is sent to the Actor

#### Scenario: A prior authorization exists

- **WHEN** the exact revision was Authorized in an earlier invocation but is not Authorized in the current invocation
- **THEN** no request is sent to the Actor

### Requirement: PAR-4 At most one transmission

One invocation SHALL transmit the application request to the Actor at most once and SHALL NOT retry after the Actor might have received it.

#### Scenario: Actor receipt is uncertain

- **WHEN** communication fails after the Actor might have received the request
- **THEN** the capability does not retransmit the request

### Requirement: PAR-5 Request-interaction result

The result SHALL describe only the application-request interaction and SHALL distinguish AuthorizationDenied, AuthorizationUndecidable, ReceiptAcknowledged, ReceiptRefused, KnownNotReceived, and ReceiptUncertain.

#### Scenario: Actor explicitly acknowledges

- **WHEN** the Actor explicitly acknowledges the request
- **THEN** the result is ReceiptAcknowledged without claiming the external Plan changed

#### Scenario: Actor explicitly refuses

- **WHEN** the Actor explicitly refuses the request
- **THEN** the result is ReceiptRefused

#### Scenario: Receipt cannot be determined

- **WHEN** the capability cannot establish whether the Actor received the request
- **THEN** the result is ReceiptUncertain

#### Scenario: Request was definitely not received

- **WHEN** the capability establishes that the Actor did not receive the request
- **THEN** the result is KnownNotReceived

### Requirement: PAR-6 Cancellation preserves uncertainty

Cancellation SHALL prevent transmission when it occurs before sending and preserve uncertainty when it occurs after receipt may have happened.

#### Scenario: Cancelled before transmission

- **WHEN** the invocation is cancelled before transmission begins
- **THEN** no request is sent to the Actor

#### Scenario: Cancelled after possible receipt

- **WHEN** the invocation is cancelled after the Actor might have received the request and no explicit acknowledgment or refusal exists
- **THEN** the result is ReceiptUncertain

#### Scenario: Receipt evidence is established before cancellation

- **WHEN** ReceiptAcknowledged, ReceiptRefused, or KnownNotReceived is established before cancellation is observed
- **THEN** the result preserves that exact receipt evidence and no retry occurs

#### Scenario: Cancellation wins before Actor invocation

- **WHEN** cancellation is observed before Actor invocation begins
- **THEN** the supplied cancellation result is returned and no request is sent

#### Scenario: Actor observes cancellation before transmission

- **WHEN** the Actor has been invoked but observes cancellation before beginning transmission
- **THEN** the Actor does not transmit and the result is KnownNotReceived

### Requirement: PAR-7 External state requires fresh observation

No application result SHALL establish the current external Plan; only a later current observation may establish it.

#### Scenario: Application was acknowledged

- **WHEN** the caller needs to know the current external Plan after acknowledgment
- **THEN** the caller must use a fresh authoritative observation

### Requirement: PAR-8 Stateless and provider-independent meaning

The capability SHALL retain no authoritative application state and SHALL express the request without provider-specific mutation semantics.

#### Scenario: A later invocation begins

- **WHEN** another application request is considered
- **THEN** no prior result is treated as authoritative input for that request
