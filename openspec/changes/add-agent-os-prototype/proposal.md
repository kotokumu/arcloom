## Why

The Agent OS direction needs a runnable foundation on which an operator can replace business loops and external agents without changing the host's business logic. Existing Plan capabilities own disposable, Plan-specific operations; they do not provide durable application-owned work or local resource ownership.

## Intended Outcomes

An operator can start, inspect and resume local work with an explicit application and permissions. Applications determine business progression; interchangeable execution and feedback drivers supply results. Interrupted external work remains visibly uncertain instead of being repeated blindly.

## Success Criteria

- SC-1: Two business applications produce different progressions without changing the host.
- SC-2: Two execution providers and one feedback provider can be selected explicitly without provider-specific business branching in the host.
- SC-3: Restart preserves committed state and results, rejects changed run bindings, and never automatically repeats an uncertain external action.
- SC-4: Missing grants prevent external dispatch; expired knowledge is excluded and confirmed context remains separate from drafts.
- SC-5: Durable mailboxes, immutable artifacts and stale lease rejection are verifiable through local operations.

## Scope

### In Scope

- A locally runnable, single-owner prototype with inspectable Intent, Task, AgentProcess, Artifact, KnowledgeNode, Signal and Lease categories.
- Replaceable business progression, execution and feedback; explicit named grants and bounded local actions.
- Durable state and consumer cursors, queue/coalescing delivery, restart uncertainty, knowledge expiry and separated context.
- Operator-only knowledge ingestion and one owner scope per state directory.

### Out of Scope

- Completion of Stage 1; full asynchronous syscalls, parallel work and interruption guarantees, live provider hooks or production sandboxing.
- UI, semantic knowledge GC, OKF compatibility, remote installation, multi-tenant access or live grant revocation.
- Automatic replay of uncertain effects and authority over external business facts.
- Changes to accepted Go Plan capabilities.

## Capabilities

### New Capabilities

- `agent-os-operation`: An operator starts, inspects and resumes application-controlled local work with durable evidence and explicit authority.

### Modified Capabilities

None. Existing `plan-feedback-loop-operation`, `reconciliation-control-loop` and `authorization` contracts retain their current scope and Requirement IDs.

## Affected Concepts

| Concept | Candidate owner capability | Change |
|---|---|---|
| Intent, Task, AgentProcess | agent-os-operation | Add prototype-owned intent binding and work lifecycle; distinct from a Plan Task |
| Artifact, KnowledgeNode | agent-os-operation | Add immutable evidence and expiring operator-curated knowledge |
| Signal, Mailbox | agent-os-operation | Add durable correlated events and independent consumption |
| Lease | agent-os-operation | Add local resource ownership with stale-holder rejection |
| Application, driver capability, owner scope | agent-os-operation | Add replaceability and explicit permission boundaries |

## Decisions Required

None. The reviewed limited increment uses one operator scope and operator-only knowledge ingestion; the full original roadmap remains authoritative for subsequent increments.

## Impact

Operators gain a separate prototype beside the existing product. Existing Go consumers and historical external systems require no migration. The original direction remains preserved in [Agent OS v0.1](../../../docs/design/agent-os-v0.1.original.md); the implementation boundary and deferred guarantees are recorded in [design.md](design.md).
