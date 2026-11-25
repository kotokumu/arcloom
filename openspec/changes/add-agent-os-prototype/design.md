## Context

The preserved [Agent OS v0.1 original](../../../docs/design/agent-os-v0.1.original.md) governs the new direction. This local prototype retains Rust, Wasmtime Component Model/WIT, SQLite WAL and local CAS beside the existing Go product. The strengthened increment requires real consumer-owned kernel ports, open capability invocation, provider dependency composition, typed guest services and Mailbox/Lease integration in actual dispatch. These abstraction requirements are implementation acceptance conditions, not deferred roadmap items.

## Goals / Non-Goals

### Goals

Run replaceable business applications against interchangeable capability implementations. Persist application progress and independently consumed signals; reserve resources before effects; expose pending external work without waiting inside the kernel. Let applications request only the knowledge and artifacts they need through explicitly granted typed guest services. Preserve uncertainty rather than replaying possible external effects.

### Non-Goals

Completion of Stage 1, full synchronous/asynchronous syscall coverage, parallel fan-out/join, live Codex/Claude hooks and four standard traps, native-driver sandboxing, portable token issuance, live grant revocation, multi-host coordination, UI, semantic knowledge GC and OKF compatibility. Native programs remain trusted local programs with host-user authority. Guest knowledge/artifact writes are not included in the current imported service surface; knowledge ingestion remains an operator operation.

## Conceptual Model and Minimality

| Concept | Decision and state owner | Invariant |
|---|---|---|
| Intent | Operator owns immutable requirements snapshot per run | Guest cannot commit or overwrite it |
| Task | Kernel owns durable run binding and budget; guest owns business state | Resume requires identical application/configuration binding |
| AgentProcess | Kernel owns durable action identity and disposition; provider owns external effect | Pending intent precedes dispatch; no uncertain replay |
| Artifact | Owner-scoped CAS owns immutable content | Successful reads verify digest and guest reads require exact evidence scope |
| KnowledgeNode | Operator owns ingestion and trust; context service owns filtering | Expired knowledge is excluded; confirmed and draft collections remain separate |
| Signal | Kernel owns append-only occurrence and correlation | Acknowledgement and next application state commit atomically |
| Mailbox | Consumer owns subscription and delivery progress | Coalescing one kind/target cannot discard queued occurrences of another |
| Lease | Reservation authority owns resource exclusion and fencing | A possibly running or uncertain effect retains reconciliation protection |
| Capability invocation | Application requests named ability; selected provider owns implementation | Kernel has no execution/evaluation/provider switch |
| Provider dependency | Declaring provider owns required ability scope | Only explicitly declared and granted dependencies are callable |
| Guest lease/ticket | Callback-local host resources stage proposed work | A trapped or rejected callback cannot dispatch or acquire a durable lease |

Task and AgentProcess retain different restart meanings; Signal and Mailbox have different ownership; Artifact and Knowledge differ in mutability and authority. Lease cannot be folded into the host lock because it protects the dispatched resource and possible effects, not just a local runner. The graph exists for real provider delegation, not speculative extension. Guest handles protect capability and callback lifetimes rather than representing a second durable process model.

## Responsibilities, Boundaries and Dependencies

The kernel consumes contracts for application transitions, run state and atomic commits, Mailbox delivery, Lease reservation, capability resolution, dispatch and requested context. Those contracts belong to the consumer and expose domain values rather than SQLite, Wasmtime or command-process types. The kernel does not import or construct concrete Journal or CommandDriver implementations. Its contracts and decisions live in the separate `arcloom-kernel` crate at `agent-os/kernel`, whose direct dependencies are only `anyhow`, `serde` and `serde_json`. Concrete storage, Component and provider adapters live outside that crate, implement the contracts and are wired by the CLI composition root. This dependency boundary is checked by compiling the kernel independently and inspecting its dependency graph.

The CLI host also owns the scheduling choice after the kernel returns pending work. It may collect a pending result to run the local demonstration, but this wait is outside the kernel. Its transport scheduling does not decide business success, retries or transitions.

| Boundary | Consumer and protected constraint | Details kept outside the kernel |
|---|---|---|
| Application transition | Kernel needs a substitutable business decision | Wasmtime stores, bindings and guest resources |
| Durable run/Mailbox/Lease contracts | Kernel needs atomic progress and exclusion | SQL schema, transactions and filesystem layout |
| Capability resolution/dispatch | Kernel needs explicit authority and pending outcomes | Adapter configuration, process management and graph implementation |
| Requested context | Guest needs authorized evidence on demand | Knowledge queries, CAS paths and digest implementation |

The implementation may group related contracts, but shared transaction ownership must not become duplicated policy. SQLite transaction authority belongs to the persistence adapter implementing the atomic reservation contract. The journal consumes an injected immutable evidence-store contract, with LocalCas as the default adapter; completion, artifact reads and knowledge-source validation all use that same backend. Artifact bytes are persisted and verified before the journal reference commits. A failed transaction may leave an unreferenced immutable blob but cannot commit a reference before its content exists. The kernel supplies the selected delivery, state and proposed invocation; it does not reconstruct persistence transactions.

## Decisions

### Typed guest services and staged invocation

The authoritative ABI is [business-app.wit](../../../agent-os/wit/business-app.wit). The exported callback uses typed Signal, Transition and outcome records/variants. Imported services expose requested knowledge, artifact reads, resource acquisition and dispatch staging. Business state and provider input remain application-owned opaque data; OS-owned discriminants and handles are typed.

Lease acquisition and dispatch imports create callback-local lease/ticket resources. They do not spawn work, acknowledge a Signal or acquire a durable Lease. Only a valid ticket returned in the successful transition can become an open capability invocation. A trap, malformed return, invalid resource or denied request cannot commit those staged effects. Guest memory/fuel limits and an empty ambient WASI surface remain enforced.

Knowledge and artifact access require explicit grants. Context is requested through services rather than injected unconditionally into every callback or provider request. Knowledge reads remain within the run's single operator scope and retain expiry and confirmed/draft separation. Artifact reads additionally require the exact artifact attached to the current Signal. No arbitrary CAS enumeration or filesystem access is exposed.

### Open capability graph and scoped implementations

A manifest declares application requirements, explicit grants and providers with provided abilities, required dependencies and adapter configuration. There is no fixed execution or feedback provider slot. Command transport, direct input identity and dependency delegation are concrete substitutable implementations of the same invocation contract; the kernel interprets none of their configuration.

Resolution rejects missing dependencies, ambiguous providers and cycles before work. The application's directly callable abilities are its declared requirements. A provider receives handles only for its declared dependencies, not the full registry. Every root and transitively requested dependency still requires an explicit grant; declaring a dependency never grants it. Each provider may declare its resource key; absent that declaration, its effective resource is the invoked capability. Every dependency edge must share its consumer's effective resource. Composition across distinct capability names therefore declares the same explicit resource on the participating providers. Missing resource agreement is rejected before effects. One root reservation/fencing token covers the complete dependency invocation; independent resource graphs remain separate. Atomic reservation of multiple resources is outside this increment.

Run binding fixes the resolved manifest and Component content. Native executable/source bytes remain operator-managed and are not included in this content binding.

### Mailbox selection, reservation and completion

The runner consumes its real consumer Mailbox, including configured queue/coalescing behavior. Coalescing groups explicitly selected replaceable notifications by kind and target. Acknowledgement tracks the delivered selection rather than blindly jumping over interleaved queued Signals. A newer notification arriving during delivery remains pending.

Reservation atomically commits the selected Mailbox acknowledgement, next application state, pending action and resource Lease. A busy resource commits none of them, leaving the input available. The recorded action retains its root capability, selected provider, generation and fencing identity before external submission. The run binding fixes the resolved manifest; dependency scope is enforced by its scoped handles rather than duplicated as a graph in every action record. The same transaction authority protects guest wait/complete transitions and their acknowledgements without inventing an external action.

The kernel submits through a dispatch port and returns pending work to the host. Transport workers and pending completion collection remain outside the kernel. Completion is correlated to the recorded action and generation; it records immutable artifact evidence and the Signal and releases only the corresponding current reservation. A late or repeated result cannot acknowledge unrelated work or release a newer holder's Lease.

A restart or dispatch failure without an established outcome records uncertainty and a stall Signal for operator inspection. Its reservation becomes reconciling; ordinary expiry cannot make a possibly active effect safe to duplicate. Explicit operator reconciliation is required to release that reservation. Inspection exposes action provider, resource, token and disposition; `lease-reconcile` frees only the matching uncertain reservation after the operator verifies the external outcome, without replaying work or rewriting the recorded result. Automatic replay remains prohibited. Terminal uncertainty is not a promised application failure callback.

### Bounded local operation

Guest execution permits one linear memory and two tables for the Rust guest and canonical resource adapter. Each table receives half the total table-entry budget, and a third table is rejected. Memory and fuel remain bounded; Component compilation has no global memory/time quota. Commands use argv, explicit environment and workdir, deadline and bounded output without shell interpolation. Normal cleanup kills the owned process group and reaps its direct child. Abrupt host death kills the direct child on Linux, but grandchild reclamation is not guaranteed. A finite kernel action budget prevents unlimited accepted invocations; exhaustion persists budget_hit and its Signal.

## Independent Change Scenarios and Review

The independent reviewer identified these concrete propagation tests without assuming Class or Package structure:

1. Replace a command provider with a direct implementation or provider delegation: kernel policy and application business progression remain unchanged.
2. Add a dependency that lacks a provider or grant, or forms a cycle: reject before external work; no dependency can enlarge authority.
3. Change a guest to request knowledge only after feedback: earlier callbacks and unrelated providers receive no unconditional context.
4. Trap after staging a guest ticket: no dispatch, durable Lease or acknowledgement occurs.
5. Interleave a queued result with coalesced notifications and append another notification during delivery: acknowledging the selection loses none of the remaining work.
6. Contend for the same resource and interrupt its holder: one reservation wins atomically; pending/uncertain work remains protected until reconciliation.
7. Complete an old pending handle after its ownership changes: correlation/fencing rejects it without releasing current ownership.

The review accepted the ownership direction with these transactional constraints. Guest writes are outside this increment, so its read imports and staged invocation resources do not require a second guest-write transaction mechanism. Full Stage 1 omissions do not waive any of the mandatory abstraction boundaries above.

## Contracts and Verification

| Contract | Observable proof |
|---|---|
| Consumer-owned kernel ports | Kernel can operate with alternative state/dispatch implementations and contains no concrete Journal/CommandDriver dependency |
| Open graph | Direct, command and delegated implementations; missing/ambiguous/cyclic dependency rejection; denied dependency cannot execute |
| Typed WIT and staging | Real Components call imported services; denied access and trapped/invalid callback create no committed effect |
| Requested context | No implicit context; explicit knowledge/artifact grants and exact current-Signal artifact scope |
| Atomic reservation | Busy Lease leaves state/ACK unchanged; accepted invocation commits action, Lease and selected Mailbox progress together |
| Pending work | Kernel returns pending without collecting transport completion; exact completion correlation |
| Mailbox and Lease integration | Real runner coalescing with interleaved queue events; competing dispatch and reconciling recovery |
| Existing invariants | CAS verification and substituted evidence backend through real completion/context paths, expiry separation, budget limits, binding mismatch and no replay |

## Migration / Rollback

Keep the existing Go API and tests intact. Manifest version 2 and WIT package `arcloom:os@0.2.0` are incompatible with the preceding prototype. Rebuild example Components and use new run IDs; do not mutate bindings of historical runs. Legacy state remains available for inspection, not automatic migration into the new ABI. Keep generated Components and local state outside tracked files. The original design file remains byte-identical. Publication of the new main specification remains governed by the official archive command.

## Risks / Trade-offs

Native drivers retain host-user authority. Local grants do not issue portable Capability Tokens. Durable external execution means committed progress and conservative uncertainty, not snapshotting guest stacks or exactly-once effects. One local host owns a state directory; remote coordination and complete fan-out/join remain outside this increment.

## Open Questions

None blocking the reviewed responsibility and transaction model. Exact typed ABI and adapter contracts remain authoritative in their code/IDL rather than duplicated field schemas in this document.
