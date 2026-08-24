## MODIFIED Requirements

### Requirement: GPCD-2 Canonical Plan narrative
The dry-run SHALL encode a versioned machine-readable payload before the human-readable Plan narrative in the Milestone description or parent Issue body. The machine-readable block SHALL be exactly `<!-- arcloom-plan:v1\n` followed by an RFC 4648 base64url encoding without padding of a UTF-8 JSON object, followed by `\n-->\n\n`. The base64url text SHALL be canonical: it SHALL use only the URL-safe alphabet, SHALL contain no padding, and re-encoding the decoded bytes SHALL reproduce the same text. The JSON SHALL contain no insignificant whitespace and SHALL preserve every decoded Plan string exactly. Its object members SHALL occur in the declared order. A Milestone payload SHALL contain `goal` and `acceptance_conditions`. An Issue payload SHALL contain `goal`, `acceptance_conditions`, and `target_date`, whose value is the target-date string or `null`. JSON strings SHALL escape quotation mark and reverse solidus, SHALL use the short escapes for backspace, form feed, line feed, carriage return, and tab, SHALL encode every other `U+0000` through `U+001F` control as a lowercase four-hex-digit `\u00xx` escape, and SHALL emit every other Unicode scalar value as UTF-8 without escaping it. Solidus and HTML-sensitive characters SHALL NOT be escaped. The payload SHALL NOT duplicate the Plan name or Tasks because GitHub represents them as native titles. The human-readable portion SHALL remain presentation content and SHALL NOT be the source from which a Provider reconstructs Plan meaning.

After the machine-readable block, the dry-run SHALL encode the Goal and acceptance conditions as deterministic Markdown. It SHALL neither escape nor trim inserted text. The exact byte concatenation SHALL be `"## Goal\n\n" + goal + "\n\n## Acceptance Conditions"`, followed for each declared condition at one-based index `n` by `"\n\n### " + decimal(n) + "\n\n" + condition`. The Issue representation SHALL then append `"\n\n## Target Date\n\n" + targetDate` when the date exists. Both representations SHALL finally append `"\n"`. An inserted value's own trailing line feeds remain in addition to these fixed delimiters.

```text
<!-- arcloom-plan:v1
{base64url machine-readable payload}
-->

## Goal

{goal}

## Acceptance Conditions

### 1

{first acceptance condition}
```

For two conditions, the complete human-readable structural form before the optional target-date delimiter and final line feed SHALL be:

```text
## Goal

{goal}

## Acceptance Conditions

### 1

{first acceptance condition}

### 2

{second acceptance condition}
```

For the Issue representation, an existing target date SHALL append the following human-readable delimiter and value:

```text

## Target Date

{YYYY-MM-DD}
```

#### Scenario: Milestone machine-readable payload
- **WHEN** the Milestone representation is planned
- **THEN** its description starts with a version-one payload containing the exact Goal and ordered acceptance conditions and does not duplicate the Plan name, Tasks, or target date

#### Scenario: Issue machine-readable payload
- **WHEN** the Issue representation is planned
- **THEN** its parent Issue body starts with a version-one payload containing the exact Goal, ordered acceptance conditions, and present or absent target date and does not duplicate the Plan name or Tasks

#### Scenario: Multiple acceptance conditions
- **WHEN** a Plan contains multiple acceptance conditions
- **THEN** the machine-readable payload and human-readable narrative each contain every preserved statement once in declared order

#### Scenario: Preserved multi-line text
- **WHEN** a Goal or acceptance condition contains CRLF, a Markdown heading, leading whitespace, a trailing LF, or text resembling the machine-readable delimiter
- **THEN** decoding the payload recovers the exact input text and the human-readable narrative retains the unchanged input bytes

#### Scenario: Canonical JSON escaping
- **WHEN** a payload string contains control characters, quotation mark, reverse solidus, solidus, HTML-sensitive characters, or non-ASCII Unicode scalar values
- **THEN** the dry-run emits the one canonical JSON and base64url byte sequence defined by this requirement

#### Scenario: Issue representation has a target date
- **WHEN** the Issue representation is selected for a Plan with a target date
- **THEN** the parent Issue payload and human-readable narrative both contain that target date

### Requirement: GPCD-6 Dry-run boundary and GitHub contract
Producing a request plan SHALL perform no network request, external mutation, authorization decision, observation, persistence, or Provider-assigned identifier generation. The result SHALL target GitHub.com REST API version `2022-11-28` and the create-Milestone, create-Issue, and add-sub-issue contracts. The corresponding operations SHALL represent `POST /repos/{owner}/{repo}/milestones`, `POST /repos/{owner}/{repo}/issues`, and `POST /repos/{owner}/{repo}/issues/{parent_number}/sub_issues`. The result SHALL conform structurally to those known client inputs but SHALL NOT assert Repository existence, permission sufficiency, server acceptance, absence of conflicts, or eventual creation success. GitHub Enterprise Server SHALL remain outside this contract. Reverse reconstruction from the human-readable Markdown SHALL remain outside this contract; a separate observation capability MAY decode only the versioned machine-readable payload.

#### Scenario: Request plan is produced
- **WHEN** a valid creation dry-run succeeds
- **THEN** no GitHub resource or other external fact is created or changed

#### Scenario: GitHub would reject a request
- **WHEN** a structurally valid request plan targets a Repository that is absent, inaccessible, exceeds a Provider limit, or rejects an operation
- **THEN** the dry-run remains a non-executed plan and makes no success claim about the external operation

#### Scenario: API contract is inspected
- **WHEN** a Host inspects a returned request plan
- **THEN** it can identify the fixed API version and each request's operation-specific input, including Milestone `number`, parent Issue `number`, and child Issue `id` result semantics

#### Scenario: Machine-readable payload is inspected
- **WHEN** a Host inspects a Milestone description or parent Issue body in the request plan
- **THEN** it can identify a versioned payload that can recover every payload-backed Plan value without parsing the human-readable Markdown
