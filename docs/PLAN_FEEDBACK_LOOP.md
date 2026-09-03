# Run the Plan Feedback Loop

## 1. Runtime and Authority

`arcloom-plan` performs one explicit evaluation of one GitHub milestone. It acquires a fresh Snapshot, reacquires current progress for Delivery Observations, requests a read-only Codex assessment, and delivers that assessment to an operator-owned output reader. It never edits GitHub, applies a Proposed Plan, or schedules another evaluation.

| Input | Required value |
|---|---|
| Operating system | Linux or macOS; other platforms exit 1 |
| Build | Go version declared in `go.mod` |
| `--owner`, `--repo`, `--milestone` | Exact GitHub repository and positive milestone number |
| `--codex` | Absolute executable path for the SDK-compatible Codex version, currently `0.149.1` |
| `--shutdown-grace` | Explicit positive finite SDK shutdown duration, such as `5s` |
| `--model` | Model available to the operator's Codex account |
| `--effort` | `low`, `medium`, `high`, `xhigh`, `max`, or `ultra`; the selected model must support it |
| `--workdir` | Existing absolute directory containing the intended read-only assessment context |
| `--output-fifo` | Existing named pipe with a reader started by the operator |
| `--delivery-evidence` | Optional current UTF-8 regular file with Goal/Acceptance Condition evidence |
| `GITHUB_TOKEN` | Environment-only credential when needed for repository read access; never a command-line flag |

The existing SDK owns executable compatibility, read-only policy, process lifecycle, and bounded shutdown. An unsafe or incompatible session cannot start a Turn. Codex authentication must already be configured. A real assessment sends planning material to Codex and consumes the operator's model usage; deterministic tests require neither credentials nor a live model.

The command's HTTP credential transport authorizes only HTTPS GET requests to `api.github.com`. Codex runs in the operator's environment; that HTTP restriction is not an environment-secret isolation boundary. Use a dedicated read-only credential and assessment directory, and keep secrets out of operator evidence. Treat output as potentially sensitive planning material.

---

## 2. Build and Invoke

The following operator examples are authored instructions, not captured live-run evidence. Replace the absolute paths and model with values from the intended installation. The build and command boundaries are covered by automated tests; the live GitHub/Codex run belongs to the post-merge verification tasks.

In the repository checkout, create a new private run directory and build the executable:

```sh
plan_run_dir=$(mktemp -d)
go build -o "$plan_run_dir/arcloom-plan" ./cmd/arcloom-plan
mkfifo "$plan_run_dir/output.fifo"
printf '%s\n' "$plan_run_dir"
```

In a second terminal, substitute that printed directory and start the reader. This command waits for the writer to open the FIFO:

```sh
cat /absolute/run-directory/output.fifo \
  | tee /absolute/run-directory/evidence.ndjson
```

In the first terminal, with `GITHUB_TOKEN` already supplied through the operator's environment, invoke one cycle:

```sh
"$plan_run_dir/arcloom-plan" \
  --owner kotokumu --repo arcloom --milestone 2 \
  --codex /absolute/path/to/codex \
  --shutdown-grace 5s \
  --model YOUR_AVAILABLE_MODEL --effort medium \
  --workdir /absolute/path/to/arcloom \
  --output-fifo "$plan_run_dir/output.fifo" \
  --delivery-evidence /absolute/path/to/current-delivery-evidence.txt
plan_exit=$?
printf 'arcloom-plan exit: %s\n' "$plan_exit"
```

Omit `--delivery-evidence` when there is no additional current evidence. The file is read afresh after the Snapshot, not used as authoritative Plan state. Supply concrete Goal and Acceptance Condition evidence; a prior assessment or closed Tasks alone does not establish completion.

The process exits after one cycle. A later evaluation requires a new invocation and a newly started reader. Use a new run directory for each retained evidence set. No background loop, automatic retry, or implicit Actor action is present.

---

## 3. Output, Failure, and Cancellation

The FIFO carries newline-delimited JSON. An assessed cycle emits `assessment` first and then `processed_report`, preserving the exact target, assessed Plan, outcome, and any Proposed Plan. A no-current or Attempt Failure cycle emits only `processed_report`. No-current is distinct from `insufficient_information`, which is an assessment of an established current Plan.

| Exit | Meaning |
|---|---|
| `0` | An assessment was handed off successfully and processed evidence was written; this does **not** imply outcome `complete` |
| `1` | Invalid configuration, observation/assessment failure, or output failure |
| `2` | Current Plan is not established; available representation progress remains in the processed record |
| `130` | Caller cancellation, including SIGINT or SIGTERM |

The command does not write diagnostics to inherited stdout/stderr. If it exits 1 without a complete record, verify flag spelling, target values, executable/version, model/effort, directory, evidence-file type/encoding, FIFO, and reader. Invalid arguments are not echoed. SDK/observation failures that produce a Report use stable failure codes rather than raw provider errors.

The command opens only its own independent nonblocking FIFO descriptor. Missing output paths, symlinks, regular files, missing readers, or unsupported deadlines fail before GitHub/Codex work. Output files are never created or truncated. SIGINT/SIGTERM cancels an active blocked write and waits for owned Host/SDK work; the operator retains ownership of the reader. Cancel a hung invocation explicitly if its reader has stopped consuming.

A complete write means that the FIFO accepted the bytes, not that `tee` saved them durably or that the operator acted on them. Cancellation or failure can leave a partial record, a received assessment without processed evidence, or another uncertain destination effect. Keep such evidence marked incomplete; do not infer no effect or replay it automatically. The command does not alter the parent's stdout flags or close the reader.

Each record includes acquisition start/end times, the native target URL, GitHub API version, configured model/effort/executable/workdir/shutdown bound, build metadata, and the SDK-supported Codex version. `codexObservedVersion: null` means the SDK does not expose observed version metadata; it does not claim version validation was skipped. Unavailable build fields remain explicitly unavailable. Snapshot and Delivery acquisition intervals are separate and do not assert an atomic cross-source observation.

---

## 4. Post-Merge Verification

Implementation issues [#51](https://github.com/kotokumu/arcloom/issues/51)–[#56](https://github.com/kotokumu/arcloom/issues/56) are completed by the reviewed implementation PR. Live verification follows the dependency order below and remains separate from that PR's completion.

| Ticket | Required action and evidence |
|---|---|
| [#44](https://github.com/kotokumu/arcloom/issues/44) | After the implementation is merged, build the merged revision and capture a baseline run against milestone 2. Retain complete records, exit status, acquisition times, target identity, configuration, exact executable version output, and build revision. |
| [#45](https://github.com/kotokumu/arcloom/issues/45) | Make at least two explicitly authorized real Actor changes to GitHub state. For each, retain the Actor action and source revision, refresh current evidence, start a new reader, and run a fresh evaluation. Record the observed progress difference; do not use request count or a previous assessment as current facts. |
| [#46](https://github.com/kotokumu/arcloom/issues/46) | After every other milestone Task is complete and its evidence is ready, close this final coordination Issue. Then run a later ordinary Request with a fresh Snapshot showing every Task closed and current evidence for each Acceptance Condition. Verify outcome `complete` for the exact final Plan and retain the result; coordination Issue closure or exit 0 alone is insufficient. |

This implementation PR leaves #44–#46 open. Complete #44/#45 only after their evidence is reviewed. For #46, distinguish the coordination Issue closure before the final fresh observation from the final Complete verification afterward. Completing the implementation PR does not establish live convergence or Milestone 2 completion.
