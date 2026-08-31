## Purpose

Provides a generic, recalculable authorization decision for the exact subject supplied by a consumer without retaining authoritative authorization state.

## ADDED Requirements

### Requirement: exact-authorization-subject

Authorization MUST evaluate the exact established Authorization Subject supplied by the consumer and bind the resulting Authorization Evaluation to that subject.

- **Input and Acceptance**: The consumer supplies an Authorization Subject whose domain validity has already been established by its owner and a valid Authorization Policy.
- **Behavioral Rules**: Authorization neither interprets nor substitutes the subject's domain meaning. The resulting Evaluation contains that exact subject and its aggregate Decision.
- **Failure Handling**: An empty or otherwise invalid Policy cannot produce Authorized.

#### Scenario: Valid subject and policy [happy]

- **GIVEN** a consumer has an established Authorization Subject and a valid Authorization Policy
- **WHEN** the consumer requests authorization
- **THEN** the resulting Evaluation refers to that exact subject

#### Scenario: Policy is invalid [error]

- **GIVEN** an established Authorization Subject and an empty or otherwise invalid Policy
- **WHEN** the consumer requests authorization
- **THEN** the result is not Authorized

### Requirement: authorization-decision-semantics

Authorization MUST aggregate current Rule Conclusions using deny-overrides, all-permit, otherwise-undecidable semantics.

- **Behavioral Rules**: At least one Deny produces Denied; every applicable Rule yielding Permit produces Authorized; every other combination produces Undecidable.

#### Scenario: A rule denies [happy]

- **GIVEN** a valid Policy whose applicable Rules include at least one Deny
- **WHEN** the Policy is evaluated for an exact Authorization Subject
- **THEN** the Decision is Denied

#### Scenario: All rules permit [happy]

- **GIVEN** a valid Policy whose every applicable Rule yields Permit
- **WHEN** the Policy is evaluated for an exact Authorization Subject
- **THEN** the Decision is Authorized

#### Scenario: No definitive result exists [boundary]

- **GIVEN** a valid Policy with no Deny and at least one applicable Rule that cannot yield Permit
- **WHEN** the Policy is evaluated for an exact Authorization Subject
- **THEN** the Decision is Undecidable

### Requirement: missing-authorization-evidence

Missing, unavailable, invalid, or failed authorization evidence MUST NOT produce Permit.

- **Behavioral Rules**: A Rule that cannot establish the evidence it requires yields no permission for the subject.

#### Scenario: Required evidence is unavailable [error]

- **GIVEN** an applicable Rule requires evidence for an exact Authorization Subject
- **WHEN** that evidence cannot be established
- **THEN** the Rule does not permit the subject

### Requirement: authorization-cancellation

Cancellation before authorization completion MUST establish no Authorization Decision.

- **Failure Handling**: The caller observes its cancellation outcome and receives no established Evaluation.

#### Scenario: Evaluation is cancelled [error]

- **GIVEN** authorization evaluation has not completed
- **WHEN** the caller cancels the evaluation
- **THEN** no Authorization Decision is established

### Requirement: authorization-recalculation-and-isolation

Each Authorization Evaluation MUST be recalculated from its current supplied subject, Policy, and evidence and remain isolated from other evaluations.

- **Behavioral Rules**: Prior Decisions are not inputs to a later Evaluation. Authorization does not modify the supplied subject, whose reachable semantic state remains unchanged for the Evaluation's lifetime.
- **Concurrency and Idempotency**: Concurrent evaluations exchange no subject, evidence, or Decision state.

#### Scenario: Prior authorization exists [happy]

- **GIVEN** the same Authorization Subject has a prior Evaluation and its Policy or evidence has changed
- **WHEN** the subject is evaluated again
- **THEN** the new Decision is based only on the current Evaluation inputs

#### Scenario: Subjects are evaluated concurrently [concurrency]

- **GIVEN** distinct Authorization Subjects and their current Policies and evidence
- **WHEN** consumers evaluate them concurrently
- **THEN** each Evaluation refers only to its own subject and evidence
