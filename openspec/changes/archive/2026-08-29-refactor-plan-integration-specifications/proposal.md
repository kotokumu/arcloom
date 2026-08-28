## Why

The accepted `github-plan-creation-dry-run`, `github-plan-representation-observation`, and `plan-control` specifications still embed shared product meaning in long Requirement paragraphs. They also use implementation-shaped language such as Go-toolchain whitespace, constructors, accessors, and configured boundaries, which makes stable product contracts harder to distinguish from one physical implementation.

## Intended Outcomes

- Consumers can interpret GitHub Plan request planning, GitHub Plan observation, and Plan Control from explicit, uniquely owned concepts.
- Requirements state observable acceptance, results, failures, side effects, and isolation without prescribing internal construction or access mechanisms.
- Existing external compatibility, Plan meaning, evidence localization, AI assessment, and Scenario outcomes remain unchanged.

## Success Criteria

- SC-1: Every non-trivial concept shared by multiple Requirements in the three capabilities has one definition in its owning main spec or an explicit reference to its existing owner.
- SC-2: No normative statement depends on a Go toolchain, constructor, accessor, package boundary, Controller, Port, or internal data representation.
- SC-3: All 23 existing Requirement headings and every existing Scenario heading remain represented with equivalent preconditions, interactions, and observable results.
- SC-4: No active OpenSpec change has a delta spec for an affected capability when publication begins.
- SC-5: The change and all accepted main specs pass strict OpenSpec validation after publication.

## Scope

### In Scope

- Add a Conceptual Model to `github-plan-creation-dry-run` for repository targets, representation selection, versioned Plan narrative meaning, creation request plans, planned requests, and symbolic result references.
- Add a Conceptual Model to `github-plan-representation-observation` for GitHub Plan targets, native and payload-backed facts, collection establishment, and observation outcomes while reusing Plan and reconciliation concepts.
- Add a Conceptual Model to `plan-control` for Plan snapshots, delivery observations, Plan Control assessments, proposed Plans, and control failures.
- Rewrite all existing Requirements and Scenarios in the three capabilities into the structured observable-contract format.
- Preserve existing Requirement and Scenario headings so archived history and the current OpenSpec version retain valid identities.

### Out of Scope

- Product behavior, runtime code, public interface, external data, or architecture changes.
- Redefining Plan, Observation, Evidence, or Plan Validation Violation concepts owned by other capabilities.
- Changing the persisted `arcloom-plan:v1` compatibility contract or the declared GitHub.com compatibility version.
- Requirement ID migration or Scenario heading tags.
- Refactoring capabilities introduced by the active `control-real-github-plan` change.

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `github-plan-creation-dry-run`: `GPCD-1` through `GPCD-7` — separate stable request-planning meaning from construction and accessor mechanisms.
- `github-plan-representation-observation`: `GHPO-1` through `GHPO-8` — centralize GitHub target, fact-source, collection, and outcome concepts while preserving conservative observation behavior.
- `plan-control`: `PLC-1` through `PLC-8` — centralize control inputs, assessment states, proposed Plan meaning, and failure categories without changing AI-owned judgment.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| GitHub Repository Target | `github-plan-creation-dry-run` | Define local identity and validation once for request planning and reuse it from observation. |
| GitHub Plan Representation | `github-plan-creation-dry-run` | Define the explicit Milestone or Issue selection once. |
| Versioned Plan Narrative | `github-plan-creation-dry-run` | Define persisted machine meaning separately from human narrative. |
| Creation Request Plan | `github-plan-creation-dry-run` | Define passive ordered operations and their dependency graph. |
| Symbolic Result Reference | `github-plan-creation-dry-run` | Define dependency identity without implementation accessors or Provider-assigned values. |
| GitHub Plan Target | `github-plan-representation-observation` | Define the immutable repository, representation, and resource-number binding. |
| Observed GitHub Fact | `github-plan-representation-observation` | Distinguish native and payload-backed sources of Plan meaning. |
| Task Collection Establishment | `github-plan-representation-observation` | Define completeness and external-resource coherence before projection into Observation. |
| Plan Snapshot | `plan-control` | Define the current valid Plan supplied for one control decision. |
| Delivery Observation | `plan-control` | Define caller-supplied evidence whose semantics remain external-AI owned. |
| Plan Control Assessment | `plan-control` | Define Complete, Retain, Revise, and Insufficient Information outcomes once. |
| Proposed Plan | `plan-control` | Define the valid, meaningfully different Plan carried by Revise. |

## Decisions Required

なし。

## Impact

Specification readers and future changes receive clearer concept ownership and requirements that can be evaluated without knowing the Go implementation. Runtime behavior, persisted artifacts, GitHub operations, AI interactions, `PRODUCT.md`, and `ARCHITECTURE.md` remain unchanged.
