## MODIFIED Requirements

### Requirement: PLC-1 Plan control subject

A Plan Control caller MUST receive an assessment concerning one valid Plan Snapshot supplied for that decision.

- **前提条件**: The Plan Snapshot has been established from authoritative external facts. Plan creation and removal remain outside this capability.
- **入力と受理**: One valid Plan Snapshot and the Delivery Observations for the decision are supplied.
- **振る舞いの規則**: The result concerns adjustment or completion of exactly that Plan Snapshot.
- **失敗の扱い**: If no valid Plan Snapshot exists, Plan Control returns a stable invalid-input failure, requests no AI assessment, and returns no valid result.
- **参照**: [related] `plan` Conceptual Model for Plan validity and identity.

#### Scenario: Existing Plan is controlled

- **GIVEN** a valid Plan Snapshot and Delivery Observations
- **WHEN** a caller requests Plan Control assessment
- **THEN** the result concerns adjustment or completion of that Plan Snapshot

#### Scenario: Plan does not exist

- **GIVEN** no current valid Plan Snapshot can be established
- **WHEN** a caller requests Plan Control assessment
- **THEN** Plan Control returns a stable invalid-input failure without requesting an AI assessment

### Requirement: PLC-2 Plan completion meaning

A Plan Control caller MUST receive Complete only when the external AI determines from available Delivery Observations that the Plan Goal and Acceptance Conditions have been achieved.

- **振る舞いの規則**: Plan Control preserves the AI-owned semantic judgment without independently re-evaluating Delivery Observations. Task completion is evidence available to the AI but is neither necessary nor sufficient by itself for Complete.
- **副作用**: Complete does not constitute Delivery Acceptance.

#### Scenario: Goal and acceptance conditions are achieved

- **GIVEN** available Delivery Observations establish the Goal and Acceptance Conditions to the external AI
- **WHEN** the AI returns Complete
- **THEN** the Plan Control Assessment records Complete

#### Scenario: All Tasks are complete without outcome evidence

- **GIVEN** every observed Task is complete but evidence required to establish the Goal or Acceptance Conditions is unavailable
- **WHEN** the AI returns Insufficient Information
- **THEN** the Plan Control Assessment records Insufficient Information rather than Complete

#### Scenario: Obsolete Task remains incomplete

- **GIVEN** one obsolete Task remains incomplete while other observations establish the Goal and Acceptance Conditions to the external AI
- **WHEN** the AI returns Complete
- **THEN** the Plan Control Assessment records Complete

### Requirement: PLC-3 AI control assessment

The external AI MUST select exactly one Plan Control Assessment defined by this capability for the supplied Plan Snapshot and Delivery Observations.

- **入力と受理**: Complete, Retain, and Insufficient Information contain no Proposed Plan. Revise contains exactly one Proposed Plan.
- **振る舞いの規則**: Complete records achieved Plan outcomes; Retain records that no adjustment is required; Revise records one candidate next Plan; Insufficient Information records that required evidence is unavailable.
- **副作用**: An assessment does not change external Plan state. This Plan-specific contract neither constrains other reconciliation results nor exposes a Change concept.

#### Scenario: Plan remains suitable

- **GIVEN** the external AI determines that the current Plan remains suitable
- **WHEN** it returns Retain
- **THEN** the Plan Control Assessment records Retain without changing external Plan state

#### Scenario: Plan adjustment is required

- **GIVEN** the external AI determines that the current Plan requires adjustment
- **WHEN** it returns Revise with one valid Proposed Plan
- **THEN** the Plan Control Assessment contains that Proposed Plan

#### Scenario: AI cannot decide

- **GIVEN** required Delivery Observation meaning is unavailable to the external AI
- **WHEN** it returns Insufficient Information
- **THEN** the Plan Control Assessment records Insufficient Information without inventing a Proposed Plan

#### Scenario: Another Reconciliation Module returns another result

- **GIVEN** a different Reconciliation capability defines outcomes unsupported by Plan Control
- **WHEN** that capability produces one of its outcomes
- **THEN** the Plan Control Assessment contract places no constraint on it

### Requirement: PLC-4 Progress-sensitive adjustment

Plan Control MUST make supplied Delivery Observations available to the external AI without imposing a fixed observation set or restricting valid Plan revision beyond Plan invariants.

- **入力と受理**: Delivery Observations may concern schedule, progress, remaining work, quality, or requirement changes.
- **振る舞いの規則**: The external AI owns their semantic interpretation and may revise any Plan element subject to Plan validity. Scope remains expressed through Goal, Acceptance Conditions, and Tasks rather than a separate required Scope element.
- **参照**: [related] `plan` Conceptual Model for Plan composition and validity.

#### Scenario: Plan is delayed

- **GIVEN** schedule and progress Delivery Observations are supplied
- **WHEN** the external AI returns Revise with a valid Proposed Plan
- **THEN** the AI received those observations and the Plan Control Assessment contains that Proposed Plan

#### Scenario: Plan scope is insufficient

- **GIVEN** Delivery Observations concern Goal, Acceptance Condition, or Task coverage
- **WHEN** the external AI returns Revise with a valid Proposed Plan
- **THEN** the AI received those observations and the Plan Control Assessment contains that Proposed Plan

#### Scenario: Task amount is unsuitable

- **GIVEN** Delivery Observations concern excessive, insufficient, or overly coarse Tasks
- **WHEN** the external AI returns Revise with a valid Proposed Plan
- **THEN** the AI received those observations and the Plan Control Assessment contains that Proposed Plan

### Requirement: PLC-5 AI result contract

Plan Control MUST validate the external AI response form and Proposed Plan structure without replacing or semantically re-evaluating a valid AI-owned assessment.

- **入力と受理**: The response contains exactly one Plan Control Assessment. Only Revise contains exactly one Proposed Plan, which is valid and differs from the Plan Snapshot in at least one exact Plan element value.
- **振る舞いの規則**: A formally valid Complete, Retain, Revise, or Insufficient Information assessment is preserved as returned.
- **失敗の扱い**: An empty, malformed, contradictory, or otherwise invalid response produces AI Contract Failure and no valid Plan Control Assessment.
- **参照**: [related] `plan` Conceptual Model for exact Plan element identity and validity.

#### Scenario: Proposed Plan equals the snapshot

- **GIVEN** the external AI returns Revise with a Proposed Plan equal to the Plan Snapshot
- **WHEN** Plan Control validates the response
- **THEN** it returns AI Contract Failure and no valid Plan Control Assessment

#### Scenario: Proposed Plan is invalid

- **GIVEN** the external AI returns Revise with a structurally invalid Proposed Plan
- **WHEN** Plan Control validates the response
- **THEN** it returns AI Contract Failure and no valid Plan Control Assessment

#### Scenario: AI returns contradictory outcomes

- **GIVEN** the external AI response claims more than one outcome such as Complete and a Proposed Plan
- **WHEN** Plan Control validates the response
- **THEN** it returns AI Contract Failure and no valid Plan Control Assessment

#### Scenario: AI returns a formally valid complete assessment

- **GIVEN** the external AI returns exactly one formally valid Complete assessment
- **WHEN** Plan Control validates the response
- **THEN** it preserves Complete without independently re-evaluating the semantic judgment

### Requirement: PLC-6 AI boundary failure

A Plan Control caller MUST be able to distinguish AI Boundary Failure, AI Contract Failure, valid Insufficient Information, and its own lifecycle outcome.

- **振る舞いの規則**: A valid result remains associated with the Plan Snapshot supplied to that AI request. Detecting a newer authoritative Plan or revalidating a result before external application remains outside this capability.
- **失敗の扱い**: AI unavailability, Provider-side timeout while the caller lifecycle remains active, or another failure before an AI response becomes AI Boundary Failure and returns no result. Caller cancellation or deadline expiration before a valid response returns the supplied lifecycle outcome and no result. A valid Insufficient Information response remains an assessment rather than a failure.

#### Scenario: AI is unavailable

- **GIVEN** the caller lifecycle remains active and no AI response can be established because of unavailability, Provider-side timeout, or another external failure
- **WHEN** Plan Control completes the request
- **THEN** it returns AI Boundary Failure and no Plan Control Assessment

#### Scenario: Control request is cancelled

- **GIVEN** no valid AI response has been established
- **WHEN** the caller cancels or its deadline expires
- **THEN** Plan Control returns the supplied lifecycle outcome and no Plan Control Assessment

#### Scenario: AI reports insufficient information

- **GIVEN** the external AI successfully returns a valid Insufficient Information response
- **WHEN** Plan Control validates it
- **THEN** it returns the Insufficient Information assessment rather than AI Boundary Failure

### Requirement: PLC-7 Control boundary

Plan Control MUST remain an assessment capability and cause none of the external effects excluded by this Requirement.

- **副作用**: Plan Control does not authorize or apply a Proposed Plan, perform Tasks, decide Delivery Acceptance, collect observations, persist an authoritative Plan, or expose a Change concept or contract. It does not prescribe how a result becomes effective outside this capability.

#### Scenario: Valid adjustment is proposed

- **GIVEN** Plan Control has established a valid Revise assessment
- **WHEN** it returns the Proposed Plan
- **THEN** no external Plan state has changed and Plan Control has made no authorization decision

### Requirement: PLC-8 Stateless control

Plan Control MUST derive each assessment only from the Plan Snapshot, Delivery Observations, and AI response established for that decision.

- **振る舞いの規則**: A previous runtime decision is never authoritative input to a later decision.
- **副作用**: Plan Control retains no authoritative Plan or assessment state between requests.

#### Scenario: Runtime state is discarded

- **GIVEN** prior Plan Control runtime state has been discarded
- **WHEN** the same externally established inputs and AI response are supplied again
- **THEN** Plan Control can establish the same assessment
