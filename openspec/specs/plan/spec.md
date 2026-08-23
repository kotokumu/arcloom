## Purpose

Defines the provider-independent meaning and minimum structural validity of a Plan that Arcloom can use without owning an external planning system's facts.

## Requirements

### Requirement: PLN-1 Plan vocabulary
A Plan SHALL contain one name, one Goal, one or more acceptance conditions, zero or more Tasks, and an optional target date. A Milestone SHALL NOT be a Plan element. Each Plan element SHALL remain independent of Provider identifiers, resource types, and API fields.

#### Scenario: Minimum Plan
- **WHEN** a Plan has a valid name, Goal, and one acceptance condition
- **THEN** it is structurally valid without a Task or target date

#### Scenario: Milestone is not a Plan element
- **WHEN** a consumer describes how a Plan is represented by a Provider-native Milestone
- **THEN** that Milestone belongs to the external representation and does not become part of the Plan

### Requirement: PLN-2 Plan text
Plan names, Goals, acceptance conditions, and Task names SHALL be valid UTF-8 and SHALL contain at least one code point outside the Unicode `White_Space` property recognized by the repository's Go toolchain. Plan names and Task names SHALL NOT contain CR (`U+000D`), LF (`U+000A`), NEL (`U+0085`), line separator (`U+2028`), or paragraph separator (`U+2029`). Valid text, including leading or trailing whitespace and original line endings in Goals and acceptance conditions, SHALL be preserved without case folding, whitespace rewriting, or Unicode normalization.

#### Scenario: Unicode Plan
- **WHEN** a Plan contains valid non-ASCII text
- **THEN** the Plan preserves the exact text

#### Scenario: Blank text
- **WHEN** a required text value is empty or consists only of Unicode whitespace
- **THEN** Plan construction fails with a validation result identifying the invalid element

#### Scenario: Invalid UTF-8
- **WHEN** a required text value contains an invalid UTF-8 byte sequence
- **THEN** Plan construction fails with a validation result identifying the invalid element

#### Scenario: Multi-line title
- **WHEN** a Plan name or Task name contains CR, LF, NEL, line separator, or paragraph separator
- **THEN** Plan construction fails with a validation result identifying the invalid element

#### Scenario: Preserved surrounding whitespace
- **WHEN** valid text contains leading or trailing whitespace and at least one non-whitespace code point
- **THEN** the Plan preserves that whitespace exactly

### Requirement: PLN-3 Collection invariants
Acceptance-condition statements and Task names SHALL each be unique by exact preserved text within one Plan. Their declared order SHALL be preserved. A zero-value or otherwise invalid Plan SHALL NOT cross a public package boundary as a valid Plan. Acceptance-condition and Task collection validation SHALL be reusable independently of Plan aggregate construction for publicly constructible element values and their zero values. It SHALL return the same typed zero-element or duplicate validation result, including the affected input index, as Plan construction. Empty collections SHALL satisfy standalone collection validation; the requirement for at least one acceptance condition remains an aggregate Plan invariant. When standalone input contains simultaneous independent zero-element and duplicate violations, no ordering among the applicable validation results is part of the contract.

#### Scenario: Duplicate acceptance condition
- **WHEN** two acceptance conditions in one Plan have the same exact statement
- **THEN** Plan construction fails with a duplicate-acceptance-condition result

#### Scenario: Duplicate Task name
- **WHEN** two Tasks in one Plan have the same exact name
- **THEN** Plan construction fails with a duplicate-Task result

#### Scenario: Similar Unicode text
- **WHEN** two text values differ by case or Unicode code-point sequence
- **THEN** the values remain distinct and are not normalized for uniqueness

#### Scenario: Standalone collection uniqueness validation
- **WHEN** a consumer validates valid Plan acceptance-condition or Task values independently of Plan aggregate construction
- **THEN** exact duplicates return the same typed duplicate result as Plan construction and an empty collection passes the uniqueness rule

#### Scenario: Standalone collection contains a zero element
- **WHEN** standalone collection validation receives a zero acceptance condition or Task at a known input index
- **THEN** it returns the same typed invalid-element result and index as Plan construction

### Requirement: PLN-4 Target date
A target date SHALL be exactly ten ASCII characters in `YYYY-MM-DD` form and SHALL represent a valid proleptic Gregorian calendar date from `0001-01-01` through `9999-12-31`. It SHALL contain no whitespace, time, time zone, or Provider-specific deadline semantics.

#### Scenario: Leap-day target
- **WHEN** `2028-02-29` is provided as a target date
- **THEN** it is accepted and its canonical text is `2028-02-29`

#### Scenario: Invalid calendar date
- **WHEN** a date such as `2027-02-29` is provided
- **THEN** target-date construction fails with an invalid-date result

#### Scenario: Invalid lexical form
- **WHEN** a value contains a non-zero-padded month, surrounding whitespace, a time, a time zone, year `0000`, or more than ten characters
- **THEN** target-date construction fails with an invalid-date result

### Requirement: PLN-5 Validation result and Plan validity
A failed public Plan-value constructor SHALL return its zero result value and a validation error with a stable violation category, affected Plan element kind, and collection index when the affected element belongs to a collection. Consumers SHALL be able to identify the validation error with Go error inspection. No ordering among simultaneous independent violations is part of the contract. A Plan SHALL report whether it is valid without requiring a consumer to inspect or reconstruct its element invariants. A successfully constructed Plan SHALL report valid, and the zero Plan SHALL report invalid. Plan-name validation SHALL be reusable without constructing a Plan aggregate and SHALL return the same typed validation result and preserve the same validity semantics as Plan construction.

#### Scenario: Invalid Task in a collection
- **WHEN** a Task at a known input index is invalid
- **THEN** construction returns a zero result and a validation error identifying the Task element and its index

#### Scenario: Multiple independent violations
- **WHEN** an input contains more than one independent violation
- **THEN** construction returns one applicable typed validation error without promising which independent violation is reported first

#### Scenario: Constructed Plan validity
- **WHEN** a consumer asks a successfully constructed Plan whether it is valid
- **THEN** the Plan reports valid

#### Scenario: Zero Plan validity
- **WHEN** a consumer asks the zero Plan whether it is valid
- **THEN** the Plan reports invalid without requiring the consumer to reconstruct Plan rules

#### Scenario: Standalone Plan-name validation
- **WHEN** a consumer validates a Plan name without constructing a Plan aggregate
- **THEN** the same valid names are accepted and the same invalid-text or multiline-name validation result is returned as during Plan construction
