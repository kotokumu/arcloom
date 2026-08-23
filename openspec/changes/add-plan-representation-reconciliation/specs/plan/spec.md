## MODIFIED Requirements

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
