## 1. Implementation

- [x] 1.1 `[[agent-os-operation/replaceable-business-progression]]` Provide the reviewed Rust/WIT application and driver contracts; verify two different applications and interchangeable execution providers without host workflow edits.
- [x] 1.2 `[[agent-os-operation/durable-action-outcomes]]` Implement durable run binding, action lifecycle and inspection with SQLite WAL; verify pending/completed restart boundaries, changed-binding rejection and exclusive local runner ownership.
- [x] 1.3 `[[agent-os-operation/explicit-dispatch-authority]]` Enforce exact grants and single-owner context boundaries; verify denied dispatch creates no process or action artifact and no cross-directory context is supplied.
- [x] 1.4 `[[agent-os-operation/bounded-local-work]]` Enforce application and command resource limits and action budgets; verify invalid output, fuel exhaustion, timeout, output overflow and budget exhaustion.
- [x] 1.5 `[[agent-os-operation/independent-signal-delivery]]` Implement durable independent queue/coalescing mailboxes; verify filtered delivery, acknowledgement persistence and consumer independence.
- [x] 1.6 `[[agent-os-operation/immutable-artifact-evidence]]` Implement immutable local CAS; verify identical content identity and rejection of corrupt bytes.
- [x] 1.7 `[[agent-os-operation/filtered-knowledge-context]]` Implement operator knowledge ingestion, expiry and separate context packs; verify expiry equality and human/stable versus draft separation.
- [x] 1.8 `[[agent-os-operation/local-resource-leases]]` Implement local lease acquisition/release; verify active-owner exclusion, expiry and stale release rejection.
- [x] 1.9 `[design-only]` Document build/run/inspect steps and deferred original-roadmap guarantees; verify a fresh local walkthrough and preservation of the original design document.

## 2. Verification and Publication

- [x] 2.1 `[[agent-os-operation/replaceable-business-progression]]` Run the prototype tests and runnable application/driver examples; record reproducible results without claiming full Stage 1.
- [x] 2.2 `[[agent-os-operation/durable-action-outcomes]]` Run failure and restart scenarios and inspect durable evidence; retain the legacy Go regression result.
- [ ] 2.3 `[design-only]` Publish the approved reader-facing Conceptual Model from model.md into the new owning main spec during apply; validate all main specs, leaving Requirement merging to the official archive command.
- [x] 2.4 `[design-only]` Run Markdown lint and `openspec validate add-agent-os-prototype --strict --no-interactive`; resolve all errors and review scope consistency. Full `npm run lint` passed, including all main, active and archived specs.
- [ ] 2.5 `[design-only]` When the change is ready to archive, use `openspec archive add-agent-os-prototype --yes` and `npm run lint:openspec`; do not mark publication complete before both succeed.

Publication note: [specification rules section 8](../../specs/README.md#8-delta-and-publication) reserves Requirement merging for the official archive command. A new model-only main spec fails strict validation because a spec must have at least one Requirement. The complete approved Conceptual Model therefore remains in model.md until publication can accompany archival; no invalid main spec or premature Requirement copy is retained.

Local validation evidence: [Agent OS prototype verification](../../../docs/verification/agent-os-prototype.md). Tasks 2.3 and 2.5 are publication operations deferred to official archival; the proposed change remains open for PR review.
