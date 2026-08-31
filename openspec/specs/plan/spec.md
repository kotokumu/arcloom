## Purpose

Defines the provider-independent meaning and minimum structural validity of a Plan that Arcloom can use without owning an external planning system's facts.

## Conceptual Model

### Plan composition

A Plan is provider-independent. It contains exactly one name, one Goal, one or more ordered Acceptance Conditions, zero or more ordered Tasks, and an optional Target Date. Provider resources, identifiers, and fields are not Plan elements.

### Plan Text

Plan names, Goals, Acceptance Conditions, and Task names are Plan Text. Plan Text is valid UTF-8 and contains at least one code point outside the Unicode `White_Space` property. Its exact value is preserved without case folding, whitespace rewriting, or Unicode normalization. Plan names and Task names are single-line and therefore exclude CR (`U+000D`), LF (`U+000A`), NEL (`U+0085`), line separator (`U+2028`), and paragraph separator (`U+2029`).

### Plan Collections

Acceptance Conditions and Tasks are ordered Plan Collections. Acceptance Condition identity is its exact statement. Task identity is its exact name. Each identity is unique within its collection, and declared order is part of the Plan.

### Target Date

A Target Date is either absent or one canonical calendar date. A present value is exactly ten ASCII characters in `YYYY-MM-DD` form and represents a valid proleptic Gregorian date from `0001-01-01` through `9999-12-31`. It carries no time, time zone, whitespace, or Provider-specific deadline meaning.

### Plan validity and violations

A Plan is valid exactly when its composition and every contained value satisfy the rules above. A Plan Validation Violation identifies a stable violation category, the affected Plan element kind, and the input index when a collection member is affected. No ordering among simultaneous independent violations is part of the Plan contract.

## Requirements

### Requirement: plan-vocabulary

A Plan consumer MUST be able to use the provider-independent Plan composition defined by this capability.

- **Input and Acceptance**: A Plan accepts one name, one Goal, one or more Acceptance Conditions, zero or more Tasks, and an optional Target Date.
- **Behavioral Rules**: The accepted values retain the composition, collection order, and provider independence defined by the Plan Conceptual Model.
- **Failure Handling**: Input that violates the Plan composition produces a Plan Validation Violation and no valid Plan.

#### Scenario: Minimum Plan [happy]

- **GIVEN** a consumer supplies a valid name, Goal, and one Acceptance Condition
- **WHEN** the consumer forms a Plan without a Task or Target Date
- **THEN** the Plan is valid

#### Scenario: Milestone is not a Plan element [compatibility]

- **GIVEN** a Provider-native Milestone represents a Plan
- **WHEN** a consumer interprets the represented Plan
- **THEN** the Milestone remains external representation and is not a Plan element

### Requirement: plan-text

A Plan consumer MUST receive exact preservation of valid Plan Text and rejection of invalid Plan Text.

- **Input and Acceptance**: Plan Text is accepted exactly when it satisfies the Plan Text definition in the Conceptual Model.
- **Behavioral Rules**: Accepted text retains its exact UTF-8 value, including permitted leading or trailing whitespace and original Goal or Acceptance Condition line endings.
- **Failure Handling**: Invalid text produces a Plan Validation Violation identifying the affected element.

#### Scenario: Unicode Plan [happy]

- **GIVEN** a consumer supplies valid non-ASCII Plan Text
- **WHEN** the text becomes part of a Plan
- **THEN** the Plan preserves the exact text

#### Scenario: Blank text [error]

- **GIVEN** a required Plan Text value is empty or contains only Unicode `White_Space`
- **WHEN** a consumer submits the value
- **THEN** no valid value is returned and the Plan Validation Violation identifies the affected element

#### Scenario: Multi-line title [error]

- **GIVEN** a Plan name or Task name contains CR, LF, NEL, line separator, or paragraph separator
- **WHEN** a consumer submits the name
- **THEN** no valid value is returned and the Plan Validation Violation identifies the affected element

#### Scenario: Preserved surrounding whitespace [boundary]

- **GIVEN** valid Plan Text contains leading or trailing whitespace and at least one non-whitespace code point
- **WHEN** the text becomes part of a Plan
- **THEN** the Plan preserves that whitespace exactly

### Requirement: collection-invariants

A Plan consumer MUST receive ordered, exact-identity Plan Collections whose members satisfy Plan validity.

- **Input and Acceptance**: Acceptance Condition and Task collections accept individually valid members; independently validated empty collections are valid, while a complete Plan still requires at least one Acceptance Condition.
- **Behavioral Rules**: Member identity is exact preserved text, duplicate identities are rejected, and declared order is preserved.
- **Failure Handling**: An invalid or duplicate member produces the same stable Plan Validation Violation whether the collection is validated independently or as part of a Plan. A collection-member violation includes its input index. No ordering among simultaneous independent violations is guaranteed.

#### Scenario: Duplicate acceptance condition [error]

- **GIVEN** two Acceptance Conditions in one Plan have the same exact statement
- **WHEN** a consumer submits the Plan
- **THEN** no valid Plan is returned and the violation identifies a duplicate Acceptance Condition

#### Scenario: Similar Unicode text [happy]

- **GIVEN** two collection members differ by case or Unicode code-point sequence
- **WHEN** their identities are compared
- **THEN** they remain distinct and are not normalized for uniqueness

### Requirement: target-date

A Plan consumer MUST receive a Target Date only when it satisfies the Target Date definition in the Conceptual Model.

- **Input and Acceptance**: A present value is canonical `YYYY-MM-DD` text representing a valid date from `0001-01-01` through `9999-12-31`.
- **Behavioral Rules**: The accepted value retains its canonical text and has no time, time-zone, whitespace, or Provider-specific deadline meaning.
- **Failure Handling**: A non-canonical or invalid date produces an invalid-target-date Plan Validation Violation and no valid Target Date.

#### Scenario: Leap-day target [happy]

- **GIVEN** the value is `2028-02-29`
- **WHEN** a consumer submits it as a Target Date
- **THEN** it is accepted with canonical text `2028-02-29`

#### Scenario: Invalid calendar date [error]

- **GIVEN** the value is an impossible date such as `2027-02-29`
- **WHEN** a consumer submits it as a Target Date
- **THEN** no valid Target Date is returned and the result contains an invalid-target-date violation

### Requirement: validation-result-and-plan-validity

A Plan consumer MUST be able to distinguish valid Plan values from invalid input through the Plan validity and Plan Validation Violation concepts.

- **Input and Acceptance**: Complete Plan input and independently validatable Plan values use the same applicable validity rules.
- **Behavioral Rules**:

  | Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
  |---|---|---|---|---|
  | Valid complete Plan | Every composition and element invariant holds | Consumer requests validity | Valid | None |
  | Invalid complete Plan candidate | At least one applicable invariant fails | Consumer validates submitted input | One applicable Plan Validation Violation and no valid Plan | None |
  | Independently validatable value | One Plan value or collection is supplied outside a complete Plan | Its owning rules are evaluated | The same success or violation category as complete Plan validation | None |
  | Multiple independent violations | More than one applicable rule fails | Validation is requested | One applicable violation without promised selection order | None |

- **Invariants**: A valid Plan reports valid without requiring the consumer to reconstruct its invariants. A Plan Validation Violation exposes its stable category, affected element kind, and applicable collection index.
- **Failure Handling**: Failed validation returns no valid value. When independent violations coexist, one applicable violation is returned without guaranteeing which is selected first.

#### Scenario: Invalid Task in a collection [error]

- **GIVEN** a Task at a known input index violates Plan validity
- **WHEN** a consumer submits the containing Plan
- **THEN** no valid Plan is returned and the violation identifies the Task and its index

#### Scenario: Multiple independent violations [boundary]

- **GIVEN** submitted Plan input contains more than one independent violation
- **WHEN** a consumer validates the input
- **THEN** one applicable Plan Validation Violation is returned without a promised ordering among the violations

#### Scenario: Valid Plan reports validity [happy]

- **GIVEN** a Plan satisfies every Plan invariant
- **WHEN** a consumer asks whether it is valid
- **THEN** the Plan reports valid
