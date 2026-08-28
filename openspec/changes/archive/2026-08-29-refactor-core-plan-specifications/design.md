## Context

This change edits accepted specification structure without changing runtime behavior or architecture. The quality-spec publisher must apply each Conceptual Model replacement and its Requirement delta together. OpenSpec 1.10 also treats a Scenario heading change as removal plus addition during a MODIFIED Requirement, so existing Requirement and Scenario headings remain exact.

## Goals / Non-Goals

### Goals

- Publish each capability's Conceptual Model and complete MODIFIED Requirements atomically.
- Preserve every existing Requirement and Scenario identity while clarifying its observable contract.
- Detect behavior loss, implementation-detail leakage, and concurrent main-spec changes before publication completes.

### Non-Goals

- Change Go code, public interfaces, tests, architecture, or runtime behavior.
- Rename Requirement headings or add tags to Scenario headings.
- Modify active legacy-schema changes or archived historical artifacts.

## Conceptual-Model-to-Implementation Mapping

| Specification concept / Requirement | Owning component | Physical representation | Notes |
|---|---|---|---|
| Plan, Plan Text, Plan Collection, Target Date, Plan Validation Violation / PLN-1–PLN-5 | Existing ownership unchanged | `plan` main spec Conceptual Model and Requirements | No runtime representation changes. |
| Observation, Plan Location, Difference, Unavailable Information, Evidence, Reconciliation Determination / PRR-1–PRR-8 | Existing ownership unchanged | `plan-representation-reconciliation` main spec Conceptual Model and Requirements | No runtime representation changes. |

## Decisions

### Decision: Preserve existing Requirement and Scenario headings

- **Choice**: Keep PLN-1–PLN-5, PRR-1–PRR-8, and every current Scenario heading exact.
- **Rationale**: Active and archived artifacts use the legacy identifiers, and OpenSpec 1.10 rejects a MODIFIED Requirement that omits an existing Scenario name. Heading migration is independent from responsibility separation.
- **Alternatives**: Rename headings in the same change; rejected because it expands traceability updates and conflicts with active legacy-schema work. Add duplicate tagged Scenarios; rejected because it creates redundant specification text.
- **Consequences**: The two main specs gain the new content structure but retain legacy heading conventions until a later migration.

### Decision: Publish conceptual and behavioral text as one operation

- **Choice**: Use `node tools/archive-change.mjs refactor-core-plan-specifications` after all tasks and preconditions are satisfied.
- **Rationale**: Direct OpenSpec archive does not publish Conceptual Model replacements. The repository publisher locks main-spec publication, validates snapshots, applies both kinds of change, and rolls back only when its own published state remains current.
- **Alternatives**: Edit main specs during apply or run direct `openspec archive`; rejected because either can expose a partial specification.
- **Consequences**: Apply changes only planning artifacts. Main specs remain unchanged until archive.

### Decision: Verify equivalence from the accepted specification boundary

- **Choice**: Compare the legacy and delta Requirement and Scenario heading sets, inspect all removed physical terms, and run strict change and main-spec validation before publication.
- **Rationale**: No code change exists to verify. The relevant regression is loss or alteration of accepted guarantees during document restructuring.
- **Alternatives**: Rely only on OpenSpec syntax validation; rejected because it cannot identify conceptual ownership or implementation-detail leakage.
- **Consequences**: Verification combines automated identity checks with specification review against `openspec/specs/REVIEW.md`.

## Risks / Trade-offs

- A concurrent or newly created active delta targets either capability → Recheck active delta paths immediately before publication and stop on overlap.
- Moving rules into Conceptual Model accidentally weakens a Requirement → Require complete MODIFIED blocks, preserve all Scenario identities, and review each Scenario outcome against the accepted main spec.
- Removing Go terminology accidentally removes an observable failure guarantee → Preserve stable category, affected element, index, precedence, and no-valid-result outcomes while removing only their physical representation.
- Legacy headings remain inconsistent with the new authoring convention → Defer one coordinated heading migration until active legacy-schema changes are archived.

## Migration / Rollback

No runtime or data migration exists. Publication creates the new main-spec state only after strict validation. On validation failure, the repository publisher restores the prior main specs and returns the change to its active location when no concurrent edit conflicts with rollback.

## Open Questions

なし。
