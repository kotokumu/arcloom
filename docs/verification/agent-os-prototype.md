# Agent OS prototype verification

This validates the limited Stage 1 increment in `add-agent-os-prototype`, not the full original roadmap. The original supplied design is preserved byte-for-byte at [agent-os-v0.1.original.md](../design/agent-os-v0.1.original.md), SHA-256 `2308209098e6f0d4e8fe4fbf357adfbf39604da520bacc496c9f3c91f2f6d4e0`.

## Reproduce

From the repository root on Linux with the pinned Rust toolchain, a C compiler and the example Python interpreter available:

```sh
cargo fmt --all --check
cargo clippy --workspace --all-targets --locked -- -D warnings
cargo test --workspace --locked
bash agent-os/examples/demo.sh
python3 agent-os/tests/e2e.py "${CARGO_TARGET_DIR:-target}/debug/arcloom-os"
npm run lint
npm test
go test ./...
```

## Observed results

| Check | Result |
|---|---|
| Rust tests | 34 passed; no failed or ignored tests |
| Actual CLI and Component tests | 10 passed; no skips |
| Wasmtime example runs | Feedback loop completes after execution and evaluation; observe-only completes with zero driver actions |
| Formatting and Clippy | Passed, with warnings treated as errors for Clippy |
| Markdown and OpenSpec | Passed for main specs, active changes and archived changes |
| Markdown rule tests | 30 passed |
| Existing Go regression | All 14 packages passed |

The CLI suite exercises driver substitution, immutable completed-run resume, binding mismatch, denial before effects, budget exhaustion, invalid application output, WASM fuel exhaustion, timeout uncertainty and simultaneous runner exclusion. Native tests additionally exercise per-Component memory count restrictions, process ownership during cleanup, persisted action uncertainty and completion, corruption detection, knowledge expiry, lease fencing, consumer-independent coalescing and concurrent transition exclusion. Native guest tests specify business transitions separately from host dispatch.

## Review outcomes and limitations

Independent scenario review completed before implementation. Implementation review found and corrected aggregate guest-memory limits, process-group PID reuse during cleanup, and documentation that incorrectly promised application callbacks after uncertain driver failure. Persistence review corrected acknowledgement of a coalesced notification when a newer notification arrives during delivery, and migration transaction atomicity.

No real Codex/Claude session, Hook Shim, Web UI, portable Capability Token, native process sandbox, full asynchronous syscall implementation or OKF compatibility was tested or delivered. Mailbox coalescing and leases have tested persistence APIs; application-selected delivery modes and driver lease acquisition remain unwired. The CLI uses a durable queue and one state-directory host lock. See the [prototype boundary](../../agent-os/README.md#prototype-boundary).

Results describe local validation. GitHub-hosted CI and publication of the OpenSpec change through the archive command are separate follow-up operations.
