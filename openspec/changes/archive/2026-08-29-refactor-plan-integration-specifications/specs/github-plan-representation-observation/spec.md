## MODIFIED Requirements

### Requirement: GHPO-1 Explicit target binding

A GitHub Plan observation caller MUST bind one valid GitHub Plan Target before any external fact is accessed.

- **入力と受理**: The target satisfies the GitHub Plan Target definition and remains bound for the observation capability lifetime.
- **振る舞いの規則**: Every Observation concerns only that target, while Provider-native identity remains outside the provider-independent Observation.
- **失敗の扱い**: Invalid local binding input is rejected before GitHub access. Local acceptance makes no claim about remote existence, access, permissions, or lifecycle.
- **参照**: [related] `github-plan-creation-dry-run` Conceptual Model for GitHub Repository Target and GitHub Plan Representation.

#### Scenario: Valid target is bound

- **GIVEN** a caller selects a valid GitHub Repository Target, representation, and positive Resource Number
- **WHEN** it establishes GitHub Plan observation
- **THEN** every resulting Observation can concern only that selected target

#### Scenario: Local binding is invalid

- **GIVEN** at least one required GitHub Plan Target value is invalid
- **WHEN** a caller establishes GitHub Plan observation
- **THEN** the request is rejected before GitHub is accessed

### Requirement: GHPO-2 Versioned payload meaning

GitHub Plan observation MUST interpret `arcloom-plan:v1` meaning only from a valid machine-readable block at the beginning of native content and MUST preserve independently established facts when other payload meaning is unusable.

- **入力と受理**: A valid block exposes the exact representation-specific values defined by the Versioned Plan Narrative. Unknown version-one members do not affect known members. Human-readable narrative contributes no fact.
- **振る舞いの規則**: An absent, unsupported, or ambiguous block makes all payload-backed locations Unavailable. One unusable required member makes only its Plan Location Unavailable. A decoded value that violates Plan meaning becomes a known Plan Validation Violation. An empty Acceptance Condition collection is known Complete and empty.
- **参照**: [related] `github-plan-creation-dry-run` Conceptual Model for Versioned Plan Narrative; [related] `plan` Conceptual Model for Plan validity; [related] `plan-representation-reconciliation` Conceptual Model for Observation and Unavailable Information.

#### Scenario: Version-one payload is valid

- **GIVEN** the bound resource begins with a valid version-one block for its representation
- **WHEN** GitHub Plan observation interprets the content
- **THEN** the exact payload-backed Plan values are available

#### Scenario: Human-readable narrative changes

- **GIVEN** a valid machine-readable block is unchanged while its following narrative changes
- **WHEN** GitHub Plan observation interprets the content
- **THEN** the reconstructed Observation is unchanged

#### Scenario: Payload block is unusable

- **GIVEN** the machine-readable block is absent, unsupported, or ambiguous
- **WHEN** GitHub Plan observation interprets the content
- **THEN** all payload-backed Plan Locations are Unavailable while independently observed native facts remain usable

#### Scenario: One required member is unusable

- **GIVEN** one required payload member is unusable and other members remain independently established
- **WHEN** GitHub Plan observation interprets the block
- **THEN** only the Plan Location backed by that member becomes Unavailable

#### Scenario: Decoded value violates Plan meaning

- **GIVEN** a decoded payload value violates a Plan invariant
- **WHEN** GitHub Plan observation classifies the fact
- **THEN** the affected Plan Location contains the known Plan Validation Violation rather than Unavailable Information

### Requirement: GHPO-3 Milestone representation meaning

GitHub Plan observation MUST project a Milestone representation into the Plan meaning assigned by the Conceptual Model.

- **振る舞いの規則**: Milestone title supplies Plan name; its Versioned Plan Narrative supplies Goal and Acceptance Conditions; its native target date supplies present or absent Target Date; and the Complete assigned-Issue collection supplies Tasks. Open and closed Issues are included and Pull Requests are excluded.
- **失敗の扱い**: A present native target date that cannot represent a valid Plan date becomes a known target-date violation. Absence remains known absence.
- **参照**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model.

#### Scenario: Complete Milestone representation is observed

- **GIVEN** all facts required by the selected Milestone representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning

#### Scenario: Closed Issue and Pull Request are assigned

- **GIVEN** one closed Issue and one Pull Request are assigned to the bound Milestone
- **WHEN** GitHub Plan observation establishes Task membership
- **THEN** the closed Issue is an observed Task and the Pull Request is not

#### Scenario: Milestone target date is absent or invalid

- **GIVEN** the Milestone target date is absent or cannot represent a valid Plan date
- **WHEN** GitHub Plan observation classifies Target Date meaning
- **THEN** the Observation distinguishes known absence from a known Plan Validation Violation

### Requirement: GHPO-4 Issue representation meaning

GitHub Plan observation MUST project a parent Issue representation into the Plan meaning assigned by the Conceptual Model.

- **振る舞いの規則**: Parent Issue title supplies Plan name; its Versioned Plan Narrative supplies Goal, Acceptance Conditions, and optional Target Date; and the Complete Sub-issue collection supplies Tasks. A completely established empty Sub-issue collection is known Complete and empty.
- **失敗の扱い**: A selected parent resource that is a Pull Request makes the Plan root Unavailable and contributes no descendant state.
- **参照**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model.

#### Scenario: Complete Issue representation is observed

- **GIVEN** all facts required by the selected parent Issue representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning

#### Scenario: Parent is a Pull Request

- **GIVEN** the selected parent resource is a Pull Request
- **WHEN** a caller requests observation
- **THEN** the Plan root is Unavailable and no descendant state is asserted

#### Scenario: Issue has no Sub-issues

- **GIVEN** the entire Sub-issue collection has been established and is empty
- **WHEN** GitHub Plan observation classifies Task membership
- **THEN** Task membership is known Complete and empty

### Requirement: GHPO-5 Conservative failure observation

A GitHub Plan observation caller MUST receive unavailable GitHub facts as localized Unavailable Information without Provider detail, except for its own cancellation or deadline outcome before success.

- **振る舞いの規則**: Any inability to establish a fact becomes Unavailable Information at the narrowest affected Plan Location while independently established facts remain usable. A GitHub outcome that does not unambiguously establish absence remains Unavailable and never becomes authoritative Plan absence.
- **副作用**: Target binding remains unchanged. Provider errors, response content, credentials, rate-limit information, request identifiers, redirect targets, and target identity do not enter the Observation.
- **失敗の扱い**: Caller cancellation or deadline expiration before success returns the supplied caller-lifecycle outcome and no successful Observation. No other Provider failure crosses the observation boundary.
- **参照**: [related] `plan-representation-reconciliation` Conceptual Model for localized Unavailable Information and covering rules.

#### Scenario: Root facts are inaccessible

- **GIVEN** GitHub cannot establish the selected root representation
- **WHEN** a caller requests observation
- **THEN** the Plan root is Unavailable and no Provider failure detail crosses the boundary

#### Scenario: Task membership is only partly established

- **GIVEN** coherent Task members are known but the entire collection cannot be established
- **WHEN** GitHub Plan observation classifies membership
- **THEN** those members are preserved and Task membership is Incomplete

#### Scenario: GitHub does not establish authoritative absence

- **GIVEN** a GitHub outcome may represent either absent or inaccessible data
- **WHEN** GitHub Plan observation classifies the root fact
- **THEN** the Plan root is Unavailable rather than authoritatively absent

#### Scenario: Caller cancels observation

- **GIVEN** no successful Observation has been established
- **WHEN** the caller cancels or its deadline expires
- **THEN** the supplied caller-lifecycle outcome is returned and no successful Observation is returned

### Requirement: GHPO-6 Collection integrity

GitHub Plan observation MUST expose Task membership as Complete only after the whole external collection is coherently established and otherwise preserve only the facts justified by Task Collection Establishment.

- **振る舞いの規則**: Unavailable or contradictory later evidence preserves prior coherent members and makes membership Incomplete. Repeated observation of the same external resource with the same title contributes one Task and makes membership Incomplete; conflicting titles contribute neither. Distinct external resources remain distinct until Plan rules classify equal Task names. Collection order does not affect correspondence.
- **参照**: [related] `plan` Conceptual Model for Task identity and duplicate validity; [related] `plan-representation-reconciliation` Conceptual Model for Complete and Incomplete collection meaning.

#### Scenario: Complete collection spans multiple Provider responses

- **GIVEN** coherent portions of the external Task collection are established separately
- **WHEN** every portion has been established
- **THEN** Task membership is Complete and contains every coherent Task

#### Scenario: Later collection evidence is unavailable

- **GIVEN** coherent Tasks are established before remaining collection evidence becomes Unavailable
- **WHEN** GitHub Plan observation completes
- **THEN** the coherent Tasks are preserved and membership is Incomplete

#### Scenario: External resource repeats

- **GIVEN** the same external Task resource is observed more than once
- **WHEN** GitHub Plan observation establishes its title facts
- **THEN** equal titles contribute one Task while conflicting titles contribute neither, and membership is Incomplete

#### Scenario: Distinct resources share a title

- **GIVEN** distinct external Task resources have the same exact title
- **WHEN** GitHub Plan observation establishes their facts
- **THEN** they remain distinct observations so the Plan duplicate-Task rule can apply

### Requirement: GHPO-7 Supported GitHub context

A GitHub Plan observation caller MUST receive read-only observation of the declared Milestone and Issue representations on GitHub.com using access supplied by the Host.

- **前提条件**: The selected GitHub Plan Target uses a supported GitHub.com representation. GitHub Enterprise Server is outside this capability.
- **振る舞いの規則**: Observation uses only external read operations and produces the provider-independent Observation contract.
- **失敗の扱い**: When the external GitHub contract cannot establish a required fact, the affected Plan Location follows GHPO-5.

#### Scenario: Supported representation is observed

- **GIVEN** a declared GitHub.com Milestone or Issue representation is bound
- **WHEN** a caller requests observation
- **THEN** only external read operations occur and the result follows the provider-independent Observation contract

#### Scenario: Required external behavior is unsupported

- **GIVEN** GitHub cannot establish a fact required by the selected representation
- **WHEN** a caller requests observation
- **THEN** the affected Plan Location is Unavailable without exposing Provider-specific failure detail

### Requirement: GHPO-8 Read-only stateless observation

GitHub Plan observation MUST remain read-only, stateless between requests, and isolated across concurrent callers.

- **振る舞いの規則**: Each request derives its result from current GitHub facts and does not depend on a prior Observation or Arcloom-owned durable state.
- **副作用**: Observation performs no external mutation, authorization decision, persistence, Provider identifier generation, or credential storage and does not modify Host-owned GitHub access state.
- **排他・冪等**: Concurrent requests through the same configured observation capability do not mix facts, results, failures, or caller-lifecycle outcomes.

#### Scenario: GitHub facts change between observations

- **GIVEN** two observation requests establish different current GitHub facts
- **WHEN** both produce results
- **THEN** each Observation reflects only the facts established for its request

#### Scenario: Observations run concurrently

- **GIVEN** callers share the same configured observation capability
- **WHEN** they request observation concurrently
- **THEN** facts, results, failures, and cancellation remain isolated by request

#### Scenario: Prior runtime state is lost

- **GIVEN** all prior Arcloom runtime state has been discarded
- **WHEN** a caller requests a later observation
- **THEN** it can be established again from current GitHub facts
