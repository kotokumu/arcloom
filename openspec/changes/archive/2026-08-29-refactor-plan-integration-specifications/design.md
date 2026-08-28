## Context

This change restructures three accepted specifications without changing runtime behavior, architecture, or external contracts. The quality-spec publisher must apply each complete Conceptual Model replacement and its Requirement delta together. OpenSpec 1.10 also treats a Scenario heading change inside a MODIFIED Requirement as removal plus addition, so existing Requirement and Scenario headings remain exact.

## Goals / Non-Goals

### Goals

- Publish each capability's Conceptual Model and complete MODIFIED Requirements atomically.
- Preserve all 23 Requirement identities, all 68 Scenario identities, and their accepted observable outcomes.
- Detect implementation-detail leakage, behavior loss, active-delta conflicts, and concurrent main-spec changes before publication completes.

### Non-Goals

- Change Go code, public interfaces, tests, architecture, persisted `arcloom-plan:v1` data, GitHub compatibility, or runtime behavior.
- Rename Requirement headings or add tags to Scenario headings.
- Modify archived historical artifacts or the active `control-real-github-plan` change.

## Conceptual-Model-to-Implementation Mapping

| Specification concept / Requirement | Owning component | Physical representation | Notes |
|---|---|---|---|
| GitHub Repository Target, GitHub Plan Representation, Versioned Plan Narrative, Creation Request Plan, Planned Request, Symbolic Result Reference / GPCD-1–GPCD-7 | Existing ownership unchanged | `github-plan-creation-dry-run` main spec Conceptual Model and Requirements | No runtime representation or payload change. |
| GitHub Plan Target, Observed GitHub Fact, Task Collection Establishment, GitHub Observation Outcome / GHPO-1–GHPO-8 | Existing ownership unchanged | `github-plan-representation-observation` main spec Conceptual Model and Requirements | Existing `plan` and reconciliation concepts are referenced, not copied. |
| Plan Snapshot, Delivery Observation, Plan Control Assessment, Proposed Plan, Plan Control Failure / PLC-1–PLC-8 | Existing ownership unchanged | `plan-control` main spec Conceptual Model and Requirements | No AI boundary or assessment representation change. |

## Decisions

### Decision: Preserve existing Requirement and Scenario headings

- **Choice**: Keep GPCD-1–GPCD-7, GHPO-1–GHPO-8, PLC-1–PLC-8, and every current Scenario heading exact.
- **Rationale**: Archived artifacts use these identities, and OpenSpec 1.10 rejects a MODIFIED Requirement that omits an existing Scenario name.
- **Alternatives**: Rename headings and add Scenario tags in this change; rejected because identity migration is independent from separating product meaning and implementation mechanisms.
- **Consequences**: The main specs gain the new structure while legacy heading conventions remain for a later coordinated migration.

### Decision: Reuse core Plan and reconciliation concepts by reference

- **Choice**: Define provider-specific targets, fact sources, planning results, and control assessments in these three capabilities while referencing Plan, Plan Validation Violation, Observation, Plan Location, and Unavailable Information from their existing owners.
- **Rationale**: Repeating those definitions would create competing SSOTs and make later changes ambiguous.
- **Alternatives**: Copy the core concepts into every integration capability; rejected because identical terms could then evolve independently.
- **Consequences**: Requirements contain explicit related-spec references and rely on the accepted core Conceptual Models.

### Decision: Publish conceptual and behavioral text as one operation

- **Choice**: Use `node tools/archive-change.mjs refactor-plan-integration-specifications` after all tasks and preconditions pass.
- **Rationale**: Direct OpenSpec archive does not publish Conceptual Model replacements. The repository publisher locks publication, validates snapshots, applies Conceptual Model and Requirement changes, and fails closed on concurrent edits.
- **Alternatives**: Edit main specs during apply or run direct `openspec archive`; rejected because either can expose a partial specification.
- **Consequences**: Apply changes only planning artifacts. Main specs remain unchanged until archive.

### Decision: Verify equivalence at the accepted-spec boundary

- **Choice**: Compare accepted and delta Requirement and Scenario heading sets, review every accepted guarantee and outcome, scan normative text for physical mechanisms, and run strict validation before and after publication.
- **Rationale**: This change has no runtime implementation to test; the relevant regression is altered or lost specification meaning.
- **Alternatives**: Rely only on OpenSpec syntax validation; rejected because it cannot detect conceptual duplication, implementation leakage, or weakened observable guarantees.
- **Consequences**: Publication requires automated identity checks plus review under `openspec/specs/REVIEW.md`.

## Risks / Trade-offs

- A new active delta targets one of the three capabilities → Recheck every active change path immediately before publication and stop on overlap.
- Moving text into Conceptual Models weakens a Requirement → Publish complete MODIFIED blocks and compare all 68 Scenario outcomes with the accepted specs.
- External compatibility detail is removed as an implementation mechanism → Preserve `arcloom-plan:v1`, representation-specific meaning, GitHub.com scope, and compatibility version as observable external contracts.
- Shared core concepts are accidentally redefined → Keep provider-specific models narrow and reference the accepted `plan` and `plan-representation-reconciliation` Conceptual Models.
- Legacy headings remain inconsistent with current authoring conventions → Defer one coordinated identity migration until no active or historical validation constraint blocks it.

## Migration / Rollback

No runtime, data, or external-contract migration exists. Publication creates the new main-spec state only after strict validation. On validation failure, the repository publisher restores the prior main specs and returns the change to its active location when no concurrent edit conflicts with rollback.

## Open Questions

なし。
