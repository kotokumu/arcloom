## MODIFIED Requirements

### Requirement: PRR-1 Reconciliation subject and target binding

A reconciliation caller MUST receive a result concerning one valid expected Plan and one externally associated Plan target.

- **前提条件**: The caller supplies a valid expected Plan derived from the applicable proposed Change and an observation source bound to that same Change target. The caller owns this association, and the observation source remains bound to exactly that target for its lifetime.
- **入力と受理**: Target selection and Provider-native identity remain outside Plan, Observation, Evidence, and result meaning. Target existence, access, and lifecycle remain observation facts rather than input-validity checks.
- **振る舞いの規則**: Every result applies only to the supplied Plan and the Observation of the bound target.
- **失敗の扱い**: A missing or unusable observation source produces a stable invalid-observer failure. An invalid expected Plan produces a stable invalid-expected-Plan failure. Both return no valid result before observation begins.

#### Scenario: Target-bound reconciliation

- **GIVEN** a valid expected Plan and an observation source bound to one external target
- **WHEN** a caller requests reconciliation
- **THEN** the result applies only to the Observation of that bound target

#### Scenario: Observation boundary is missing

- **GIVEN** no observation source has been supplied
- **WHEN** a caller configures reconciliation
- **THEN** configuration fails with the stable invalid-observer category and no observation occurs

#### Scenario: Zero Controller is called

- **GIVEN** the caller has no usable configured reconciliation boundary and its lifecycle remains active
- **WHEN** the caller requests reconciliation
- **THEN** the request fails with the stable invalid-observer category, performs no observation, and returns no valid result

#### Scenario: Expected Plan is invalid

- **GIVEN** the supplied expected Plan is invalid
- **WHEN** a caller requests reconciliation
- **THEN** the request fails with the stable invalid-expected-Plan category before observation begins

### Requirement: PRR-2 Provider-independent observation contract

An observation source MUST provide one coherent logical Observation that satisfies the Observation classifications in the Conceptual Model.

- **入力と受理**: Multiple external reads, pages, or retries may contribute facts, but every known fact belongs to the same bound target and every detected contradiction is resolved as Unavailable Information.
- **振る舞いの規則**: A scalar inconsistency affects that scalar, an uncertain collection affects that collection, and an inconsistency spanning top-level locations affects the Plan root. Facts outside the affected location remain usable. Provider identifiers, resource types, fields, and failures do not enter the Observation.
- **失敗の扱い**: An impossible combination of Observation states produces a stable observation-contract failure and no valid reconciliation result. Concurrent external changes that the observation source cannot detect are outside this guarantee.
- **参照**: [related] `plan` Conceptual Model for Plan validity and Plan Validation Violations.

#### Scenario: Provider-native representation is observed

- **GIVEN** a Provider exposes a native representation such as a Milestone, parent Issue, or Project
- **WHEN** the observation source projects its facts
- **THEN** the Observation contains only provider-independent Plan meaning and availability

#### Scenario: External value violates Plan meaning

- **GIVEN** an external value violates a Plan invariant
- **WHEN** the observation source classifies the fact
- **THEN** the affected location contains the known Plan Validation Violation rather than Unavailable Information

#### Scenario: Collection contains a duplicate member

- **GIVEN** a complete external collection repeats the same exact valid member
- **WHEN** the observation source forms the collection fact
- **THEN** the Observation retains one valid representative and one duplicate-member violation at the collection location

#### Scenario: Invalid external member is repeated

- **GIVEN** the same exact invalid raw member occurs more than once in an external collection
- **WHEN** the observation source classifies the occurrences
- **THEN** each occurrence contributes only its element-validity violation, duplicate detection does not apply, and Evidence retains one item for that violation category at the collection location

#### Scenario: Paged collection changes during observation

- **GIVEN** coherent collection members have been observed
- **WHEN** the observation source detects membership change between pages and cannot establish one complete view
- **THEN** the Observation preserves the coherent members and marks the collection Incomplete

#### Scenario: Invalid observation crosses the boundary

- **GIVEN** an Observation claims contradictory root states, invalid known values, duplicate valid members, or another impossible classification
- **WHEN** reconciliation receives it
- **THEN** reconciliation fails with the stable observation-contract category and returns no valid result

### Requirement: PRR-3 Semantic correspondence

Reconciliation MUST compare expected Plan meaning with known observed meaning using the identity rules owned by Plan and Observation.

- **振る舞いの規則**: Plan name, Goal, and Target Date use exact preserved value. Acceptance Conditions use exact statement and Tasks use exact name. Complete collections use order-independent membership. Case, whitespace, line endings, and Unicode code-point sequences are not normalized.
- **失敗の扱い**: Unknown or Unavailable meaning does not become a known mismatch under this Requirement.
- **参照**: [related] `plan` Conceptual Model for exact Plan Text and Plan Collection identity.

#### Scenario: Equal meaning in a different order

- **GIVEN** complete observed Acceptance Condition and Task collections contain the expected members in a different order
- **WHEN** reconciliation compares collection meaning
- **THEN** collection order alone produces no Difference

#### Scenario: Text differs only by normalization

- **GIVEN** expected and observed text use different Unicode code-point sequences that render similarly
- **WHEN** reconciliation compares them
- **THEN** the values are not equal

#### Scenario: Task name changes

- **GIVEN** a complete observed Task collection lacks one expected name and contains a different name
- **WHEN** reconciliation compares membership
- **THEN** the expected Task is absent and the observed Task is unexpected

### Requirement: PRR-4 Difference evidence

Reconciliation MUST represent every known mismatch as the Difference classification defined in the Conceptual Model.

- **振る舞いの規則**: Each Difference identifies one Plan Location and carries only the payload permitted by its kind. Differences are unique and canonically ordered by root, Plan name, Goal, Acceptance Condition collection and members by ascending raw UTF-8 bytes, Task collection and members by ascending raw UTF-8 bytes, then Target Date. Multiple violations at one location are ordered by violation-category bytes.
- **副作用**: An authoritative root absence produces one covering root `ExpectedAbsent` carrying the complete immutable expected Plan and suppresses descendant Evidence.
- **失敗の扱い**: Missing expected members in an Incomplete collection are not classified as absent. An empty Complete collection produces member-level absence without producing `InvalidObserved` solely for emptiness.

#### Scenario: Scalar value differs

- **GIVEN** expected and known observed Plan name or Goal values differ
- **WHEN** reconciliation compares them
- **THEN** Evidence contains one `ValueDifferent` at that Plan Location with both values

#### Scenario: Expected target date is absent externally

- **GIVEN** the expected Plan has a Target Date and the Observation knows it is absent
- **WHEN** reconciliation compares Target Date meaning
- **THEN** Evidence contains `ExpectedAbsent` at the Target Date location

#### Scenario: Unexpected target date is present externally

- **GIVEN** the expected Plan has no Target Date and the Observation has a known valid Target Date
- **WHEN** reconciliation compares Target Date meaning
- **THEN** Evidence contains `UnexpectedPresent` at the Target Date location

#### Scenario: Known target-date values differ

- **GIVEN** expected and observed Target Dates are both present and unequal
- **WHEN** reconciliation compares them
- **THEN** Evidence contains `ValueDifferent` at the Target Date location with both dates

#### Scenario: Expected member is absent

- **GIVEN** a Complete observed collection lacks an expected Acceptance Condition or Task
- **WHEN** reconciliation compares membership
- **THEN** Evidence contains `ExpectedAbsent` at the member location

#### Scenario: External Plan is authoritatively absent

- **GIVEN** the Observation authoritatively establishes that the bound external representation is absent
- **WHEN** reconciliation compares the expected Plan
- **THEN** Evidence contains exactly one covering `ExpectedAbsent` at the Plan root

#### Scenario: Observed scalar violates a Plan invariant

- **GIVEN** a known observed Plan name, Goal, or Target Date has a Plan Validation Violation
- **WHEN** reconciliation evaluates the Observation
- **THEN** Evidence contains `InvalidObserved` at that scalar location with the stable violation category

#### Scenario: Observed collection violates a Plan invariant

- **GIVEN** a known Acceptance Condition or Task member is invalid or duplicated
- **WHEN** reconciliation evaluates the Observation
- **THEN** Evidence contains one `InvalidObserved` for each unique violation category at that collection location and preserves Differences established from valid members

#### Scenario: Complete observed collection is empty

- **GIVEN** a Complete observed collection contains no valid member or violation
- **WHEN** reconciliation compares it with expected members
- **THEN** Evidence contains `ExpectedAbsent` for each expected member and no `InvalidObserved` solely because the collection is empty

#### Scenario: Multiple observed violations share one location

- **GIVEN** one collection has repeated invalid members with one violation category and another member with a different category
- **WHEN** reconciliation aggregates Evidence
- **THEN** Evidence contains one `InvalidObserved` per unique category at that location ordered by raw UTF-8 bytes of the categories

### Requirement: PRR-5 Unavailable information

Reconciliation MUST preserve facts that cannot be established as Unavailable Information rather than a known Difference.

- **振る舞いの規則**: Unavailable Information identifies only Plan root, Plan name, Goal, Acceptance Condition collection, Task collection, or Target Date. Items are unique and ordered in that sequence.
- **副作用**: Root Unavailable Information covers every descendant unavailable item and every Difference. An Incomplete collection produces one unavailable item at its collection location, prevents omitted expected members from becoming absent, and preserves Differences established from observed members.
- **失敗の扱い**: Provider failure detail never becomes Unavailable Information payload.

#### Scenario: Plan existence is unavailable

- **GIVEN** the Observation cannot establish whether the bound external representation exists
- **WHEN** reconciliation evaluates it
- **THEN** Evidence contains only root Unavailable Information and no Difference

#### Scenario: Expected member is omitted from an incomplete collection

- **GIVEN** an expected Task is not observed and Task membership is Incomplete
- **WHEN** reconciliation compares membership
- **THEN** Evidence does not report that Task as absent and contains Unavailable Information for the Task collection

#### Scenario: Known unexpected member is present in an incomplete collection

- **GIVEN** an observed Task is absent from the expected Plan and Task membership is Incomplete
- **WHEN** reconciliation compares membership
- **THEN** Evidence preserves `UnexpectedPresent` for the Task and contains Unavailable Information for the Task collection

#### Scenario: Target-date existence is unavailable

- **GIVEN** the Observation cannot establish whether a Target Date exists
- **WHEN** reconciliation compares Target Date meaning
- **THEN** Evidence contains Target Date Unavailable Information and no Target Date Difference

### Requirement: PRR-6 Evidence-derived immutable result

A result consumer MUST receive an immutable Reconciliation Determination derived only from its Evidence.

- **振る舞いの規則**: The determination follows the derivation table in the Conceptual Model. The caller cannot select it independently or alter returned Evidence so that later reads contradict it.
- **排他・冪等**: Reading or attempting to alter one returned collection does not change the stored result or another read.

#### Scenario: Equal complete representation

- **GIVEN** every required fact is available and no Difference exists
- **WHEN** reconciliation produces a result
- **THEN** the determination is `Satisfied` with empty Evidence

#### Scenario: Different complete representation

- **GIVEN** every required fact is available and at least one Difference exists
- **WHEN** reconciliation produces a result
- **THEN** the determination is `NotSatisfied` with the known Differences

#### Scenario: Difference and unavailable information coexist

- **GIVEN** at least one non-covered Difference and some Unavailable Information exist
- **WHEN** reconciliation produces a result
- **THEN** the determination is `Undecidable` and Evidence preserves both kinds

#### Scenario: Caller mutates returned evidence collection

- **GIVEN** a consumer has read an Evidence collection from a result
- **WHEN** the consumer changes its local collection value
- **THEN** subsequent result reads expose the original determination-consistent Evidence

### Requirement: PRR-7 Observation outcomes and failures

A reconciliation caller MUST be able to distinguish unavailable external facts, caller lifecycle termination, invalid inputs, and observation-contract failures.

- **振る舞いの規則**: Access denial, rate limiting, temporary service failure, detected concurrent change, and other inability to establish facts become Unavailable Information at the least affected Plan Location. Authoritative not-found becomes root absence. Structurally invalid binding configuration fails before observation and does not become Evidence.
- **失敗の扱い**: A failed or invalid Observation produces a stable observation-contract failure and no valid result. Caller cancellation or deadline expiration observed before return takes precedence over unusable observation source, invalid expected Plan, and observation-contract failure. An unusable observation source takes precedence over an invalid expected Plan. Cancellation after return does not alter the result. A cancellation report not caused by the caller is an observation-contract failure.

#### Scenario: Provider cannot establish current facts

- **GIVEN** access, service health, rate limits, or concurrent updates prevent coherent facts
- **WHEN** the observation source reports the outcome
- **THEN** the affected Plan Location is Unavailable and no Provider-specific detail crosses the boundary

#### Scenario: Observer violates its contract

- **GIVEN** the observation source returns a non-caller failure or an invalid Observation
- **WHEN** reconciliation evaluates the outcome
- **THEN** it returns no valid result and exposes the stable observation-contract failure without Provider detail

#### Scenario: Caller cancels before return

- **GIVEN** reconciliation has not returned a valid result
- **WHEN** the caller cancels or its deadline expires
- **THEN** reconciliation returns the caller lifecycle outcome and no valid result

#### Scenario: Caller context is nil

- **GIVEN** no valid caller lifecycle input accompanies the request
- **WHEN** a caller requests reconciliation
- **THEN** the request fails with the stable invalid-context category, performs no observation, and returns no valid result

#### Scenario: Nil context and invalid expected Plan coexist

- **GIVEN** caller lifecycle input is absent and the expected Plan is invalid
- **WHEN** a caller requests reconciliation
- **THEN** the stable invalid-context category takes precedence, no observation occurs, and no valid result is returned

#### Scenario: Cancellation coincides with observer failure

- **GIVEN** the observation source fails before a valid result is established
- **WHEN** the caller also cancels before return
- **THEN** reconciliation returns the caller lifecycle outcome and no valid result

#### Scenario: Cancellation coincides with an invalid expected Plan

- **GIVEN** the expected Plan is invalid
- **WHEN** the caller cancels before reconciliation returns
- **THEN** reconciliation returns the caller lifecycle outcome, performs no observation, and returns no valid result

#### Scenario: Observer returns an unrelated context error

- **GIVEN** the caller lifecycle remains active
- **WHEN** the observation source reports cancellation or deadline expiration
- **THEN** reconciliation returns the stable observation-contract failure and no valid result

### Requirement: PRR-8 Read-only stateless behavior

Reconciliation MUST remain read-only, stateless between calls, and isolated across concurrent callers.

- **副作用**: Reconciliation does not mutate the expected Plan, request or authorize external change, persist Observation or result, or depend on prior runtime state. Each call obtains a new logical Observation.
- **排他・冪等**: Concurrent calls share no per-call Observation or result state and do not mix facts.

#### Scenario: External facts change

- **GIVEN** two calls concern the same expected Plan
- **WHEN** they receive different current Observations
- **THEN** each result reflects only its own Observation

#### Scenario: Runtime state is discarded

- **GIVEN** all prior Arcloom runtime state has been lost
- **WHEN** a caller supplies the expected Plan and obtains a new Observation
- **THEN** reconciliation can produce a result without restoring prior runtime state

#### Scenario: Difference is found

- **GIVEN** reconciliation identifies a Difference
- **WHEN** it returns the result
- **THEN** no external mutation or authorization has been requested

#### Scenario: Reconciliations run concurrently

- **GIVEN** callers independently request reconciliation using the same configured observation capability
- **WHEN** the calls run concurrently
- **THEN** each caller receives one result derived only from its own logical Observation without cross-call state
