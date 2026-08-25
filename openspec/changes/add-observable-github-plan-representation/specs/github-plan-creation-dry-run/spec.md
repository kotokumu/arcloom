## MODIFIED Requirements

### Requirement: GPCD-2 Versioned Plan narrative
The dry-run SHALL place a version-one machine-readable Plan block at the beginning of the Milestone description or parent Issue body, before the human-readable narrative. The version-one block SHALL remain compatible with persisted `arcloom-plan:v1` artifacts and SHALL losslessly represent its Plan values. A Milestone payload SHALL contain the exact Goal and ordered acceptance conditions. An Issue payload SHALL additionally contain the present or absent target date. The payload SHALL omit Plan name and Tasks because GitHub represents them natively, and the Milestone payload SHALL omit the natively represented target date. The same Plan and representation SHALL produce the same payload. The narrative SHALL present the exact Goal and acceptance conditions in declared order and, for an Issue, the target date when present; it SHALL NOT be a reconstruction source.

#### Scenario: Milestone payload is reversible
- **WHEN** a Milestone representation is planned
- **THEN** its leading version-one payload recovers the exact Goal and ordered acceptance conditions without duplicating native Plan values

#### Scenario: Issue payload is reversible
- **WHEN** an Issue representation is planned
- **THEN** its leading version-one payload recovers the exact Goal, ordered acceptance conditions, and present or absent target date without duplicating native Plan values

#### Scenario: Narrative text is preserved
- **WHEN** Plan text contains multiple lines or Markdown-sensitive content
- **THEN** the payload recovers the exact input and the narrative presents the unchanged values in declared order

#### Scenario: Same Plan is represented twice
- **WHEN** the same Plan and representation are supplied twice
- **THEN** both request plans contain the same machine-readable payload

### Requirement: GPCD-6 Dry-run boundary and GitHub compatibility
Producing a creation request plan SHALL perform no network access, external mutation, authorization decision, observation, persistence, or Provider-assigned identifier generation. The returned passive request plan SHALL identify GitHub.com REST compatibility version `2022-11-28` and contain the operation inputs and result dependencies needed for the selected Milestone or Issue representation. It SHALL NOT claim Repository existence, access, permission sufficiency, absence of conflicts, Provider acceptance, or creation success. GitHub Enterprise Server SHALL remain outside this capability.

#### Scenario: Request plan is produced
- **WHEN** a valid creation dry-run succeeds
- **THEN** no GitHub resource or other external fact is accessed, created, or changed

#### Scenario: External operation would fail
- **WHEN** the passive request plan describes operations that GitHub would reject
- **THEN** the dry-run makes no claim that the operations can or will succeed

#### Scenario: Host inspects compatibility
- **WHEN** a Host inspects a returned request plan
- **THEN** it can identify the declared GitHub compatibility version, required operation inputs, and result dependencies
