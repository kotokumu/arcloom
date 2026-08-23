## Purpose

Defines how Arcloom determines whether an externally observed representation matches an expected Plan while preserving external authority and distinguishing known differences from unavailable information.

## Requirements

### Requirement: PRR-1 Reconciliation subject and target binding
Plan representation reconciliation SHALL receive one valid expected Plan and one observation boundary whose implementation remains bound to exactly one external Plan target for its lifetime. Target selection and Provider-native identity SHALL remain outside the observation Port and SHALL NOT appear in the expected Plan, observation, evidence, or result. This capability assumes that the invoking Host supplies both the Plan derived from the applicable proposed Change and an observation boundary bound to that same Change target; this association is an external precondition and is not constructed or validated by this capability. The Provider implementation SHALL only validate the structural configuration of the binding and preserve that immutable binding. Target existence, access, and lifecycle SHALL remain observation outcomes. A missing observation boundary or an invalid expected Plan SHALL fail before external observation and SHALL return no valid result. Normal construction SHALL reject a missing observation boundary. A reconciliation call on a zero or otherwise unconstructed Controller SHALL fail with the same stable invalid-observer category.

#### Scenario: Target-bound reconciliation
- **WHEN** reconciliation receives a valid expected Plan and an observation boundary bound to one external target
- **THEN** the result applies only to the observation of that bound target

#### Scenario: Observation boundary is missing
- **WHEN** reconciliation is configured without an observation boundary
- **THEN** configuration fails with a stable invalid-observer category and no observation occurs

#### Scenario: Zero Controller is called
- **WHEN** reconciliation is called on a zero or otherwise unconstructed Controller with a non-nil live context
- **THEN** it fails with the stable invalid-observer category, performs no observation, and returns no valid result

#### Scenario: Expected Plan is invalid
- **WHEN** reconciliation is requested with the zero or otherwise invalid Plan
- **THEN** it fails with a stable invalid-expected-Plan category before requesting an observation

### Requirement: PRR-2 Provider-independent observation contract
One reconciliation SHALL consume one immutable logical observation of the bound target. An implementation MAY use multiple Provider reads, pages, or retries to form it. The observation SHALL be coherent when every asserted fact refers to the same bound target and the Provider has no detected contradiction among the facts asserted as known. A Provider that cannot establish coherence SHALL identify the potentially inconsistent facts using Plan locations, and Plan Representation Controller-owned observation construction SHALL mark unavailable the least Plan location whose scope contains them all. A scalar inconsistency SHALL affect that scalar, an uncertain collection read SHALL affect that collection, and an inconsistency spanning multiple top-level locations SHALL affect the Plan root. Facts outside the unavailable location SHALL remain usable. Concurrent external changes that the Provider cannot detect are outside this guarantee.

The observation root SHALL be exactly one of known present, authoritatively absent, or unavailable. Absent and unavailable roots SHALL carry no descendant state. A present root SHALL classify Plan name and Goal as a known valid value, a known Plan-invariant violation, or unavailable. Target date SHALL be known present with one valid date, known absent with no value, a known invalid-target-date violation, or unavailable. Each acceptance-condition and Task collection SHALL contain zero or more distinct valid members, zero or more known Plan-invariant violations, and complete or incomplete membership. A duplicate external member SHALL retain one valid representative and one duplicate-member violation. Invalid members SHALL appear only as violations and SHALL NOT appear in the valid-member set. An unavailable scalar SHALL carry neither a value nor a violation. Provider identifiers, resource types, API fields, and Provider errors SHALL NOT appear in the observation.

#### Scenario: Provider-native representation is observed
- **WHEN** a Provider reads a Milestone, parent Issue, Project, or another native representation
- **THEN** it exposes only the corresponding provider-independent Plan values, collection membership, and availability

#### Scenario: External value violates Plan meaning
- **WHEN** an external value is blank, invalid UTF-8, an invalid target date, duplicated within a collection, or otherwise violates a Plan invariant
- **THEN** the observation preserves a known violation at the affected scalar or collection and does not classify the fact as unavailable

#### Scenario: Collection contains a duplicate member
- **WHEN** a complete external collection contains the same exact valid member more than once
- **THEN** the observation retains one valid representative and one duplicate-member violation at the collection location

#### Scenario: Invalid external member is repeated
- **WHEN** the same exact invalid raw member occurs more than once in an external collection
- **THEN** each occurrence is classified only by its element-validity violation, duplicate detection does not apply, and evidence aggregation retains one item for that violation code at the collection location

#### Scenario: Paged collection changes during observation
- **WHEN** the Provider detects that collection membership changed between pages and cannot establish one complete view
- **THEN** the observation preserves coherent members already observed and marks that collection incomplete

#### Scenario: Invalid observation crosses the boundary
- **WHEN** an observation claims contradictory root states, invalid known values, duplicate known members, or another impossible availability combination
- **THEN** reconciliation fails with a stable observation-contract category and returns no valid result

### Requirement: PRR-3 Semantic correspondence
Reconciliation SHALL compare known Plan name, Goal, and target date values by exact preserved value. It SHALL pair acceptance conditions by exact statement and Tasks by exact name. Collection order SHALL NOT affect correspondence. Case, whitespace, line endings, and Unicode code-point sequences SHALL NOT be normalized. A complete observed collection SHALL have set semantics because duplicate exact members cannot cross the observation contract as known values.

#### Scenario: Equal meaning in a different order
- **WHEN** complete observed acceptance-condition and Task collections contain the same members as the expected Plan in a different order
- **THEN** collection order alone produces no difference

#### Scenario: Text differs only by normalization
- **WHEN** expected and observed text use different Unicode code-point sequences that render similarly
- **THEN** the text is not treated as equal

#### Scenario: Task name changes
- **WHEN** a complete observed Task collection lacks one expected name and contains a different name
- **THEN** the expected Task is absent and the observed Task is unexpected

### Requirement: PRR-4 Difference evidence
A known difference SHALL be exactly one of `ExpectedAbsent`, `UnexpectedPresent`, `ValueDifferent`, or `InvalidObserved`. Each difference SHALL identify one provider-independent Plan location. Locations SHALL be limited to the Plan root; Plan name; Goal; acceptance-condition collection and a member identified by exact statement; Task collection and a member identified by exact name; and target date. Locations SHALL NOT use a collection index or Provider identifier. `ExpectedAbsent` SHALL contain only expected meaning, `UnexpectedPresent` SHALL contain only observed meaning, `ValueDifferent` SHALL contain both meanings, and `InvalidObserved` SHALL contain one stable Plan-invariant violation category. Evidence SHALL contain no duplicates.

Differences SHALL be ordered by root, Plan name, Goal, acceptance-condition collection, acceptance-condition members in ascending raw UTF-8 byte order, Task collection, Task members in ascending raw UTF-8 byte order, and target date. Multiple `InvalidObserved` items at the same location SHALL be ordered by the raw UTF-8 bytes of their stable Plan-violation codes. An authoritative root absence SHALL produce only the covering root `ExpectedAbsent` difference and SHALL suppress every descendant difference. Its expected payload SHALL be the complete immutable expected Plan. An empty complete observed collection SHALL produce member-level `ExpectedAbsent` differences for expected members and SHALL NOT by itself produce `InvalidObserved`; `InvalidObserved` represents an observed violation rather than missing expected meaning. Multiple invalid observed members with the same violation code at one collection location SHALL produce one `InvalidObserved` item because evidence identifies the violated Plan rule, not Provider-native invalid values.

#### Scenario: Scalar value differs
- **WHEN** a known observed Plan name or Goal differs from the expected value
- **THEN** reconciliation reports one `ValueDifferent` item at that scalar location with both values

#### Scenario: Expected target date is absent externally
- **WHEN** the expected Plan has a target date and the observation knows the target date is absent
- **THEN** reconciliation reports `ExpectedAbsent` at the target-date location

#### Scenario: Unexpected target date is present externally
- **WHEN** the expected Plan has no target date and the observation has a known valid target date
- **THEN** reconciliation reports `UnexpectedPresent` at the target-date location

#### Scenario: Known target-date values differ
- **WHEN** expected and observed target dates are both present and unequal
- **THEN** reconciliation reports `ValueDifferent` at the target-date location with both dates

#### Scenario: Expected member is absent
- **WHEN** a complete observed collection lacks an expected acceptance condition or Task
- **THEN** reconciliation reports `ExpectedAbsent` at the member location

#### Scenario: External Plan is authoritatively absent
- **WHEN** the Planning Context authoritatively confirms that the bound external representation does not exist
- **THEN** reconciliation reports exactly one covering `ExpectedAbsent` difference at the Plan root

#### Scenario: Observed scalar violates a Plan invariant
- **WHEN** a known observed Plan name, Goal, or target date has a Plan-invariant violation
- **THEN** reconciliation reports `InvalidObserved` at that scalar location with the stable violation category

#### Scenario: Observed collection violates a Plan invariant
- **WHEN** a known acceptance-condition or Task member is invalid or duplicated
- **THEN** reconciliation reports one `InvalidObserved` item for each unique violation category at that collection location while preserving differences established from valid members

#### Scenario: Complete observed collection is empty
- **WHEN** a complete observed collection contains no valid or invalid observed member
- **THEN** reconciliation reports `ExpectedAbsent` for each expected member and reports no `InvalidObserved` solely because the collection is empty

#### Scenario: Multiple observed violations share one location
- **WHEN** one collection contains multiple invalid observed members with the same violation code and another violation with a different code
- **THEN** reconciliation reports one `InvalidObserved` item for each unique violation code at that collection location ordered by raw UTF-8 bytes of the codes

### Requirement: PRR-5 Unavailable information
Unavailable information SHALL identify only the Plan root, Plan name, Goal, acceptance-condition collection, Task collection, or target-date location. It SHALL remain separate from known differences and SHALL contain no Provider error. Root unavailability SHALL be a covering item that suppresses every descendant unavailable item and every difference. An incomplete collection SHALL produce one unavailable item at its collection location, SHALL prevent an omitted expected member from being reported as absent, and SHALL preserve differences established from observed members. Unavailable items SHALL be unique and ordered by root, Plan name, Goal, acceptance-condition collection, Task collection, then target date.

#### Scenario: Plan existence is unavailable
- **WHEN** the Provider cannot determine whether the bound external representation exists
- **THEN** reconciliation reports only the Plan root as unavailable and reports no difference

#### Scenario: Expected member is omitted from an incomplete collection
- **WHEN** an expected Task is not observed and Task collection membership is incomplete
- **THEN** reconciliation does not report that Task as absent and reports the Task collection as unavailable

#### Scenario: Known unexpected member is present in an incomplete collection
- **WHEN** an observed Task is absent from the expected Plan and Task collection membership is incomplete
- **THEN** reconciliation preserves the `UnexpectedPresent` difference and reports the Task collection as unavailable

#### Scenario: Target-date existence is unavailable
- **WHEN** the Provider cannot determine whether a target date exists
- **THEN** reconciliation reports the target-date location as unavailable and no target-date difference

### Requirement: PRR-6 Evidence-derived immutable result
The complete evidence set for a result SHALL consist only of known differences and unavailable information. No known difference and no unavailable information SHALL derive `Satisfied`. At least one known difference and no unavailable information SHALL derive `NotSatisfied`. Any unavailable information SHALL derive `Undecidable` while preserving every non-covered known difference. No other fact SHALL influence the determination. A caller SHALL NOT select the determination independently or mutate a returned result so that its determination contradicts its evidence.

#### Scenario: Equal complete representation
- **WHEN** all required information is available and no difference exists
- **THEN** the result is `Satisfied` with empty evidence

#### Scenario: Different complete representation
- **WHEN** all required information is available and at least one difference exists
- **THEN** the result is `NotSatisfied` with the known differences

#### Scenario: Difference and unavailable information coexist
- **WHEN** at least one non-covered known difference exists and other required information is unavailable
- **THEN** the result is `Undecidable` and preserves both kinds of evidence

#### Scenario: Caller mutates returned evidence collection
- **WHEN** a caller changes a collection returned by a result accessor
- **THEN** subsequent reads expose the original determination-consistent evidence

### Requirement: PRR-7 Observation outcomes and failures
A Provider SHALL encode access denial, rate limiting, temporary service failure, detected concurrent external change, and other inability to establish facts as unavailable information using the localization rule in PRR-2. An authoritative not-found fact SHALL be known root absence. Structurally invalid target binding or Provider configuration SHALL prevent construction of the bound observation implementation and SHALL NOT become reconciliation evidence; target existence, access, and lifecycle facts SHALL NOT be checked as construction validity. A non-context error or invalid observation returned across the observation boundary SHALL terminate reconciliation with a stable observation-contract category and no valid result. An Observer-returned context error SHALL be a caller context outcome only when it equals the supplied context's current error; any unrelated context error SHALL be an observation-contract failure. A nil caller context SHALL fail with a stable invalid-context category before observation. For a non-nil context, caller cancellation or deadline expiration SHALL take precedence over invalid observer state, invalid expected Plan, and observer-contract failure. Invalid observer state SHALL take precedence over invalid expected Plan. Cancellation after a result is returned SHALL not alter that result.

#### Scenario: Provider cannot establish current facts
- **WHEN** access, service health, rate limits, or concurrent updates prevent a coherent observation
- **THEN** the affected location is unavailable and no Provider-specific error crosses the boundary

#### Scenario: Observer violates its contract
- **WHEN** the observation implementation returns a non-context error or an invalid observation
- **THEN** reconciliation returns no valid result and exposes a stable observation-contract failure without wrapping Provider details

#### Scenario: Caller cancels before return
- **WHEN** caller cancellation or deadline expiration is observed before reconciliation returns
- **THEN** reconciliation returns the applicable context error and no valid result

#### Scenario: Caller context is nil
- **WHEN** reconciliation receives a nil caller context
- **THEN** it fails with a stable invalid-context category, performs no observation, and returns no valid result

#### Scenario: Nil context and invalid expected Plan coexist
- **WHEN** reconciliation receives both a nil caller context and an invalid expected Plan
- **THEN** it fails with the stable invalid-context category, performs no observation, and returns no valid result

#### Scenario: Cancellation coincides with observer failure
- **WHEN** the caller context is cancelled before return and the observer also returns a contract failure
- **THEN** reconciliation returns the context error and no valid result

#### Scenario: Cancellation coincides with an invalid expected Plan
- **WHEN** the caller context is cancelled before return and the expected Plan is invalid
- **THEN** reconciliation returns the context error, performs no observation, and returns no valid result

#### Scenario: Observer returns an unrelated context error
- **WHEN** the Observer returns cancellation or deadline expiration while the supplied caller context remains live
- **THEN** reconciliation fails with the stable observation-contract category and returns no valid result

### Requirement: PRR-8 Read-only stateless behavior
Plan representation reconciliation SHALL NOT mutate the expected Plan, request or authorize an external Change, persist an observation or result, or require prior runtime state. Each reconciliation SHALL request a new logical observation and reconstruct its result. A Controller and its observation boundary SHALL support concurrent reconciliations without sharing per-call Observation or Result state or mixing facts between calls. Loss of all Arcloom runtime state SHALL NOT change correctness.

#### Scenario: External facts change
- **WHEN** two reconciliations of the same expected Plan receive different current observations
- **THEN** each result reflects only its own observation

#### Scenario: Runtime state is discarded
- **WHEN** reconciliation runs after all prior Arcloom runtime state is lost
- **THEN** it can produce a result from the expected Plan and a new observation without restoring Arcloom-owned state

#### Scenario: Difference is found
- **WHEN** reconciliation reports a difference
- **THEN** no external mutation or authorization is requested

#### Scenario: Reconciliations run concurrently
- **WHEN** callers reconcile independently using the same Controller at the same time
- **THEN** each call receives one Result derived only from its own logical Observation without a data race or cross-call state
