## Purpose

Defines how Arcloom produces an inspectable, dependency-aware plan of GitHub creation requests for a provider-independent Plan without sending requests or claiming that external state changed.

## Requirements

### Requirement: GPCD-1 Explicit GitHub target and representation
A creation dry-run SHALL require a locally valid GitHub Repository target and exactly one explicit representation: Milestone or Issue. It SHALL NOT infer the representation from Plan content. Each Repository segment SHALL be valid UTF-8, contain at least one code point outside the Unicode `White_Space` property recognized by the repository's Go toolchain, contain no `/` or line-break code point defined by PLN-2, and otherwise be preserved exactly. Local validation SHALL NOT enforce GitHub's remote naming rules; existence, access, and permissions SHALL remain unverified.

#### Scenario: Milestone representation selected
- **WHEN** a valid Plan, Repository target, and Milestone representation are supplied
- **THEN** the dry-run returns a Milestone creation request plan

#### Scenario: Issue representation selected
- **WHEN** a valid Plan, Repository target, and Issue representation are supplied
- **THEN** the dry-run returns an Issue creation request plan

#### Scenario: Representation is absent
- **WHEN** no supported representation is supplied
- **THEN** the dry-run fails without returning a request plan

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

### Requirement: GPCD-3 Milestone creation request plan
For the Milestone representation, the request plan SHALL contain one create-Milestone request followed by one create-Issue request for each Task in declared order. The Milestone title SHALL equal the Plan name, its description SHALL equal the versioned Plan narrative, and an existing target date SHALL be represented as the Milestone target date. Each Task Issue title SHALL equal the Task name, its body SHALL be absent, and its Milestone input SHALL reference the created Milestone.

#### Scenario: Milestone representation without Tasks
- **WHEN** a Plan has no Tasks
- **THEN** the request plan contains only the create-Milestone request

#### Scenario: Milestone representation with Tasks
- **WHEN** a Plan contains two Tasks
- **THEN** the request plan contains one create-Milestone request followed by two create-Issue requests in declared Task order
- **AND** each Issue request references the Milestone request result

### Requirement: GPCD-4 Issue creation request plan
For the Issue representation, the request plan SHALL support at most 100 Tasks and SHALL reject a larger Plan for this representation without returning a request plan. For a supported Plan, the request plan SHALL contain one create-Issue request for the Plan followed, for each Task in declared order, by one create-Issue request for the Task and one add-sub-issue request. The parent Issue title SHALL equal the Plan name and its body SHALL equal the versioned Plan narrative. Each Task Issue title SHALL equal the Task name; its body and Milestone input SHALL be absent. Each add-sub-issue request SHALL reference the parent Issue and corresponding Task Issue using their required distinct result kinds.

#### Scenario: Issue representation without Tasks
- **WHEN** a Plan has no Tasks
- **THEN** the request plan contains only the parent create-Issue request

#### Scenario: Issue representation with Tasks
- **WHEN** a Plan contains two Tasks
- **THEN** the request plan contains the parent create-Issue request and two adjacent Task-Issue and add-sub-issue request pairs in declared Task order
- **AND** each relationship request distinguishes the parent Issue number from the corresponding Task Issue identity

#### Scenario: Issue representation reaches its Task limit
- **WHEN** a Plan contains 100 Tasks
- **THEN** the Issue representation returns a valid request plan containing all 100 Task Issues and relationships

#### Scenario: Issue representation exceeds its Task limit
- **WHEN** a Plan contains 101 Tasks
- **THEN** the Issue representation fails with an unsupported-representation result and returns no request plan

### Requirement: GPCD-5 Request-plan integrity
A request plan SHALL be deterministic, dependency-ordered, and immutable from a consumer's perspective. Every returned symbolic result reference SHALL identify an earlier request in the same plan and the exact result kind required by its input. A dry-run SHALL NOT invent a GitHub-assigned Milestone number, Issue number, or Issue ID. Callers SHALL NOT be able to construct a dangling, forward, cross-plan, or result-kind-incompatible reference through the public contract.

#### Scenario: Same input is planned twice
- **WHEN** the same valid Plan, Repository target, and representation are supplied twice
- **THEN** the two request plans contain equal requests, values, ordering, and symbolic references

#### Scenario: Number and ID remain distinct
- **WHEN** the Issue representation contains a Task
- **THEN** the relationship request distinguishes the parent Issue number reference from the child Issue identity reference

#### Scenario: Caller mutates returned collections
- **WHEN** a caller changes a collection returned by a request-plan accessor
- **THEN** a subsequent read of the request plan is unchanged

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

### Requirement: GPCD-7 Validation result
A failed dry-run construction SHALL return no request plan and a stable validation result identifying the affected input. An invalid Plan SHALL remain distinguishable from an unsupported selected representation. No partial request plan SHALL be returned.

#### Scenario: Invalid Plan input
- **WHEN** an invalid Plan is supplied
- **THEN** construction returns no request plan and identifies the Plan input as invalid

#### Scenario: Invalid Repository input
- **WHEN** an invalid Repository target is supplied
- **THEN** construction returns no request plan and identifies the Repository input
