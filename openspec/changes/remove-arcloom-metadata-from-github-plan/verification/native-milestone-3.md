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

Pending task 6.4. Authenticated GitHub CLI access succeeds from an approved
execution context on 2026-09-03. The public Snapshot proof must run against the
implementation commit and receive human review. Browser and connector reads
used to capture the deterministic fixture do not satisfy this live proof.

---

## 5. Completion Metadata-Dependency Inventory

Task 5.2 remains pending until official OpenSpec publication. The
pre-publication search on 2026-09-02 found no production `payloadFor` or
`decodePayload` definition or call. Its only runtime-test matches are the
intentional black-box compatibility inputs in:

- `providers/github/plan/native_narrative_observation_test.go`
- `providers/github/plan/snapshot_observation_test.go`

The other matches are this approved change's migration-control records and the
two owning main specifications' accepted payload Requirements. The main
Requirement matches must disappear only through task 6.7's official archive
command; manually removing them during implementation would bypass the
publication workflow. After publication, rerun the frozen command and replace
this pre-publication audit with the final result and reviewer decision.

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
| `go test -race ./...` | Pass, all 11 packages |
| `go vet ./...` | Pass |
| `golangci-lint run` | Pass, 0 issues |
| `npm run lint` | Pass |
| `npm test` | Pass, 30 tests |
| `git diff --check` | Pass |
