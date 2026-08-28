## 1. Publication preconditions

- [x] 1.1 `[design-only]` Confirm no active change has a delta path under `specs/plan/` or `specs/plan-representation-reconciliation/`; verify from `openspec status --change <active-change> --json` for every active change
- [x] 1.2 `[design-only]` Compare accepted and delta heading sets; verify all 13 Requirement names and all 63 Scenario names are preserved exactly
- [x] 1.3 `[change]` Confirm `model.md` contains exactly `None.` under Unresolved Decisions and complete Conceptual Model replacements for both affected capabilities

## 2. Specification review

- [x] 2.1 `[plan: PLN-1–PLN-5]` Review every Plan concept, Requirement block, and Scenario against the accepted spec; verify composition, text, collection, Target Date, validity, and violation outcomes are preserved
- [x] 2.2 `[plan-representation-reconciliation: PRR-1–PRR-8]` Review every reconciliation concept, Requirement block, and Scenario against the accepted spec; verify Observation, comparison, Evidence, failure precedence, and isolation outcomes are preserved
- [x] 2.3 `[change]` Search normative delta and replacement text for Go, package, constructor, zero value, accessor, Port, Controller, API field, and data-race mechanisms; remove each implementation dependency or record why the term denotes an observable external contract
- [x] 2.4 `[change]` Apply every criterion in `openspec/specs/REVIEW.md`; verify no P0 or P1 finding remains

## 3. Verification

- [x] 3.1 `[change]` Run `openspec validate refactor-core-plan-specifications --strict` and resolve every error
- [x] 3.2 `[change]` Run `openspec schema validate quality-spec`, `openspec validate --specs --strict`, and `node --test tools/archive-change.test.mjs`; verify the schema, current main specs, and publication boundary remain valid
- [x] 3.3 `[design-only]` Confirm apply changes only planning artifacts and leaves main-spec publication to `node tools/archive-change.mjs refactor-core-plan-specifications`
