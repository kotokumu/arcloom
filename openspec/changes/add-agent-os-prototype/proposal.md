## Why

The Agent OS direction needs a runnable foundation on which an operator can replace business loops and external agents without changing the host's business logic. Existing Plan capabilities own disposable, Plan-specific operations; they do not provide durable application-owned work or local resource ownership.

## Intended Outcomes

An operator can start, inspect and resume local work with an explicit application and permissions. Applications determine business progression; interchangeable execution and feedback drivers supply results. Interrupted external work remains visibly uncertain instead of being repeated blindly.

## Success Criteria

- SC-1: Two business applications produce different progressions without changing the host.
- SC-2: Command, direct and delegated capability implementations can be selected through explicit dependencies without provider-specific business branching in the kernel.
- SC-3: Restart preserves committed state and results, rejects changed run bindings, and never automatically repeats an uncertain external action.
- SC-4: Missing root or dependency grants prevent dispatch; guests request knowledge and current-signal artifacts explicitly, and expired or unauthorized context is excluded.
- SC-5: Real dispatch atomically consumes its Mailbox selection and reserves its resource Lease; busy or uncertain ownership prevents conflicting work.
- SC-6: A guest trap after staging an invocation commits no effect, and pending external work returns to the host without a kernel transport wait.

## Scope

### In Scope

- A locally runnable, single-owner prototype with inspectable Intent, Task, AgentProcess, Artifact, KnowledgeNode, Signal and Lease categories.
- Replaceable business progression and open named abilities, composed provider dependencies, explicit scoped grants and bounded local actions.
- Durable state and actual queue/coalescing dispatch, atomic resource reservations, restart uncertainty, requested knowledge and separated context.
- Typed guest services with staged invocation resources and pending external-work handoff.
- Operator-only knowledge ingestion and one owner scope per state directory.

### Out of Scope

- Completion of Stage 1; full asynchronous syscalls, parallel work and interruption guarantees, live provider hooks or production sandboxing.
- UI, semantic knowledge GC, OKF compatibility, remote installation, multi-tenant access or live grant revocation.
- Guest knowledge/artifact mutation, beyond the imported read services in this increment.
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
| Application, capability invocation, provider dependency, owner scope | agent-os-operation | Add open composition, scoped authority and requested context |
| Pending operation and staged guest resource | agent-os-operation | Distinguish proposed work, durable action and external completion |

## Decisions Required

None. The reviewed increment uses consumer-owned kernel contracts, open scoped dependencies, typed guest read/staging services and real Mailbox/Lease dispatch. One operator scope and operator-only knowledge ingestion remain explicit; none of the mandatory abstraction boundaries are deferred.

## Impact

Operators gain a separate prototype beside the existing product. Existing Go consumers and historical external systems require no migration. The original direction remains preserved in [Agent OS v0.1](../../../docs/design/agent-os-v0.1.original.md); the implementation boundary and deferred guarantees are recorded in [design.md](design.md).
