## Purpose

Establishes a current Plan and provider-independent progress from one authoritative GitHub Milestone observation without making Arcloom authoritative for either.

## ADDED Requirements

### Requirement: exact-milestone-target

GitHub Plan Snapshot observation MUST concern the exact valid GitHub Milestone Target supplied by the caller.

- **入力と受理**: The target contains a valid repository identity and positive Milestone number.
- **失敗の扱い**: An invalid target is rejected before GitHub access.

#### Scenario: Valid target is observed [happy]

- **GIVEN** a caller has one valid GitHub Milestone Target
- **WHEN** the caller requests a GitHub Plan Snapshot
- **THEN** the observation concerns exactly that repository and Milestone number

#### Scenario: Invalid target [error]

- **GIVEN** a target with an invalid repository identity or Milestone number
- **WHEN** the caller requests a GitHub Plan Snapshot
- **THEN** the capability reports invalid target input without accessing GitHub

### Requirement: current-authoritative-snapshot

Each successful GitHub Plan Snapshot MUST be derived from a fresh observation of the targeted GitHub Milestone and its currently observable membership.

- **振る舞いの規則**: Earlier GitHub Plan Snapshots are not fact sources for a later observation.

#### Scenario: Re-observe changed facts [happy]

- **GIVEN** GitHub facts have changed since an earlier GitHub Plan Snapshot
- **WHEN** the caller requests a new GitHub Plan Snapshot
- **THEN** it reflects the newly observed facts rather than the earlier GitHub Plan Snapshot

### Requirement: current-plan-eligibility

A GitHub Plan Snapshot MUST expose a current Plan only when all required current GitHub facts and membership are known, coherent, valid, and complete.

- **振る舞いの規則**: A valid complete representation yields that current Plan. A coherent root and Representation Progress remain observable without a current Plan when Plan meaning or complete membership cannot be established.
- **失敗の扱い**: Missing, conflicting, invalid, incomplete, unsupported, or unusable required Plan facts never produce a current Plan and never establish authoritative Plan absence.
- **参照**: [related] `openspec/specs/plan/spec.md`; [related] `openspec/specs/github-plan-representation-observation/spec.md`

#### Scenario: Plan facts are complete [happy]

- **GIVEN** the current Milestone representation contains a valid and complete Plan
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** the GitHub Plan Snapshot contains that current Plan

#### Scenario: Plan facts are not trustworthy [error]

- **GIVEN** required Plan facts are missing, conflicting, invalid, or incomplete
- **WHEN** the caller obtains a successful GitHub Plan Snapshot from a coherent current root and progress
- **THEN** it contains no current Plan, preserves coherent Representation Progress, and does not claim authoritative Plan absence

#### Scenario: Payload cannot establish Plan meaning [error]

- **GIVEN** the current root and membership progress are coherent but required Plan narrative meaning is missing, unsupported, or unusable
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains that progress without a current Plan

#### Scenario: Membership cannot be completed [boundary]

- **GIVEN** the current root and some coherent member progress are established but complete membership cannot be established
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains Incomplete Representation Progress without a current Plan

#### Scenario: Known Plan values violate Plan invariants [error]

- **GIVEN** the current root and membership progress are coherent but known Plan values violate Plan invariants
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains the coherent progress without a current Plan

### Requirement: provider-independent-progress

A GitHub Plan Snapshot MUST express Representation Progress without GitHub-specific identity or lifecycle meaning and MUST NOT treat representation state as Plan completion.

- **振る舞いの規則**: Progress contains overall state, ordered observed member names and states, and Complete or Incomplete membership. Overall and member states are Open, Closed, or Unknown.
- **参照**: [related] `openspec/specs/plan-control/spec.md` (PLC-2)

#### Scenario: Current membership is complete [happy]

- **GIVEN** all current Milestone members and their states are known
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains provider-independent member names and states with Complete membership

#### Scenario: Current membership is incomplete [boundary]

- **GIVEN** complete Milestone membership cannot be established after coherent progress is known
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** its Representation Progress has Incomplete membership

#### Scenario: Distinct members have the same name [boundary]

- **GIVEN** distinct observed members have the same name
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** progress preserves every observed member and the GitHub Plan Snapshot contains no current Plan when those names violate Plan invariants

### Requirement: observation-failure

An unavailable or unsuccessful GitHub observation MUST NOT become a successful current GitHub Plan Snapshot or authoritative absence.

- **振る舞いの規則**: A coherent current root with partial member progress produces a successful GitHub Plan Snapshot with Incomplete progress and no current Plan.
- **失敗の扱い**: When no coherent current root and Representation Progress can be established, observation returns no successful GitHub Plan Snapshot and makes no absence claim.

#### Scenario: GitHub cannot be observed [error]

- **GIVEN** no coherent current root and Representation Progress can be obtained or interpreted safely
- **WHEN** the caller requests a GitHub Plan Snapshot
- **THEN** no successful current GitHub Plan Snapshot is returned

#### Scenario: Root cannot be established [error]

- **GIVEN** the current root fact cannot be established
- **WHEN** the caller requests a GitHub Plan Snapshot
- **THEN** observation fails without a GitHub Plan Snapshot and without claiming authoritative absence

#### Scenario: Member collection fails after the root [boundary]

- **GIVEN** a coherent current root and some member progress are established before complete membership becomes unavailable
- **WHEN** the caller requests a GitHub Plan Snapshot
- **THEN** a successful GitHub Plan Snapshot contains Incomplete progress and no current Plan

### Requirement: observation-isolation

Each GitHub Plan Snapshot observation MUST remain bound to its own target and caller lifecycle.

- **排他・冪等**: Concurrent observations exchange no target or result facts.
- **失敗の扱い**: Cancellation before success yields no successful GitHub Plan Snapshot for that invocation.

#### Scenario: Observation is cancelled [error]

- **GIVEN** a GitHub Plan Snapshot observation has not completed
- **WHEN** its caller cancels the observation
- **THEN** no successful GitHub Plan Snapshot is returned for that invocation

#### Scenario: Targets are observed concurrently [concurrency]

- **GIVEN** distinct valid GitHub Milestone Targets
- **WHEN** callers observe them concurrently
- **THEN** each result contains facts only for its own target
