## Purpose

Defines how a read-only GitHub Planning Provider reconstructs a provider-independent Plan observation from one bound GitHub Milestone or parent Issue without owning or mutating the external facts.

## Requirements

### Requirement: GHPO-1 Explicit target binding
A GitHub Plan observation SHALL require one explicit target consisting of a locally valid Repository, either a Milestone or Issue representation, and a positive GitHub resource number distinct from other GitHub identifiers. The observation SHALL remain bound to that target, reject an invalid local binding before accessing GitHub, make no claim about remote existence, access, permissions, or lifecycle, and exclude provider-native identity from the provider-independent Observation.

#### Scenario: Valid target is bound
- **WHEN** a caller selects a valid Repository, representation, and resource number
- **THEN** the resulting observation can concern only that selected target

#### Scenario: Local binding is invalid
- **WHEN** any required local target input is invalid
- **THEN** observation is rejected before GitHub is accessed

### Requirement: GHPO-2 Versioned payload meaning
The Provider SHALL interpret the version-one machine-readable Plan block defined by GPCD-2 only when it appears at the beginning of the native content and SHALL ignore the human-readable narrative. A valid block SHALL make its exact payload-backed Plan values available. Unknown version-one members SHALL NOT affect known members. An unusable or ambiguous block SHALL make all payload-backed locations unavailable while preserving independently observed native facts. An unusable required member SHALL affect only its Plan location while preserving other independently established members. A decoded value that violates a Plan invariant SHALL be reported as that known violation rather than as unavailable. An empty acceptance-condition collection SHALL be known complete and empty.

#### Scenario: Version-one payload is valid
- **WHEN** the bound resource begins with a valid version-one payload for its representation
- **THEN** the exact payload-backed Plan values are available

#### Scenario: Human-readable narrative changes
- **WHEN** narrative content changes without changing the machine-readable block
- **THEN** the reconstructed Plan observation is unchanged

#### Scenario: Payload block is unusable
- **WHEN** the machine-readable block is absent, unsupported, or ambiguous
- **THEN** all payload-backed locations are unavailable while independently observed native facts remain usable

#### Scenario: One required member is unusable
- **WHEN** one required payload member is unusable and other members remain independently established
- **THEN** only the Plan location backed by that member becomes unavailable

#### Scenario: Decoded value violates Plan meaning
- **WHEN** a decoded value violates a Plan invariant
- **THEN** the affected location contains that known Plan violation rather than unavailable information

### Requirement: GHPO-3 Milestone representation meaning
For a Milestone representation, the observation SHALL derive the Plan name from the Milestone title, Goal and acceptance conditions from its version-one payload, the optional target date from the Milestone target date, and Tasks from the complete collection of assigned Issues. Both open and closed Issues SHALL be included, and Pull Requests SHALL be excluded. An absent Milestone target date SHALL be known absent; a present value that cannot represent a valid Plan date SHALL be reported as a known target-date violation.

#### Scenario: Complete Milestone representation is observed
- **WHEN** all facts required by the selected Milestone representation are available
- **THEN** the observation contains the exact represented Plan meaning

#### Scenario: Closed Issue and Pull Request are assigned
- **WHEN** a closed Issue and a Pull Request are both assigned to the Milestone
- **THEN** the closed Issue is an observed Task and the Pull Request is not

#### Scenario: Milestone target date is absent or invalid
- **WHEN** the target date is absent or cannot represent a valid Plan date
- **THEN** the observation distinguishes known absence from a known Plan violation

### Requirement: GHPO-4 Issue representation meaning
For an Issue representation, the observation SHALL derive the Plan name from the parent Issue title, Goal, acceptance conditions, and optional target date from its version-one payload, and Tasks from the complete Sub-issue collection. A Pull Request SHALL NOT satisfy the parent Issue representation and SHALL make the Plan root unavailable. A completely observed empty Sub-issue collection SHALL produce a known complete empty Task collection.

#### Scenario: Complete Issue representation is observed
- **WHEN** all facts required by the selected parent Issue representation are available
- **THEN** the observation contains the exact represented Plan meaning

#### Scenario: Parent is a Pull Request
- **WHEN** the selected parent resource is a Pull Request
- **THEN** the Plan root is unavailable and no descendant state is asserted

#### Scenario: Issue has no Sub-issues
- **WHEN** the complete Sub-issue collection is empty
- **THEN** Task membership is known complete and empty

### Requirement: GHPO-5 Conservative failure observation
Caller cancellation or deadline expiration SHALL be the only Provider failure returned through the observation boundary. Any other inability to establish a GitHub fact SHALL be represented as unavailable information at the narrowest affected Plan location while preserving independently established facts. A GitHub outcome that does not unambiguously establish absence SHALL NOT become authoritative Plan absence. Observation SHALL NOT rebind the target or expose Provider errors, response content, credentials, rate-limit information, request identifiers, redirect targets, or target identity.

#### Scenario: Root facts are inaccessible
- **WHEN** GitHub cannot establish the selected root representation
- **THEN** the Plan root is unavailable and no Provider failure detail crosses the boundary

#### Scenario: Task membership is only partly established
- **WHEN** coherent Task members are known but the complete collection cannot be established
- **THEN** those members are preserved and Task membership is incomplete

#### Scenario: GitHub does not establish authoritative absence
- **WHEN** GitHub reports an outcome that may also represent inaccessible or hidden data
- **THEN** the observation reports unavailable information rather than authoritative absence

#### Scenario: Caller cancels observation
- **WHEN** the caller cancels or its deadline expires before a result is established
- **THEN** cancellation is reported and no successful Observation is returned

### Requirement: GHPO-6 Collection integrity
Task membership SHALL be reported complete only after the complete external collection has been established. If the collection becomes unavailable or contradictory after coherent members are established, those members SHALL be preserved and membership SHALL be incomplete. Repeated observations of the same external resource with the same title SHALL contribute one Task and mark membership incomplete; conflicting titles for the same resource SHALL contribute neither title and mark membership incomplete. Distinct external resources with equal titles SHALL remain distinct so Plan rules can report duplicate Tasks. Collection order SHALL NOT affect correspondence.

#### Scenario: Complete collection spans multiple Provider responses
- **WHEN** all parts of the external Task collection are established
- **THEN** membership is complete and contains every coherent Task

#### Scenario: Later collection evidence is unavailable
- **WHEN** coherent Tasks are established before remaining collection evidence becomes unavailable
- **THEN** coherent Tasks are preserved and membership is incomplete

#### Scenario: External resource repeats
- **WHEN** the same external resource is observed repeatedly
- **THEN** equal titles contribute one Task while conflicting titles contribute neither, and membership is incomplete

#### Scenario: Distinct resources share a title
- **WHEN** distinct external resources have the same exact title
- **THEN** they remain distinct observations so the Plan duplicate-Task rule can apply

### Requirement: GHPO-7 Supported GitHub context
The Provider SHALL support read-only observation of the declared Milestone and Issue representations on GitHub.com using Host-supplied access configuration. GitHub Enterprise Server SHALL remain outside this capability. If the external GitHub contract cannot establish a required fact, the affected Plan location SHALL follow GHPO-5.

#### Scenario: Supported representation is observed
- **WHEN** a declared GitHub.com representation is observed
- **THEN** only read operations are used and the result follows the provider-independent Observation contract

#### Scenario: Required external behavior is unsupported
- **WHEN** GitHub cannot establish a fact required by the selected representation
- **THEN** the affected Plan location is unavailable without exposing Provider-specific failure detail

### Requirement: GHPO-8 Read-only stateless observation
Each observation SHALL derive its result from current GitHub facts and SHALL NOT depend on a prior observation or Arcloom-owned durable state. Observation SHALL perform no external mutation, authorization decision, persistence, Provider identifier generation, or credential storage and SHALL NOT modify Host-owned access configuration. Concurrent observations through the same configured boundary SHALL remain isolated.

#### Scenario: GitHub facts change between observations
- **WHEN** two observations establish different current GitHub facts
- **THEN** each result reflects only the facts established for that observation

#### Scenario: Observations run concurrently
- **WHEN** callers observe through the same configured boundary concurrently
- **THEN** facts, results, failures, and cancellation remain isolated by invocation

#### Scenario: Prior runtime state is lost
- **WHEN** all prior Arcloom runtime state is discarded
- **THEN** a later observation can be established again from current GitHub facts
