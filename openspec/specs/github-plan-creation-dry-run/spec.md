## Purpose

Defines how Arcloom produces an inspectable, dependency-aware plan of GitHub creation requests for a provider-independent Plan without sending requests or claiming that external state changed.

## Conceptual Model

### GitHub Repository Target

A GitHub Repository Target identifies a GitHub.com repository locally without asserting that it exists or is accessible. It consists of exact owner and repository-name segments. Each segment is valid UTF-8, contains at least one code point outside Unicode `White_Space`, excludes `/` and the line-break code points prohibited for a Plan name, and is otherwise preserved exactly. GitHub remote naming rules are not local validity rules.

### GitHub Plan Representation

A GitHub Plan Representation is exactly Milestone or Issue. It is selected explicitly and is never inferred from Plan content.

### Versioned Plan Narrative

A Versioned Plan Narrative begins with a machine-readable `arcloom-plan:v1` block and is followed by a human-readable narrative. The block losslessly preserves the exact Plan values assigned to it. For Milestone representation these are Goal and ordered Acceptance Conditions. For Issue representation they are Goal, ordered Acceptance Conditions, and present or absent Target Date. Plan name and Tasks are represented natively in both representations, and a Milestone Target Date is also native. The human narrative presents Plan meaning but is never a reconstruction source.

### Creation Request Plan

A Creation Request Plan is a passive, deterministic, immutable description of GitHub creation operations. It contains one GitHub Repository Target, one GitHub Plan Representation, the declared GitHub.com compatibility version, ordered Planned Requests, and their dependencies. It neither contains Provider-assigned identifiers nor asserts that an operation can or will succeed.

A Planned Request is one create-Milestone, create-Issue, or add-Sub-issue operation with its exact Plan-derived inputs. A Symbolic Result Reference identifies a required result kind from an earlier Planned Request in the same Creation Request Plan. It cannot be dangling, forward, cross-plan, or result-kind incompatible.

## Requirements

### Requirement: explicit-github-target-and-representation

A GitHub creation planner MUST accept a dry-run only for one valid Plan, one valid GitHub Repository Target, and one explicitly selected GitHub Plan Representation.

- **Input and Acceptance**: The Repository target and representation satisfy their Conceptual Model definitions. Representation is never inferred from Plan content.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Milestone representation | Plan and Repository Target are valid | Milestone is explicitly selected | Milestone Creation Request Plan | None |
  | Issue representation | Plan and Repository Target are valid | Issue is explicitly selected | Issue Creation Request Plan | None |
  | Representation absent or unsupported | Plan and Repository Target are valid | No supported representation is selected | Stable invalid-representation result and no Creation Request Plan | None |

- **Invariants**: Accepted Repository segments retain their exact values, and representation is never inferred from Plan content.
- **Failure Handling**: An absent or invalid target or representation produces a stable validation result and no Creation Request Plan. Local validation makes no claim about remote Repository naming, existence, access, or permissions.
- **References**: [related] `plan` Conceptual Model for Plan validity and Plan-name line breaks.

#### Scenario: Milestone representation selected [happy]

- **GIVEN** a valid Plan and GitHub Repository Target with Milestone explicitly selected
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a Milestone Creation Request Plan

#### Scenario: Representation is absent [error]

- **GIVEN** a valid Plan and GitHub Repository Target but no supported representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation input as invalid

### Requirement: versioned-plan-narrative

A GitHub creation planner MUST represent the applicable exact Plan meaning in a deterministic Versioned Plan Narrative compatible with persisted `arcloom-plan:v1` artifacts.

- **Input and Acceptance**: The machine-readable block contains exactly the representation-specific Plan values defined by the Conceptual Model and begins the native content before the human narrative.
- **Behavioral Rules**: The block losslessly preserves exact values. The narrative presents the exact Goal and ordered Acceptance Conditions and, for Issue representation, the present Target Date, but remains non-authoritative for reconstruction. Equal Plan and representation inputs produce equal machine-readable blocks.
- **References**: [related] `plan` Conceptual Model for Plan element identity.

#### Scenario: Issue payload is reversible [happy]

- **GIVEN** a Plan and Issue representation
- **WHEN** the planner produces the Versioned Plan Narrative
- **THEN** its leading version-one block recovers the exact Goal, ordered Acceptance Conditions, and present or absent Target Date without duplicating native Plan values

#### Scenario: Narrative text is preserved [happy]

- **GIVEN** Plan Text contains multiple lines or Markdown-sensitive content
- **WHEN** the planner produces the Versioned Plan Narrative
- **THEN** the machine-readable block recovers the exact input and the human narrative presents the unchanged values in declared order

#### Scenario: Same Plan is represented twice [idempotency]

- **GIVEN** the same Plan and GitHub Plan Representation are supplied twice
- **WHEN** the planner produces both Creation Request Plans
- **THEN** both contain the same machine-readable block

### Requirement: milestone-creation-request-plan

A GitHub creation planner MUST describe Milestone representation with the ordered Planned Requests and dependencies defined by this Requirement.

- **Input and Acceptance**: The Plan and Milestone representation satisfy [[github-plan-creation-dry-run/explicit-github-target-and-representation]].
- **Behavioral Rules**: The Creation Request Plan starts with one create-Milestone request whose title is the Plan name, narrative is the Versioned Plan Narrative, and optional target date is the Plan Target Date. It then contains one create-Issue request per Task in declared order; each Task Issue title is the Task name, its body is absent, and its Milestone input references the planned Milestone result.

#### Scenario: Milestone representation without Tasks [boundary]

- **GIVEN** a valid Plan with no Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains only the create-Milestone request

#### Scenario: Milestone representation with Tasks [happy]

- **GIVEN** a valid Plan with two ordered Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains one create-Milestone request followed by two create-Issue requests in declared Task order, and each Issue request references the Milestone request result

### Requirement: issue-creation-request-plan

A GitHub creation planner MUST describe Issue representation with the ordered Planned Requests and dependencies defined by this Requirement for a Plan containing at most 100 Tasks.

- **Input and Acceptance**:

  | Partition | Condition or range | Acceptance or result |
  |---|---|---|
  | Empty through maximum | Task count is 0 through 100 inclusive | Accept Issue representation. |
  | Above maximum | Task count is greater than 100 | Reject as unsupported representation and return no Creation Request Plan. |
- **Behavioral Rules**: The Creation Request Plan starts with one create-Issue request whose title is the Plan name and whose body is the Versioned Plan Narrative. For each Task in declared order it then contains one Task create-Issue request followed by one add-Sub-issue request. A Task Issue has the Task name as title and no body or Milestone input. Each relationship references the parent Issue number and corresponding Task Issue identity as distinct result kinds.
- **Failure Handling**: More than 100 Tasks produces a stable unsupported-representation result and no Creation Request Plan.

#### Scenario: Issue representation with Tasks [happy]

- **GIVEN** a valid Plan with two ordered Tasks and Issue representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains the parent create-Issue request and two adjacent Task-Issue and add-Sub-issue request pairs in declared Task order, and each relationship request distinguishes the parent Issue number from the corresponding Task Issue identity

#### Scenario: Issue representation reaches its Task limit [boundary]

- **GIVEN** a valid Plan with 100 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a valid Creation Request Plan containing all 100 Task Issues and relationships

#### Scenario: Issue representation exceeds its Task limit [boundary]

- **GIVEN** a valid Plan with 101 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation as unsupported

### Requirement: request-plan-integrity

A Creation Request Plan consumer MUST receive a deterministic, dependency-ordered, and immutable plan whose Symbolic Result References satisfy the Conceptual Model.

- **Behavioral Rules**: Equal accepted inputs produce equal Planned Requests, values, order, and references. Every reference identifies an earlier request in the same plan and its exact required result kind. No Provider-assigned Milestone number, Issue number, or Issue identity is invented.
- **Concurrency and Idempotency**: Changing a consumer-owned copy of returned plan data does not affect later reads.
- **Failure Handling**: The public contract does not admit a Creation Request Plan containing a dangling, forward, cross-plan, or result-kind-incompatible reference.

#### Scenario: Same input is planned twice [idempotency]

- **GIVEN** the same valid Plan, GitHub Repository Target, and representation
- **WHEN** a planner produces two Creation Request Plans
- **THEN** both plans contain equal requests, values, ordering, and Symbolic Result References

#### Scenario: Number and ID remain distinct [happy]

- **GIVEN** an Issue Creation Request Plan contains a Task
- **WHEN** a consumer inspects its relationship request
- **THEN** the parent Issue number reference and child Issue identity reference remain distinct result kinds

#### Scenario: Caller mutates returned collections [idempotency]

- **GIVEN** a consumer has read a collection from a Creation Request Plan
- **WHEN** the consumer changes its local collection value
- **THEN** a subsequent read exposes the original plan unchanged

### Requirement: dry-run-boundary-and-github-compatibility

A GitHub creation planner MUST produce only a passive Creation Request Plan with the declared GitHub.com compatibility and without claiming or changing external state.

- **Behavioral Rules**: The plan identifies REST compatibility version `2022-11-28` and contains the operation inputs and result dependencies required by the selected representation. GitHub Enterprise Server is outside this capability.
- **Side Effects**: Planning performs no network access, external mutation, authorization decision, observation, persistence, or Provider-assigned identifier generation.
- **Failure Handling**: The plan makes no claim about Repository existence, access, permissions, conflicts, Provider acceptance, or operation success.

#### Scenario: Request plan is produced [happy]

- **GIVEN** valid dry-run inputs
- **WHEN** the planner successfully produces a Creation Request Plan
- **THEN** no GitHub resource or other external fact has been accessed, created, or changed

#### Scenario: Host inspects compatibility [compatibility]

- **GIVEN** a Host has a Creation Request Plan
- **WHEN** it inspects the plan contract
- **THEN** it can identify the declared GitHub compatibility version, required operation inputs, and result dependencies

### Requirement: validation-result

A GitHub creation planner MUST return no Creation Request Plan and identify the affected input when dry-run input is invalid or unsupported.

- **Input and Acceptance**: Plan, GitHub Repository Target, and GitHub Plan Representation are validated under their owning rules.
- **Failure Handling**: The stable validation result distinguishes an invalid Plan, invalid Repository target, and unsupported representation. No partial Creation Request Plan is returned.

#### Scenario: Invalid Plan input [error]

- **GIVEN** an invalid Plan is supplied
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the Plan input as invalid
