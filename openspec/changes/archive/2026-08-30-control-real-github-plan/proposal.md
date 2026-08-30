## Why

Arcloom can compare a GitHub representation with a caller-supplied Plan and can establish a provider-independent Plan Control assessment, but it cannot yet obtain the current Plan from GitHub, obtain that assessment through Codex, authorize a proposed revision independently of `Change`, request its application from an external Actor, or re-observe the authoritative result. A real Plan therefore cannot yet be controlled to completion.

## Intended Outcomes

- Establish one current Plan and provider-neutral progress evidence from fresh GitHub planning facts without making Arcloom authoritative for either.
- Add provider-independent authorization whose exact target-bound subject is supplied by its consumer and which does not depend on the experimental `Change` concept.
- Add the Plan application-request boundary that evaluates authorization for one exact target-bound revision and requests application from an external Actor without applying or claiming the external change itself.
- Leave GitHub Milestone and Issue mutation with an external Actor and keep application-request receipt distinct from external target state.
- Enable the existing provider-independent Plan Control assessment through a trusted compatible Codex installation without exposing Codex-specific meaning to its consumer.
- Add a disposable proof composition that demonstrates observation, Codex Plan Control, an authorized external-Actor application request, external reflection, re-observation, and completion against one real GitHub Plan without defining a fixed product workflow or owning durable control state.

## Success Criteria

- SC-1: A caller can establish a current Plan and provider-independent progress from a fresh observation of one exact GitHub Milestone target.
- SC-2: A caller can obtain one valid Plan Control assessment for the exact current Plan and caller-owned observation material through Codex without exposing provider protocol meaning.
- SC-3: Only a currently Authorized exact Plan revision can be requested from an external Actor, and the result preserves request-receipt uncertainty without claiming external state.
- SC-4: A later fresh observation can establish the resulting current Plan independently of the application request.
- SC-5: Repeated and concurrent operations retain no authoritative Plan, authorization, AI-session, request, or loop state in Arcloom.

## Scope

### In Scope

- Current Plan and progress observation from one GitHub Milestone target.
- Generic subject-bound authorization and exact Plan application-request contracts.
- Read-only Codex-backed Plan Control assessment.
- Disposable verification of one real Plan revision and re-observation.
- Product and architecture terminology affected by generic Authorization.

### Out of Scope

- Direct GitHub mutation by an Arcloom Provider.
- A long-running controller, scheduler, retry coordinator, or durable loop history.
- Task execution, Delivery Acceptance, and treating Task completion as Goal completion.
- A universal Observation schema or common Provider interface.
- Stabilization of the experimental `Change` concept.

## Capabilities

### New Capabilities

- `github-plan-snapshot`: Establishes a transient current Plan and provider-neutral progress evidence from authoritative GitHub planning facts.
- `authorization`: Establishes a generic, recalculable authorization decision for one consumer-supplied proposed external mutation without depending on `Change` or target-specific data.
- `plan-application-request`: Accepts one exact target-bound Plan revision, evaluates authorization for that revision, and requests application from an external Actor without treating request acceptance as proof of external state.
- `codex-plan-control-assessment`: Obtains a read-only Plan Control assessment through Codex for one exact current Plan and caller-supplied observation material.

### Modified Capabilities

None.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| GitHub Plan Snapshot | `github-plan-snapshot` | Add current-Plan eligibility and provider-independent progress meaning without redefining the Plan Snapshot owned by `plan-control`. |
| Authorization Evaluation | `authorization` | Add exact-subject binding and recalculable decision semantics. |
| Plan Revision | `plan-application-request` | Add exact target/current/proposed association. |
| Request Receipt Evidence | `plan-application-request` | Add receipt-only outcomes that do not establish external state. |
| Codex Assessment Interaction | `codex-plan-control-assessment` | Add a read-only, invocation-isolated use of the existing Plan Control contract. |

## Decisions Required

None.

## Impact

- Affected consumers: Hosts that compose observation, assessment, authorization, application request, and later observation.
- Affected external systems: read-only GitHub observation, an external Actor using GitHub's native interface, and a trusted compatible Codex installation.
- Affected contracts: four new capabilities; no existing capability behavior is modified.
- Affected canonical documents: `PRODUCT.md` and `ARCHITECTURE.md` require terminology and ownership updates for generic Authorization.
- No authoritative Plan, authorization decision, application request/result, AI session, or loop state is persisted by Arcloom.
