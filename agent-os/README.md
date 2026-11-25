# Arcloom Agent OS prototype

This is a runnable subset of Stage 1 from the [original Agent OS v0.1 proposal](../docs/design/agent-os-v0.1.original.md). It uses Rust, Wasmtime Component Model/WIT, SQLite WAL and a local SHA-256 CAS. The existing Go feedback controllers remain available.

**Run only trusted native driver programs.** WASM applications have no WASI imports and are constrained by fuel and memory. Native command drivers retain the host user's filesystem and network authority; Capability grants gate dispatch but do not sandbox those programs. This prototype is not ready to run hostile agents or third-party native drivers.

## Run the examples

Prerequisites: Linux, a Rust toolchain supported by the locked dependencies, `rustup`, a C toolchain for SQLite, and Python 3 at `/usr/bin/python3` for the example drivers. Run from the repository root:

```sh
bash agent-os/examples/demo.sh
```

The script builds the host and two WASM applications, converts the core WASM files to Components with the embedded WIT metadata, and runs both against a fresh temporary state directory. It prints that directory for later inspection. It does not call a paid model or require credentials.

The first application owns the sequence of text transformation, feedback and completion. The transformation, expected result, driver commands, grants, budget and runtime limits come from [feedback-loop.json](examples/feedback-loop.json). A negative evaluation causes the application to request another execution, subject to the kernel's action budget. The included transformer is deliberately deterministic; it does not pretend to be a self-improving AI. An impossible expectation demonstrates budget exhaustion.

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
| Goal, input, expected result | Manifest `intent`, interpreted only by the selected application and drivers |
| Business sequence and retry decision | Replace the Component implementing [business-app.wit](wit/business-app.wit) |
| Agent or execution provider | Replace a named execution driver's `command`; preserve its JSON input/output contract |
| Feedback criterion implementation | Replace a named feedback driver's `command`; the application interprets its response |
| Permission and budget | Manifest `grants` and `limits`; absent grants never authorize dispatch |

A driver receives one JSON object on stdin containing `input`, `intent`, `context`, `action` and `generation`. It writes one JSON value to stdout and exits successfully. The executable and work directory are resolved by the manifest loader; subsequent arguments are passed unchanged. Environment variables are explicit. A command string is never interpolated through a shell. See [execute.py](examples/execute.py) and [evaluate.py](examples/evaluate.py).

An application receives its committed state and a Signal through WIT, then returns its next state and one effect. Protocol effect names are fixed contracts; business targets, provider selection and outcome criteria are not kernel constants. The application chooses `execute`, `evaluate`, `wait` or `complete`; a required capability must resolve unambiguously and be granted before dispatch.

## Durable state and uncertainty

Application state, consumed Signal and pending action are committed atomically before a command is started. Successful command results become immutable CAS artifacts and correlated Signals. A restart can resume a committed completion without repeating the external action. If the previous action was pending when the host stopped, its outcome is uncertain: the kernel records this and does not replay it automatically.

Driver errors, timeouts and interrupted effects do not establish that no external effect occurred. Inspect the retained state before deciding on further work. The local process lock excludes concurrent hosts for one state directory; SQLite records local leases and fencing counters but does not claim that external tools enforce those tokens.

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

Build and pack the example Components before running the CLI tests, using the demo script above. The CLI suite runs without conditional skips and uses temporary state directories. It exercises completion and resume, the observe-only workflow, provider substitution, denial before effects, budget exhaustion, binding mismatch, WASM fuel exhaustion, timeout uncertainty and concurrent host exclusion.

Native application tests cover completion, retry decisions, malformed signals and the different observe-only workflow. Host tests cover the actual WIT component boundary, authorization, persistence and process lifecycle; building and running the demo exercises the real Component artifacts and subprocess drivers together.

This is not the completed Stage 1. Deferred items include the full sync/async syscall surface, Codex/Claude hook shims and all four standard traps, live-provider authentication, native-process sandboxing, full fan-out/join and interrupt handling, app-side Knowledge writes and OKF v0.2 compatibility. Stage 2 UI and semantic GC, and Stage 3 ecosystem features remain deferred. These omissions do not replace the original design with another architecture; the [implementation design](../openspec/changes/add-agent-os-prototype/design.md) records this increment's limits.

## Prototype boundary

The business-app runner currently consumes its durable queue. The independently tested Mailbox API supports consumer filters and coalescing, but application manifest delivery-mode selection is not wired into the runner yet. Lease acquisition, expiry and fencing are available in the persistence API; driver dispatch currently relies on the single-host state-directory lock rather than acquiring a resource Lease. Neither feature is claimed as complete Stage 1 integration.

Grants are local manifest authority for a single operator-owned state directory; portable Capability Token issuance/revocation is deferred. The Component ABI currently returns one effect per callback, not the full synchronous/asynchronous syscall surface. No Hook Shim or four-trap provider integration is implemented. Native drivers are trusted programs with host-user permissions. On normal error/timeout their process group is terminated; on abrupt host death the direct child is killed on Linux, but grandchild reclamation is not guaranteed.

Guest execution permits one linear memory and one table, with configured memory, table-size and fuel limits. This is not a total host compiler-memory or compilation-time quota. Source files used by trusted native commands are operator-managed and are not hashed into run identity; do not replace them during an existing run.
