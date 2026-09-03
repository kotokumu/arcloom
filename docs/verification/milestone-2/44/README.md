# Milestone 2: Real Baseline

## 1. Result

[Issue #44](https://github.com/kotokumu/arcloom/issues/44) has one successful real GitHub/Codex Attempt on the reviewed correction commit `0a2fcb254a55cf47aef6915522a8012609eba7e9`, based on merged Host commit `6318ca9fbc4a53c29767c0c0ec6a59106ea17877`. This is baseline evidence, not Milestone completion.

| Item | Captured value |
|---|---|
| Target Identity | `github-milestone`, `kotokumu/arcloom/milestones/2` |
| Snapshot acquisition | `2026-09-03T14:21:25.486329Z`–`2026-09-03T14:21:26.359157Z` |
| Later Delivery acquisition | `2026-09-03T14:21:26.359163Z`–`2026-09-03T14:21:27.146115Z` |
| Current progress | Complete membership, 7 closed Tasks and 3 open Tasks (#44–#46) |
| Assessment | `retain`, exact current Plan association, no Proposed Plan |
| Published Report / directive | `assessed` / `await_request` |
| Command completion | `2026-09-03T14:21:32.720Z`, exit 0, no signal |
| Output | Two complete NDJSON records, no inherited stdout/stderr output |

`baseline-records.ndjson` preserves the bytes received by the FIFO reader, including the full Plan, all Task progress, assessment, Report, directive, and provenance. `baseline-manifest.json` records the exact command, times, input/output/launcher/binary hashes, and process outcome. `operator-input.json` contains current verification facts supplied to Delivery Observations; no earlier Plan or assessment is used as current state.

The command uses the public Controller and Plan Attempt path. Its GitHub transport permits only GET requests, and it has no Authorization or Plan-application wiring. The `processed_report` represents the Controller's published completion; automated Controller tests establish publication-before-directive commitment. Neither the output nor this document claims an external mutation or durable exactly-once delivery.

---

## 2. Runtime and Reproduction

`runtime-preflight.json` records observed version/configuration and the official download digest. The exact Codex binary reports `codex-cli 0.149.1`; the Host binary carries `vcs.modified=false`. The SDK does not expose its observed version, so the output's `codexObservedVersion` remains null. The manifest's version and user-agent values come from separate operator preflight commands, not an added SDK field.

`codex-launcher.sh` and `capture.mjs` preserve the operator files used for this run. Their absolute temporary paths describe this invocation; they are not portable installation defaults. The collector starts an owned FIFO reader, keeps a writer descriptor open without writing bytes, invokes the command once, and closes that keeper after command exit. It records all received bytes without changing the assessment or retrying.

For a new run, follow the [operator runbook](https://github.com/kotokumu/arcloom/blob/main/docs/PLAN_FEEDBACK_LOOP.md), build the identified source revision, use new private directories, and substitute the actual executable, evidence, and FIFO paths. Reacquire current operator evidence rather than reusing the baseline input as current facts. The repository is private and requires a GitHub credential; the launcher removes `GITHUB_TOKEN` and `GH_TOKEN` before Codex starts.

The dedicated Codex state directory has mode 0700 and is separate from the empty assessment directory. Only the verification child receives its documented state-root setting; the parent environment and global configuration remain unchanged. A local `auth.json` symlink reuses the existing ChatGPT login. This is shared authentication-cache use, including normal token refresh, **not** authentication-write isolation or cross-process refresh serialization. Credentials and runtime databases are excluded from this evidence set.

The isolated state produces no effective MCP servers or Hooks, disabled Apps, and disabled Web Search. The SDK independently checks this configuration and fixes read-only sandbox, no agent network, and approval `never`. This operator setup does not relax any SDK rejection rule. [Official state-root configuration](https://learn.chatgpt.com/docs/config-file/environment-variables), [authentication cache behavior](https://learn.chatgpt.com/docs/auth).

---

## 3. Correction and Verification

The SDK correction preserves verified named MCP/Apps denials when starting a Thread, without forwarding connection details or secrets. Deterministic regression tests establish the named-configuration correction; the successful isolated live run establishes the end-to-end baseline. These are distinct evidence scopes.

`failed-before-fix.ndjson` is the captured `2026-09-03T13:53:05Z` failure on merged source `6318ca9`. It contains one `ai_boundary_failure` Report and no assessment. Other setup findings are the system Codex update to 0.153.0 and nonempty user hook-trust metadata that the unchanged validator rejects. A pinned official executable and dedicated state directory resolve those operator constraints without editing personal configuration.

Verification commands that pass for the SDK correction:

- `go test ./...`
- `go test -race ./...`
- `golangci-lint run ./...`
- `npm run lint`
- `npm test`

`node --test docs/verification/milestone-2/evidence.test.mjs` verifies retained bytes and hashes, exact assessment association, complete progress, outcome/directive, build identity, and ordered acquisition intervals. It validates the captured artifacts, not a new live run or a cryptographic attestation of execution. Independent review covers the implementation and operational configuration; the PR also requires evidence review and CI before merge.

---

## 4. Remaining Work

[#45](https://github.com/kotokumu/arcloom/issues/45) requires at least two later ordinary Requests, each after an actual external GitHub change, with fresh observations and exact progress differences. [#46](https://github.com/kotokumu/arcloom/issues/46) requires every Task closed, separate evidence for each Acceptance Condition, and a later exact Complete assessment. The Retain baseline does not satisfy either final obligation.
