## Why

GitHub Plan creation currently writes an `arcloom-plan:v1` block into Milestone descriptions and parent Issue bodies, and GitHub Plan observation treats that block as the only source for Goal, Acceptance Conditions, and an Issue-represented Target Date. This makes the external planning system act as Arcloom-specific storage and prevents a GitHub-native Plan such as Milestone #3 from being established as a current Plan.

## Intended Outcomes

- A GitHub Milestone or parent Issue remains a meaningful Plan representation when Arcloom is absent or removed.
- Creation produces human-readable GitHub content without Arcloom-specific metadata.
- Observation establishes Plan meaning from the GitHub-native title, description or body, date, and Task relationships.
- Legacy content has explicit compatibility behavior and does not remain required storage.

## Success Criteria

- SC-1: A creation dry-run for either supported representation contains no `arcloom-plan:v1` marker or equivalent opaque Arcloom state.
- SC-2: Observation establishes the supported Plan facts from GitHub-native information when Arcloom-specific metadata is absent or has been removed.
- SC-3: Milestone #3 is observable as a current Plan from its existing GitHub title, description, assigned Issues, and native state.
- SC-4: Observation handles legacy metadata present, absent, and removed according to one explicit compatibility rule without requiring the metadata.
- SC-5: Every affected main Requirement, fixture, and verification artifact no longer defines the metadata block as authoritative Plan storage.

## Scope

### In Scope

- GitHub-native representation of Goal and ordered Acceptance Conditions in Milestone descriptions and parent Issue bodies.
- GitHub-native representation of an Issue-represented Target Date.
- Creation dry-run output for Milestone and Issue representations.
- Read-only observation of native content for Milestone and Issue representations.
- Compatibility behavior for existing descriptions and bodies that contain legacy `arcloom-plan:v1` content.
- Localized unavailable information and validation violations when native content cannot establish a Plan fact.
- Migration of affected specifications, tests, fixtures, and real-Plan verification evidence.

### Out of Scope

- Mutation or backfill of existing GitHub Milestones and Issues.
- Authorization, application requests, or direct GitHub mutation.
- Additional GitHub Plan representation kinds.
- Changes to provider-independent Plan meaning or validity.
- General-purpose Markdown interpretation outside the declared GitHub Plan representation.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `github-plan-creation-dry-run`: `[[github-plan-creation-dry-run/versioned-plan-narrative]]`, `[[github-plan-creation-dry-run/milestone-creation-request-plan]]`, `[[github-plan-creation-dry-run/issue-creation-request-plan]]` — replace metadata-backed narrative creation with a GitHub-native, human-readable Plan representation.
- `github-plan-representation-observation`: `[[github-plan-representation-observation/versioned-payload-meaning]]`, `[[github-plan-representation-observation/milestone-representation-meaning]]`, `[[github-plan-representation-observation/issue-representation-meaning]]`, `[[github-plan-representation-observation/conservative-failure-observation]]` — establish Plan facts from native content and define compatibility and incomplete-content outcomes.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Versioned Plan Narrative | `github-plan-creation-dry-run` | Replace the metadata-backed definition with a human-readable GitHub Plan Narrative. |
| Observed GitHub Fact | `github-plan-representation-observation` | Replace recovered payload meaning with meaning established from declared native content. |
| GitHub Plan Representation | `github-plan-creation-dry-run` | Constrain representability by the native narrative contract while preserving the Milestone and Issue variants. |
| GitHub Observation Outcome | `github-plan-representation-observation` | Define localized outcomes for absent, malformed, ambiguous, and legacy native content. |

## Decisions Required

- Define the native narrative structure and the exact representability boundary for Plan Text that can conflict with structural markers.
- Decide whether legacy `arcloom-plan:v1` content is ignored, accepted as a fallback, or rejected when native meaning is present or absent.
- Define how native section absence, duplication, ordering, and malformed numbering affect each Plan Location.

## Impact

- Existing consumers of creation request descriptions and Issue bodies receive different exact content.
- Existing GitHub resources that rely only on the metadata block may no longer expose a complete current Plan without a human-readable native narrative.
- The two affected capabilities and the GitHub Plan Adapter implementation, tests, fixtures, and verification evidence change together.
- `PRODUCT.md` remains unchanged because provider independence and external authority already require this direction.
- `ARCHITECTURE.md` requires a terminology update only if the accepted design removes payload-version ownership from the GitHub Plan Adapter boundary.
