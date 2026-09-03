## MODIFIED Requirements

### Requirement: explicit-github-target-and-representation

A GitHub creation planner MUST accept a dry-run only for one valid and representable Plan, one valid GitHub Repository Target, and one explicitly selected GitHub Plan Representation.

- **Input and Acceptance**: The Repository target and representation satisfy their Conceptual Model definitions. The Plan satisfies the `plan` capability and the selected representation's native-narrative acceptance rules. Representation is never inferred from Plan content.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Milestone representation | Plan and Repository Target are valid and representable | Milestone is explicitly selected | Milestone Creation Request Plan | None |
  | Issue representation | Plan and Repository Target are valid and representable | Issue is explicitly selected | Issue Creation Request Plan | None |
  | Representation absent or unsupported | Plan and Repository Target are valid | No supported representation is selected | Stable invalid-representation result and no Creation Request Plan | None |
  | Plan is not natively representable | Plan and Repository Target are valid | Exact Plan Text collides with the selected representation's structural lines | Stable unsupported-representation result identifying the representation input and no Creation Request Plan | None |

- **Invariants**: Accepted Repository segments retain their exact values, Plan validity remains provider-independent, and representation is never inferred from Plan content.
- **Failure Handling**: An absent or invalid target or representation produces a stable validation result and no Creation Request Plan. A valid but unrepresentable Plan produces `UnsupportedRepresentation` for `RepresentationField` and no Creation Request Plan. Local validation makes no claim about remote Repository naming, field limits, existence, access, permissions, or acceptance.
- **References**: [related] `plan` Conceptual Model for Plan validity and exact Plan Text.

#### Scenario: Milestone representation selected [happy]

- **GIVEN** a valid natively representable Plan and GitHub Repository Target with Milestone explicitly selected
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a Milestone Creation Request Plan

#### Scenario: Representation is absent [error]

- **GIVEN** a valid Plan and GitHub Repository Target but no supported representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation input as invalid

#### Scenario: Plan Text collides with native structure [error]

- **GIVEN** a valid Plan whose exact Goal or Acceptance Condition contains a reserved structural line for the selected GitHub representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation input as unsupported

## ADDED Requirements

### Requirement: github-plan-narrative

A GitHub creation planner MUST represent every accepted Plan through one deterministic, human-readable GitHub Plan Narrative without Arcloom-specific metadata or another opaque reconstruction source.

- **Input and Acceptance**:

  | Partition | Condition or range | Acceptance or result |
  |---|---|---|
  | Natively representable Plan Text | No Goal or Acceptance Condition contains a full LF-delimited line equal to `## Goal`, `## Acceptance Conditions`, or `## Target Date`, and no Acceptance Condition contains a full LF-delimited line beginning `### ` | Accept and preserve every exact Plan Text byte without normalization. |
  | Structurally colliding Plan Text | At least one Goal or Acceptance Condition line equals a reserved H2 line, or an Acceptance Condition line begins `### ` | Reject the selected representation as unsupported and return no Creation Request Plan. |

- **Behavioral Rules**:

  | Representation state | Exact GitHub Plan Narrative |
  |---|---|
  | Milestone or undated Issue with conditions `C₁` through `Cₙ` | `"## Goal\n\n" + G + "\n\n## Acceptance Conditions" + Σ("\n\n### " + decimal(i) + "\n\n" + Cᵢ) + "\n"` for contiguous `i` from 1 through `n` |
  | Dated Issue with conditions `C₁` through `Cₙ` and date `D` | The undated form through `Cₙ`, followed by `"\n## Target Date\n\n" + D + "\n"` |

  `G`, every `Cᵢ`, and `D` are the exact Plan values. Structural bytes use LF. The document has exactly one final framing LF. Creation emits no preamble, HTML comment, encoded block, duplicate Plan value, or field whose purpose is opaque reconstruction.
- **Invariants**: The accepted narrative recovers the exact Goal, ordered Acceptance Conditions, and applicable Issue Target Date under `[[github-plan-representation-observation/native-narrative-meaning]]`. Plan Text equality remains exact byte equality under the `plan` capability.
- **Concurrency and Idempotency**: Equal accepted Plan and representation inputs produce byte-equal narratives.
- **Failure Handling**: A structural-line collision produces the unsupported-representation result defined by `[[github-plan-creation-dry-run/explicit-github-target-and-representation]]` and no partial Creation Request Plan.
- **References**: [related] `plan` Conceptual Model for exact Plan Text, ordered Acceptance Conditions, and canonical Target Date; [related] `[[github-plan-representation-observation/native-narrative-meaning]]` for native observation.

#### Scenario: Canonical Milestone narrative [happy]

- **GIVEN** a representable Plan with Goal `Goal` and two Acceptance Conditions `First` and `Second`
- **WHEN** the planner produces a Milestone Creation Request Plan
- **THEN** the description is exactly `## Goal\n\nGoal\n\n## Acceptance Conditions\n\n### 1\n\nFirst\n\n### 2\n\nSecond\n`

#### Scenario: Canonical dated Issue narrative [happy]

- **GIVEN** the same Plan has Target Date `2028-02-29` and Issue representation is selected
- **WHEN** the planner produces an Issue Creation Request Plan
- **THEN** the body appends exactly `\n## Target Date\n\n2028-02-29\n` after the final Acceptance Condition framing

#### Scenario: Exact value bytes are preserved [boundary]

- **GIVEN** representable Goal and Acceptance Condition values contain Unicode, CRLF, leading whitespace, trailing whitespace, and additional LF bytes
- **WHEN** the planner produces the GitHub Plan Narrative
- **THEN** every value byte is preserved exactly and only the declared structural bytes are added

#### Scenario: Acceptance Condition collides with an ordinal marker [error]

- **GIVEN** an Acceptance Condition contains a full line beginning `### `
- **WHEN** the planner requests either GitHub representation
- **THEN** the selected representation is unsupported and no Creation Request Plan is returned

#### Scenario: Same Plan is represented twice [idempotency]

- **GIVEN** the same accepted Plan and GitHub Plan Representation are supplied twice
- **WHEN** the planner produces both Creation Request Plans
- **THEN** both contain the same exact GitHub Plan Narrative

## MODIFIED Requirements

### Requirement: milestone-creation-request-plan

A GitHub creation planner MUST describe Milestone representation with the ordered Planned Requests and dependencies defined by this Requirement.

- **Input and Acceptance**: The Plan and Milestone representation satisfy [[github-plan-creation-dry-run/explicit-github-target-and-representation]].
- **Behavioral Rules**: The Creation Request Plan starts with one create-Milestone request whose title is the Plan name, description is the GitHub Plan Narrative, and optional target date is the Plan Target Date. It then contains one create-Issue request per Task in declared order; each Task Issue title is the Task name, its body is absent, and its Milestone input references the planned Milestone result.

#### Scenario: Milestone representation without Tasks [boundary]

- **GIVEN** a valid representable Plan with no Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains only the create-Milestone request and its description contains the native narrative without metadata

#### Scenario: Milestone representation with Tasks [happy]

- **GIVEN** a valid representable Plan with two ordered Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains one create-Milestone request followed by two create-Issue requests in declared Task order, and each Issue request references the Milestone request result

### Requirement: issue-creation-request-plan

A GitHub creation planner MUST describe Issue representation with the ordered Planned Requests and dependencies defined by this Requirement for a representable Plan containing at most 100 Tasks.

- **Input and Acceptance**:

  | Partition | Condition or range | Acceptance or result |
  |---|---|---|
  | Empty through maximum | Task count is 0 through 100 inclusive and Plan Text is natively representable | Accept Issue representation. |
  | Above maximum | Task count is greater than 100 | Reject as unsupported representation and return no Creation Request Plan. |
  | Structural collision | Plan Text is not natively representable | Reject as unsupported representation and return no Creation Request Plan. |

- **Behavioral Rules**: The Creation Request Plan starts with one create-Issue request whose title is the Plan name and whose body is the GitHub Plan Narrative, including the Target Date section only when the Plan Target Date is present. For each Task in declared order it then contains one Task create-Issue request followed by one add-Sub-issue request. A Task Issue has the Task name as title and no body or Milestone input. Each relationship references the parent Issue number and corresponding Task Issue identity as distinct result kinds.
- **Failure Handling**: More than 100 Tasks or structurally colliding Plan Text produces a stable unsupported-representation result and no Creation Request Plan.

#### Scenario: Issue representation with Tasks [happy]

- **GIVEN** a valid representable Plan with two ordered Tasks and Issue representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains the parent create-Issue request with its native narrative and two adjacent Task-Issue and add-Sub-issue request pairs in declared Task order

#### Scenario: Issue representation reaches its Task limit [boundary]

- **GIVEN** a valid representable Plan with 100 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a valid Creation Request Plan containing all 100 Task Issues and relationships

#### Scenario: Issue representation exceeds its Task limit [boundary]

- **GIVEN** a valid representable Plan with 101 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation as unsupported

### Requirement: validation-result

A GitHub creation planner MUST return no Creation Request Plan and identify the affected input when dry-run input is invalid or the selected GitHub representation is unsupported.

- **Input and Acceptance**: Plan, GitHub Repository Target, and GitHub Plan Representation are validated under their owning rules. Native representability belongs to GitHub Plan Representation and does not change Plan validity.
- **Failure Handling**: The stable validation result distinguishes an invalid Plan, invalid Repository target, invalid representation, and unsupported representation. Structural-line collision and the Issue Task-count limit both identify `UnsupportedRepresentation` for `RepresentationField`. No partial Creation Request Plan is returned. No precedence among simultaneous independent invalid inputs is guaranteed.

#### Scenario: Invalid Plan input [error]

- **GIVEN** an invalid Plan is supplied
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the Plan input as invalid

#### Scenario: Valid Plan is not natively representable [error]

- **GIVEN** a valid Plan contains a reserved structural-line collision
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies `UnsupportedRepresentation` for `RepresentationField`

## REMOVED Requirements

### Requirement: versioned-plan-narrative

- **Reason**: GitHub Plan content no longer stores or recovers Plan meaning through `arcloom-plan:v1` or another opaque payload.
- **Migration**: Use `[[github-plan-creation-dry-run/github-plan-narrative]]`. Existing stored metadata is not mutated and has no authority during native observation.
