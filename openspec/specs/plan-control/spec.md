## Purpose

Plan Control supplies an externally established Plan snapshot and delivery observations to an external AI that controls the Plan toward completion, while Arcloom validates and represents the AI result without becoming the executor or authoritative Plan store.

## Conceptual Model

### Plan Snapshot and Delivery Observations

A Plan Snapshot is one valid current Plan established from authoritative external facts for one control decision. The resulting assessment remains associated with that exact Plan Snapshot. A Delivery Observation is caller-supplied evidence made available to the external AI without Plan Control defining a fixed observation vocabulary or reinterpreting its semantics.

### Plan Control Assessment

A Plan Control Assessment is exactly Complete, Retain, Revise, or Insufficient Information. Complete means the external AI determines that the Plan Goal and Acceptance Conditions have been achieved. Retain means it determines that the current Plan remains suitable. Revise means it supplies exactly one Proposed Plan. Insufficient Information means it cannot decide from available observations. Task completion is evidence but is neither necessary nor sufficient for Complete.

A Proposed Plan is structurally valid and differs from the Plan Snapshot in at least one exact Plan element value. It exists only for Revise.

### Plan Control Failure

A Plan Control Failure produces no valid assessment. Invalid input identifies an unusable Plan Snapshot. AI Boundary Failure means no AI response was established because the external interaction failed while the caller lifecycle remained active. AI Contract Failure means a returned response has an invalid or contradictory form or Proposed Plan. Caller cancellation or deadline expiration before a valid response remains the caller lifecycle outcome rather than either AI failure.

## Requirements

### Requirement: plan-control-subject

A Plan Control caller MUST receive an assessment concerning one valid Plan Snapshot supplied for that decision.

- **Preconditions**: The Plan Snapshot has been established from authoritative external facts. Plan creation and removal remain outside this capability.
- **Input and Acceptance**: One valid Plan Snapshot and the Delivery Observations for the decision are supplied.
- **Behavioral Rules**: The result concerns adjustment or completion of exactly that Plan Snapshot.
- **Failure Handling**: If no valid Plan Snapshot exists, Plan Control returns a stable invalid-input failure, requests no AI assessment, and returns no valid result.
- **References**: [related] `plan` Conceptual Model for Plan validity and identity.

#### Scenario: Existing Plan is controlled [happy]

- **GIVEN** a valid Plan Snapshot and Delivery Observations
- **WHEN** a caller requests Plan Control assessment
- **THEN** the result concerns adjustment or completion of that Plan Snapshot

#### Scenario: Plan does not exist [error]

- **GIVEN** no current valid Plan Snapshot can be established
- **WHEN** a caller requests Plan Control assessment
- **THEN** Plan Control returns a stable invalid-input failure without requesting an AI assessment

### Requirement: plan-completion-meaning

A Plan Control caller MUST receive Complete only when the external AI determines from available Delivery Observations that the Plan Goal and Acceptance Conditions have been achieved.

- **Behavioral Rules**: Plan Control preserves the AI-owned semantic judgment without independently re-evaluating Delivery Observations. Task completion is evidence available to the AI but is neither necessary nor sufficient by itself for Complete.
- **Side Effects**: Complete does not constitute Delivery Acceptance.

#### Scenario: Goal and acceptance conditions are achieved [happy]

- **GIVEN** available Delivery Observations establish the Goal and Acceptance Conditions to the external AI
- **WHEN** the AI returns Complete
- **THEN** the Plan Control Assessment records Complete

#### Scenario: All Tasks are complete without outcome evidence [boundary]

- **GIVEN** every observed Task is complete but evidence required to establish the Goal or Acceptance Conditions is unavailable
- **WHEN** the AI returns Insufficient Information
- **THEN** the Plan Control Assessment records Insufficient Information rather than Complete

#### Scenario: Obsolete Task remains incomplete [boundary]

- **GIVEN** one obsolete Task remains incomplete while other observations establish the Goal and Acceptance Conditions to the external AI
- **WHEN** the AI returns Complete
- **THEN** the Plan Control Assessment records Complete

### Requirement: ai-control-assessment

The external AI MUST select exactly one Plan Control Assessment defined by this capability for the supplied Plan Snapshot and Delivery Observations.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Complete | Exact Snapshot and observations are supplied | AI judges the Goal and Acceptance Conditions achieved | Complete with no Proposed Plan | None |
  | Retain | Exact Snapshot and observations are supplied | AI judges no adjustment is required | Retain with no Proposed Plan | None |
  | Revise | Exact Snapshot and observations are supplied | AI judges adjustment is required | Revise with exactly one Proposed Plan | None |
  | Insufficient Information | Exact Snapshot is supplied | Required evidence is unavailable | Insufficient Information with no Proposed Plan | None |

- **Invariants**: Exactly one classification is established and remains associated with the exact supplied Snapshot and Delivery Observations.
- **Side Effects**: An assessment does not change external Plan state. This Plan-specific contract neither constrains other reconciliation results nor exposes a Change concept.

#### Scenario: Plan adjustment is required [happy]

- **GIVEN** the external AI determines that the current Plan requires adjustment
- **WHEN** it returns Revise with one valid Proposed Plan
- **THEN** the Plan Control Assessment contains that Proposed Plan

#### Scenario: Another Reconciliation Module returns another result [compatibility]

- **GIVEN** a different Reconciliation capability defines outcomes unsupported by Plan Control
- **WHEN** that capability produces one of its outcomes
- **THEN** the Plan Control Assessment contract places no constraint on it

### Requirement: progress-sensitive-adjustment

Plan Control MUST make supplied Delivery Observations available to the external AI without imposing a fixed observation set or restricting valid Plan revision beyond Plan invariants.

- **Input and Acceptance**: Delivery Observations may concern schedule, progress, remaining work, quality, or requirement changes.
- **Behavioral Rules**: The external AI owns their semantic interpretation and may revise any Plan element subject to Plan validity. Scope remains expressed through Goal, Acceptance Conditions, and Tasks rather than a separate required Scope element.
- **References**: [related] `plan` Conceptual Model for Plan composition and validity.

#### Scenario: Plan is delayed [happy]

- **GIVEN** schedule and progress Delivery Observations are supplied
- **WHEN** the external AI returns Revise with a valid Proposed Plan
- **THEN** the AI received those observations and the Plan Control Assessment contains that Proposed Plan

### Requirement: ai-result-contract

Plan Control MUST validate the external AI response form and Proposed Plan structure without replacing or semantically re-evaluating a valid AI-owned assessment.

- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Non-Revise valid | Exactly one Complete, Retain, or Insufficient Information exists | No Proposed Plan exists | Preserve the assessment | None |
  | Revise valid | Exactly one Revise exists | Exactly one valid Proposed Plan differs from the Snapshot | Preserve Revise and the Proposed Plan | None |
  | Equal proposal | Exactly one Revise exists | Proposed Plan equals the Snapshot | AI Contract Failure and no Assessment | None |
  | Invalid proposal | Exactly one Revise exists | Proposed Plan is invalid | AI Contract Failure and no Assessment | None |
  | Invalid classification | Response is empty, malformed, or contradictory | Zero or multiple outcomes exist | AI Contract Failure and no Assessment | None |
- **Failure Handling**: An empty, malformed, contradictory, or otherwise invalid response produces AI Contract Failure and no valid Plan Control Assessment.
- **References**: [related] `plan` Conceptual Model for exact Plan element identity and validity.

#### Scenario: Proposed Plan equals the snapshot [boundary]

- **GIVEN** the external AI returns Revise with a Proposed Plan equal to the Plan Snapshot
- **WHEN** Plan Control validates the response
- **THEN** it returns AI Contract Failure and no valid Plan Control Assessment

#### Scenario: AI returns a formally valid complete assessment [happy]

- **GIVEN** the external AI returns exactly one formally valid Complete assessment
- **WHEN** Plan Control validates the response
- **THEN** it preserves Complete without independently re-evaluating the semantic judgment

### Requirement: ai-boundary-failure

A Plan Control caller MUST be able to distinguish AI Boundary Failure, AI Contract Failure, valid Insufficient Information, and its own lifecycle outcome.

- **Behavioral Rules**: A valid result remains associated with the Plan Snapshot supplied to that AI request. Detecting a newer authoritative Plan or revalidating a result before external application remains outside this capability.
- **Failure Handling**: AI unavailability, Provider-side timeout while the caller lifecycle remains active, or another failure before an AI response becomes AI Boundary Failure and returns no result. Caller cancellation or deadline expiration before a valid response returns the supplied lifecycle outcome and no result. A valid Insufficient Information response remains an assessment rather than a failure.

#### Scenario: AI is unavailable [error]

- **GIVEN** the caller lifecycle remains active and no AI response can be established because of unavailability, Provider-side timeout, or another external failure
- **WHEN** Plan Control completes the request
- **THEN** it returns AI Boundary Failure and no Plan Control Assessment

#### Scenario: Control request is cancelled [error]

- **GIVEN** no valid AI response has been established
- **WHEN** the caller cancels or its deadline expires
- **THEN** Plan Control returns the supplied lifecycle outcome and no Plan Control Assessment

#### Scenario: AI reports insufficient information [happy]

- **GIVEN** the external AI successfully returns a valid Insufficient Information response
- **WHEN** Plan Control validates it
- **THEN** it returns the Insufficient Information assessment rather than AI Boundary Failure

### Requirement: control-boundary

Plan Control MUST remain an assessment capability and cause none of the external effects excluded by this Requirement.

- **Side Effects**: Plan Control does not authorize or apply a Proposed Plan, perform Tasks, decide Delivery Acceptance, collect observations, persist an authoritative Plan, or expose a Change concept or contract. It does not prescribe how a result becomes effective outside this capability.

#### Scenario: Valid adjustment is proposed [happy]

- **GIVEN** Plan Control has established a valid Revise assessment
- **WHEN** it returns the Proposed Plan
- **THEN** no external Plan state has changed and Plan Control has made no authorization decision

### Requirement: stateless-control

Plan Control MUST derive each assessment only from the Plan Snapshot, Delivery Observations, and AI response established for that decision.

- **Behavioral Rules**: A previous runtime decision is never authoritative input to a later decision.
- **Side Effects**: Plan Control retains no authoritative Plan or assessment state between requests.

#### Scenario: Runtime state is discarded [compatibility]

- **GIVEN** prior Plan Control runtime state has been discarded
- **WHEN** the same externally established inputs and AI response are supplied again
- **THEN** Plan Control can establish the same assessment
