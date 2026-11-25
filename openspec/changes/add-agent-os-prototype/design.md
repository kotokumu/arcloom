## Context

The user's Agent OS v0.1 is the governing new direction. This increment implements a local Stage 1 prototype beside the legacy Go implementation. It preserves Rust, Wasmtime Component Model/WIT, SQLite WAL, local CAS, external execution/feedback drivers and application-owned transitions. No Web UI, semantic GC, production sandbox, OKF compatibility, or complete Stage 1 claim is made.

## Goals / Non-Goals

### Goals

Run interchangeable WASM business applications with persisted application state, correlated execution/feedback signals, explicit scoped grants, bounded subprocess execution, restart detection and inspection from a CLI. Include the seven object categories, queue/coalescing mailboxes, leases, knowledge expiry and separated context packs at minimum useful depth.

### Non-Goals

Live provider authentication, full hook interception for Codex/Claude, remote installation, UI, automatic replay of uncertain effects, multi-host coordination and semantic knowledge GC. External command drivers are explicitly trusted local programs, not sandboxed untrusted agents. This limitation cannot silently fall back to granting sandbox capabilities.

## Conceptual-Model-to-Implementation Mapping

| Concept | Owner / representation | Invariant |
|---|---|---|
| Intent | Operator, immutable JSON snapshot per run | Applications cannot commit or overwrite it |
| Task | Kernel run record with app hash, config hash, budget, state and status | A run resumes only under identical binding; one local runner holds an OS file lock |
| AgentProcess | Kernel action record with generation and pending/completed/uncertain status | Persist before spawn; pending after restart becomes uncertain, never auto-replayed |
| Artifact | CAS addressed by SHA-256 | Writes are immutable and reads verify digest |
| KnowledgeNode | SQLite record, source hashes, status/trust/expiry | Knowledge writes are operator CLI only in this increment; context separates human/stable from drafts |
| Signal | Append-only log, correlated run and action IDs | Consumption and next app state commit atomically |
| Lease | SQLite expiry/fencing counter | Expired holders cannot release newer leases; this prototype does not claim external fencing |
| Mailbox | Consumer cursor and filtered queue/coalescing selection | Durable cursor; coalescing only explicitly selected signals |

Removal analysis: Task and AgentProcess have different restart semantics; Artifact and Knowledge differ in mutability and authority; Signal and consumer cursor have different ownership; Lease protects independent resource ownership. No extra universal workflow model is introduced.

## Decisions

### Decision: Consumer-owned contracts and application-owned transitions

Rust crate `agent-os` has `model` (JSON value contracts and manifest validation), `journal` (SQLite/CAS/knowledge/mailbox/lease persistence), `drivers` (bounded external JSON subprocess transport), `application` (WIT/Wasmtime adapter), and `kernel` (authorization and action lifecycle). CLI is composition and dispatch only. These are Rust modules, not seven separate frameworks.

WIT world `business-app` exports `on-signal(state: string, signal: string) -> string`. State and signal strings carry versioned JSON; result contains next state and exactly one tagged effect: wait, complete, execute or evaluate. Driver requests receive the opaque application payload plus immutable Intent and filtered Context. The kernel never interprets business input, success criteria or retry policy. Application artifacts own those decisions. A second application must demonstrate changed workflow without kernel edits.

This increment has one operator-owned scope per state directory, not multiple tenants. Context queries and artifacts never cross state directories. The app receives only its run Intent, its own Signal and that owner-scoped filtered Context. It has no arbitrary filesystem, CAS or Knowledge query imports. Knowledge ingestion is operator CLI only; app-side knowledge write syscalls are deferred rather than falsely exposed.

Execution and feedback drivers are selected by manifest names and provides/requires capability identifiers, not provider switches. Missing or ambiguous provider selection is rejected before work. Manifest grants authorize exact named driver capabilities, with no default grants. Commands use argv arrays, explicit environment, explicit workdir, deadline and output bound; no shell interpolation. External programs are trusted and keep host-user authority; full process isolation is deferred, visibly.

### Decision: Persistence before side effects

App transition, input acknowledgement and pending action record commit in one SQLite transaction. A completed action creates an immutable artifact and correlated Signal in a transaction. Restart with a pending action marks uncertainty and stops without replay. Restart after completion consumes its persisted Signal without rerunning the command. Configuration and component content hashes bind the run. Per-state-directory file locking excludes concurrent hosts. Durable execution means resuming committed transitions, not snapshotting WASM stacks or claiming exactly-once external effects.

### Decision: Bounded runtime

WASM has no WASI imports, receives only explicit data, and has memory/fuel limits. A command timeout kills its process group and reaps the direct child on Linux. Kernel budget caps accepted external actions; an exhausted budget persists a terminal budget_hit status and signal. Driver failure records terminal uncertainty and a stall Signal for operator inspection, without another application callback or automatic retry. Invalid app output never causes an external action. Untrusted subprocess sandboxing and live four-trap provider integration remain incomplete.

## Contracts and Test Specification

| Contract | Consumer / protected constraint | Evidence |
|---|---|---|
| App on-signal | Kernel / workflow substitution, no ambient WASI | Two components; invalid output; fuel exhaustion |
| Driver invocation | Kernel / provider-independent payload and bounded process lifecycle | Two execution scripts; feedback script; timeout and output overflow |
| Journal transition | Kernel / atomic state, cursor and pending action | Restart at pending and completed boundaries; binding mismatch |
| Capability check | Kernel / fail-closed named grants | Denied driver has no process or artifact side effect |
| CAS and knowledge | Context assembly / provenance and expiry | Digest verification, draft expiry, separate stable/draft contexts |
| Lease | Resource users / fenced local ownership | Expiry and stale release tests |
| Mailbox | Apps / queue vs coalescing delivery | Consumer independence and persisted cursors |

Independent scenario analysis completed by /root/scenarios under DEVELOPMENT.md section 4.3: replacing agent, application, workflow and provider must not change unrelated owners; late feedback must retain correlation; crash uncertainty must not be replayed; permissions must be checked at dispatch. Scenario-driven limitations: frozen manifests do not support live grant revocation; changes require new runs. No incompatible artifact or config may resume a historical run.

## Risks / Trade-offs

- Trusted native driver programs can escape the intended scope: this is a local prototype, explicitly not a security boundary for hostile executables.
- No external operation idempotency: interruption is terminal uncertainty, requiring operator investigation.
- JSON inside WIT allows flexible payloads but validates envelopes at runtime; it is not the final full syscall ABI.
- Seven object categories are minimal; production trap routing, full async syscall surface and OKF v0.2 are not implemented.

## Migration / Rollback

Keep the legacy Go API and tests intact. Document the new scope explicitly in PRODUCT.md and ARCHITECTURE.md. Removing the new crate does not modify historical external systems. Keep prototype state outside tracked files.

## Open Questions

None blocking this limited increment. Deferred Stage 1 features are explicitly listed above, not substituted with alternate designs.

## Implementation review clarifications

Mailbox queue/coalescing and Lease fencing are implemented and independently tested persistence contracts. Runner integration of manifest-selected coalescing and resource Lease acquisition remains deferred; the runner uses a durable queue and a state-directory OS lock. Portable Capability Token issuance/revocation, Hook Shim and full syscall imports remain deferred. These omissions do not replace the v0.1 design; they delimit this increment.

The Wasmtime execution boundary permits one memory and one table and bounds their sizes and execution fuel. Component compilation resources are not globally quota-controlled. Native driver errors remain uncertain because side effects may have occurred. Driver source/executable content is operator-managed and not included in the manifest/component binding hash.
