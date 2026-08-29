## Purpose

Obtains a read-only Codex-backed Plan Control judgment with provider-independent meaning and without retaining authoritative AI session state.

## ADDED Requirements

### Requirement: valid-plan-control-judgment

A successful Codex-backed assessment MUST yield exactly one valid provider-independent Plan Control Assessment: Complete, Retain, Revise, or Insufficient Information.

- **振る舞いの規則**: Revise contains exactly one valid Proposed Plan. Complete, Retain, and Insufficient Information contain no Proposed Plan.
- **参照**: [related] `openspec/specs/plan-control/spec.md` (PLC-3 and PLC-5)

#### Scenario: Plan should be revised [happy]

- **GIVEN** Codex establishes a valid Revise judgment for the assessed material
- **WHEN** the assessment succeeds
- **THEN** it contains exactly one valid Proposed Plan

#### Scenario: Plan should not be revised [happy]

- **GIVEN** Codex establishes Complete, Retain, or Insufficient Information
- **WHEN** the assessment succeeds
- **THEN** it contains no Proposed Plan

### Requirement: exact-current-assessment-material

A Codex-backed assessment MUST concern the exact current Plan and caller-owned observation material supplied for that invocation.

- **入力と受理**: The caller supplies one valid current Plan and immutable observation material whose vocabulary and meaning remain caller-owned.
- **振る舞いの規則**: The judgment is associated only with that Plan and observation material.

#### Scenario: Assessment material is supplied [happy]

- **GIVEN** a caller has one valid current Plan and caller-owned observation material
- **WHEN** the caller requests a Codex-backed assessment
- **THEN** the judgment is associated only with that exact supplied material

### Requirement: read-only-assessment-boundary

A Codex-backed assessment MUST remain read-only and MUST NOT authorize or apply a Plan revision.

- **副作用**: Assessment modifies neither supplied material nor any external provider and makes no authorization or application request.

#### Scenario: Assessment proposes a revision [happy]

- **GIVEN** the assessment returns Revise with one valid Proposed Plan
- **WHEN** the caller receives the assessment
- **THEN** the Proposed Plan remains only a proposal and no external state has changed

### Requirement: invalid-ai-output

Missing, malformed, ambiguous, conflicting, or failed AI output MUST NOT produce a successful Plan Control Assessment.

- **失敗の扱い**: No provider protocol detail becomes a successful provider-independent judgment.

#### Scenario: Output has conflicting meanings [error]

- **GIVEN** returned AI output cannot represent exactly one valid Plan Control Assessment
- **WHEN** the output is considered
- **THEN** no successful assessment is established

### Requirement: independent-assessments

Each Codex-backed assessment MUST be independent of prior and concurrent assessments and retain no authoritative AI session state.

- **振る舞いの規則**: A later assessment depends only on its current supplied material.
- **排他・冪等**: Concurrent assessments exchange no Plan, observation, or judgment state.

#### Scenario: A Plan is reassessed [happy]

- **GIVEN** one or more earlier assessments exist
- **WHEN** the caller requests another assessment with current material
- **THEN** the new result is based on that current material rather than retained prior session state

#### Scenario: Assessments run concurrently [concurrency]

- **GIVEN** distinct current Plans and observation material
- **WHEN** callers request their assessments concurrently
- **THEN** each result refers only to its own Plan and observation material

### Requirement: bounded-assessment-cancellation

A cancelled in-progress assessment MUST establish no successful judgment and terminate within its accepted finite shutdown bound.

- **入力と受理**: The supplied Codex app-server SDK Client is configured with a positive finite shutdown bound before it is composed with the Plan Control adapter.
- **状態と遷移**: Success is established only after the correlated Turn reports `completed`, one translatable final output is established, and cancellation has not already occurred. Cancellation after that establishment does not replace success.
- **失敗の扱い**: Cancellation before success returns no assessment with an error matching the supplied context error no later than that bound plus scheduling tolerance. A concurrent shutdown failure remains observable without hiding the context error.

#### Scenario: Assessment is cancelled [error]

- **GIVEN** an assessment is in progress under an accepted finite shutdown bound
- **WHEN** the caller cancels it before success
- **THEN** no successful judgment is returned, the error matches the supplied context error, and the interaction terminates within that bound plus scheduling tolerance

### Requirement: safe-host-configuration

The capability MUST begin an external-AI Turn only with Safe Host Configuration and MUST provide no input that relaxes its fixed read-only constraints.

- **前提条件**: The Host supplies an absolute executable path for Codex 0.149.1 through a Codex app-server SDK Client that satisfies the accepted lifecycle contract.
- **入力と受理**: Local assessment inputs must be valid and compatible. The SDK validates the executable path and positive finite shutdown bound before process start, verifies the app-server version before starting a Thread, and revalidates mutable request paths before each call. Every Turn fixes approval policy to `never` and uses a read-only sandbox with agent-initiated network access disabled. Read-only local tool activity may occur within that sandbox.
- **構成の隔離**: Effective MCP servers, Apps, Hooks, and Web Search are rejected before starting a Turn. These constraints are not caller-relaxable.
- **失敗の扱い**: Incompatible, invalid, or unsafe local input prevents an external-AI Turn. Configuration that only the app-server can validate may start and initialize the process but cannot start a Thread or Turn after incompatibility is established.

#### Scenario: Configuration is unsafe [error]

- **GIVEN** local assessment input is incompatible or attempts to relax the read-only boundary
- **WHEN** the Host requests an assessment
- **THEN** no external-AI Turn begins

#### Scenario: Codex app-server is incompatible [error]

- **GIVEN** the configured executable starts an app-server that is not Codex 0.149.1 or does not implement the accepted protocol
- **WHEN** the Host requests an assessment
- **THEN** the SDK returns no completed Turn and starts no Codex Thread
