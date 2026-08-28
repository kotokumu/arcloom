## Purpose

Plan Control supplies an externally established Plan snapshot and delivery observations to an external AI that controls the Plan toward completion, while Arcloom validates and represents the AI result without becoming the executor or authoritative Plan store.

## ADDED Requirements

### Requirement: PLC-1 Plan control subject
Plan Control SHALL control one existing Plan toward completion. Its input SHALL be a valid snapshot of the current Plan established from authoritative external facts for that control decision. Plan creation and removal are outside this capability.

#### Scenario: Existing Plan is controlled
- **WHEN** a valid Plan snapshot and delivery observations are supplied
- **THEN** the control result concerns adjustment or completion of that Plan snapshot

#### Scenario: Plan does not exist
- **WHEN** no current Plan snapshot can be established
- **THEN** Plan Control fails with a stable input category without requesting an AI assessment

### Requirement: PLC-2 Plan completion meaning
The complete assessment SHALL mean that the external AI determines from available observations that the Plan's Goal and acceptance conditions have been achieved. Plan Control SHALL preserve that semantic judgment without independently re-evaluating the observations. Task completion SHALL be evidence available to the AI but SHALL be neither necessary nor sufficient by itself for the AI's judgment. Plan completion SHALL NOT constitute Delivery Acceptance.

#### Scenario: Goal and acceptance conditions are achieved
- **WHEN** the AI establishes from available observations that the Goal and acceptance conditions are achieved
- **THEN** the control result records Plan completion

#### Scenario: All Tasks are complete without outcome evidence
- **WHEN** every observed Task is complete, evidence required to establish the Goal or acceptance conditions is unavailable, and the AI returns insufficient information
- **THEN** the control result records insufficient information rather than completion

#### Scenario: Obsolete Task remains incomplete
- **WHEN** one obsolete Task remains incomplete and the AI determines from other observations that the Goal and acceptance conditions are achieved
- **THEN** the control result records the AI's complete assessment

### Requirement: PLC-3 AI control assessment
The external AI SHALL decide how to control the Plan toward completion from the Plan snapshot and available observations. Plan Control SHALL represent exactly one of the initially supported Plan-specific assessments: complete, retain the current Plan, propose one revised Plan, or insufficient information. This assessment contract SHALL NOT define a common result contract for other Reconciliation Modules and SHALL NOT expose a Change Concept.

#### Scenario: Plan remains suitable
- **WHEN** the AI determines that no adjustment is required
- **THEN** the control result records that assessment without changing external Plan state

#### Scenario: Plan adjustment is required
- **WHEN** the AI determines that the Plan requires adjustment
- **THEN** the control result contains one structurally valid proposed next Plan

#### Scenario: AI cannot decide
- **WHEN** the AI reports that required information is unavailable
- **THEN** the control result records insufficient information without inventing a Plan adjustment

#### Scenario: Another Reconciliation Module returns another result
- **WHEN** a different Reconciliation Module defines results that Plan Control does not support
- **THEN** the Plan Control result contract places no constraint on those results

### Requirement: PLC-4 Progress-sensitive adjustment
Plan Control SHALL make supplied observations available to the external AI without requiring a fixed observation set. Observations MAY describe schedule, progress, remaining work, quality, or requirement changes. The AI SHALL own the semantic judgment about whether those observations require Plan adjustment. Plan Control SHALL NOT restrict which Plan elements the AI may revise beyond the structural invariants of Plan. Scope SHALL be expressed through the Plan's Goal, acceptance conditions, and Tasks rather than through a separate required Scope element.

#### Scenario: Plan is delayed
- **WHEN** schedule and progress observations are supplied and the AI returns a structurally valid revised Plan
- **THEN** the AI boundary receives those observations and the control result contains the revised Plan

#### Scenario: Plan scope is insufficient
- **WHEN** observations about Goal, acceptance-condition, or Task coverage are supplied and the AI returns a structurally valid revised Plan
- **THEN** the AI boundary receives those observations and the control result contains the revised Plan

#### Scenario: Task amount is unsuitable
- **WHEN** observations about excessive, insufficient, or overly coarse Tasks are supplied and the AI returns a structurally valid revised Plan
- **THEN** the AI boundary receives those observations and the control result contains the revised Plan

### Requirement: PLC-5 AI result contract
The external AI SHALL own the semantic judgment expressed by its assessment. Plan Control SHALL validate only the response form, the exclusivity required by PLC-3, and the structural validity of a proposed revised Plan. It SHALL NOT replace or semantically re-evaluate a structurally valid AI assessment. A proposed revised Plan SHALL differ from the supplied Plan in at least one preserved Plan element value. A successfully returned but empty, malformed, or contradictory AI response SHALL fail with a stable AI-contract category.

#### Scenario: Proposed Plan equals the snapshot
- **WHEN** the AI proposes a next Plan equal to the supplied Plan snapshot
- **THEN** Plan Control fails with the stable AI-contract category and returns no valid control result

#### Scenario: Proposed Plan is invalid
- **WHEN** the AI proposes a structurally invalid next Plan
- **THEN** Plan Control fails with the stable AI-contract category and returns no valid control result

#### Scenario: AI returns contradictory outcomes
- **WHEN** the AI response claims more than one outcome such as complete and proposed next Plan
- **THEN** Plan Control fails with the stable AI-contract category and returns no valid control result

#### Scenario: AI returns a formally valid complete assessment
- **WHEN** the AI returns one complete assessment in a valid response form
- **THEN** Plan Control preserves that assessment without independently re-evaluating its semantic judgment

### Requirement: PLC-6 AI boundary failure
Failure of the AI interaction before a response is returned SHALL be distinct from both an AI-contract failure and a valid insufficient-information assessment. AI unavailability, a Provider-side timeout while the supplied request context remains active, or another Provider failure SHALL produce a stable AI-boundary failure and no Plan Control result. Cancellation or deadline expiry of the supplied request context before a valid response is established SHALL return that context error and no Plan Control result. A result SHALL remain associated with the Plan snapshot supplied to that AI request. Detecting a newer authoritative Plan or revalidating a result before external application is outside this capability.

#### Scenario: AI is unavailable
- **WHEN** the external AI cannot return a response because of unavailability, a Provider-side timeout while the supplied request context remains active, or another Provider failure
- **THEN** Plan Control returns a stable AI-boundary failure and no Plan Control result

#### Scenario: Control request is cancelled
- **WHEN** the supplied request context is cancelled or its deadline expires before a valid AI response is established
- **THEN** Plan Control returns the supplied context error and no Plan Control result

#### Scenario: AI reports insufficient information
- **WHEN** the external AI successfully returns the valid insufficient-information assessment
- **THEN** Plan Control returns that assessment rather than an AI-boundary failure

### Requirement: PLC-7 Control boundary
Plan Control SHALL NOT authorize or apply a proposed next Plan, perform Tasks, decide Delivery Acceptance, collect observations, persist an authoritative Plan, or expose a Change Concept or contract. Plan Control SHALL NOT prescribe how its result becomes effective outside this capability.

#### Scenario: Valid adjustment is proposed
- **WHEN** Plan Control returns a proposed next Plan
- **THEN** no external Plan state has been changed and no authorization decision has been made by Plan Control

### Requirement: PLC-8 Stateless control
Each Plan control result SHALL be derived from the Plan snapshot, observations, and AI response established for that decision. A previous runtime decision SHALL NOT be authoritative input to a later decision.

#### Scenario: Runtime state is discarded
- **WHEN** the same externally established inputs and AI response are supplied after prior runtime state is discarded
- **THEN** Plan Control can establish the same control result
