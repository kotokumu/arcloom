## Purpose

Provide a generic, recalculable authorization decision for the exact subject supplied by a consumer without retaining authoritative authorization state.

## ADDED Requirements

### Requirement: AUTH-1 Exact authorization subject

Authorization SHALL evaluate the exact subject supplied by the consumer against the supplied policy without interpreting or substituting its domain meaning. The consumer SHALL supply a subject whose validity is already established by the subject's owner.

#### Scenario: Valid subject and policy

- **WHEN** a consumer requests authorization with an established subject and valid policy
- **THEN** the resulting decision refers to that exact subject

#### Scenario: Policy is invalid

- **WHEN** the policy is empty or otherwise invalid
- **THEN** the result is not Authorized

### Requirement: AUTH-2 Decision semantics

Authorization SHALL decide Denied when any applicable rule denies the subject, Authorized when every applicable rule permits it, and Undecidable otherwise.

#### Scenario: A rule denies

- **WHEN** at least one applicable rule returns Deny
- **THEN** the decision is Denied

#### Scenario: All rules permit

- **WHEN** every applicable rule returns Permit
- **THEN** the decision is Authorized

#### Scenario: No definitive result exists

- **WHEN** no rule denies and at least one applicable rule cannot permit
- **THEN** the decision is Undecidable

### Requirement: AUTH-3 Missing evidence fails closed

Missing, unavailable, invalid, or failed evidence SHALL NOT produce Permit.

#### Scenario: Required evidence is unavailable

- **WHEN** a rule cannot establish its required evidence
- **THEN** that rule does not permit the subject

### Requirement: AUTH-4 Cancellation establishes no decision

Cancellation before completion SHALL establish no authorization decision.

#### Scenario: Evaluation is cancelled

- **WHEN** authorization evaluation is cancelled before completion
- **THEN** no decision is established

### Requirement: AUTH-5 Recalculation and isolation

Each authorization decision SHALL be recalculated from the current supplied subject, policy, and evidence, and concurrent decisions SHALL remain isolated.

Evaluation SHALL NOT modify the supplied subject. The subject and state reachable from it SHALL remain semantically immutable for the complete lifetime of the returned subject-bound Evaluation. A policy used concurrently SHALL NOT exchange subject, evidence, or decision state between evaluations.

#### Scenario: Prior authorization exists

- **WHEN** the same subject is evaluated again with changed policy or evidence
- **THEN** the new decision is based only on the current evaluation inputs

#### Scenario: Subjects are evaluated concurrently

- **WHEN** distinct subjects are evaluated concurrently
- **THEN** each decision refers only to its own subject and evidence
