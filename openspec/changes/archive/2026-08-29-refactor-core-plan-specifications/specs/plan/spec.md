## MODIFIED Requirements

### Requirement: PLN-1 Plan vocabulary

A Plan consumer MUST be able to use the provider-independent Plan composition defined by this capability.

- **入力と受理**: A Plan accepts one name, one Goal, one or more Acceptance Conditions, zero or more Tasks, and an optional Target Date.
- **振る舞いの規則**: The accepted values retain the composition, collection order, and provider independence defined by the Plan Conceptual Model.
- **失敗の扱い**: Input that violates the Plan composition produces a Plan Validation Violation and no valid Plan.

#### Scenario: Minimum Plan

- **GIVEN** a consumer supplies a valid name, Goal, and one Acceptance Condition
- **WHEN** the consumer forms a Plan without a Task or Target Date
- **THEN** the Plan is valid

#### Scenario: Milestone is not a Plan element

- **GIVEN** a Provider-native Milestone represents a Plan
- **WHEN** a consumer interprets the represented Plan
- **THEN** the Milestone remains external representation and is not a Plan element

### Requirement: PLN-2 Plan text

A Plan consumer MUST receive exact preservation of valid Plan Text and rejection of invalid Plan Text.

- **入力と受理**: Plan Text is accepted exactly when it satisfies the Plan Text definition in the Conceptual Model.
- **振る舞いの規則**: Accepted text retains its exact UTF-8 value, including permitted leading or trailing whitespace and original Goal or Acceptance Condition line endings.
- **失敗の扱い**: Invalid text produces a Plan Validation Violation identifying the affected element.

#### Scenario: Unicode Plan

- **GIVEN** a consumer supplies valid non-ASCII Plan Text
- **WHEN** the text becomes part of a Plan
- **THEN** the Plan preserves the exact text

#### Scenario: Blank text

- **GIVEN** a required Plan Text value is empty or contains only Unicode `White_Space`
- **WHEN** a consumer submits the value
- **THEN** no valid value is returned and the Plan Validation Violation identifies the affected element

#### Scenario: Invalid UTF-8

- **GIVEN** a required Plan Text value contains an invalid UTF-8 byte sequence
- **WHEN** a consumer submits the value
- **THEN** no valid value is returned and the Plan Validation Violation identifies the affected element

#### Scenario: Multi-line title

- **GIVEN** a Plan name or Task name contains CR, LF, NEL, line separator, or paragraph separator
- **WHEN** a consumer submits the name
- **THEN** no valid value is returned and the Plan Validation Violation identifies the affected element

#### Scenario: Preserved surrounding whitespace

- **GIVEN** valid Plan Text contains leading or trailing whitespace and at least one non-whitespace code point
- **WHEN** the text becomes part of a Plan
- **THEN** the Plan preserves that whitespace exactly

### Requirement: PLN-3 Collection invariants

A Plan consumer MUST receive ordered, exact-identity Plan Collections whose members satisfy Plan validity.

- **入力と受理**: Acceptance Condition and Task collections accept individually valid members; independently validated empty collections are valid, while a complete Plan still requires at least one Acceptance Condition.
- **振る舞いの規則**: Member identity is exact preserved text, duplicate identities are rejected, and declared order is preserved.
- **失敗の扱い**: An invalid or duplicate member produces the same stable Plan Validation Violation whether the collection is validated independently or as part of a Plan. A collection-member violation includes its input index. No ordering among simultaneous independent violations is guaranteed.

#### Scenario: Duplicate acceptance condition

- **GIVEN** two Acceptance Conditions in one Plan have the same exact statement
- **WHEN** a consumer submits the Plan
- **THEN** no valid Plan is returned and the violation identifies a duplicate Acceptance Condition

#### Scenario: Duplicate Task name

- **GIVEN** two Tasks in one Plan have the same exact name
- **WHEN** a consumer submits the Plan
- **THEN** no valid Plan is returned and the violation identifies a duplicate Task

#### Scenario: Similar Unicode text

- **GIVEN** two collection members differ by case or Unicode code-point sequence
- **WHEN** their identities are compared
- **THEN** they remain distinct and are not normalized for uniqueness

#### Scenario: Standalone collection uniqueness validation

- **GIVEN** a consumer has valid Acceptance Condition or Task values outside a complete Plan
- **WHEN** the consumer validates their collection
- **THEN** exact duplicates produce the same duplicate violation as Plan validation and an empty collection satisfies the collection rule

#### Scenario: Standalone collection contains a zero element

- **GIVEN** an independently validated collection contains an invalid Acceptance Condition or Task at a known input index
- **WHEN** the consumer validates the collection
- **THEN** the result contains the same invalid-element category and index as validation of the collection within a Plan

### Requirement: PLN-4 Target date

A Plan consumer MUST receive a Target Date only when it satisfies the Target Date definition in the Conceptual Model.

- **入力と受理**: A present value is canonical `YYYY-MM-DD` text representing a valid date from `0001-01-01` through `9999-12-31`.
- **振る舞いの規則**: The accepted value retains its canonical text and has no time, time-zone, whitespace, or Provider-specific deadline meaning.
- **失敗の扱い**: A non-canonical or invalid date produces an invalid-target-date Plan Validation Violation and no valid Target Date.

#### Scenario: Leap-day target

- **GIVEN** the value is `2028-02-29`
- **WHEN** a consumer submits it as a Target Date
- **THEN** it is accepted with canonical text `2028-02-29`

#### Scenario: Invalid calendar date

- **GIVEN** the value is an impossible date such as `2027-02-29`
- **WHEN** a consumer submits it as a Target Date
- **THEN** no valid Target Date is returned and the result contains an invalid-target-date violation

#### Scenario: Invalid lexical form

- **GIVEN** a value contains a non-zero-padded month, surrounding whitespace, a time, a time zone, year `0000`, or more than ten characters
- **WHEN** a consumer submits it as a Target Date
- **THEN** no valid Target Date is returned and the result contains an invalid-target-date violation

### Requirement: PLN-5 Validation result and Plan validity

A Plan consumer MUST be able to distinguish valid Plan values from invalid input through the Plan validity and Plan Validation Violation concepts.

- **入力と受理**: Complete Plan input and independently validatable Plan values use the same applicable validity rules.
- **振る舞いの規則**: A valid Plan reports valid without requiring the consumer to reconstruct its invariants. A Plan Validation Violation exposes its stable category, affected element kind, and applicable collection index.
- **失敗の扱い**: Failed validation returns no valid value. When independent violations coexist, one applicable violation is returned without guaranteeing which is selected first.

#### Scenario: Invalid Task in a collection

- **GIVEN** a Task at a known input index violates Plan validity
- **WHEN** a consumer submits the containing Plan
- **THEN** no valid Plan is returned and the violation identifies the Task and its index

#### Scenario: Multiple independent violations

- **GIVEN** submitted Plan input contains more than one independent violation
- **WHEN** a consumer validates the input
- **THEN** one applicable Plan Validation Violation is returned without a promised ordering among the violations

#### Scenario: Constructed Plan validity

- **GIVEN** a Plan satisfies every Plan invariant
- **WHEN** a consumer asks whether it is valid
- **THEN** the Plan reports valid

#### Scenario: Zero Plan validity

- **GIVEN** a Plan value does not satisfy the Plan invariants
- **WHEN** a consumer asks whether it is valid
- **THEN** it reports invalid without requiring the consumer to reconstruct those invariants

#### Scenario: Standalone Plan-name validation

- **GIVEN** a consumer has a candidate Plan name outside a complete Plan
- **WHEN** the consumer validates that name
- **THEN** the same names are accepted and the same invalid-text or multiline-name violation is returned as during complete Plan validation
