## Purpose

Establish a current Plan and provider-independent progress evidence from one authoritative GitHub Milestone observation without making Arcloom authoritative for either.

## ADDED Requirements

### Requirement: GHPS-1 Exact Milestone target

The capability SHALL observe the exact GitHub repository and positive Milestone number supplied by the caller.

#### Scenario: Invalid target

- **WHEN** the repository identity or Milestone number is invalid
- **THEN** the capability reports an invalid target without accessing GitHub

### Requirement: GHPS-2 Current authoritative snapshot

Each snapshot SHALL be derived from a current observation of the targeted GitHub Milestone and its current membership.

#### Scenario: Re-observe changed facts

- **WHEN** GitHub facts have changed since an earlier snapshot
- **THEN** a new snapshot reflects the newly observed facts rather than the earlier snapshot

### Requirement: GHPS-3 Current Plan eligibility

The capability SHALL expose a current Plan only when the required GitHub facts are known, coherent, valid, and complete.

#### Scenario: Plan facts are complete

- **WHEN** the current Milestone representation contains a valid and complete Plan
- **THEN** the snapshot contains that Plan

#### Scenario: Plan facts are not trustworthy

- **WHEN** required Plan facts are missing, conflicting, invalid, or incomplete
- **THEN** the snapshot does not contain a Plan, preserves any coherent representation progress, and does not claim that an authoritative Plan is absent

#### Scenario: Payload cannot establish Plan meaning

- **WHEN** the current root and membership progress are coherent but the Plan payload is missing, unsupported, or unusable
- **THEN** a successful snapshot contains that progress without a current Plan

#### Scenario: Membership cannot be completed

- **WHEN** the current root and some coherent member progress are established but complete membership cannot be established
- **THEN** a successful snapshot contains incomplete progress without a current Plan

#### Scenario: Known Plan values violate Plan invariants

- **WHEN** the current root and membership progress are coherent but known Plan values violate Plan invariants
- **THEN** a successful snapshot contains the coherent progress without a current Plan

### Requirement: GHPS-4 Provider-independent progress

The snapshot SHALL express overall representation state, member names and states, and membership completeness without exposing GitHub-specific identities or lifecycle terms. Representation state SHALL remain observation material and SHALL NOT establish that the Plan is Complete.

#### Scenario: Current membership is complete

- **WHEN** all current Milestone members and their states are known
- **THEN** the snapshot contains provider-independent item names, states, and a complete progress indication

#### Scenario: Current membership is incomplete

- **WHEN** complete Milestone membership cannot be established
- **THEN** the snapshot marks progress as incomplete

#### Scenario: Distinct members have the same name

- **WHEN** distinct observed members have the same name
- **THEN** progress preserves each observed member and the snapshot exposes no current Plan when the duplicate names violate Plan invariants

### Requirement: GHPS-5 Observation failure does not become state

An unavailable or unsuccessful GitHub observation SHALL NOT be represented as a successful current snapshot or as authoritative absence. Observation SHALL fail when no coherent current root and representation progress can be established. Inability to complete member collection after a coherent root is established SHALL instead produce a successful snapshot with incomplete progress and no current Plan.

#### Scenario: GitHub cannot be observed

- **WHEN** GitHub facts cannot be obtained or interpreted safely
- **THEN** no successful current snapshot is returned

#### Scenario: Root cannot be established

- **WHEN** the current root fact cannot be established
- **THEN** observation fails without a snapshot and without claiming authoritative absence

#### Scenario: Member collection fails after the root

- **WHEN** a coherent current root is established but member collection later becomes unavailable
- **THEN** a successful snapshot contains incomplete progress and no current Plan

### Requirement: GHPS-6 Isolation of observations

Cancellation or concurrency SHALL NOT cause one observation to yield another observation's result.

#### Scenario: Observation is cancelled

- **WHEN** the caller cancels an observation before completion
- **THEN** no successful snapshot is returned for that observation

#### Scenario: Targets are observed concurrently

- **WHEN** distinct targets are observed concurrently
- **THEN** each result contains facts only for its own target
