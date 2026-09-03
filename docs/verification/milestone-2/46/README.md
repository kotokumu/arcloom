# Milestone 2: Final Complete Assessment

## 1. Result

[Issue #46](https://github.com/kotokumu/arcloom/issues/46) has a successful later ordinary Request with a fresh, complete GitHub Snapshot and an exact `complete` assessment. All ten Tasks are closed. The assessed Plan is identical to the current Plan and to the baseline/repeated Plan; no Proposed Plan is present.

| Event or result | Captured value (UTC) |
|---|---|
| Prerequisite proof PR #64 merged | `2026-09-03T16:07:40Z` |
| Issue #45 completed | `2026-09-03T16:07:41Z` |
| Final coordination Issue #46 completed | `2026-09-03T16:09:20Z`; every other Task is already complete |
| New ordinary command | `2026-09-03T16:09:38.119Z`–`2026-09-03T16:09:46.420Z` |
| Fresh Snapshot acquisition | `2026-09-03T16:09:38.128794Z`–`2026-09-03T16:09:39.030798Z` |
| Later Delivery acquisition | `2026-09-03T16:09:39.030799Z`–`2026-09-03T16:09:39.870969Z` |
| Assessment / processed Report / directive | `complete` / `assessed` / `await_request` |
| Process and output | Exit 0, no signal, two complete NDJSON records, no inherited stdout/stderr bytes |

The GitHub Milestone remains administratively open in this observation (`progress.overallState` is `open`). This does not contradict semantic Plan completion: the ten Tasks are closed and separate Goal/Acceptance Condition evidence supports Complete. Milestone closure follows review and merge of this proof; neither Issue closure nor exit 0 substitutes for the assessment.

---

## 2. Acceptance Evidence

`operator-input.json` preserves the exact caller-owned Delivery evidence supplied to the command. It maps each of the five current Acceptance Conditions separately, including exact requirement text and source/test names. It contains no previous Snapshot or Report as current Plan state and does not preselect an outcome.

| Condition | Verified evidence |
|---|---|
| 1. Explicit triggers and target-specific semantic Results | [Host](https://github.com/kotokumu/arcloom/blob/main/internal/planhost/host.go) starts no Attempt before Trigger and submits only its binding's identity. [Plan-owned delivery](https://github.com/kotokumu/arcloom/blob/main/controllers/plan/assessmentdelivery/delivery.go) preserves exact Assessments. Idle lifecycle, pending-trigger and every-assessment-occurrence tests protect the boundary. No Host scheduler, reconciliation policy or external mutation decision is added. |
| 2. Deterministic local environment | [Host integration tests](https://github.com/kotokumu/arcloom/blob/main/internal/planhost/host_test.go) use a test-owned Planning Context, a current-facts assessor, channels and `testing/synctest`; they require no credential, external service/process, wall-clock sleep or manual operation. This claim concerns that local environment, not live verification or SDK process tests. |
| 3. Fresh multi-cycle convergence | `TestHostConvergesOnlyAfterActorChangesAndAcceptance` verifies two explicit Actor changes, source revisions 0/1/2 in Snapshot and assessment, exact Result delivery, and Retain → Retain → Complete. `TestHostCurrentFactsDetermineAllOutcomes` rejects closed Tasks without acceptance evidence as insufficient information. |
| 4. Baseline and two later real Attempts | The reviewed [baseline](https://github.com/kotokumu/arcloom/blob/main/docs/verification/milestone-2/44/README.md) and [two later Attempts](https://github.com/kotokumu/arcloom/blob/main/docs/verification/milestone-2/45/README.md) use the same public contracts and immutable runtime. The later Requests follow actual external changes; their exact progress differences are retained, including the empty metadata-only difference. [PR #63](https://github.com/kotokumu/arcloom/pull/63) and [PR #64](https://github.com/kotokumu/arcloom/pull/64) are merged before final coordination closes. |
| 5. Development environment for Milestone 3 | The [runnable command](https://github.com/kotokumu/arcloom/tree/main/cmd/arcloom-plan), local integration tests, [operator runbook](https://github.com/kotokumu/arcloom/blob/main/docs/PLAN_FEEDBACK_LOOP.md), complete real-run artifacts and CI integrity checks are available. This establishes readiness for evidence-driven remodelling/refactoring, not completion of Milestone 3 work. |

Independent acceptance audit for conditions 1, 2, 3 and 5 reports no P1/P2 findings. It reruns Host, command, assessmentdelivery and attempt packages with `go test -race -count=10`. The independent Issue #45 evidence review passes before PR #64 merges.

---

## 3. Provenance and Verification

`final-records.ndjson` preserves the complete bytes from the owned FIFO reader. `final-manifest.json` records the exact invocation, times, input/output/launcher/binary hashes and exit outcome. `task-closure.json` preserves selected native GitHub fields from operator GETs before and after Issue #46 closure; these records are separate from the command's subsequent fresh Snapshot.

The immutable Host revision is `0a2fcb254a55cf47aef6915522a8012609eba7e9` with `vcs.modified=false`, identical in Go sources and module files to prerequisite main revision `aefb5d5b6b8b2472ac08149dd1987f367a0d12af`. Its binary, launcher and pinned Codex hashes match the [baseline runtime evidence](https://github.com/kotokumu/arcloom/blob/main/docs/verification/milestone-2/44/runtime-preflight.json). A new preflight confirms Codex 0.149.1, no MCP servers or Hooks, disabled Apps and Web Search. The unchanged SDK independently validates effective configuration before its Turn. The same isolated state directory/shared existing authentication-cache assumptions apply; no credentials are included.

Verification commands:

- `go test ./...` and `go test -race ./...`: all packages pass on unchanged product sources.
- `golangci-lint run ./...`: no issues.
- `npm run lint` and `npm test`: Markdown, all 12 accepted specs, all 8 archived changes and 30 Markdown-rule tests pass; there is no active OpenSpec change.
- `node --test docs/verification/milestone-2/evidence.test.mjs`: the Issue #46 case first fails without its retained evidence, then all four cases pass with the exact captured artifacts. It checks hashes, all ten Task identities/states, prerequisite/closure/acquisition order, all five acceptance mappings and exact Complete/Plan/Report association.

Artifact tests validate retained bytes and relationships; they do not rerun a live model, cryptographically attest execution or independently decide semantic acceptance. Source inspection, behavioral tests, the real assessment and independent review provide those distinct evidence layers. The Host's report-before-directive rule is covered by `TestControllerCommitsDirectivesOnlyAfterCompletionPublication`; acquisition timestamps alone do not prove that internal publication order.

---

## 4. Publication Boundary

Issue #46 coordination closure precedes the final Request, as required by its Done conditions. Final proof publication is a dedicated Issue #46 PR with independent evidence review and passing final-head CI. The operator closes Milestone 2 only after that PR is merged. No product behavior, public contract or accepted OpenSpec requirement changes in this evidence-only work.
