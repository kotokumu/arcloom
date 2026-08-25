## Why

Arcloom can compare a GitHub representation with a caller-supplied Plan and can establish a provider-independent Plan Control assessment, but it cannot yet obtain the current Plan from GitHub, use a concrete AI Agent Provider, authorize a proposed revision independently of `Change`, request its application from an external Actor, or re-observe the authoritative result. A real Plan therefore cannot yet be controlled to completion.

## What Changes

- Establish one current Plan and provider-neutral progress evidence from fresh GitHub planning facts without making Arcloom authoritative for either.
- Add provider-independent authorization whose exact target-bound subject is supplied by its consumer and which does not depend on the experimental `Change` concept.
- Add the Plan application-request boundary that evaluates authorization for one exact target-bound revision and requests application from an external Actor without applying or claiming the external change itself.
- Keep GitHub Milestone and Issue mutation with an external Actor because GitHub provides no documented compare-and-swap or create-if-absent contract for those resources.
- Add a concrete Codex app-server AI Agent Provider adapter for the existing Plan Control Assessor Port, with structured output and no Codex protocol types crossing the Provider boundary.
- Add a disposable proof composition that demonstrates observation, Codex Plan Control, an authorized external-Actor application request, external reflection, re-observation, and completion against one real GitHub Plan without defining a fixed product workflow or owning durable control state.
- Update the product and architecture documents so Authorization is independent of `Change` and remains a safety boundary before external application.

## Capabilities

### New Capabilities

- `github-plan-snapshot`: Establishes a transient current Plan and provider-neutral progress evidence from authoritative GitHub planning facts.
- `authorization`: Establishes a generic, recalculable authorization decision for one consumer-supplied proposed external mutation without depending on `Change` or target-specific data.
- `plan-application-request`: Accepts one exact target-bound Plan revision, evaluates authorization for that revision, and requests application from an external Actor without treating request acceptance as proof of external state.
- `codex-plan-control-assessment`: Adapts the experimental Codex app-server protocol to the Plan Control Assessor Port using constrained structured output and caller-supplied observation material.

### Modified Capabilities

None.

## Impact

- Affected components: Plan Controller integration, Plan Snapshot Observation Module, Authorization, Plan Application Request Module, GitHub Planning Provider Module, Codex AI Agent Provider Module, external Actor boundary, and Composition Root.
- Affected code: new provider-independent packages and public contracts, read-side additions to `githubplan`, a concrete Codex adapter, tests, and a disposable host-facing proof harness.
- Affected external systems: read-only GitHub REST observation, an external Actor using GitHub's native interface, and the locally installed experimental `codex app-server` protocol.
- Affected canonical documents: `PRODUCT.md` and `ARCHITECTURE.md` require terminology and dependency updates for generic Authorization.
- No authoritative Plan, authorization decision, application request/result, AI session, or loop state is persisted by Arcloom.
