## Purpose

Defines how a read-only GitHub Planning Provider reconstructs a provider-independent Plan observation from one bound GitHub Milestone or parent Issue without owning or mutating the external facts.

## Conceptual Model

### GitHub Plan Target

A GitHub Plan Target binds one GitHub Repository Target, one GitHub Plan Representation, and one positive GitHub Resource Number for its lifetime. Resource Number is distinct from every other GitHub identifier. The binding identifies the external subject but never enters the provider-independent Observation.

### Observed GitHub Facts

An Observed GitHub Fact is meaning established from native GitHub fields, relationships, or the exact raw UTF-8 GitHub Plan Narrative grammar. Milestone native facts provide Plan name, Target Date, and Task membership; its native narrative provides Goal and Acceptance Conditions. Issue native facts provide Plan name and Task membership; its native narrative provides Goal, Acceptance Conditions, and Target Date. Legacy marker or payload text is generic nonfact preamble. It is neither authoritative nor a fallback, and conflicting payload content is ignored. Marker-only content leaves narrative-backed facts unavailable.

Required Goal and Acceptance Conditions H2 inventory, order, and exact framing are global structural constraints. Their failure makes Goal Unavailable, Acceptance Conditions Incomplete and empty, and Issue Target Date Unavailable while independently established name, Tasks, and Milestone Target Date remain usable. With established global structure, exact values are classified under the `plan` capability. A known value that violates Plan meaning becomes a Plan Validation Violation. An Issue Target Date is Absent only when its section is authoritatively omitted, Present when exact framed text is valid, and Unavailable when its local boundary or final framing cannot be established.

### Acceptance Condition Sequence Establishment

Acceptance Condition Sequence Establishment preserves the longest fully framed contiguous sequence of exact `### 1` through `### n` members. Ordinals start at 1, have no leading zero, and remain contiguous in source order. The first wrong ordinal or member-framing defect ends establishment and makes membership Incomplete without discarding the preceding prefix or resuming later. Membership is Complete only when the declared EOF or Issue Target Date boundary establishes the end of the sequence; defined zero-member forms can therefore be Complete and empty. Each established member retains its exact text and is classified under `plan` rules.

### Task Collection Establishment

Task Collection Establishment contains coherent external Task resources and Complete or Incomplete membership. Membership is Complete only when the entire external collection is established. Repeated observation of one external resource with the same title contributes one Task and makes membership Incomplete; conflicting titles for one resource contribute neither title and make membership Incomplete. Distinct resources remain distinct until Plan collection rules are applied, even when their titles are equal.

### GitHub Observation Outcome

One observation request produces either a valid provider-independent Observation or the supplied caller cancellation or deadline outcome before success. Other inability to establish GitHub facts is represented inside a valid Observation as localized Unavailable Information. This capability never treats a GitHub outcome as authoritative root absence and never exposes Provider failure detail, credentials, request metadata, redirect targets, or target identity.

## Requirements

### Requirement: explicit-target-binding

A GitHub Plan observation caller MUST bind one valid GitHub Plan Target before any external fact is accessed.

- **Input and Acceptance**: The target satisfies the GitHub Plan Target definition and remains bound for the observation capability lifetime.
- **Behavioral Rules**: Every Observation concerns only that target, while Provider-native identity remains outside the provider-independent Observation.
- **Failure Handling**: Invalid local binding input is rejected before GitHub access. Local acceptance makes no claim about remote existence, access, permissions, or lifecycle.
- **References**: [related] `github-plan-creation-dry-run` Conceptual Model for GitHub Repository Target and GitHub Plan Representation.

#### Scenario: Valid target is bound [happy]

- **GIVEN** a caller selects a valid GitHub Repository Target, representation, and positive Resource Number
- **WHEN** it establishes GitHub Plan observation
- **THEN** every resulting Observation can concern only that selected target

#### Scenario: Local binding is invalid [error]

- **GIVEN** at least one required GitHub Plan Target value is invalid
- **WHEN** a caller establishes GitHub Plan observation
- **THEN** the request is rejected before GitHub is accessed

### Requirement: versioned-payload-meaning

GitHub Plan observation MUST interpret `arcloom-plan:v1` meaning only from a valid machine-readable block at the beginning of native content and MUST preserve independently established facts when other payload meaning is unusable.

- **Input and Acceptance**: A valid block exposes the exact representation-specific values defined by the Versioned Plan Narrative. Unknown version-one members do not affect known members. Human-readable narrative contributes no fact.
- **Behavioral Rules**: An absent, unsupported, or ambiguous block makes all payload-backed locations Unavailable. One unusable required member makes only its Plan Location Unavailable. A decoded value that violates Plan meaning becomes a known Plan Validation Violation. An empty Acceptance Condition collection is known Complete and empty.
- **References**: [related] `github-plan-creation-dry-run` Conceptual Model for Versioned Plan Narrative; [related] `plan` Conceptual Model for Plan validity; [related] `plan-representation-reconciliation` Conceptual Model for Observation and Unavailable Information.

#### Scenario: Version-one payload is valid [happy]

- **GIVEN** the bound resource begins with a valid version-one block for its representation
- **WHEN** GitHub Plan observation interprets the content
- **THEN** the exact payload-backed Plan values are available

#### Scenario: Payload block is unusable [error]

- **GIVEN** the machine-readable block is absent, unsupported, or ambiguous
- **WHEN** GitHub Plan observation interprets the content
- **THEN** all payload-backed Plan Locations are Unavailable while independently observed native facts remain usable

#### Scenario: Decoded value violates Plan meaning [error]

- **GIVEN** a decoded payload value violates a Plan invariant
- **WHEN** GitHub Plan observation classifies the fact
- **THEN** the affected Plan Location contains the known Plan Validation Violation rather than Unavailable Information

### Requirement: milestone-representation-meaning

GitHub Plan observation MUST project a Milestone representation into the Plan meaning assigned by the Conceptual Model.

- **Behavioral Rules**: Milestone title supplies Plan name; its Versioned Plan Narrative supplies Goal and Acceptance Conditions; its native target date supplies present or absent Target Date; and the Complete assigned-Issue collection supplies Tasks. Open and closed Issues are included and Pull Requests are excluded.
- **Failure Handling**: A present native target date that cannot represent a valid Plan date becomes a known target-date violation. Absence remains known absence.
- **References**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model.

#### Scenario: Complete Milestone representation is observed [happy]

- **GIVEN** all facts required by the selected Milestone representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning

#### Scenario: Closed Issue and Pull Request are assigned [boundary]

- **GIVEN** one closed Issue and one Pull Request are assigned to the bound Milestone
- **WHEN** GitHub Plan observation establishes Task membership
- **THEN** the closed Issue is an observed Task and the Pull Request is not

#### Scenario: Milestone target date is absent or invalid [boundary]

- **GIVEN** the Milestone target date is absent or cannot represent a valid Plan date
- **WHEN** GitHub Plan observation classifies Target Date meaning
- **THEN** the Observation distinguishes known absence from a known Plan Validation Violation

### Requirement: issue-representation-meaning

GitHub Plan observation MUST project a parent Issue representation into the Plan meaning assigned by the Conceptual Model.

- **Behavioral Rules**: Parent Issue title supplies Plan name; its Versioned Plan Narrative supplies Goal, Acceptance Conditions, and optional Target Date; and the Complete Sub-issue collection supplies Tasks. A completely established empty Sub-issue collection is known Complete and empty.
- **Failure Handling**: A selected parent resource that is a Pull Request makes the Plan root Unavailable and contributes no descendant state.
- **References**: [related] `plan` Conceptual Model; [related] `plan-representation-reconciliation` Conceptual Model.

#### Scenario: Complete Issue representation is observed [happy]

- **GIVEN** all facts required by the selected parent Issue representation are available
- **WHEN** a caller requests observation
- **THEN** the Observation contains the exact represented Plan meaning

#### Scenario: Parent is a Pull Request [error]

- **GIVEN** the selected parent resource is a Pull Request
- **WHEN** a caller requests observation
- **THEN** the Plan root is Unavailable and no descendant state is asserted

### Requirement: conservative-failure-observation

A GitHub Plan observation caller MUST receive unavailable GitHub facts as localized Unavailable Information without Provider detail, except for its own cancellation or deadline outcome before success.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Root unavailable | No coherent root fact is established | GitHub cannot establish the root | Plan root is Unavailable; no descendant state is asserted | No authoritative absence is claimed. |
  | Localized unavailable | Coherent facts exist outside one affected location | One fact cannot be established | Only the narrowest affected Plan Location is Unavailable | Other coherent facts remain usable. |
  | Partial collection | Some coherent members are established | Complete membership cannot be established | Members remain observable and membership is Incomplete | Omitted members do not become authoritative absence. |
  | Non-authoritative absence | GitHub does not unambiguously establish absence | A resource cannot be selected conclusively | The affected location is Unavailable | No authoritative Plan absence is claimed. |

- **Invariants**: Provider failure detail never crosses the observation boundary, and an inability to establish a fact never becomes authoritative absence.
- **Side Effects**: Target binding remains unchanged. Provider errors, response content, credentials, rate-limit information, request identifiers, redirect targets, and target identity do not enter the Observation.
- **Failure Handling**: Caller cancellation or deadline expiration before success returns the supplied caller-lifecycle outcome and no successful Observation. No other Provider failure crosses the observation boundary.
- **References**: [related] `plan-representation-reconciliation` Conceptual Model for localized Unavailable Information and covering rules.

#### Scenario: Root facts are inaccessible [error]

- **GIVEN** GitHub cannot establish the selected root representation
- **WHEN** a caller requests observation
- **THEN** the Plan root is Unavailable and no Provider failure detail crosses the boundary

#### Scenario: GitHub does not establish authoritative absence [boundary]

- **GIVEN** a GitHub outcome may represent either absent or inaccessible data
- **WHEN** GitHub Plan observation classifies the root fact
- **THEN** the Plan root is Unavailable rather than authoritatively absent

#### Scenario: Caller cancels observation [error]

- **GIVEN** no successful Observation has been established
- **WHEN** the caller cancels or its deadline expires
- **THEN** the supplied caller-lifecycle outcome is returned and no successful Observation is returned

### Requirement: collection-integrity

GitHub Plan observation MUST expose Task membership as Complete only after the whole external collection is coherently established and otherwise preserve only the facts justified by Task Collection Establishment.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Complete aggregation | Every response is coherent | Complete membership is established across one or more responses | Preserve every coherent Task and classify membership Complete | None |
  | Later evidence unavailable | Coherent members already exist | Later evidence cannot establish the rest | Preserve coherent Tasks and classify membership Incomplete | None |
  | Repeated same resource | One external resource is observed repeatedly | Titles agree | Contribute one Task and classify membership Incomplete | None |
  | Repeated resource conflicts | One external resource is observed repeatedly | Titles conflict | Contribute neither conflicting title and classify membership Incomplete | None |
  | Distinct resources share a title | Distinct external identities are observed | Titles are equal | Preserve distinct observations for Plan duplicate classification | None |

- **Invariants**: Collection order does not affect correspondence, and external-resource identity is not replaced by Task-name equality during observation.
- **References**: [related] `plan` Conceptual Model for Task identity and duplicate validity; [related] `plan-representation-reconciliation` Conceptual Model for Complete and Incomplete collection meaning.

#### Scenario: Complete collection spans multiple Provider responses [happy]

- **GIVEN** coherent portions of the external Task collection are established separately
- **WHEN** every portion has been established
- **THEN** Task membership is Complete and contains every coherent Task

#### Scenario: Later collection evidence is unavailable [error]

- **GIVEN** coherent Tasks are established before remaining collection evidence becomes Unavailable
- **WHEN** GitHub Plan observation completes
- **THEN** the coherent Tasks are preserved and membership is Incomplete

#### Scenario: External resource repeats [idempotency]

- **GIVEN** the same external Task resource is observed more than once
- **WHEN** GitHub Plan observation establishes its title facts
- **THEN** equal titles contribute one Task while conflicting titles contribute neither, and membership is Incomplete

#### Scenario: Distinct resources share a title [boundary]

- **GIVEN** distinct external Task resources have the same exact title
- **WHEN** GitHub Plan observation establishes their facts
- **THEN** they remain distinct observations so the Plan duplicate-Task rule can apply

### Requirement: supported-github-context

A GitHub Plan observation caller MUST receive read-only observation of the declared Milestone and Issue representations on GitHub.com using access supplied by the Host.

- **Preconditions**: The selected GitHub Plan Target uses a supported GitHub.com representation. GitHub Enterprise Server is outside this capability.
- **Behavioral Rules**: Observation uses only external read operations and produces the provider-independent Observation contract.
- **Failure Handling**: When the external GitHub contract cannot establish a required fact, the affected Plan Location follows [[github-plan-representation-observation/conservative-failure-observation]].

#### Scenario: Supported representation is observed [happy]

- **GIVEN** a declared GitHub.com Milestone or Issue representation is bound
- **WHEN** a caller requests observation
- **THEN** only external read operations occur and the result follows the provider-independent Observation contract

### Requirement: read-only-stateless-observation

GitHub Plan observation MUST remain read-only, stateless between requests, and isolated across concurrent callers.

- **Behavioral Rules**: Each request derives its result from current GitHub facts and does not depend on a prior Observation or Arcloom-owned durable state.
- **Side Effects**: Observation performs no external mutation, authorization decision, persistence, Provider identifier generation, or credential storage and does not modify Host-owned GitHub access state.
- **Concurrency and Idempotency**: Concurrent requests through the same configured observation capability do not mix facts, results, failures, or caller-lifecycle outcomes.

#### Scenario: GitHub facts change between observations [happy]

- **GIVEN** two observation requests establish different current GitHub facts
- **WHEN** both produce results
- **THEN** each Observation reflects only the facts established for its request

#### Scenario: Observations run concurrently [concurrency]

- **GIVEN** callers share the same configured observation capability
- **WHEN** they request observation concurrently
- **THEN** facts, results, failures, and cancellation remain isolated by request

#### Scenario: Prior runtime state is lost [compatibility]

- **GIVEN** all prior Arcloom runtime state has been discarded
- **WHEN** a caller requests a later observation
- **THEN** it can be established again from current GitHub facts
