# Milestone 2: Repeated Real Attempts

## 1. Result and Scope

[Issue #45](https://github.com/kotokumu/arcloom/issues/45) has two successful later ordinary Requests after two real external GitHub changes. Each uses the same reviewed runtime as the [baseline](https://github.com/kotokumu/arcloom/blob/main/docs/verification/milestone-2/44/README.md), with fresh Snapshot and later Delivery acquisition. Each produces its own exact Retain assessment, assessed Report, `await_request` directive, and exit 0.

| External change | Later command interval (UTC) | Exact progress difference from the preceding Snapshot |
|---|---|---|
| PR #63 merges at `14:29:29`; #44 completes at `14:29:30` | `2026-09-03T14:33:15.307Z`–`2026-09-03T14:33:25.993Z` | Only the #44 Task changes from open to closed: 7 → 8 closed |
| Milestone description loses only its obsolete metadata comment at `14:34:34` | `2026-09-03T14:34:44.157Z`–`2026-09-03T14:34:51.696Z` | Empty difference; exact Plan and progress remain unchanged at 8 closed / 2 open |

All events occur on 2026-09-03 UTC. The second change is a real source mutation, not another Task completion. The accepted native-description observation contract requires legacy-metadata removal to preserve the same Plan/Snapshot. [Accepted specification](https://github.com/kotokumu/arcloom/blob/main/openspec/specs/github-plan-representation-observation/spec.md).

Issue #45 and final verification #46 remain open during both observations. These two Retain results establish repeated feedback, not final Complete or Milestone completion.

---

## 2. Evidence and Reproduction

`attempt-1-records.ndjson` and `attempt-2-records.ndjson` preserve the complete bytes received by each new FIFO reader. Each matching manifest records its exact command, runtime and input/output hashes, timestamps, and exit status. Matching input files are the current operator facts used by that invocation; historical verification events are labeled as such and are not substituted for current Plan or Task state.

`operations.json` records the deliberate actions and observed GitHub timestamps. `milestone-before.json` contains the selected fields from a fresh GitHub GET. `milestone-after.json` contains the same fields from the actual PATCH response. Their descriptions and hashes establish that only the leading encoded comment is removed; the readable Goal and all five Acceptance Conditions are byte-identical. The saved original description also permits restoration of that comment if required.

The operator uses the same preserved collector, exec-only launcher, fixed Codex 0.149.1, immutable Host source `0a2fcb254a55cf47aef6915522a8012609eba7e9`, safe configuration and shared login-cache assumptions as the baseline. Neither Attempt performs a GitHub mutation. The operator launches each command explicitly after its corresponding Actor change; there is no automatic wake-up, scheduler, retry, or previous-result replay.

Run `node --test docs/verification/milestone-2/evidence.test.mjs` to check raw hashes, current-Plan association, complete Task membership, ordered changes/Requests/acquisitions, and both exact progress differences. This validates retained evidence rather than performing new live Requests. For a new real run, use the [operator runbook](https://github.com/kotokumu/arcloom/blob/main/docs/PLAN_FEEDBACK_LOOP.md) with fresh directories and current evidence.

---

## 3. Corrected Administrative Error

GitHub interprets a negated closing-keyword phrase in PR #63 as an automatic #45 closure at `2026-09-03T14:29:31Z`. The PR body is corrected and the unfinished Issue is reopened at `2026-09-03T14:32:01Z`, before either later Request. This transient closure/reopening is excluded from the two deliberate proof changes. Both retained Snapshots correctly show #45 open.

Subsequent PRs verify that GitHub's `closingIssuesReferences` contains only the intended Issue before merge. [Correction record](https://github.com/kotokumu/arcloom/issues/45).

---

## 4. Completion Gate

The verification plan has an independent acceptance review: two real external changes are required; two nonzero progress increases are not. The final evidence requires independent review and CI before the #45 PR merges. Final convergence remains [Issue #46](https://github.com/kotokumu/arcloom/issues/46).
