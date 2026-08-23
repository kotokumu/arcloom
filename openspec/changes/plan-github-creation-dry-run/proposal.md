## Why

Arcloom has no executable Plan model or GitHub planning boundary to validate before external mutation is introduced. Requiring manually prepared GitHub resources would test observation first and would not establish whether one provider-independent Plan can be represented safely through either GitHub Milestones or Issues.

---

## What Changes

- Add the provider-independent Plan vocabulary required for an initial creation request: name, Goal, acceptance conditions, optional target date, and Tasks.
- **BREAKING**: Treat a Milestone as one possible external representation of a Plan rather than an element owned by the Plan model.
- Add a GitHub creation dry-run that represents one Plan as either a Repository Milestone with associated Issues or a parent Issue with sub-issues.
- Produce a deterministic, dependency-aware GitHub API request plan without sending a request or inventing GitHub-assigned identifiers.
- Keep observation, reconciliation with existing GitHub facts, authorization, request execution, update, deletion, and persistence outside this change.

---

## Capabilities

### New Capabilities

- `plan`: Defines the provider-independent meaning and structural invariants of a Plan used by this change.
- `github-plan-creation-dry-run`: Defines complete GitHub request planning for the supported Milestone and Issue representations without external mutation.

### Modified Capabilities

None.

---

## Impact

- Revises the Plan descriptions in `PRODUCT.md` and `ARCHITECTURE.md` so Provider-native Milestones are not owned by the provider-independent Planning Model.
- Adds the first product packages beyond the build-check placeholder.
- Adds a GitHub Planning Provider boundary and public contracts for inspecting a creation dry-run.
- Does not add a GitHub SDK, network access, authentication, durable state, a CLI, a server, or a third-party production dependency. Test code uses `go-cmp` as required by the repository's Go test-authoring workflow.
