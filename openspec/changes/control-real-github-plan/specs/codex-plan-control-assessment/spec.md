## Purpose

Obtain a read-only Codex-backed Plan Control judgment with provider-independent meaning and without retaining authoritative AI session state.

## ADDED Requirements

### Requirement: CPCA-1 Valid Plan Control judgment

An assessment SHALL yield exactly one valid provider-independent Plan Control judgment: Complete, Retain, Revise, or InsufficientInformation.

#### Scenario: Plan should be revised

- **WHEN** the judgment is Revise
- **THEN** the assessment includes one valid proposed Plan

#### Scenario: Plan should not be revised

- **WHEN** the judgment is Complete, Retain, or InsufficientInformation
- **THEN** the assessment does not include a proposed Plan

### Requirement: CPCA-2 Exact current assessment material

The assessment SHALL consider the exact current Plan and caller-owned observation material supplied for that invocation.

#### Scenario: Assessment material is supplied

- **WHEN** an assessment is requested
- **THEN** its judgment is associated only with that Plan and observation material

### Requirement: CPCA-3 Read-only assessment boundary

Assessment SHALL NOT modify the Plan, authorize a revision, request application, or mutate an external provider.

#### Scenario: Assessment proposes a revision

- **WHEN** the judgment is Revise
- **THEN** the proposed Plan remains only a proposal

### Requirement: CPCA-4 Invalid AI output is not a successful assessment

Missing, malformed, ambiguous, conflicting, or failed AI output SHALL NOT produce a successful Plan Control judgment.

#### Scenario: Output has conflicting meanings

- **WHEN** AI output cannot be interpreted as exactly one valid judgment
- **THEN** no successful assessment is returned

### Requirement: CPCA-5 Independent assessments

Each assessment SHALL be independent of prior assessments and SHALL retain no authoritative AI session state.

#### Scenario: A Plan is reassessed

- **WHEN** an assessment is requested after an earlier assessment
- **THEN** the new result is based on the current supplied material rather than retained prior session state

### Requirement: CPCA-6 Cancellation and concurrency

Assessment SHALL respond to caller cancellation within a bounded interval, and concurrent assessments SHALL not exchange Plans, observations, or judgments.

#### Scenario: Assessment is cancelled

- **WHEN** the caller cancels an in-progress assessment
- **THEN** no successful judgment is returned and the assessment terminates within the configured bound

#### Scenario: Assessments run concurrently

- **WHEN** distinct assessments run concurrently
- **THEN** each result refers only to its own Plan and observation material

### Requirement: CPCA-7 Compatible safe Host configuration

The Host SHALL supply a trusted compatible Codex installation and valid assessment inputs. The adapter SHALL always select the declared read-only assessment constraints and SHALL expose no configuration that relaxes them. Incompatible or unsafe local inputs SHALL prevent interaction.

#### Scenario: Configuration is unsafe

- **WHEN** local assessment input is incompatible with or attempts to relax the read-only assessment boundary
- **THEN** no AI interaction begins
