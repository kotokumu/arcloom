# plan-reconciliation-loop Specification

## Purpose
Provides a standard read-only Plan attempt that distinguishes successful observation without a current Plan from actual Plan Reconciliation of one current Plan, while leaving Authorization, application, receipt handling, and later wake-up causes outside the attempt.

## Conceptual Model

### Plan Target Binding

A Plan Target Binding associates one exact Target Identity with its target-bound Plan Snapshot Observer and Delivery Observer. The Plan Attempt resolves the binding under its caller lifecycle, validates exact identity equality, and returns a stable boundary failure when the association is absent, invalid, or mismatched. Plan Control remains the target-specific judgment authority and is supplied as a Plan Attempt dependency. The Composition Root supplies the resolver and Plan Control boundary without orchestrating their invocation order.

### Plan Attempt and Result

A Plan Attempt is one read-only target-specific Attempt. It resolves the requested identity and begins with exactly one fresh Plan Snapshot. Its successful value is an explicit sum of Current Plan Not Established and Current Plan Assessed.

Current Plan Not Established preserves a successful Snapshot that establishes coherent representation progress without a valid current Plan. It contains no Delivery Observation or Assessment, performs no semantic Plan Reconciliation, and is not Failure, Insufficient Information, or authoritative absence.

Current Plan Assessed preserves the successful Snapshot and exact Plan Control Assessment for its current Plan. This branch acquires current Delivery Observations after the Snapshot and performs exactly one semantic Plan Reconciliation through Plan Control. Complete, Retain, Revise, and Insufficient Information retain their Plan-specific meanings; Revise retains its exact Proposed Plan.

Both successful branches select Await Another Request. The Attempt performs no Authorization, external application, Task execution, or Delivery Acceptance. External interactions and application results have no special relationship with a later Request and are never inputs to a Plan Attempt.

## Requirements

### Requirement: fresh-plan-observation

Each Plan Attempt MUST begin with one fresh snapshot observation of its bound external Plan target and preserve every successful snapshot outcome in exactly one explicit Plan Attempt Result branch.

- **前提条件**: Before control starts, the caller supplies a valid Plan Target, an exact accepted target kind, a Plan Target Resolver, and a Plan Control Assessor. A successful binding contains the exact requested Target Identity with one target-bound snapshot Observer and Delivery Observer. The Resolver observes the supplied caller context and returns within its documented cancellation bound after that context ends.
- **振る舞いの規則**: The Plan Attempt resolves that binding, validates exact identity equality, invokes the snapshot Observer exactly once, and establishes the fresh Snapshot before any Delivery Observation. Earlier snapshots and request causes are not fact sources. A successful Snapshot without a valid current Plan establishes Current Plan Not Established with no Delivery Observation, no Assessment request, and no semantic Plan Reconciliation. A successful Snapshot with a valid current Plan may establish only Current Plan Assessed after satisfying [[plan-reconciliation-loop/current-plan-assessment]].
- **失敗の扱い**: Invalid Plan Target, target-kind, Resolver, or Assessor configuration is rejected before an Attempt boundary or Controller lifecycle exists. Unavailable, invalid, or identity-mismatched resolved binding and Delivery Observation failure are Plan Attempt failures with stable Plan-owned codes and do not expose an underlying boundary failure as their own identity. Existing Snapshot and Plan Control failures preserve their existing contracts. Caller cancellation observed before or after each boundary interaction takes precedence. An Attempt failure establishes neither a Plan Attempt Result nor a Directive.
- **参照**: [[reconciliation-control-loop/level-based-attempt]]; [[github-plan-snapshot/current-authoritative-snapshot]]; [[github-plan-snapshot/current-plan-eligibility]]

#### Scenario: PRL-FPO-1 Current Plan is established [happy]

- **GIVEN** one bound target can provide a successful fresh snapshot containing a valid current Plan
- **WHEN** a Plan Attempt begins
- **THEN** the Attempt preserves that fresh Snapshot and may proceed to assessment of that exact current Plan

#### Scenario: PRL-FPO-0 Invalid Plan reconciliation configuration is rejected [error]

- **GIVEN** a Plan Target or Plan Attempt is configured with an invalid target identity, a missing observation boundary, an invalid accepted kind, no Resolver, or no Assessor
- **WHEN** the caller submits that configuration for use
- **THEN** it is rejected with the corresponding stable setup failure before any Attempt or Controller lifecycle begins

#### Scenario: PRL-FPO-2 Snapshot has no current Plan [boundary]

- **GIVEN** one fresh observation establishes coherent progress without a valid current Plan
- **WHEN** a Plan Attempt completes
- **THEN** it establishes Current Plan Not Established, preserves that Snapshot, contains no Delivery Observation or Assessment, performs no semantic Plan Reconciliation, and selects Await Another Request

#### Scenario: PRL-FPO-3 Snapshot observation fails [error]

- **GIVEN** no successful coherent snapshot can be established
- **WHEN** a Plan Attempt begins
- **THEN** it fails without a Plan Attempt Result or Plan Control Assessment

#### Scenario: PRL-FPO-4 Target binding is unavailable [error]

- **GIVEN** the Plan Target Resolver cannot establish a valid binding for the requested Target Identity
- **WHEN** a Plan Attempt begins
- **THEN** it returns a stable Plan reconciliation boundary failure without snapshot observation or Plan Control Assessment

#### Scenario: PRL-FPO-5 Exact target binding is used [isolation]

- **GIVEN** two Plan Target Identities resolve to distinct self-identifying Snapshot and Delivery Observation bindings
- **WHEN** each Plan Attempt runs
- **THEN** each resolver receives the exact requested identity, each resolved binding identifies that same target, and each Attempt invokes only that binding's Snapshot and Delivery boundaries before the configured Assessor

#### Scenario: PRL-FPO-6 Wrong target kind is rejected [error]

- **GIVEN** a requested Target Identity has a kind different from the Plan Attempt boundary's exact configured kind
- **WHEN** a Plan Attempt begins
- **THEN** it returns Target Binding Unavailable without invoking the Target Resolver or any bound observation or assessment capability

#### Scenario: PRL-FPO-7 Caller cancels at a boundary [error]

- **GIVEN** caller cancellation is observed before, during, or immediately after Target resolution, Snapshot observation, Delivery observation, or Plan Control assessment and each boundary honors its cancellation contract
- **WHEN** the Plan Attempt returns
- **THEN** it returns the exact caller context error with zero Plan Attempt Result and zero Directive instead of a Plan-owned or composed boundary failure

### Requirement: current-plan-assessment

A Plan Attempt with a valid current Plan MUST next acquire current caller-selected Delivery Observations, perform exactly one semantic Plan Reconciliation through Plan Control, and establish Current Plan Assessed from that exact Plan and Assessment.

- **入力と受理**: The exact current Plan from the fresh snapshot and the Delivery Observations acquired after that Snapshot are supplied to Plan Control.
- **振る舞いの規則**: A valid Complete, Retain, Revise, or Insufficient Information Assessment is the target-specific Reconciliation Result and is preserved unchanged in Current Plan Assessed with the assessed Plan and successful Snapshot. Revise preserves its exact Proposed Plan.
- **失敗の扱い**: Delivery Observation failure, Plan Control failure, or caller lifecycle termination produces no Current Plan Assessed branch and no successful Plan Attempt Result.
- **参照**: [related] `openspec/specs/plan-control/spec.md`

#### Scenario: PRL-CPA-1 Current Plan is assessed [happy]

- **GIVEN** a fresh snapshot contains a valid current Plan and current Delivery Observations are available
- **WHEN** Plan Control returns one valid Assessment
- **THEN** Current Plan Assessed associates that exact Assessment with that exact current Plan and records that one semantic Plan Reconciliation occurred

#### Scenario: PRL-CPA-2 Plan Control cannot establish an Assessment [error]

- **GIVEN** a fresh snapshot contains a valid current Plan
- **WHEN** Plan Control fails or the caller lifecycle ends before a valid Assessment exists
- **THEN** no successful Plan Attempt Result or semantic Plan Reconciliation Result is established

#### Scenario: PRL-CPA-3 Delivery observation fails [error]

- **GIVEN** a fresh snapshot contains a valid current Plan
- **WHEN** current Delivery Observations cannot be established
- **THEN** the Attempt returns a stable Plan reconciliation boundary failure and does not request Plan Control Assessment

### Requirement: read-only-plan-attempt

Every successful Plan Attempt Result branch MUST remain read-only, select Await Another Request, and leave every proposed external application to a separate explicit interaction.

- **振る舞いの規則**: Current Plan Not Established and Current Plan Assessed with Complete, Retain, Revise, or Insufficient Information all select Await Another Request.
- **副作用**: Read-only means the Attempt performs no Authorization evaluation, application request, Actor contact for mutation, target mutation, Task execution, or Delivery Acceptance and does not convert a Proposed Plan into an externally effective revision. Observation and AI assessment may retain the operational I/O, cost, and telemetry effects of their existing Provider contracts.
- **参照**: [[reconciliation-control-loop/explicit-control-directive]]; [[plan-application-request/current-authorization-required]]; [[plan-application-request/external-state-requires-fresh-observation]]

#### Scenario: PRL-RPA-1 Assessment proposes revision [happy]

- **GIVEN** Plan Control returns Revise with one exact valid Proposed Plan
- **WHEN** the Plan Attempt returns Current Plan Assessed
- **THEN** the branch preserves the Assessment, selects Await Another Request, and no Authorization or application interaction has occurred

#### Scenario: PRL-RPA-2 Assessment is complete [happy]

- **GIVEN** Plan Control returns Complete
- **WHEN** the Plan Attempt returns Current Plan Assessed
- **THEN** it selects Await Another Request without deciding Delivery Acceptance or mutating the Plan

### Requirement: ordinary-plan-reentry

Every later Plan evaluation requested after an external interaction or event MUST use the same ordinary Target Identity request and fresh-observation behavior as any other request.

- **入力と受理**: The caller supplies only the valid Plan Target Identity required by the generic request contract.
- **振る舞いの規則**: Event type, Authorization Decision, Plan Application Result, request receipt, and prior Plan Attempt Result do not enter the Attempt as current Plan facts or scheduling instructions.
- **副作用**: No external interaction or application result automatically creates a Reconciliation Request or Control Directive.
- **参照**: [[reconciliation-control-loop/exact-target-request]]; [[reconciliation-control-loop/level-based-attempt]]; [[plan-application-request/request-interaction-result]]

#### Scenario: PRL-OPR-1 Caller requests after application interaction [happy]

- **GIVEN** a separate Plan application interaction has produced any defined result
- **WHEN** the caller explicitly requests reconciliation for the same Plan target
- **THEN** the new Attempt begins with fresh observation and receives neither the application result nor its receipt as a decision input

#### Scenario: PRL-OPR-2 No later request is submitted [boundary]

- **GIVEN** a separate external interaction or event has occurred
- **WHEN** the caller submits no Reconciliation Request and no prior successful Directive made the target eligible
- **THEN** that occurrence alone starts no Plan Attempt
