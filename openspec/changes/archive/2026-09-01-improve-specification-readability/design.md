## Context

This is a medium-risk, behavior-preserving documentation and repository-tooling refactor. The risk trigger is coordinated change across the Quality Workflow, authoring methodology, structural validation, CI, and all eleven accepted main specs. Product concepts, responsibilities, state ownership, public Go contracts, and Architecture boundaries remain unchanged, so the lightweight responsibility check applies instead of new product conceptual modeling or independent evolution-scenario discovery.

The latest `~/openspec-template` is the source for representation-selection semantics. Arcloom retains its project context and established Japanese Requirement block labels to avoid an unrelated corpus-wide language migration.

## Goals / Non-Goals

### Goals

- Keep one coherent representation-selection rule across methodology, schema instructions, project rules, templates, review criteria, and machine-readable structure.
- Make every main spec readable from its Conceptual Model and Requirement rules before Scenarios.
- Preserve all seventy-five existing observable guarantees and give the thirty-six legacy numbered Requirement titles stable kebab-case IDs.
- Enforce allowed Requirement blocks, block order, and Scenario tags in local and CI validation.

### Non-Goals

- Change product behavior, implementation code, tests, public contracts, Components, or dependency direction.
- Convert rules that have no partition, decision, lifecycle, or invariant structure into tables, or reduce Scenarios to a target count.
- Add a generic parsing, Markdown, helper, utility, shared, or common package.
- Mechanically decide whether an author selected the semantically correct normative representation.

## Conceptual-Model-to-Implementation Mapping

| Specification concept / Requirement | Owning component | Physical representation | Notes |
|---|---|---|---|
| Representation-selection methodology | OpenSpec authoring methodology | `openspec/specs/README.md`, `MODELING.md`, and `REVIEW.md` | Arcloom-adapted content from the latest local template is authoritative for authors and reviewers. |
| Artifact-generation guidance | Quality Workflow | `openspec/config.yaml`, `openspec/schemas/quality-spec/schema.yaml`, and schema templates | Every generation stage receives the same representation and Scenario constraints. |
| Machine-readable structure | Specification methodology | `openspec/specs/_schema/requirement-structure.json` version 2 | Keeps Japanese block labels and adds `不変条件` plus representation metadata. |
| Structural conformance | Repository validation | `tools/validate-spec-structure.mjs` and its test | Exact responsibility: validate main-spec Requirement structure; no generic helper boundary. |
| Scenario retirement | Quality Workflow publication | Change-local `scenario-retirements.json` applied by `tools/archive-change.mjs` | Removes only explicitly named examples after ordinary delta archive and before final strict validation. |
| CI conformance | OpenSpec Check | `.github/workflows/openspec-check.yml` | Runs structural validator tests and validates the repository corpus. |
| Main-spec presentation | Existing owning capabilities | All eleven existing main `spec.md` files | IDs and guarantees are preserved; only organization and representative examples change. |

## Decisions

### Decision: Adapt template semantics without a language migration

- **Choice**: Port the latest template's Conceptual Model, representation-selection, Invariant, and Scenario constraints into the existing Japanese methodology and block vocabulary. Preserve Arcloom-specific `context` and operation guidance.
- **Rationale**: The source update addresses the readability defect, while byte-copying the English template would create mixed block labels and overwrite project-specific configuration.
- **Alternatives**: Copy every source file verbatim; rejected because it changes language and project context. Keep the current template; rejected because it permits Scenario-centered specifications.
- **Consequences**: Upstream updates require semantic comparison rather than blind file replacement. The repository remains internally consistent.

### Decision: Validate structure with one responsibility-named command

- **Choice**: Add `tools/validate-spec-structure.mjs`. With an optional repository-root argument, it loads `openspec/specs/_schema/requirement-structure.json`, discovers main specs, ignores fenced code and HTML comments, and validates Requirement block labels and order plus Scenario tags. It exits nonzero with file and line diagnostics on any violation.
- **Rationale**: `openspec validate --strict` does not read the repository's machine-readable structure; unsupported labels and `[isolation]` currently pass. A dedicated command closes that exact gap without broadening `archive-change.mjs`.
- **Alternatives**: Add validation to `archive-change.mjs`; rejected because publication and corpus linting have different lifecycles. Extract a shared Markdown helper; rejected because there is no second justified owner and the abstraction would be broader than the required contract. Rely on reviewer instructions only; rejected because current invalid structures reached main.
- **Consequences**: The parser intentionally validates structural syntax, not semantic representation choice. Semantic selection remains a review responsibility.

### Decision: Migrate the complete corpus from guarantees, not from Scenario deletion

- **Choice**: For each of the seventy-five Requirements, first restate every accepted rule in prose, a Partition Table, Decision Table, State Transition Table, or Invariant according to `model.md`; only then retain representative Scenarios derivable from those rules.
- **Rationale**: Deleting Scenarios first risks deleting the only current statement of a guarantee. A preservation map makes semantic equivalence reviewable.
- **Alternatives**: Reduce Scenario count mechanically; rejected because quantity is not a quality criterion. Leave all Scenarios and add tables; rejected because it duplicates normative information and remains difficult to read.
- **Consequences**: The diff is documentation-heavy but behavior-preserving. Review must compare each old Scenario outcome against the new normative rule or retained example.

### Decision: Publish clarified normative presentation through the Quality Workflow

- **Choice**: Treat the corpus-wide reader-facing reorganization as specification clarification: record every needed complete Conceptual Model replacement in `model.md`, every rewritten Requirement in `MODIFIED Requirements`, and publish them together with `tools/archive-change.mjs` after implementation and review.
- **Rationale**: Even when product behavior is preserved, direct Apply edits to a main spec's Conceptual Model would bypass the repository's atomic publication contract. The modified text remains normative and therefore belongs in a reviewed specification Change.
- **Alternatives**: Use `skip_specs: true` and edit main specs directly; rejected because Apply instructions prohibit Conceptual Model publication. Avoid an OpenSpec Change entirely; rejected because the multi-file migration and equivalence checks need an auditable plan.
- **Consequences**: The Change declares all existing capabilities modified in presentation while explicitly preserving their Requirement IDs and observable guarantees. Apply changes methodology and tooling; archive performs the main-spec publication.

### Decision: Retire redundant Scenarios as an atomic publication operation

- **Choice**: A Change may include `scenario-retirements.json` version 1 with exact capability, Requirement slug, and full Scenario heading. `archive-change.mjs` validates the manifest against the delta and current main spec, runs ordinary strict Change validation and archive with every existing Scenario still present, removes the explicitly retired Scenario blocks from the published main specs, confirms at least one Scenario remains in each Requirement, then runs final strict and structural validation before committing publication.
- **Rationale**: OpenSpec 1.10.0 deliberately rejects a MODIFIED Requirement that omits an existing Scenario. Keeping all fifty examples preserves validation but leaves the readability defect. A manifest makes the otherwise unsupported removal explicit, reviewable, bounded, and rollback-safe without weakening strict validation.
- **Alternatives**: Archive with `--no-validate`; rejected because it weakens the publication gate. Directly remove Scenarios after archive; rejected because it is not atomic and the archived Change would not explain the difference. Keep redundant Scenarios indefinitely; rejected because they remain the dominant document structure.
- **Consequences**: Publication tooling gains one narrowly defined migration operation. The manifest is archived with the Change. Scenario retirement changes examples only; every retired THEN outcome must already be preserved in the replacement Requirement.

### Decision: Normalize the complete corpus under one publication

- **Choice**: Rewrite all eleven main specs through capability deltas, rename legacy numbered Requirement titles with explicit `RENAMED Requirements`, normalize unsupported Requirement blocks, add supported tags to retained Scenarios, and retire examples only after their outcomes are represented normatively.
- **Rationale**: A repository-wide methodology is credible only when the accepted corpus follows it without legacy exceptions.
- **Alternatives**: Allow legacy exceptions; rejected because they preserve ambiguity. Apply only syntactic tags; rejected because it leaves rules encoded only in examples.
- **Consequences**: The Change is documentation-heavy and must publish all affected capabilities atomically.

## Responsibility and Interface Checks

### Lightweight Responsibility Check

| Changed behavior | Existing owner | Information / authority used | State / invariant affected | Public contract / boundary impact | Verdict |
|---|---|---|---|---|---|
| Select specification representation | OpenSpec authoring methodology | Rule structure and published methodology | Documentation completeness only | None | Pass |
| Reject unsupported spec structure | Repository validation tooling | Machine-readable block and tag definitions | Repository conformance only | One repository command consumed by CI | Pass |
| Present accepted guarantees | Existing owning main specs | Already accepted Requirement content | No product state or invariant changes | None | Pass, subject to equivalence review |

No changed behavior requires a new product Concept, responsibility, state owner, Component, Package, Port, or public Go contract. Full independent evolution-scenario discovery and Architecture-boundary analysis are N/A because the Change preserves those elements.

### Repository Command Contract

| Interface | Consumer | Input / precondition | Output / postcondition | Failure | Constraint protected |
|---|---|---|---|---|---|
| `node tools/validate-spec-structure.mjs [repository-root]` | Local authors and OpenSpec Check | Root contains the machine-readable structure and main-spec directory; omitted root means current working directory | Exit 0 and a concise checked-file summary | Exit nonzero with exact file, line, and structural violation | Unsupported authoring structure cannot enter main unnoticed |

## Test Specification and TDD Plan

### Structural Validator Behavior

| Case | Expected observation |
|---|---|
| Canonical blocks in configured order and an allowed Scenario tag | Validation succeeds. |
| Unknown Requirement block | Validation fails at its file and line. |
| Canonical blocks out of order or repeated | Validation fails at the first structural violation. |
| Unknown or missing Scenario tag | Validation fails at its heading line. |
| Bold labels or headings inside HTML comments or fenced examples | They are ignored. |
| Repository main-spec corpus | All discovered specs pass after normalization. |

### Scenario Retirement Publication Behavior

| Case | Expected observation |
|---|---|
| Manifest names an existing Scenario included in the corresponding MODIFIED Requirement | Ordinary strict Change validation succeeds; publication removes that Scenario before final validation. |
| Manifest is absent | Publication behavior remains unchanged. |
| Manifest has an unsupported version, duplicate entry, invalid capability or Requirement, missing Scenario, or capability without a delta | Publication fails before modifying main specs. |
| Retirement would leave a Requirement without a Scenario | Publication fails and preserves the active Change and original main spec. |
| Retirement or final structural/OpenSpec validation fails after archive | Publisher restores every original main spec and moves the archived Change back to active. |

### Documentation Conformance

- All seventy-five accepted Requirements occur exactly once under stable kebab-case slugs in the rewritten files.
- Every old normative outcome maps to a new Requirement rule, table row, Invariant, or retained Scenario.
- No remaining Scenario supplies a result absent from its Requirement.
- Every retained or replaced Conceptual Model defines the vocabulary, classifications, relationships, states, and structural Invariants needed before Requirements.
- No `.go` file or public contract changes.

### Construction Sequence

1. **Red**: Add validator tests for canonical input, unknown labels, ordering, unknown tags, and ignored examples; confirm failure because the command does not exist.
2. **Green**: Implement the minimum responsibility-specific validator and make its tests pass.
3. **Refactor**: Improve diagnostics and parsing names without introducing a generic Markdown abstraction.
4. **Red**: Add archive workflow tests for valid retirement, invalid manifests, last-Scenario rejection, and rollback; confirm the valid-retirement case fails before publisher support exists.
5. **Green**: Add the minimum manifest parsing, exact Scenario-block removal, and rollback integration to `archive-change.mjs`.
6. Integrate structural validation into OpenSpec Check and publication, then confirm current corpus failures identify the known violations.
7. Adapt the methodology and templates, normalize required legacy violations, and make corpus validation pass.
8. Complete every capability delta from the guarantee-preservation audit and explicit retirement manifest, then run semantic review and all validation.

## Detailed Design Gate

| Implementation unit | Implements concept / responsibility / contract | Dependencies | Behavior / test | Migration or rollback impact |
|---|---|---|---|---|
| Methodology and workflow documents | Representation selection and Scenario responsibility | Latest local template plus Arcloom context | Cross-file terminology and rule consistency review | Revert documentation files |
| `requirement-structure.json` | Canonical block order, representations, and tags | None | Validator fixtures consume it | Revert to version 1 with validator removal |
| `validate-spec-structure.mjs` | Main-spec structural conformance | Node standard library and structure JSON | Node behavior tests and corpus execution | Remove command and CI step |
| `scenario-retirements.json` and archive publication support | Explicit removal of redundant examples while preserving strict Change validation | Existing archive parser, atomic snapshots, and Node standard library | Archive workflow tests for success, invalid input, last Scenario, and rollback | Remove manifest support; existing Changes without the file are unaffected |
| Full-corpus delta specs and applicable model replacements | Reader-facing normative presentation | Existing accepted specs and guarantee-preservation audit | Slug/equivalence review, strict Change validation, post-publication structural and OpenSpec validation, Go tests | Publisher restores main specs on failure; revert publication commit after success |
| OpenSpec Check | Continuous structural enforcement | Node command and tests | Workflow command review plus local equivalent execution | Remove added steps |

| Implementation unit | Simplest viable representation | State / identity / lifecycle / boundary need | Rejected simpler alternative | Verdict |
|---|---|---|---|---|
| Methodology update | Direct document edits | One authoritative rule set must stay coherent across generated instructions | Add a new documentation service or generator | Pass |
| Structural validation | One executable Node module plus one test module | CI needs deterministic parsing and diagnostics; no long-lived state | Reviewer-only checklist | Pass |
| Scenario retirement | One optional versioned Change-local manifest handled by the existing publisher | OpenSpec 1.10.0 has no Scenario-removal delta operation; atomic lifecycle and rollback already belong to publication | New general migration framework or `--no-validate` | Pass |
| Full-corpus rewrite | Existing Requirements expressed as complete `MODIFIED` blocks plus only the necessary full Conceptual Model replacements | Product concepts and rules already exist; atomic publication must preserve them | Direct main-spec edits, syntax-only tagging, or new capabilities | Pass |

## Risks / Trade-offs

- A concise rewrite may accidentally drop a rare concurrency or cancellation guarantee → map every old Scenario outcome before removal and review the resulting rule tables against tests.
- Tables may become another form of exhaustive test catalog → include only reachable combinations with distinct guarantees and keep examples out of normative tables.
- Structural validation may misread Markdown examples → ignore fenced code and HTML comments and test those cases.
- Scenario retirement may remove the wrong block or leave an invalid Requirement → require exact manifest identity, reject ambiguous or missing matches, preserve one Scenario, and run final structural and strict validation under the existing rollback boundary.
- Arcloom may drift from the upstream template wording → preserve a source-to-target checklist in tasks and compare every changed upstream methodology file.
- GitHub Actions may remain unavailable because of account billing → run the exact CI commands locally and report the external execution limitation separately.

## Migration / Rollback

No product or data migration exists. During publication, `archive-change.mjs` stages Conceptual Model replacements, applies Requirement deltas, validates all main specs, and restores the active Change and original specs on failure. After a successful commit, rollback reverts the methodology, validator, CI, and published specification presentation together; existing implementation and runtime state are unaffected.

## Open Questions

None.
