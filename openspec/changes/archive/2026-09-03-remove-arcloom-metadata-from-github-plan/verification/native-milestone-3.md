## 1. Verification Scope

This record contains the migration inventory and the deterministic and live evidence required by the approved DesignDoc. It does not make legacy metadata authoritative.

---

## 2. Baseline Metadata-Dependency Inventory

The baseline was captured on 2026-09-02 before production or test implementation changed.

```sh
rg -l 'arcloom-plan:v1|decodePayload|payloadFor' --glob '!openspec/changes/archive/**' | sort
```

| File | Baseline role |
|---|---|
| `providers/github/plan/payload.go` | Production payload authority |
| `providers/github/plan/payload_decode.go` | Production payload authority |
| `providers/github/plan/scheme.go` | Production payload authority |
| `providers/github/plan/snapshot_observer.go` | Production payload authority |
| `providers/github/plan/concurrency_statelessness_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/failure_context_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/native_text_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/pagination_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/payload_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/payload_producer_test.go` | Current test or fixture authority |
| `providers/github/plan/response_shape_observation_test.go` | Current test or fixture authority |
| `providers/github/plan/snapshot_observation_test.go` | Current test or fixture authority |
| `openspec/specs/github-plan-creation-dry-run/spec.md` | Current specification authority |
| `openspec/specs/github-plan-representation-observation/spec.md` | Current specification authority |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/proposal.md` | Approved migration control |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/model.md` | Approved migration control |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/design.md` | Approved migration control |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/tasks.md` | Approved migration control |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/specs/github-plan-creation-dry-run/spec.md` | Approved migration control |
| `openspec/changes/remove-arcloom-metadata-from-github-plan/specs/github-plan-representation-observation/spec.md` | Approved migration control |

The completion inventory will distinguish intentional black-box compatibility inputs from migration-control references and will contain no production or current-specification authority.

---

## 3. Deterministic Milestone 3 Evidence

The fixture was captured from `kotokumu/arcloom` Milestone #3 and Issues #47,
#48, #57, and #58 on 2026-09-02. The repository test does not make a network
request.

| Fact | Captured value |
|---|---|
| Title | `Arcloom Reconciliation Model and Module Finalization` |
| Target Date | Absent |
| Overall state | Open |
| Metadata marker | Absent |
| Task membership | Complete: #47, #48, #57, #58 |
| Task states | All open |

The exact captured native narrative is:

```md
## Goal

Use the deterministic and real Plan Feedback Loop evidence from Milestone #2 to remodel and refactor Reconciliation, then finalize only the modules and interfaces required by verified behavior.

## Acceptance Conditions

### 1

Milestone #2 provides a runnable reference Host, deterministic local test environment, multi-cycle convergence test, and real GitHub Plan evidence.

### 2

The responsibilities of Reconciliation, Control, Feedback Loop composition, Result Destination, External Actor, and Observation are re-audited against that evidence.

### 3

The accepted conceptual model, specifications, and Architecture contain no unresolved responsibility or boundary decision required by implementation.

### 4

The implementation is refactored to the accepted model while the deterministic and real-loop verification remains green.

### 5

Only evidence-backed module responsibilities, boundaries, and consumer-owned interfaces remain.
```

The ordered captured Tasks are:

1. #47 `Re-audit Reconciliation, Control, Feedback Loop, Result Destination, External Actor, and Observation responsibilities.`
2. #48 `Finalize the modules and interfaces required for Controller development.`
3. #57 `Remodel Reconciliation from verified Feedback Loop evidence.`
4. #58 `Refactor Reconciliation to the accepted model without weakening loop behavior.`

The no-network proof is
`TestMilestoneSnapshotObserverReconstructsCapturedMilestoneThree`. It passed as
part of `go test -v -race ./providers/github/plan` on 2026-09-02 and established
the exact current Plan plus complete open progress.

---

## 4. Live Milestone 3 Evidence

The authenticated public Snapshot proof passes against implementation commit
`fab5efe4c2caa3ca3a5461360c9bc6d05ed1c5c9`. The exact observed Plan, progress,
native Task identities, and response evidence are recorded in
`live-milestone-3-result.json`. Goal, all five ordered Acceptance Conditions,
and all four ordered Task names equal the captured values in section 3 byte
for byte. Target Date is absent, root and all Tasks are open, and membership is
Complete. No GitHub resource is changed.

The executed secret-free command is:

```sh
GOWORK=off GOFLAGS= GOCACHE=/private/tmp/arcloom-live-proof-go-cache go run openspec/changes/remove-arcloom-metadata-from-github-plan/verification/live_milestone_3.go --live --implementation-commit fab5efe4c2caa3ca3a5461360c9bc6d05ed1c5c9
```

The operator program uses a no-cookie client, obtains the existing GitHub CLI
credential in memory, makes only target-bound GET requests, and captures each
response body before returning the same bytes to the public Observer. The
command runs from the repository root; after archival, only its source path
changes to the dated archive directory.

| Evidence | Result |
|---|---|
| UTC observation interval | `2026-09-03T04:06:51.893103Z` through `2026-09-03T04:06:52.714904Z` |
| Go runtime | `go1.27.0` |
| GitHub API version, requested and selected | `2022-11-28` |
| Root response | HTTP 200; SHA-256 `0b0ba3a9af5523ba557c7cc355d1a51ef87e84a0d984fdf42a14d7843adc4529` |
| Complete Task-page response | HTTP 200; SHA-256 `e8fe8738eb0bd0789cf947a31c6fa2026402cfec5ae2e3883f685cd85987d8b5` |
| Source authority | Marker absent in the same fetched root; public Snapshot matches native facts |
| Drift | None against the frozen Plan and progress expectation |
| Freshness | Captured after the implementation commit; approval is recorded at `2026-09-03T05:26:08Z`, within the required 24 hours |
| Human reviewer and approval | The task owner, acting as the human reviewer in this Codex conversation, explicitly approves the presented evidence with `承認します` on 2026-09-03 |

The JSON preserves the execution-time verdict. The approval above completes
task 6.4; it does not alter the captured response or observation data.

---

## 5. Completion Metadata-Dependency Inventory

The frozen command from section 2 returns exactly these three files after
official publication on 2026-09-03:

```text
openspec/specs/github-plan-representation-observation/spec.md
providers/github/plan/native_narrative_observation_test.go
providers/github/plan/snapshot_observation_test.go
```

The occurrence-level audit uses
`rg -n 'arcloom-plan:v1|decodePayload|payloadFor' --glob '!openspec/changes/archive/**'`.

| Allowlisted file and line | Occurrences | Role and authority |
|---|---|---|
| `openspec/specs/github-plan-representation-observation/spec.md:259` | 1 | Black-box compatibility input in `native-narrative-meaning`; explicitly ignored, not a reconstruction authority |
| `providers/github/plan/native_narrative_observation_test.go:97` and `:98` | 2 | Legacy and dated legacy inputs that verify native-only authority |
| `providers/github/plan/snapshot_observation_test.go:111` | 1 | Legacy Snapshot input that verifies native-only authority |

There are zero production producer or decoder references. Every remaining
non-archive occurrence describes an intentional compatibility input, and no
current fixture or specification treats metadata as authoritative. Migration
records and the operator's marker-absence assertion remain only in archived
history. Independent reviewer `final_implementation_review` approves this
inventory on 2026-09-03 and confirms no remaining authoritative metadata
Requirement or production reference.

---

## 6. Post-implementation Review

Architecture, SOLID responsibility, interface, procedural-code, code-quality,
and Go test-specification reviews cover the implementation. The pre-PR review
findings are addressed by these changes:

- Add Snapshot coverage for invalid and duplicate Acceptance Conditions.
- Add Issue-specific legacy Target Date conflict coverage and ordinal
  no-resume coverage.
- Centralize private native narrative grammar and Milestone `due_on` text
  interpretation.
- Treat a Target Date opening without a value/final frame as Unavailable rather
  than panicking.

The final review's Medium Target Date boundary finding is resolved after the
task owner's approval on 2026-09-03. Conditions remain Incomplete until the full
date opening is established; earlier fully framed members remain observable.
Five public regression cases cover missing post-heading framing, truncation at
the heading, truncation after one LF, a malformed empty-collection boundary,
and preservation of only earlier fully framed members. All five fail against
`ab008de0548a64da93984e374b120721b5c94eed` and pass with the correction:

```sh
GOCACHE=/private/tmp/arcloom-live-proof-go-cache go test ./providers/github/plan -run '^TestNativeNarrativeObservationIssueTargetDateStates$' -count=1
```

Independent reviewer `final_implementation_review` approves the correction and
operator proof code with no remaining actionable findings on 2026-09-03. The
operator proof rejects untracked product files and resolves the supplied
revision to its full commit SHA. This agent verdict is not human approval of
live evidence.

| Verification after correction | Result |
|---|---|
| `go mod tidy -diff` | Pass in a clean archive of `fab5efe4c2caa3ca3a5461360c9bc6d05ed1c5c9`, excluding local `node_modules` |
| `go test -race ./...` | Pass, all 11 packages |
| `go vet ./...` | Pass |
| `golangci-lint run` | Pass, 0 issues |
| `npm run lint` | Pass |
| `npm test` | Pass, 30 tests |
| `git diff --check` | Pass |

---

## 7. Requirement Publication

Publication uses the official command:

```sh
openspec archive remove-arcloom-metadata-from-github-plan --yes
```

The creation delta groups all four modified Requirements under one
`MODIFIED Requirements` section. This preserves every approved Requirement
verbatim and prevents the CLI from omitting an earlier duplicate section.
The generated main specifications contain two additions, seven modifications,
and two removals. An exact Requirement-body comparison verifies all eleven
operations, all six untouched Requirements, and unchanged Conceptual Models.
No Requirement is manually merged into a main specification.

The added Requirements are `github-plan-narrative` and
`native-narrative-meaning`. The removed Requirements are
`versioned-plan-narrative` and `versioned-payload-meaning`. The generic proposal
heading warning is non-blocking for the validated `quality-spec` schema.
Independent publication review approves the exact delta publication and
retention checks. Post-publication `npm run lint:openspec` passes on
2026-09-03: all eleven main specifications and all seven archived changes
validate, with all twenty-two tasks complete. Markdown lint, the thirty
Markdown-rule tests, `git diff --check`, and all eleven Go packages under the
race detector also pass after publication.
