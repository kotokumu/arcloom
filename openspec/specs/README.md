# Product Specification Rules

`openspec/specs/` is the SSOT for guarantees that consumers can observe and verify, and for the conceptual definitions required to interpret those guarantees. Do not place implementation structures, research notes, unresolved matters, or defect records here.

## 1. Artifact Responsibilities

| Artifact | Question it answers | What it must not contain |
|---|---|---|
| proposal | Why is the change needed, and which outcomes and scope does it cover? | Technical approaches or detailed guarantees |
| model | Which concepts, states, relationships, and constraints support the guarantees? | Unsettled norms or implementation design |
| spec | What does the subject guarantee? | Implementation methods, incidental details of the current implementation, or unresolved matters |
| design | How will the specification be realized? | Redefinition of requirements |
| tasks | What will be implemented and verified, and in what order? | New requirements or design decisions |

A main spec uses the following order. The Conceptual Model is an overview needed to read the specification, not a remnant of an analysis artifact. Do not omit capability-specific terminology, states, classifications, value ranges, relationships, units, identification rules, or invariants when they exist.

```text
Purpose -> Conceptual Model (when required) -> Requirements
```

Place an external SSOT specific to a Requirement in that Requirement's `References` block. Sources for the Conceptual Model may appear in `### References` within that section. Do not use an independent `## References` section that cannot be preserved during archival.

---

## 2. Capability Boundaries and Paths

Define a capability around a responsibility that is coherent from the consumer's perspective. Consumers include users, callers, external systems, other Components, and automated processes. Search existing capabilities before creating one.

| Subject | Criterion | Location |
|---|---|---|
| Observable function | Its consumer, trigger, and result can be explained | The capability that owns the guarantee |
| Reused product rule | Multiple capabilities depend on the same meaning | The capability that defines that meaning most strongly |
| Implementation-only structure | No contract exists on which a consumer depends | Design or code |

Do not create a capability solely because a technical layer, data structure, Framework, or shared utility exists.

A capability ID is the path relative to `openspec/specs/`, ending at the directory that contains `spec.md`. Every segment uses kebab-case. Nested paths are permitted.

```text
openspec/specs/platform/search/spec.md
-> platform/search
```

---

## 3. Conceptual Model

The Conceptual Model defines the concepts, states, classifications, relationships, and constraints required to interpret a group of Requirements. It is not a place to reproduce a glossary or physical data definition.

Define only concepts that satisfy at least one of these conditions:

- Multiple Requirements reference the concept.
- The concept has states, classifications, a value set, a unit, or an identification rule.
- A relationship or invariant between concepts changes the meaning of behavior.
- Synonymy or multiple meanings of the same term change the interpretation of a Requirement.
- The meaning cannot be read unambiguously without inferring it from Scenarios.

Do not turn tables, columns, DTOs, API payloads, Classes, or Packages into the Conceptual Model. Design owns the mapping between concepts and physical structures.

| Layer | Defines | Does not define |
|---|---|---|
| Conceptual Model | Meaning, identification, state space, classification values, relationships, structural invariants, value ranges, and units | Transition triggers, operational procedures, or side effects |
| Requirement | Applicability, input acceptance, decisions, state transitions, operational invariants, results, side effects, and failure guarantees | First definitions of concepts or classification values |
| Scenario | Observable results for concrete preconditions and actions | Norms absent from the Requirement |

Each concept has exactly one defining capability. The capability that most strongly determines its meaning, invariants, and lifecycle owns it. Other capabilities reference the Requirement ID or the Conceptual Model's References instead of redefining it.

---

## 4. Requirement

A Requirement ID is `<capability-path>/<requirement-slug>`. Write a cross-reference as `[[<capability-path>/<requirement-slug>]]`. The slug and every path segment use kebab-case.

```markdown
### Requirement: <stable-kebab-case-slug>

The subject MUST <core normative guarantee>.

- **Preconditions**: <applicable prior state, permissions, and required referenced objects>
- **Input and Acceptance**: <input, allowed range, defaults, and acceptance or rejection conditions>
- **Behavioral Rules**: <decisions, calculations, state transitions, and observable output>
- **Invariants**: <conditions preserved before and after every permitted result>
- **Side Effects**: <changes and non-changes to related state, history, and notifications>
- **Concurrency and Idempotency**: <concurrent execution, retries, duplicates, and atomicity>
- **Failure Handling**: <failure result observable by the consumer>
- **References**: <another SSOT on which the Requirement depends>

#### Scenario: <concrete behavior> [happy]

- **GIVEN** <consumer and prior state>
- **WHEN** <interaction or event>
- **THEN** <observable result>
```

The code above is an original example that demonstrates the writing format. Use only the blocks that are necessary and preserve the order defined in `_schema/requirement-structure.json`. Items in the same block use the same classification axis.

Separate Requirements when their guarantees change or can be verified independently. Do not separate them when only the entry point or technical path differs and the consumer observes the same guarantee.

Write a non-functional Requirement only when its subject, conditions, measurement method, threshold, and failure guarantee are settled. Do not make unquantified adjectives such as "fast," "secure," or "high-volume" normative by themselves.

---

## 5. Selecting a Specification Representation

Select a representation based on the structure of the rule. Do not expand every rule into prose or Scenarios. When one Requirement contains multiple structures, combine the representations it needs. Use concise normative prose for rules that have none of these structures.

| Rule structure | Representation | Normative location |
|---|---|---|
| Results differ by range in a continuous or ordered domain such as numbers, dates, versions, or counts | Partition Table | Acceptance rules in `Input and Acceptance`; result rules in `Behavioral Rules` |
| Results differ across reachable combinations of conditions | Decision Table | `Behavioral Rules` |
| A concept moves through lifecycle states based on triggers and guards | State Transition Table | `Behavioral Rules`; state names and meanings in the Conceptual Model |
| A condition must always be preserved | Invariant | Concept validity in the Conceptual Model; operational guarantees in the Requirement's `Invariants` block |
| A rule is illustrated or verified concretely | Scenario | After the complete Requirement |

A Partition Table uses the following columns. State whether each boundary is included or excluded, and leave no unintended gap or overlap. Define a value's meaning, unit, precision, or time basis in the Conceptual Model when it is not self-evident.

| Partition | Condition or range | Acceptance or result |
|---|---|---|

A Decision Table uses the following columns. Cover every reachable combination that produces a distinct guarantee and state any default result. Do not create combinations prohibited by the Conceptual Model.

| Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
|---|---|---|---|---|

A State Transition Table uses the following columns. Define each state's meaning once in the Conceptual Model. State whether a transition absent from the table is rejected, ignored, or outside the contract.

| Current state | Trigger or event | Guard | Next state | Output or Side Effects |
|---|---|---|---|---|

An Invariant is not a success path or input validation rule. A Requirement that changes state states which operation preserves the Invariant and what the consumer observes when it cannot be preserved.

---

## 6. Scenario

Each Requirement has at least one primary success Scenario. Add the following perspectives only when the guarantee changes.

| Tag | Subject |
|---|---|
| `happy` | Primary successful result |
| `error` | Input rejection, unmet precondition, or dependency failure |
| `boundary` | Empty, zero, upper limit, deadline boundary, complete set, or partial set |
| `permission` | Unauthenticated, insufficient permission, or outside the permitted scope |
| `concurrency` | Concurrent update, conflict, or reversed order |
| `idempotency` | Retry or duplicate |
| `compatibility` | Existing consumer, data, or contract compatibility |

GIVEN describes state that exists before execution, WHEN describes a consumer action or event, and THEN describes an externally observable result. A Scenario is a concrete example derived from a complete Requirement. It is not the sole location for a partition, decision rule, transition, or Invariant. Do not make an internal function call or write to a particular table the only expected result.

---

## 7. External Contracts and Implementation SSOTs

| Information | SSOT | Treatment in a spec |
|---|---|---|
| Product rules, state transitions, and invariants | Main spec | Write them in the Conceptual Model or a Requirement |
| API types, statuses, and error codes | OpenAPI or IDL | Reference them from the Requirement's `References` block |
| Data types, constraints, and indexes | Migration or schema | State only conceptual meaning and reference the physical definition |
| UI placement, Components, and visual states | Design system or UI artifact | State only operations and results observable by a consumer |
| Field contracts for files, messages, and events | Machine-readable schema | Reference the existing SSOT |
| Defects, implementation divergence, and research notes | Issue tracker or audit record | Do not write them in a spec |

When no machine-readable Interface SSOT exists, copy `openspec/templates/interface-contract.md` to a location managed outside OpenSpec and make that copy authoritative. The document is not an OpenSpec capability. Place observable guarantees provided through the Interface in a Requirement of the capability that owns them.

Select a reference type from `[related]`, `[interface]`, `[api]`, `[data]`, `[code]`, `[decision]`, `[policy]`, and `[external]`. A reference does not replace a norm; it connects the norm to another SSOT.

---

## 8. Delta and Publication

A delta spec may use only the following level-two sections:

- `## Purpose`: use only for a new capability.
- `## ADDED Requirements`
- `## MODIFIED Requirements`
- `## REMOVED Requirements`
- `## RENAMED Requirements`

`MODIFIED` contains the complete Requirement after the update. Custom sections are not applied to the main spec by the OpenSpec 1.10.0 archive process. Do not add an `## References` section or a Conceptual Model delta.

Write a Conceptual Model change as the complete replacement text in `Main Spec Conceptual Model Replacements` in `model.md`. Publish with:

```bash
node tools/archive-change.mjs <change-name>
```

The publication command rejects unknown delta sections, stages Conceptual Model replacements before archiving Requirement deltas, and strictly validates every published main spec. A direct `openspec archive` does not apply Conceptual Model replacements.

---

## 9. Completion Conditions

- The model contains no unresolved matter that changes the specification.
- Capability boundaries and concept ownership do not contradict existing main specs.
- The Conceptual Model contains no undefined material term, classification value, unit, or relationship.
- Each Requirement contains one independently changeable guarantee.
- The Conceptual Model or Requirement contains a normative representation for every continuous or ordered domain, combination of conditions, lifecycle, and continuously maintained condition.
- Each Requirement has a verifiable Scenario.
- No Scenario adds a norm absent from its Requirement or substitutes for a normative representation.
- Norms contain no implementation detail, incidental detail of the current implementation, defect, or unresolved matter.
- Related Requirements and other SSOTs are traceable from each Requirement's `References` block.
- `openspec validate <change-name> --strict` succeeds.
- All P0 and P1 findings from `REVIEW.md` are resolved.
- In the Quality Workflow, strict validation succeeds after publication through `archive-change.mjs`.
