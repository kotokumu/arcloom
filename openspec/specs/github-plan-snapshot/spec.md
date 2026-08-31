# github-plan-snapshot Specification

## Purpose
Establishes a current Plan and provider-independent progress from one authoritative GitHub Milestone observation without making Arcloom authoritative for either.

## Conceptual Model

### GitHub Milestone Target

A GitHub Milestone Target identifies one exact repository and positive Milestone number for observation. It identifies the external subject but does not establish its state.

### GitHub Plan Snapshot

A GitHub Plan Snapshot is one disposable coherent result from one fresh observation of a GitHub Milestone Target. It exposes a current Plan only when every required Plan fact and the complete current membership are known, coherent, and valid. When a coherent current root and progress are established but Plan eligibility is not, the GitHub Plan Snapshot preserves that progress without a current Plan. When no coherent current root and progress can be established, no successful GitHub Plan Snapshot exists. A current Plan exposed by this capability can serve as the Plan Snapshot defined by `plan-control`; the enclosing GitHub Plan Snapshot is not that concept.

### Representation Progress

Representation Progress contains the overall representation state, ordered observed member names and states, and whether membership is Complete or Incomplete. Overall and member state are Open, Closed, or Unknown. Repeated names from distinct members remain distinct. Representation Progress is observation material and never establishes that the Plan is Complete.

## Requirements

### Requirement: exact-milestone-target

GitHub Plan Snapshot observation MUST concern the exact valid GitHub Milestone Target supplied by the caller.

- **Input and Acceptance**: The target contains a valid repository identity and positive Milestone number.
- **Failure Handling**: An invalid target is rejected before GitHub access.

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

- **Behavioral Rules**: Earlier GitHub Plan Snapshots are not fact sources for a later observation.

#### Scenario: Re-observe changed facts [happy]

- **GIVEN** GitHub facts have changed since an earlier GitHub Plan Snapshot
- **WHEN** the caller requests a new GitHub Plan Snapshot
- **THEN** it reflects the newly observed facts rather than the earlier GitHub Plan Snapshot

### Requirement: current-plan-eligibility

A GitHub Plan Snapshot MUST expose a current Plan only when all required current GitHub facts and membership are known, coherent, valid, and complete.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Current Plan established | Root, required values, and membership are coherent, valid, and complete | Snapshot is formed | Preserve the exact current Plan and Representation Progress | None |
  | Current Plan not established | A coherent root and Progress exist | Required meaning is missing, conflicting, invalid, unsupported, unusable, or membership is Incomplete | Preserve coherent Progress without a current Plan | No authoritative absence claim. |
- **Failure Handling**: Missing, conflicting, invalid, incomplete, unsupported, or unusable required Plan facts never produce a current Plan and never establish authoritative Plan absence.
- **References**: [related] `openspec/specs/plan/spec.md`; [related] `openspec/specs/github-plan-representation-observation/spec.md`

#### Scenario: Plan facts are complete [happy]

- **GIVEN** the current Milestone representation contains a valid and complete Plan
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** the GitHub Plan Snapshot contains that current Plan

#### Scenario: Membership cannot be completed [boundary]

- **GIVEN** the current root and some coherent member progress are established but complete membership cannot be established
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains Incomplete Representation Progress without a current Plan

### Requirement: provider-independent-progress

A GitHub Plan Snapshot MUST express Representation Progress without GitHub-specific identity or lifecycle meaning and MUST NOT treat representation state as Plan completion.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Complete membership | A coherent current root exists | Every current distinct member and its state are established | Preserve all members with Complete membership | None |
  | Incomplete membership | A coherent current root and some coherent members exist | Complete current membership cannot be established | Preserve coherent members with Incomplete membership | None |

- **Invariants**: Overall and member states are Open, Closed, or Unknown; distinct observed members remain distinct; Progress contains no Provider identity or metadata.
- **References**: [[plan-control/plan-completion-meaning]]

#### Scenario: Current membership is complete [happy]

- **GIVEN** all current Milestone members and their states are known
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** it contains provider-independent member names and states with Complete membership

#### Scenario: Current membership is incomplete [boundary]

- **GIVEN** complete Milestone membership cannot be established after coherent progress is known
- **WHEN** the caller obtains a successful GitHub Plan Snapshot
- **THEN** its Representation Progress has Incomplete membership

### Requirement: observation-failure

An unavailable or unsuccessful GitHub observation MUST NOT become a successful current GitHub Plan Snapshot or authoritative absence.

- **Behavioral Rules**: A coherent current root with partial member progress produces a successful GitHub Plan Snapshot with Incomplete progress and no current Plan.
- **Failure Handling**: When no coherent current root and Representation Progress can be established, observation returns no successful GitHub Plan Snapshot and makes no absence claim.

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

- **Concurrency and Idempotency**: Concurrent observations exchange no target or result facts.
- **Failure Handling**: Cancellation before success yields no successful GitHub Plan Snapshot for that invocation.

#### Scenario: Observation is cancelled [error]

- **GIVEN** a GitHub Plan Snapshot observation has not completed
- **WHEN** its caller cancels the observation
- **THEN** no successful GitHub Plan Snapshot is returned for that invocation

#### Scenario: Targets are observed concurrently [concurrency]

- **GIVEN** distinct valid GitHub Milestone Targets
- **WHEN** callers observe them concurrently
- **THEN** each result contains facts only for its own target
