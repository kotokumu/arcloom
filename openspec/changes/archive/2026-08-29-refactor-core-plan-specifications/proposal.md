## Why

The accepted `plan` and `plan-representation-reconciliation` specifications mix stable product meaning with Go-specific construction, package, and error-handling mechanisms. Their shared concepts are embedded in long Requirement paragraphs, so later specifications must infer the meaning of Plan values, observations, evidence, and reconciliation results from scattered normative text.

## Intended Outcomes

- Consumers can interpret the two capabilities from explicit, uniquely owned conceptual definitions.
- Requirements state observable guarantees without prescribing Go packages, constructors, zero values, accessors, Ports, or Controller implementation structure.
- Existing observable behavior, failure categories, evidence semantics, and Scenario outcomes remain unchanged.

## Success Criteria

- SC-1: Every non-trivial concept shared by multiple Requirements in the two capabilities has one definition in its owning main spec.
- SC-2: No normative statement depends on Go-specific package, constructor, zero-value, accessor, error-inspection, Port, or Controller terminology.
- SC-3: Every existing Scenario remains represented with the same precondition, interaction, and observable result.
- SC-4: No active OpenSpec change has a delta spec for either affected capability when publication begins.
- SC-5: The change and all main specs pass strict OpenSpec validation after publication.

## Scope

### In Scope

- Add a Conceptual Model to `plan` for Plan composition, text identity, collection invariants, target date, validity, and validation violations.
- Add a Conceptual Model to `plan-representation-reconciliation` for Observation, Plan Location, Difference, Unavailable Information, Evidence, and Reconciliation Determination.
- Rewrite the existing Requirements and Scenarios to reference those concepts and express only observable contracts.
- Preserve the existing Requirement headings during this change so unarchived artifacts retain valid references.

### Out of Scope

- Product behavior changes or implementation changes.
- Requirement ID migration to kebab-case slugs.
- Scenario heading tags; OpenSpec 1.10 treats adding a tag to a MODIFIED Scenario heading as a Scenario rename.
- Refactoring `github-plan-creation-dry-run`, `github-plan-representation-observation`, or `plan-control`.
- Refactoring the new capabilities currently defined by `control-real-github-plan`.

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `plan`: `PLN-1` through `PLN-5` — separate Plan concepts and observable guarantees from implementation mechanisms without changing their meaning.
- `plan-representation-reconciliation`: `PRR-1` through `PRR-8` — centralize observation and evidence semantics and remove physical boundary terminology without changing reconciliation outcomes.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Plan | `plan` | Define composition and provider independence once. |
| Plan Text | `plan` | Define identity, validity, and preservation once. |
| Plan Collection | `plan` | Define order, uniqueness, and minimum membership once. |
| Target Date | `plan` | Define format, range, and absence semantics once. |
| Plan Validation Violation | `plan` | Define the consumer-visible failure information independently of Go errors. |
| Observation | `plan-representation-reconciliation` | Define coherence, root state, and fact classifications once. |
| Plan Location | `plan-representation-reconciliation` | Define the provider-independent evidence location hierarchy once. |
| Difference | `plan-representation-reconciliation` | Define difference classifications and payload meaning once. |
| Unavailable Information | `plan-representation-reconciliation` | Define uncertainty separately from known differences. |
| Reconciliation Determination | `plan-representation-reconciliation` | Define derivation from Evidence once. |

## Decisions Required

なし。

## Impact

Specification readers and future OpenSpec changes receive clearer ownership and traceability. Runtime behavior, public interfaces, persisted data, external systems, `PRODUCT.md`, and `ARCHITECTURE.md` remain unchanged. Requirement headings remain stable in this change; their eventual migration is deferred until active legacy-schema changes no longer depend on them.
