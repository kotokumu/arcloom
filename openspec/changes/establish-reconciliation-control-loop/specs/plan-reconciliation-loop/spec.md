## Purpose

Provides a standard read-only Plan reconciliation attempt that associates one fresh externally authoritative Plan observation with an optional exact Plan Control assessment while leaving Authorization, application, receipt handling, and later wake-up causes outside the attempt.

## ADDED Requirements

### Requirement: fresh-plan-observation

Each Plan Reconciliation Attempt MUST begin with one fresh snapshot observation of its bound external Plan target and preserve every successful snapshot outcome in its Plan Reconciliation Result.

- **前提条件**: Before control starts, the caller supplies valid Plan Target construction, an exact accepted target kind, a non-nil Plan Target Resolver, and a non-nil Plan Control Assessor. A successful binding contains the exact requested Target Identity with one target-bound snapshot Observer and Delivery Observer. The Resolver observes the supplied caller context and returns within its documented cancellation bound after that context ends.
- **振る舞いの規則**: The Plan Reconciliation Attempt resolves that binding, validates exact identity equality, invokes the snapshot Observer exactly once, and establishes the fresh Snapshot before acquiring Delivery Observations. Earlier snapshots and request causes are not fact sources. A successful snapshot without a valid current Plan remains a successful Plan Reconciliation Result with no Delivery Observation or Assessment request.
- **失敗の扱い**: Invalid Plan Target, target-kind, Resolver, or Assessor configuration is rejected before a Reconciler or Controller lifecycle exists. Unavailable, invalid, or identity-mismatched resolved binding and Delivery Observation failure are runtime Plan Attempt failures with stable Plan-owned codes and do not expose supplied errors through error unwrapping. Existing Snapshot and Plan Control failures preserve their existing contracts. A caller context error observed before or after each boundary call wins unchanged. Every Attempt failure returns zero Result and zero Directive.
- **参照**: [[reconciliation-control-loop/level-based-attempt]]; [[github-plan-snapshot/current-authoritative-snapshot]]; [[github-plan-snapshot/current-plan-eligibility]]

#### Scenario: PRL-FPO-1 Current Plan is established [happy]

- **GIVEN** one bound target can provide a successful fresh snapshot containing a valid current Plan
- **WHEN** a Plan Reconciliation Attempt begins
- **THEN** its Result preserves that fresh snapshot and may proceed to assessment of that exact current Plan

#### Scenario: PRL-FPO-0 Invalid Plan reconciliation configuration is rejected [error]

- **GIVEN** Plan Target construction or Plan Reconciler construction receives an invalid target identity, missing observation boundary, invalid accepted kind, nil Resolver, or nil Assessor
- **WHEN** the caller constructs that value or Reconciler
- **THEN** construction returns the corresponding stable setup error before a Reconciler or Controller lifecycle exists

#### Scenario: PRL-FPO-2 Snapshot has no current Plan [boundary]

- **GIVEN** one fresh observation establishes coherent progress without a valid current Plan
- **WHEN** a Plan Reconciliation Attempt completes
- **THEN** its Result preserves that snapshot, contains no Assessment, and selects Await Another Request

#### Scenario: PRL-FPO-3 Snapshot observation fails [error]

- **GIVEN** no successful coherent snapshot can be established
- **WHEN** a Plan Reconciliation Attempt begins
- **THEN** it fails without a Plan Reconciliation Result or Plan Control Assessment

#### Scenario: PRL-FPO-4 Target binding is unavailable [error]

- **GIVEN** the Plan Target Resolver cannot establish a valid binding for the requested Target Identity
- **WHEN** a Plan Reconciliation Attempt begins
- **THEN** it returns a stable Plan reconciliation boundary failure without snapshot observation or Plan Control Assessment

#### Scenario: PRL-FPO-5 Exact target binding is used [isolation]

- **GIVEN** two Plan Target Identities resolve to distinct self-identifying Snapshot and Delivery Observation bindings
- **WHEN** each Plan Reconciliation Attempt runs
- **THEN** each resolver receives the exact requested identity, each resolved binding identifies that same target, and each Attempt invokes only that binding's Snapshot and Delivery boundaries before the configured Assessor

#### Scenario: PRL-FPO-6 Wrong target kind is rejected [error]

- **GIVEN** a requested Target Identity has a kind different from the Plan Reconciler's exact configured kind
- **WHEN** a Plan Reconciliation Attempt begins
- **THEN** it returns Target Binding Unavailable without invoking the Target Resolver or any bound observation or assessment capability

#### Scenario: PRL-FPO-7 Caller cancels at a boundary [error]

- **GIVEN** caller cancellation is observed before, during, or immediately after Target resolution, Snapshot observation, Delivery observation, or Plan Control assessment and each boundary honors its cancellation contract
- **WHEN** the Plan Reconciliation Attempt returns
- **THEN** it returns the exact caller context error with zero Result and zero Directive instead of a Plan-owned or composed boundary failure

### Requirement: current-plan-assessment

A Plan Reconciliation Attempt with a valid current Plan MUST next acquire current caller-selected Delivery Observations and request one Plan Control Assessment for that exact Plan and those Observations.

- **入力と受理**: The exact current Plan from the fresh snapshot and the Delivery Observations acquired after that Snapshot are supplied to Plan Control.
- **振る舞いの規則**: A valid Complete, Retain, Revise, or Insufficient Information Assessment is preserved unchanged in the Result and remains associated with the assessed Plan. Revise preserves its exact Proposed Plan.
- **失敗の扱い**: Plan Control failure or caller lifecycle termination produces no successful Plan Reconciliation Result.
- **参照**: [related] `openspec/specs/plan-control/spec.md`

#### Scenario: PRL-CPA-1 Current Plan is assessed [happy]

- **GIVEN** a fresh snapshot contains a valid current Plan and current Delivery Observations are available
- **WHEN** Plan Control returns one valid Assessment
- **THEN** the Plan Reconciliation Result associates that exact Assessment with that exact current Plan

#### Scenario: PRL-CPA-2 Plan Control cannot establish an Assessment [error]

- **GIVEN** a fresh snapshot contains a valid current Plan
- **WHEN** Plan Control fails or the caller lifecycle ends before a valid Assessment exists
- **THEN** no successful Plan Reconciliation Result is established

#### Scenario: PRL-CPA-3 Delivery observation fails [error]

- **GIVEN** a fresh snapshot contains a valid current Plan
- **WHEN** current Delivery Observations cannot be established
- **THEN** the Attempt returns a stable Plan reconciliation boundary failure and does not request Plan Control Assessment

### Requirement: read-only-plan-attempt

Every successful Plan Reconciliation Attempt MUST remain read-only, select Await Another Request, and leave every proposed external application to a separate explicit interaction.

- **振る舞いの規則**: Complete, Retain, Revise, Insufficient Information, and a successful snapshot without a current Plan all select Await Another Request.
- **副作用**: Read-only means the Attempt performs no Authorization evaluation, application request, Actor contact for mutation, target mutation, Task execution, or Delivery Acceptance and does not convert a Proposed Plan into an externally effective revision. Observation and AI assessment may retain the operational I/O, cost, and telemetry effects of their existing Provider contracts.
- **参照**: [[reconciliation-control-loop/explicit-control-directive]]; [[plan-application-request/current-authorization-required]]; [[plan-application-request/external-state-requires-fresh-observation]]

#### Scenario: PRL-RPA-1 Assessment proposes revision [happy]

- **GIVEN** Plan Control returns Revise with one exact valid Proposed Plan
- **WHEN** the Plan Reconciliation Attempt returns its Result
- **THEN** the Result preserves the Assessment, selects Await Another Request, and no Authorization or application interaction has occurred

#### Scenario: PRL-RPA-2 Assessment is complete [happy]

- **GIVEN** Plan Control returns Complete
- **WHEN** the Plan Reconciliation Attempt returns its Result
- **THEN** it selects Await Another Request without deciding Delivery Acceptance or mutating the Plan

### Requirement: ordinary-plan-reentry

Every later Plan reconciliation requested after an external interaction or event MUST use the same ordinary Target Identity request and fresh-observation behavior as any other request.

- **入力と受理**: The caller supplies only the valid Plan Target Identity required by the generic request contract.
- **振る舞いの規則**: Event type, Authorization Decision, Plan Application Result, request receipt, and prior Plan Reconciliation Result do not enter the Attempt as current Plan facts or scheduling instructions.
- **副作用**: No external interaction or application result automatically creates a Reconciliation Request or Control Directive.
- **参照**: [[reconciliation-control-loop/exact-target-request]]; [[reconciliation-control-loop/level-based-attempt]]; [[plan-application-request/request-interaction-result]]

#### Scenario: PRL-OPR-1 Caller requests after application interaction [happy]

- **GIVEN** a separate Plan application interaction has produced any defined result
- **WHEN** the caller explicitly requests reconciliation for the same Plan target
- **THEN** the new Attempt begins with fresh observation and receives neither the application result nor its receipt as a decision input

#### Scenario: PRL-OPR-2 No later request is submitted [boundary]

- **GIVEN** a separate external interaction or event has occurred
- **WHEN** the caller submits no Reconciliation Request and no prior successful Directive made the target eligible
- **THEN** that occurrence alone starts no Plan Reconciliation Attempt
