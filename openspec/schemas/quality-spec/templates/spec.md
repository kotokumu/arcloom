## Purpose

<!-- New capability only. Write at least 50 concrete characters. Delete this section for an existing capability delta. -->

## ADDED Requirements

### Requirement: <!-- stable-kebab-case-slug -->

The subject MUST <!-- core normative guarantee -->.

- **Preconditions**: <!-- prior state, permissions, and existence conditions required to apply the operation -->
- **Input and Acceptance**: <!-- input, permitted range, and rejection conditions -->
- **Behavioral Rules**: <!-- decisions, calculations, state transitions, and output -->
- **Invariants**: <!-- conditions preserved before and after every permitted result -->
- **Side Effects**: <!-- changes and non-changes to related state, history, and notifications -->
- **Concurrency and Idempotency**: <!-- concurrent execution, retries, and atomicity -->
- **Failure Handling**: <!-- failure result observable by a user or consumer -->
- **References**: <!-- optional: [interface]/[api]/[data]/[policy]/[external] SSOT path -->

<!--
Use these representations in the applicable block according to the rule's structure:
- Continuous or ordered domain: Partition Table (Partition | Condition or range | Acceptance or result)
- Combination of conditions: Decision Table (Rule | Preconditions or state | Input or event condition | Output or response | Side Effects)
- Lifecycle: State Transition Table (Current state | Trigger or event | Guard | Next state | Output or Side Effects)
- Continuously maintained operational guarantee: normative Invariant in `Invariants`
Use only applicable representations. Do not leave a normative table or Invariant only in model.md or a Scenario.
-->

#### Scenario: <!-- concrete behavior --> [happy]

- **GIVEN** <!-- actor and prior state -->
- **WHEN** <!-- actor action or external event -->
- **THEN** <!-- observable outcome -->

## MODIFIED Requirements

<!-- Copy the existing Requirement from its heading through every Scenario and write the complete updated content. -->

## REMOVED Requirements

### Requirement: <!-- removed-slug -->

- **Reason**: <!-- removal reason -->
- **Migration**: <!-- consumer/data migration or none -->

## RENAMED Requirements

- FROM: `### Requirement: <old-slug>`
- TO: `### Requirement: <new-slug>`
