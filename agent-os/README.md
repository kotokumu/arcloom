# Arcloom Agent OS prototype

This is a runnable subset of Stage 1 from the [original Agent OS v0.1 proposal](../docs/design/agent-os-v0.1.original.md). It uses Rust, Wasmtime Component Model/WIT, SQLite WAL and a local SHA-256 CAS. The existing Go feedback controllers remain available.

**Run only trusted native driver programs.** WASM applications have no WASI imports and are constrained by fuel and memory. Native command drivers retain the host user's filesystem and network authority; Capability grants gate dispatch but do not sandbox those programs. This prototype is not ready to run hostile agents or third-party native drivers.

## Run the examples

Prerequisites: Linux, a Rust toolchain supported by the locked dependencies, `rustup`, a C toolchain for SQLite, and Python 3 at `/usr/bin/python3` for the example drivers. Run from the repository root:

```sh
bash agent-os/examples/demo.sh
```

The script builds the host and two WASM applications, converts the core WASM files to Components with the embedded WIT metadata, and runs both against a fresh temporary state directory. It prints that directory for later inspection. It does not call a paid model or require credentials.

The first application owns the sequence of text transformation, feedback and completion. It requests named capabilities through imported guest services; these business roles are not kernel effect kinds. The transformation, expected result, provider configuration, grants, budget and runtime limits come from [feedback-loop.json](examples/feedback-loop.json). A negative evaluation causes the application to request another execution, subject to the kernel's action budget. The included transformer is deliberately deterministic; it does not pretend to be a self-improving AI. An impossible expectation demonstrates budget exhaustion.

The second [observe-only application](apps/observe-only/src/lib.rs) completes directly from its Intent. It requires no execution or feedback driver, demonstrating that the kernel does not prescribe the first application's sequence.

## Build and invoke separately

```sh
rustup target add wasm32-unknown-unknown
cargo build --locked -p arcloom-os
cargo build --locked --release --target wasm32-unknown-unknown \
  -p arcloom-feedback-loop -p arcloom-observe-only
build_dir=${CARGO_TARGET_DIR:-target}
mkdir -p agent-os/examples/generated
"$build_dir/debug/arcloom-os" pack \
  "$build_dir/wasm32-unknown-unknown/release/arcloom_feedback_loop.wasm" \
  agent-os/examples/generated/feedback-loop.component.wasm
state_dir=$(mktemp -d)
"$build_dir/debug/arcloom-os" run \
  agent-os/examples/feedback-loop.json "$state_dir" my-run
"$build_dir/debug/arcloom-os" inspect "$state_dir" my-run
"$build_dir/debug/arcloom-os" context "$state_dir"
```

Run IDs are operator-selected within one state directory. Reusing an ID resumes only with the same manifest and Component binding. A completed run does not execute its commands again. Use a new run ID after changing the manifest or application.

## Replace behavior without editing the kernel

| Change | Extension point |
|---|---|
| Goal, input, expected result | Manifest `intent`, interpreted by the selected application |
| Business sequence and retry decision | Replace the Component implementing [business-app.wit](wit/business-app.wit) |
| Capability implementation | A provider's `adapter` and `config`; the kernel sees an open capability invocation |
| Provider dependencies | Provider `requires` resolved against `provides`; each dependency needs an explicit grant |
| Permission and limits | Manifest `grants` and `limits`; missing authority never permits access |
| Notification delivery | Manifest `coalescing` selects replaceable kinds; other Signals use queue delivery |

Manifest version 2 providers declare `id`, `provides`, `requires`, `adapter`, `config` and an optional `resource`. Command execution, direct input identity and dependency delegation implement the same invocation contract. Missing or ambiguous capabilities and dependency cycles fail resolution before work. An application can invoke only its declared roots; each provider receives only handles to its declared dependencies. A dependency declaration does not grant permission. New adapter implementations are registered at composition, without adding provider switches to the kernel.

An omitted provider resource defaults to the invoked capability. Every dependency edge must share its consumer's effective resource; providers composing different capability names must explicitly select the same resource key. The complete composed invocation then uses one durable reservation/fence. Incompatible resource scopes fail configuration before effects. Atomic reservation of multiple different resources is not implemented.

Command drivers receive explicit invocation input and correlation metadata and return one JSON value on stdout. The executable and work directory are resolved from configuration; arguments are passed unchanged and environment variables are explicit. No shell interpolates a command string. Provider payloads are application-owned data; context is not attached unconditionally to every request. See [execute.py](examples/execute.py) and [evaluate.py](examples/evaluate.py).

## Typed guest boundary and requested evidence

The WIT callback accepts a typed Signal and returns a typed transition/outcome. OS-owned variants, error values, knowledge records and resource handles are typed; business state and provider input remain opaque to the kernel.

The guest imports `lease-acquire`, `dispatch`, `knowledge-query` and `artifact-get`. Lease and dispatch calls stage callback-local resources. They do not launch a provider or create a durable reservation. Only a valid dispatch ticket returned by a successful callback can proceed to the kernel's atomic reservation step. A trap or invalid return commits no staged invocation.

Knowledge reads require `knowledge.read` and return nonexpired confirmed and provisional collections separately. Artifact reads require `artifact.read` and are restricted to the artifact attached to the current Signal. The guest requests these services when needed; the runner does not inject owner-wide context into every callback. There are no guest knowledge/artifact writes in this increment.

## Kernel and host responsibilities

The separate `arcloom-kernel` crate consumes its own application, state, Mailbox, Lease, context and dispatch contracts. Its direct dependencies are only `anyhow`, `serde` and `serde_json`; Wasmtime, SQLite and concrete drivers belong to the host crate. Concrete SQLite/CAS, Wasmtime and provider implementations are wired by the host. The kernel has no concrete Journal or CommandDriver dependency and no execution/evaluation effect switch.

Submission returns pending work to the host. The CLI chooses to collect that work and continue the demonstration; provider workers and blocking collection live outside the kernel. This is a pending-operation boundary, not a claim that full parallel fan-out/join or interrupt support is complete.

## Durable state and uncertainty

The runner obtains actual deliveries from its Mailbox. Application state, selected delivery acknowledgement, resource Lease and pending action are committed atomically before a provider is started. A busy reservation leaves the input and application state unchanged. Successful provider results become immutable CAS artifacts and correlated Signals and release only the matching reservation. A restart can resume a committed completion without repeating the external action. If the previous action was pending when the host stopped, its outcome is uncertain: the kernel records this and does not replay it automatically.

Driver errors, timeouts and interrupted effects do not establish that no external effect occurred. Their resource reservation becomes reconciling and remains protected after nominal expiry; explicit operator reconciliation is required before reuse. Inspect retained state before deciding on further work. The local process lock excludes concurrent hosts for one state directory and complements per-resource Leases. External tools are not claimed to enforce fencing tokens.

Configured coalescing groups replaceable notifications by kind and target. Its acknowledgement preserves interleaved queued events and newer notifications received during delivery. A stale completion cannot release another action's reservation.

`inspect` includes the recorded actions, selected provider, resource, fencing token and disposition. After independently verifying an uncertain external outcome, the operator can release only that matching reservation:

```sh
"$build_dir/debug/arcloom-os" inspect "$state_dir" my-run
"$build_dir/debug/arcloom-os" lease-reconcile "$state_dir" my-run RESOURCE TOKEN
```

Replace `RESOURCE` and `TOKEN` with the inspected values. Reconciliation releases the matching reservation; it does not replay the action or turn an uncertain result into confirmed success.

A state directory is one operator-owned scope. Context assembly separates non-expired human/stable knowledge from provisional notes. Knowledge ingestion is operator CLI only in this increment:

```sh
"$build_dir/debug/arcloom-os" knowledge-put "$state_dir" /absolute/path/to/node.json
"$build_dir/debug/arcloom-os" context "$state_dir"
```

## Validation and remaining scope

```sh
cargo test --locked --workspace
python3 agent-os/tests/e2e.py "${CARGO_TARGET_DIR:-target}/debug/arcloom-os"
```

Build and pack the example Components before running the CLI tests, using the demo script above. Current results are recorded in [verification evidence](../docs/verification/agent-os-prototype.md). Mandatory checks include scoped dependency delegation, typed guest imports, denied evidence access, staged-ticket rollback, real Mailbox delivery, atomic Lease reservation, pending handoff and conservative recovery. Run the complete suite after rebuilding both example Components; earlier binaries do not implement the new ABI.

This is not the completed Stage 1. Deferred items include the full sync/async syscall surface, Codex/Claude hook shims and all four standard traps, live-provider authentication, native-process sandboxing, full fan-out/join and interrupt handling, app-side Knowledge writes and OKF v0.2 compatibility. Stage 2 UI and semantic GC, and Stage 3 ecosystem features remain deferred. These omissions do not replace the original design with another architecture; the [implementation design](../openspec/changes/add-agent-os-prototype/design.md) records this increment's limits.

## Prototype boundary

Real Mailbox delivery and resource Lease reservation are part of this increment's dispatch path. Consumer-owned kernel ports, open provider composition, typed guest services and requested context are also required behavior; none is deferred merely as an abstraction to implement later.

Grants are local manifest authority for one operator-owned state directory; portable Capability Token issuance/revocation is deferred. Typed imports cover requested evidence and invocation staging, not every syscall in the original roadmap. No Hook Shim or four-trap integration is implemented. Native drivers are trusted programs with host-user permissions. Normal cleanup terminates their process group and reaps the direct child; abrupt host death kills the direct child on Linux, but grandchild reclamation is not guaranteed.

Guest execution permits one linear memory and two tables, covering the Rust guest and canonical resource adapter. Each table receives half the configured table-entry budget; a third table is rejected. Memory and fuel limits remain enforced. Component compilation has no total host-memory or compilation-time quota. Native driver executables and source files are operator-managed and not hashed into run identity; do not replace them during an existing run. Manifest version 2 and WIT package `arcloom:os@0.2.0` are incompatible with the preceding prototype. Rebuild generated Components and use new run IDs; legacy state remains inspectable but is not silently migrated or resumed with a changed binding.
