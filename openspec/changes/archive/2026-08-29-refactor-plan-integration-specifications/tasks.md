## 1. Publication preconditions

- [x] 1.1 `[design-only]` Confirm no other active change has a delta path under `specs/github-plan-creation-dry-run/`, `specs/github-plan-representation-observation/`, or `specs/plan-control/`; verify from `openspec status --change <active-change> --json` for every active change
- [x] 1.2 `[design-only]` Compare accepted and delta heading sets; verify all 23 Requirement names and all 68 Scenario names are preserved exactly
- [x] 1.3 `[change]` Confirm `model.md` contains exactly `None.` under Unresolved Decisions and complete Conceptual Model replacements for all three affected capabilities

## 2. Specification review

- [x] 2.1 `[github-plan-creation-dry-run: GPCD-1–GPCD-7]` Review every concept, Requirement block, and Scenario against the accepted spec; verify target validity, narrative compatibility, request topology, reference integrity, passivity, and validation outcomes are preserved
- [x] 2.2 `[github-plan-representation-observation: GHPO-1–GHPO-8]` Review every concept, Requirement block, and Scenario against the accepted spec; verify target binding, fact mapping, conservative unavailability, collection integrity, GitHub.com scope, and isolation outcomes are preserved
- [x] 2.3 `[plan-control: PLC-1–PLC-8]` Review every concept, Requirement block, and Scenario against the accepted spec; verify Plan Snapshot, completion meaning, assessment form, observation pass-through, failure distinctions, boundaries, and statelessness are preserved
- [x] 2.4 `[change]` Search normative delta and replacement text for Go, package, constructor, zero value, accessor, Port, Controller, API field, data race, DTO, and internal call or storage mechanisms; remove every implementation dependency or record why a term denotes an observable external contract
- [x] 2.5 `[change]` Apply every criterion in `openspec/specs/REVIEW.md`; verify no P0 or P1 finding remains

## 3. Verification

- [x] 3.1 `[change]` Run `openspec validate refactor-plan-integration-specifications --strict` and resolve every error
- [x] 3.2 `[change]` Run `openspec schema validate quality-spec`, `openspec validate --specs --strict`, and `node --test tools/archive-change.test.mjs`; verify the schema, current main specs, and publication boundary remain valid
- [x] 3.3 `[design-only]` Confirm apply changes only planning artifacts and leaves main-spec publication to `node tools/archive-change.mjs refactor-plan-integration-specifications`
