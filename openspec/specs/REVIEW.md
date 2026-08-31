# Specification Review

This checklist verifies the semantic quality of a specification. `openspec validate` verifies OpenSpec syntax.

## 1. Priority

| Priority | Criterion | Approval condition |
|---|---|---|
| P0 | Incorrect norm, contradiction, unresolved matter made normative, SSOT boundary violation, or information lost during publication | Must be corrected |
| P1 | Omission or ambiguity that prevents reproducible implementation or verification | Correct unless a compelling reason is documented |
| P2 | Readability, maintainability, or risk of future misunderstanding | Correct or document the reason |

---

## 2. Problem and Scope

- Do the problem, expected outcomes, and success conditions in the proposal correspond?
- Do In Scope and Out of Scope define the boundary from the consumer's perspective?
- Does the proposal avoid fixing a technical approach as an outcome or Requirement?
- Can the reason for a new capability be explained as a difference from existing capabilities?
- Does every segment of a nested capability path use kebab-case?

---

## 3. Conceptual Model

- Are material subjects, states, classifications, values, units, and relationships defined before the Requirements that use them?
- Are identification and uniqueness explicit where required?
- Are the state space and transition conditions kept distinct?
- Is no concept redefined by multiple capabilities?
- Does the conceptual model avoid reproducing physical database, API payload, DTO, Class, or file structures?
- Are there no synonyms, multiple meanings of the same term, or unquantified adjectives?
- Does the model's `Unresolved Decisions` section contain no matter that changes the specification?
- When a main spec changes, does the model contain the complete replacement text?
- Are concepts and diagrams included because they are required, not merely because the template contains fields for them?

---

## 4. Requirement

- Does each Requirement contain one independently changeable and verifiable guarantee?
- Does the core norm include MUST and state its conditions and guarantee clearly?
- Do items in each block use the same classification axis?
- Are acceptance conditions, state transitions, outputs, side effects, and failure guarantees complete?
- Are concurrency, idempotency, permission, and compatibility contractual where relevant?
- Does each non-functional Requirement include its subject, conditions, measurement method, and threshold?
- Does the Requirement contain no implementation detail, incidental detail of the current implementation, defect, temporary note, or unresolved matter?
- Does every Requirement ID for a nested capability use the correct path?

---

## 5. Specification Representations

- When results differ in a continuous or ordered domain, is there a Partition Table with explicit boundaries and no gap or overlap?
- When results differ by combinations of conditions, is there a Decision Table covering every reachable distinct result?
- When lifecycle rules exist, is there a State Transition Table with current state, trigger, guard, next state, and result?
- Is every continuously maintained condition written as an Invariant and placed according to whether it defines concept validity or an operational guarantee?
- Are state meanings and value ranges in the Conceptual Model, and transitions and operational results in Requirements?
- Do normative tables and Invariants exist in the published main spec, not only in `model.md` or Scenarios?
- Has no representation been added without a corresponding rule structure?

---

## 6. Scenario

- Does each Requirement have a primary success case?
- Does it cover errors, boundaries, permissions, concurrency, idempotency, and compatibility that affect the guarantee?
- Does GIVEN describe the consumer and prior state, WHEN an interaction or event, and THEN an observable result?
- Can the input and expected result be constructed unambiguously from the Scenario?
- Does no Scenario introduce a norm or undefined term absent from the Requirement?
- Does each Scenario serve as a concrete example without needlessly repeating a normative table or Invariant?
- Has no perspective absent from the project been added merely to fill the template?

---

## 7. SSOT and Interfaces

- Does the spec avoid duplicating existing SSOTs for APIs, data, UI, messages, and external contracts?
- Are related Requirements and other SSOTs traceable from the Requirement's `References` block?
- Are Interface details kept distinct from observable guarantees provided through that Interface?
- When a machine-readable Interface is authoritative, does the main spec avoid reproducing its fields and types?
- Does design own the mapping between the Conceptual Model and implementation structures?

---

## 8. Workflow and Publication

- Are the proposal's capability and Requirement impacts reflected in delta specs?
- Can every difference between the model's Requirement Candidates and the actual Requirements be explained?
- Was design created after all delta specs, without redefining WHAT?
- Are the delta spec's level-two sections limited to Purpose and standard Requirement operations?
- Is each approved reader-facing Conceptual Model change already present in its owning main spec?
- Does every task identify a Requirement ID and verification method?
- Will publication use `openspec archive <change-name> --yes` followed by `npm run lint:openspec`?

---

## 9. Final Check

- [ ] There are no P0 or P1 findings.
- [ ] `openspec validate <change-name> --strict --no-interactive` succeeds.
- [ ] Concepts, terminology, and Requirement IDs are consistent between main and delta specs.
- [ ] Implementers and verifiers can begin without making additional specification decisions.
- [ ] `npm run lint:openspec` succeeds after publication.
