# Agent OS prototype verification

This verifies the limited Agent OS increment with consumer-owned kernel ports, open scoped capability composition, typed guest services, real Mailbox/Lease dispatch and pending work outside the kernel. It does not claim completion of the original Stage 1 roadmap. The [original supplied design](../design/agent-os-v0.1.original.md) is unchanged: SHA-256 `2308209098e6f0d4e8fe4fbf357adfbf39604da520bacc496c9f3c91f2f6d4e0`.

## Reproduce

From the repository root on Linux with the pinned Rust toolchain, a C compiler and the example Python interpreter available:

```sh
cargo fmt --all --check
cargo clippy --workspace --all-targets --locked -- -D warnings
cargo test --workspace --locked
cargo tree -p arcloom-kernel --depth 1 --locked
bash agent-os/examples/demo.sh
python3 agent-os/tests/e2e.py "${CARGO_TARGET_DIR:-target}/debug/arcloom-os"
npm run lint
npm test
go test ./...
```

## Current results

| Check | Result |
|---|---|
| Rust workspace tests | 61 passed: 7 guest, 44 host, 4 kernel integration and 6 manifest tests |
| Actual CLI and typed Component suite | 17 passed; zero failures or skips; 50.599 seconds |
| Actual Component demos | Both passed: feedback loop invokes execution and feedback; observe-only completes without a provider sequence |
| Kernel dependency boundary | Independently checked with cargo tree: direct dependencies are only anyhow, serde and serde_json |
| Formatting and Clippy | Passed; Clippy treats warnings as errors |
| Markdown and OpenSpec | Passed for main specs, active changes and archived changes |
| JavaScript validation | 30 Markdown-rule tests and 4 evidence tests passed |
| Legacy Go regression | All 14 packages passed |

## Requirement evidence

| Guarantee | Verification path |
|---|---|
| Consumer-owned kernel ports | Separate arcloom-kernel crate compiles without Wasmtime, SQLite or command adapters; integration tests substitute a dispatcher |
| Open capability composition | Real CLI direct/command/delegated routes and registry tests; arbitrary capability names, scoped dependency handles and explicit grants |
| Graph/resource validity | Missing, ambiguous and cyclic graph rejection; incompatible dependency resource domains rejected before work |
| Typed guest boundary | Real Components use the imported services and typed outcomes; native tests cover staged ticket validity and guest failures |
| Requested context | A kernel test rejects unsolicited knowledge queries; typed guest read tests enforce grants and current-Signal artifact scope |
| Atomic dispatch progress | Real kernel uses task Mailbox and LeaseManager; busy reservation preserves input/state; completion records evidence and releases only matching ownership |
| Queue/coalescing delivery | Interleaved queued events and notifications arriving during delivery remain pending; independent consumer progress is durable |
| Pending work and uncertainty | Kernel returns pending before completion; host collects outside it; unresolved actions retain reconciling resource protection beyond nominal expiry |
| Operator reconciliation | Inspection exposes provider, resource, token and disposition; exact matching reconciliation releases ownership without replay or invented success |
| Evidence substitution | A non-filesystem evidence backend is exercised through normal completion and knowledge-source checks; default CAS verifies immutable bytes |
| Resource limits | Real typed Components require two tables; each receives half the entry budget, a third table and second memory are rejected, and fuel/command limits remain tested |
| Durable compatibility | Changed manifest/Component binding cannot resume old state; completion is not replayed; original design checksum remains unchanged |

Independent requirement-level review traced the shipping CLI through guest imports, kernel ports, provider registry and persistence transactions. It confirmed normal-path integration rather than merely the existence of isolated abstractions. The review also checked the separate kernel dependency graph and the single-resource constraint for composed providers. Manifest version 2 and WIT package `arcloom:os@0.2.0` explicitly identify the incompatible contract revision; old state remains inspectable and new behavior uses new run IDs.

## Limits

No real Codex/Claude session, Hook Shim, four-trap integration, Web UI, portable Capability Token, native process sandbox, full asynchronous syscall set, parallel fan-out/join or OKF compatibility was tested or delivered. Guest services currently cover requested evidence and invocation staging; guest knowledge/artifact mutation is excluded. Native drivers retain host-user authority. See the [prototype boundary](../../agent-os/README.md#prototype-boundary).

Results describe local validation. GitHub-hosted CI and official OpenSpec archival remain separate operations; this work does not merge or publish the change.
