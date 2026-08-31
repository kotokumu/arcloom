## Why

The eleven published main specs contain 237 Scenarios across 75 Requirements. One hundred thirty-two Scenario headings lack a supported tag, and several normative partitions, decisions, lifecycle rules, and invariants can be understood only by reading concrete examples. The repository's specification methodology does not yet provide or enforce the representation-selection guidance added to the latest `openspec-template`.

## Intended Outcomes

- Specification authors select prose, Partition Tables, Decision Tables, State Transition Tables, Invariants, and Scenarios according to the structure of each rule.
- Readers can understand Reconciliation vocabulary, state space, decisions, and always-true guarantees before reading concrete Scenarios.
- Every existing main spec follows the new representation-selection and Scenario rules without changing its accepted behavior.
- Automated structural validation rejects unsupported Requirement blocks and Scenario tags throughout main specs.

## Success Criteria

- SC-1: Arcloom's specification methodology, Quality Workflow instructions, templates, and machine-readable Requirement structure contain one consistent representation-selection model adapted from the latest `~/openspec-template`.
- SC-2: Every normative partition, condition combination, lifecycle transition, and always-true guarantee in every existing main spec appears in the Conceptual Model or Requirement body rather than only in a Scenario.
- SC-3: Every remaining Scenario in the main-spec corpus is a tagged representative concrete example whose result is derivable from its Requirement.
- SC-4: All seventy-five accepted Requirement guarantees remain present; thirty-six legacy numbered titles are renamed to stable kebab-case IDs and all repository references follow those renames, with no implementation or public-contract change.
- SC-5: Repository validation rejects unrecognized Requirement block labels and Scenario tags, and all main specs pass that validation.
- SC-6: Strict OpenSpec validation and the existing Go test suite pass after the documentation refactor.

## Scope

### In Scope

- Adapt the latest specification-representation guidance from `~/openspec-template` into Arcloom's methodology, schema instructions, project rules, templates, and machine-readable Requirement structure.
- Add structural validation for allowed Requirement blocks and Scenario tags.
- Extend atomic specification publication with an explicit Scenario retirement manifest so a validated MODIFIED Requirement can remove examples that are redundant after their rules move into normative tables or Invariants.
- Normalize existing unsupported Requirement blocks and Scenario tags so the complete main-spec corpus passes structural validation.
- Reorganize every existing main spec with rule-appropriate normative representations; replace a Conceptual Model only where the existing overview is insufficient for the rewritten Requirements.
- Retire redundant or test-matrix Scenarios throughout the corpus through the explicit manifest while preserving their accepted guarantees in Requirement rules and implementation verification.

### Out of Scope

- Changing product behavior, Go public contracts, tests, Component ownership, or dependency direction.
- Translating the complete specification corpus or changing its established terminology solely for language consistency.
- Adding a generic Reconciliation Result, target taxonomy, durable scheduler, or external-effect workflow.

## Capabilities

### New Capabilities

None. This Change modifies specification authoring and the presentation of every existing capability without adding product behavior.

### Modified Capabilities

- `authorization`, `codex-plan-control-assessment`, `github-plan-creation-dry-run`, `github-plan-representation-observation`, `github-plan-snapshot`, `plan-application-request`, `plan-control`, `plan-reconciliation-loop`, `plan-representation-reconciliation`, `plan`, and `reconciliation-control-loop`: move complete accepted rules into the representation selected by their structure, rename legacy numbered titles to stable kebab-case IDs, tag retained Scenarios, and retire redundant examples without changing observable guarantees.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Existing product concepts | Their current owning capabilities | Presentation only; meanings, identity, states, relationships, and invariants remain unchanged. |
| Specification representation | Not a product capability | Repository authoring methodology gains explicit selection rules and structural validation. |

## Decisions Required

None. The source template, affected specifications, preserved Requirement IDs, and behavior-preserving scope determine the work without changing product acceptance.

## Impact

- Specification authors and reviewers receive explicit representation-selection and structural-conformance rules.
- All seventy-five existing Requirements are presentation targets. Thirty-six legacy numbered titles receive stable kebab-case IDs and every repository reference follows the rename; accepted guarantees remain unchanged.
- No consumer-visible behavior, external system, data, API, or migration is affected.
