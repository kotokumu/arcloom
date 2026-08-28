## MODIFIED Requirements

### Requirement: GPCD-1 Explicit GitHub target and representation

A GitHub creation planner MUST accept a dry-run only for one valid Plan, one valid GitHub Repository Target, and one explicitly selected GitHub Plan Representation.

- **入力と受理**: The Repository target and representation satisfy their Conceptual Model definitions. Representation is never inferred from Plan content.
- **振る舞いの規則**: Accepted Repository segments retain their exact values.
- **失敗の扱い**: An absent or invalid target or representation produces a stable validation result and no Creation Request Plan. Local validation makes no claim about remote Repository naming, existence, access, or permissions.
- **参照**: [related] `plan` Conceptual Model for Plan validity and Plan-name line breaks.

#### Scenario: Milestone representation selected

- **GIVEN** a valid Plan and GitHub Repository Target with Milestone explicitly selected
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a Milestone Creation Request Plan

#### Scenario: Issue representation selected

- **GIVEN** a valid Plan and GitHub Repository Target with Issue explicitly selected
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns an Issue Creation Request Plan

#### Scenario: Representation is absent

- **GIVEN** a valid Plan and GitHub Repository Target but no supported representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation input as invalid

### Requirement: GPCD-2 Versioned Plan narrative

A GitHub creation planner MUST represent the applicable exact Plan meaning in a deterministic Versioned Plan Narrative compatible with persisted `arcloom-plan:v1` artifacts.

- **入力と受理**: The machine-readable block contains exactly the representation-specific Plan values defined by the Conceptual Model and begins the native content before the human narrative.
- **振る舞いの規則**: The block losslessly preserves exact values. The narrative presents the exact Goal and ordered Acceptance Conditions and, for Issue representation, the present Target Date, but remains non-authoritative for reconstruction. Equal Plan and representation inputs produce equal machine-readable blocks.
- **参照**: [related] `plan` Conceptual Model for Plan element identity.

#### Scenario: Milestone payload is reversible

- **GIVEN** a Plan and Milestone representation
- **WHEN** the planner produces the Versioned Plan Narrative
- **THEN** its leading version-one block recovers the exact Goal and ordered Acceptance Conditions without duplicating native Plan values

#### Scenario: Issue payload is reversible

- **GIVEN** a Plan and Issue representation
- **WHEN** the planner produces the Versioned Plan Narrative
- **THEN** its leading version-one block recovers the exact Goal, ordered Acceptance Conditions, and present or absent Target Date without duplicating native Plan values

#### Scenario: Narrative text is preserved

- **GIVEN** Plan Text contains multiple lines or Markdown-sensitive content
- **WHEN** the planner produces the Versioned Plan Narrative
- **THEN** the machine-readable block recovers the exact input and the human narrative presents the unchanged values in declared order

#### Scenario: Same Plan is represented twice

- **GIVEN** the same Plan and GitHub Plan Representation are supplied twice
- **WHEN** the planner produces both Creation Request Plans
- **THEN** both contain the same machine-readable block

### Requirement: GPCD-3 Milestone creation request plan

A GitHub creation planner MUST describe Milestone representation with the ordered Planned Requests and dependencies defined by this Requirement.

- **入力と受理**: The Plan and Milestone representation satisfy GPCD-1.
- **振る舞いの規則**: The Creation Request Plan starts with one create-Milestone request whose title is the Plan name, narrative is the Versioned Plan Narrative, and optional target date is the Plan Target Date. It then contains one create-Issue request per Task in declared order; each Task Issue title is the Task name, its body is absent, and its Milestone input references the planned Milestone result.

#### Scenario: Milestone representation without Tasks

- **GIVEN** a valid Plan with no Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains only the create-Milestone request

#### Scenario: Milestone representation with Tasks

- **GIVEN** a valid Plan with two ordered Tasks and Milestone representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains one create-Milestone request followed by two create-Issue requests in declared Task order
- **AND** each Issue request references the Milestone request result

### Requirement: GPCD-4 Issue creation request plan

A GitHub creation planner MUST describe Issue representation with the ordered Planned Requests and dependencies defined by this Requirement for a Plan containing at most 100 Tasks.

- **入力と受理**: A Plan with zero through 100 Tasks is supported. A larger Plan is unsupported for Issue representation.
- **振る舞いの規則**: The Creation Request Plan starts with one create-Issue request whose title is the Plan name and whose body is the Versioned Plan Narrative. For each Task in declared order it then contains one Task create-Issue request followed by one add-Sub-issue request. A Task Issue has the Task name as title and no body or Milestone input. Each relationship references the parent Issue number and corresponding Task Issue identity as distinct result kinds.
- **失敗の扱い**: More than 100 Tasks produces a stable unsupported-representation result and no Creation Request Plan.

#### Scenario: Issue representation without Tasks

- **GIVEN** a valid Plan with no Tasks and Issue representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains only the parent create-Issue request

#### Scenario: Issue representation with Tasks

- **GIVEN** a valid Plan with two ordered Tasks and Issue representation
- **WHEN** the planner produces a Creation Request Plan
- **THEN** the plan contains the parent create-Issue request and two adjacent Task-Issue and add-Sub-issue request pairs in declared Task order
- **AND** each relationship request distinguishes the parent Issue number from the corresponding Task Issue identity

#### Scenario: Issue representation reaches its Task limit

- **GIVEN** a valid Plan with 100 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns a valid Creation Request Plan containing all 100 Task Issues and relationships

#### Scenario: Issue representation exceeds its Task limit

- **GIVEN** a valid Plan with 101 Tasks and Issue representation
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the representation as unsupported

### Requirement: GPCD-5 Request-plan integrity

A Creation Request Plan consumer MUST receive a deterministic, dependency-ordered, and immutable plan whose Symbolic Result References satisfy the Conceptual Model.

- **振る舞いの規則**: Equal accepted inputs produce equal Planned Requests, values, order, and references. Every reference identifies an earlier request in the same plan and its exact required result kind. No Provider-assigned Milestone number, Issue number, or Issue identity is invented.
- **排他・冪等**: Changing a consumer-owned copy of returned plan data does not affect later reads.
- **失敗の扱い**: The public contract does not admit a Creation Request Plan containing a dangling, forward, cross-plan, or result-kind-incompatible reference.

#### Scenario: Same input is planned twice

- **GIVEN** the same valid Plan, GitHub Repository Target, and representation
- **WHEN** a planner produces two Creation Request Plans
- **THEN** both plans contain equal requests, values, ordering, and Symbolic Result References

#### Scenario: Number and ID remain distinct

- **GIVEN** an Issue Creation Request Plan contains a Task
- **WHEN** a consumer inspects its relationship request
- **THEN** the parent Issue number reference and child Issue identity reference remain distinct result kinds

#### Scenario: Caller mutates returned collections

- **GIVEN** a consumer has read a collection from a Creation Request Plan
- **WHEN** the consumer changes its local collection value
- **THEN** a subsequent read exposes the original plan unchanged

### Requirement: GPCD-6 Dry-run boundary and GitHub compatibility

A GitHub creation planner MUST produce only a passive Creation Request Plan with the declared GitHub.com compatibility and without claiming or changing external state.

- **振る舞いの規則**: The plan identifies REST compatibility version `2022-11-28` and contains the operation inputs and result dependencies required by the selected representation. GitHub Enterprise Server is outside this capability.
- **副作用**: Planning performs no network access, external mutation, authorization decision, observation, persistence, or Provider-assigned identifier generation.
- **失敗の扱い**: The plan makes no claim about Repository existence, access, permissions, conflicts, Provider acceptance, or operation success.

#### Scenario: Request plan is produced

- **GIVEN** valid dry-run inputs
- **WHEN** the planner successfully produces a Creation Request Plan
- **THEN** no GitHub resource or other external fact has been accessed, created, or changed

#### Scenario: External operation would fail

- **GIVEN** a passive Creation Request Plan describes operations GitHub would reject
- **WHEN** a consumer inspects the plan
- **THEN** the plan makes no claim that those operations can or will succeed

#### Scenario: Host inspects compatibility

- **GIVEN** a Host has a Creation Request Plan
- **WHEN** it inspects the plan contract
- **THEN** it can identify the declared GitHub compatibility version, required operation inputs, and result dependencies

### Requirement: GPCD-7 Validation result

A GitHub creation planner MUST return no Creation Request Plan and identify the affected input when dry-run input is invalid or unsupported.

- **入力と受理**: Plan, GitHub Repository Target, and GitHub Plan Representation are validated under their owning rules.
- **失敗の扱い**: The stable validation result distinguishes an invalid Plan, invalid Repository target, and unsupported representation. No partial Creation Request Plan is returned.

#### Scenario: Invalid Plan input

- **GIVEN** an invalid Plan is supplied
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the Plan input as invalid

#### Scenario: Invalid Repository input

- **GIVEN** an invalid GitHub Repository Target is supplied
- **WHEN** the planner requests a creation dry-run
- **THEN** it returns no Creation Request Plan and identifies the Repository input as invalid
