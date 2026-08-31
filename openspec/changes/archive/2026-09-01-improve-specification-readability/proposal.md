## Why

The eleven published main specs contained 237 Scenarios across 75 Requirements. One hundred thirty-two Scenario headings lacked a supported tag, and several normative partitions, decisions, lifecycle rules, and invariants could be understood only by reading concrete examples. The repository methodology did not yet include the representation-selection guidance in the latest `openspec-template`.

## Intended Outcomes

- Authors select prose, Partition Tables, Decision Tables, State Transition Tables, Invariants, and Scenarios according to each rule's structure.
- Readers understand Reconciliation vocabulary, state space, decisions, and always-true guarantees before reading examples.
- Every main spec follows the new representation-selection and Scenario rules without changing accepted behavior.
- Standard Markdown lint detects general Markdown defects without implementing an OpenSpec-specific parser.

## Success Criteria

- SC-1: The methodology, Quality Workflow instructions, templates, and Requirement structure contain one consistent representation-selection model adapted from the latest `~/openspec-template`.
- SC-2: Every normative partition, condition combination, lifecycle transition, and always-true guarantee appears in the Conceptual Model or Requirement body rather than only in a Scenario.
- SC-3: Every remaining Scenario is a tagged representative example whose result is derivable from its Requirement.
- SC-4: All 75 accepted Requirement guarantees remain present; 36 legacy numbered titles use stable kebab-case IDs and repository references follow those renames.
- SC-5: Repository Markdown passes standard Markdown lint, while OpenSpec meaning and structure remain governed by the methodology and review.
- SC-6: Strict OpenSpec validation and the existing Go test suite pass after the documentation refactor.

## Scope

### In Scope

- Adapt the latest specification-representation guidance from `~/openspec-template` into Arcloom's methodology, schema instructions, project rules, templates, and Requirement structure.
- Add standard Markdown lint to repository validation.
- Normalize unsupported Requirement blocks and Scenario tags.
- Reorganize every main spec with rule-appropriate normative representations.
- Retain only representative Scenarios after preserving every accepted outcome in a Requirement rule, table, or Invariant.

### Out of Scope

- Implementing a repository-specific Markdown or OpenSpec parser.
- Extending publication tooling with a custom Scenario-removal protocol.
- Changing product behavior, Go public contracts, tests, Component ownership, or dependency direction.
- Translating the complete specification corpus solely for language consistency.
- Adding a generic Reconciliation Result, target taxonomy, durable scheduler, or external-effect workflow.

## Capabilities

### New Capabilities

None. This Change modifies specification authoring and presentation without adding product behavior.

### Modified Capabilities

- `authorization`, `codex-plan-control-assessment`, `github-plan-creation-dry-run`, `github-plan-representation-observation`, `github-plan-snapshot`, `plan-application-request`, `plan-control`, `plan-reconciliation-loop`, `plan-representation-reconciliation`, `plan`, and `reconciliation-control-loop`: move accepted rules into the representation selected by their structure, rename legacy numbered titles, tag retained Scenarios, and remove redundant examples without changing observable guarantees.

## Affected Concepts

| Concept | Owner | Change |
|---|---|---|
| Existing product concepts | Existing owning capabilities | Presentation only; meanings, identity, states, relationships, and invariants remain unchanged. |
| Specification representation | OpenSpec authoring methodology | Adds explicit representation-selection and review rules. |
| Markdown syntax quality | Standard Markdown lint | Detects general Markdown defects without deciding OpenSpec semantics. |

## Decisions Required

None.

## Impact

- Authors and reviewers receive explicit representation-selection rules.
- All 75 Requirements use the revised presentation.
- The 36 legacy numbered Requirement titles use stable kebab-case IDs.
- No consumer-visible behavior, external system, data, API, or migration is affected.
