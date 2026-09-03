# Milestone 2 Baseline: SDK Configuration Correction

## 1. Risk, Problem, Goal, and Context

Risk: medium, using the existing-boundary responsibility check. This is a correction to the existing `safe-host-configuration` guarantee, not a new public API or external protocol contract. It changes the SDK's private construction of an existing `thread/start` configuration. No product Concept, ownership, state lifecycle, or package boundary changes. A new OpenSpec change is unnecessary because accepted behavior is unchanged.

Issue #44 requires one successful real GitHub/Codex Plan Attempt after PR #59 is merged. The live invocation on source `6318ca9fbc4a53c29767c0c0ec6a59106ea17877` obtains current GitHub facts but fails at the AI boundary.

### Current Behavior

Codex 0.149.1 reports every configured MCP server disabled during `config/read`. The SDK then supplies an empty `mcp_servers` map in `thread/start`. The live server subsequently reports those same servers as `starting`. The SDK rejects the unexpected notifications and produces no successful assessment. The trace includes a `turn/start` request; it does not prove rejection before that request.

### Desired Behavior

The accepted safe configuration remains effective when the assessment Thread starts. Explicitly disabled named MCP and Apps settings are not erased by an empty map. Unsafe input continues to reject before a Thread or Turn.

### Constraints and Approved Assumptions

- Keep Codex 0.149.1, the existing SDK Client/CompletedTurn contracts, approval `never`, read-only sandbox, disabled agent network, and finite shutdown unchanged.
- Do not whitelist MCP startup or tool notifications to make a live run pass.
- Do not copy credentials, endpoints, commands, or environment values from effective configuration into the Thread configuration.
- Do not modify the operator's global configuration or weaken the safe-configuration validator.

---

## 2. Requirements

### Requirement Summary and Functional Requirements

| ID | Required behavior | Binary acceptance criterion |
|---|---|---|
| FIX-1 | Preserve denial of each named MCP/Apps entry admitted by safe configuration | A server that rejects missing named denials completes one valid Turn with the SDK; each name is explicitly disabled |
| FIX-2 | Preserve fail-closed safety and exact assessment association | Unsafe MCP/Apps/Hooks/Web Search still starts no Thread; unchanged material produces the exact CompletedTurn |
| FIX-3 | Complete the real baseline, not only the regression tests | The reference command emits one assessment and its matching processed Report with fresh Snapshot and later Delivery acquisition, exit 0 |

### Non-Functional Requirements

The existing cancellation bound, process reaping, per-call isolation, and no-credential-output guarantees remain unchanged. No new performance threshold or storage lifetime is introduced.

### Inputs and Outputs

Inputs are the already validated effective configuration and existing read-only Turn request. Output is a private Thread configuration containing only enforced safety values. Public output remains CompletedTurn or the existing interaction error.

### Normal Cases

Empty configuration and explicitly disabled named MCP/Apps configurations complete the same read-only interaction.

### Error Cases

Enabled, implicitly enabled, or malformed configuration rejects under the existing safety rules. Unknown notifications remain errors.

### Edge Cases

Null maps, empty maps, multiple names, named null entries, and Apps `_default` retain explicit non-enablement. No authority-bearing configuration details cross into request material.

### Non-Goals

No new notification support, authentication mechanism, configuration profile manager, Host API, retry, Plan mutation, or milestone completion shortcut.

### Risks and Open Questions

Effective configuration and Thread-level overrides have distinct merging behavior in the real server. Deterministic wire tests must cover preservation of denial; a successful live run is also required. The independent design review must confirm that this correction preserves the accepted contract before production code changes.

---

## 3. Responsibility and Interface Traceability

| Changed behavior | Existing owner | Information and authority | State/invariant | Contract impact | Verdict |
|---|---|---|---|---|---|
| Preserve disabled settings when constructing Thread configuration | Codex app-server SDK | Validated effective configuration and fixed read-only policy | No Thread override relaxes accepted safety | Private implementation only; Client signature and error semantics unchanged | PASS, independent interface review |

The existing SDK boundary owns Provider configuration and protocol behavior. Plan Control and Host remain consumers of the unchanged Client contract and receive no configuration details. The smallest representation is a private stateless configuration projection, not another object, Port, or lifecycle.

Public Interface additions and non-obvious new call sites: N/A; none are proposed. Migration: N/A; no persisted data changes. Rollback is an ordinary Git revert, with no replay of earlier assessments.

---

## 4. Test Specification and TDD Plan

| Requirement | Given | When | Then | Verification owner |
|---|---|---|---|---|
| FIX-1 | Stub reports a disabled named MCP configuration and rejects a Thread override that erases denial | SDK completes a read-only Turn | Exact CompletedTurn succeeds and denial remains present | SDK process tests |
| FIX-1 | Stub reports disabled named Apps, including `_default`/null partitions | SDK starts a Thread | Every admitted name remains explicitly disabled | SDK process tests |
| FIX-2 | Existing unsafe/malformed configuration, cancellation, concurrency, and wire fixtures | Existing test suite runs | Previous rejection, isolation, safety, and cleanup expectations remain true | SDK and full Go tests |
| FIX-3 | Merged implementation plus the reviewed correction and safe operator configuration | One explicit live command invocation | Exact target, current Plan, acquisition provenance, assessment, processed Report, and exit 0 are captured | Issue #44 operator |

The existing generated table-driven process-test scaffold is retained. `gotests -only CompleteReadOnlyTurn -use_go_cmp` confirms the function already has tests. Add regression cases without changing the CompletedTurn expected-value type. Cover multiple names, absent/empty/null maps, named null entries, invalid map and entry shapes, omitted/invalid/enabled flags, non-forwarding of synthetic secrets, and isolation between calls. The regression server emits startup notifications after the Thread response when named denials are missing, matching the observed ordering.

Construction order: make the external stub enforce the observed denial-preservation rule; record Red; implement the smallest private projection; run Green and the complete SDK suite; simplify duplicated configuration parsing if needed; independently review; rebuild and perform the real baseline. Do not replace the live verification with a stub result.

---

## 5. Current-System Evidence

Evidence packet: `M2-44-CONFIG-v1`.

| Fact | Source | Relevance |
|---|---|---|
| Exact target and safe baseline obligation | [Issue #44](https://github.com/kotokumu/arcloom/issues/44) | Live acceptance |
| Non-relaxable safety before Turn | [Accepted specification](https://github.com/kotokumu/arcloom/blob/main/openspec/specs/codex-plan-control-assessment/spec.md) | Existing guarantee, not a new requirement |
| Empty maps in Thread configuration | [SDK implementation](https://github.com/kotokumu/arcloom/blob/main/providers/codex/appserver/stdio_client.go) | Current realization |
| Config read precedes Thread start; named servers subsequently start | Local protocol-type-only trace from 2026-09-03T13:55:11Z invocation | Reproduced operational defect |
| Explicit CLI configuration layering and Thread overrides | [Official App Server documentation](https://learn.chatgpt.com/docs/app-server) and local Codex 0.149.1 execution | External behavior evidence |

### Construction Log

- Design review: PASS from independent `host_interface_review`; existing SDK ownership and Medium risk are appropriate. This verdict excludes relaxing rejection rules or allowing new notifications.
- Red: disabled MCP and Apps process cases fail with `interaction failed` against the empty-map implementation. The absent-configuration case also detects missing Apps default denial.
- Green and regression verification: SDK suite, `go test ./...`, `go test -race ./...`, `golangci-lint run ./...`, `npm run lint`, and `npm test` pass with the correction. The same Client processes the table cases without carrying names between calls.
- Independent implementation review: PASS from `host_interface_review`, with no actionable P1/P2 finding. The review confirms unchanged rejection and public error contracts, secret non-forwarding, per-call isolation, and no new boundary or state owner; SDK normal/race tests and diff checks pass independently.
- Live baseline: source `0a2fcb254a55cf47aef6915522a8012609eba7e9`, invocation `2026-09-03T14:21:25.013Z`–`2026-09-03T14:21:32.720Z`, exit 0, exact Retain assessment plus assessed Report and Await Request directive. Failed invocations are not successful evidence.
