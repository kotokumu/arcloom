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

### Requirement: GPCD-2 Canonical Plan narrative
The dry-run SHALL encode the Goal and acceptance conditions as deterministic Markdown in the Plan resource's description or body. It SHALL neither escape nor trim inserted text. The exact byte concatenation SHALL be `"## Goal\n\n" + goal + "\n\n## Acceptance Conditions"`, followed for each declared condition at one-based index `n` by `"\n\n### " + decimal(n) + "\n\n" + condition`. The Issue representation SHALL then append `"\n\n## Target Date\n\n" + targetDate` when the date exists. Both representations SHALL finally append `"\n"`. An inserted value's own trailing line feeds remain in addition to these fixed delimiters.

```text
## Goal

{goal}

## Acceptance Conditions

### 1

{first acceptance condition}
```

For two conditions, the complete structural form before the optional target-date delimiter and final line feed SHALL be:

```text
## Goal

{goal}

## Acceptance Conditions

### 1

{first acceptance condition}

### 2

{second acceptance condition}
```

For the Issue representation, an existing target date SHALL append the following fixed delimiter and value:

```text

## Target Date

{YYYY-MM-DD}
```

#### Scenario: Multiple acceptance conditions
- **WHEN** a Plan contains multiple acceptance conditions
- **THEN** the narrative contains each preserved statement once in declared order under consecutively numbered headings

#### Scenario: Preserved multi-line text
- **WHEN** a Goal or acceptance condition contains CRLF, a Markdown heading, leading whitespace, or a trailing LF
- **THEN** the narrative equals the fixed delimiters plus the unchanged input bytes and does not remove or reinterpret the inserted text

#### Scenario: Issue representation has a target date
- **WHEN** the Issue representation is selected for a Plan with a target date
- **THEN** the parent Issue body includes the canonical target-date section

### Requirement: GPCD-3 Milestone creation request plan
For the Milestone representation, the request plan SHALL contain one create-Milestone request followed by one create-Issue request for each Task in declared order. The Milestone title SHALL equal the Plan name, its description SHALL equal the canonical Plan narrative, and an existing target date SHALL be represented as `YYYY-MM-DDT00:00:00Z` in `due_on`. Each Task Issue title SHALL equal the Task name, its body SHALL be absent, and its Milestone input SHALL reference the created Milestone's `number`.

#### Scenario: Milestone representation without Tasks
- **WHEN** a Plan has no Tasks
- **THEN** the request plan contains only the create-Milestone request

#### Scenario: Milestone representation with Tasks
- **WHEN** a Plan contains two Tasks
- **THEN** the request plan contains one create-Milestone request followed by two create-Issue requests in declared Task order
- **AND** each Issue request contains a typed reference to the Milestone request's `number` result

### Requirement: GPCD-4 Issue creation request plan
For the Issue representation, the request plan SHALL support at most 100 Tasks and SHALL reject a larger Plan for this representation without returning a request plan. For a supported Plan, the request plan SHALL contain one create-Issue request for the Plan followed, for each Task in declared order, by one create-Issue request for the Task and one add-sub-issue request. The parent Issue title SHALL equal the Plan name and its body SHALL equal the canonical Plan narrative. Each Task Issue title SHALL equal the Task name; its body and Milestone input SHALL be absent. Each add-sub-issue request SHALL reference the parent Issue's `number` and the corresponding Task Issue's `id`.

#### Scenario: Issue representation without Tasks
- **WHEN** a Plan has no Tasks
- **THEN** the request plan contains only the parent create-Issue request

#### Scenario: Issue representation with Tasks
- **WHEN** a Plan contains two Tasks
- **THEN** the request plan contains the parent create-Issue request and two adjacent Task-Issue and add-sub-issue request pairs in declared Task order
- **AND** each relationship request contains typed references to the parent Issue `number` and corresponding Task Issue `id`

#### Scenario: Issue representation reaches its Task limit
- **WHEN** a Plan contains 100 Tasks
- **THEN** the Issue representation returns a valid request plan containing all 100 Task Issues and relationships

#### Scenario: Issue representation exceeds its Task limit
- **WHEN** a Plan contains 101 Tasks
- **THEN** the Issue representation fails with an unsupported-representation result and returns no request plan

### Requirement: GPCD-5 Request-plan integrity
A request plan SHALL be deterministic, topologically ordered, and immutable from a consumer's perspective. Every returned symbolic result reference SHALL identify an earlier request in the same plan and the exact result kind required by its input. A dry-run SHALL NOT invent a GitHub-assigned Milestone number, Issue number, or Issue ID. Callers SHALL NOT be able to construct a dangling, forward, cross-plan, or result-kind-incompatible reference through the public contract.

#### Scenario: Same input is planned twice
- **WHEN** the same valid Plan, Repository target, and representation are supplied twice
- **THEN** the two request plans contain equal requests, values, ordering, and symbolic references

#### Scenario: Number and ID remain distinct
- **WHEN** the Issue representation contains a Task
- **THEN** the add-sub-issue request distinguishes the parent Issue `number` reference from the child Issue `id` reference

#### Scenario: Caller mutates returned collections
- **WHEN** a caller changes a collection returned by a request-plan accessor
- **THEN** a subsequent read of the request plan is unchanged

### Requirement: GPCD-6 Dry-run boundary and GitHub contract
Producing a request plan SHALL perform no network request, external mutation, authorization decision, observation, persistence, or Provider-assigned identifier generation. The result SHALL target GitHub.com REST API version `2022-11-28` and the create-Milestone, create-Issue, and add-sub-issue contracts. The corresponding operations SHALL represent `POST /repos/{owner}/{repo}/milestones`, `POST /repos/{owner}/{repo}/issues`, and `POST /repos/{owner}/{repo}/issues/{parent_number}/sub_issues`. The result SHALL conform structurally to those known client inputs but SHALL NOT assert Repository existence, permission sufficiency, server acceptance, absence of conflicts, or eventual creation success. GitHub Enterprise Server and reverse reconstruction of a Plan from Markdown SHALL remain outside this contract.

#### Scenario: Request plan is produced
- **WHEN** a valid creation dry-run succeeds
- **THEN** no GitHub resource or other external fact is created or changed

#### Scenario: GitHub would reject a request
- **WHEN** a structurally valid request plan targets a Repository that is absent, inaccessible, or rejects an operation
- **THEN** the dry-run remains a non-executed plan and makes no success claim about the external operation

#### Scenario: API contract is inspected
- **WHEN** a Host inspects a returned request plan
- **THEN** it can identify the fixed API version and each request's operation-specific input, including Milestone `number`, parent Issue `number`, and child Issue `id` result semantics

### Requirement: GPCD-7 Validation result
A failed dry-run constructor SHALL return a zero request plan and a validation error with a stable category and affected input field. A zero Plan SHALL produce an invalid-Plan result without inventing or retaining a prior Plan-construction error. Unsupported Issue representation caused by more than 100 Tasks SHALL be distinguishable from invalid local input. No partial request plan SHALL be returned.

#### Scenario: Invalid Plan input
- **WHEN** an invalid or zero Plan is supplied
- **THEN** the constructor returns a zero request plan and an invalid-Plan result identifying the Plan field

#### Scenario: Invalid Repository input
- **WHEN** an invalid or zero Repository target is supplied
- **THEN** the constructor returns a zero request plan and identifies the Repository field
