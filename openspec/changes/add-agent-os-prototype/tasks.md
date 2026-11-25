## 1. Implementation

- [x] 1.1 `[design-only]` Make kernel contracts consumer-owned and remove concrete Journal/CommandDriver coupling; verify kernel dependencies and operation with substitute port implementations.
- [x] 1.2 `[[agent-os-operation/replaceable-business-progression]]` Replace fixed execution/evaluation effects with open capability invocation; verify different business applications without kernel workflow changes.
- [x] 1.3 `[[agent-os-operation/scoped-capability-composition]]` Implement acyclic provider resolution and declared dependency handles with explicit grants; verify direct, command and delegate providers plus missing, ambiguous, cyclic and unauthorized dependencies.
- [x] 1.4 `[[agent-os-operation/requested-guest-services]]` Implement typed WIT callback/services and callback-local staged resources; verify actual Components, on-demand evidence, exact artifact scope, denied imports and trap-after-staging rollback.
- [x] 1.5 `[[agent-os-operation/explicit-dispatch-authority]]` Enforce root/dependency authority and evidence-read grants at actual call boundaries; verify no unauthorized provider work or disclosure.
- [x] 1.6 `[[agent-os-operation/independent-signal-delivery]]` Wire actual runner delivery through configured queue/coalescing Mailboxes; verify interleaved queued events, newer notifications during delivery and independent consumer acknowledgements.
- [x] 1.7 `[[agent-os-operation/local-resource-leases]]` Atomically reserve resource, action, state and Mailbox progress; verify busy resources preserve input, exact completion releases ownership and uncertainty blocks expiry takeover until reconciliation.
- [x] 1.8 `[[agent-os-operation/pending-invocation-handoff]]` Return pending work from kernel submission and collect it in the host; verify slow work returns pending and stale completion cannot release another reservation.
- [x] 1.9 `[[agent-os-operation/durable-action-outcomes]]` Preserve application/configuration binding, durable state and conservative recovery across the new contracts; verify pending/completed boundaries and no uncertain replay.
- [x] 1.10 `[[agent-os-operation/bounded-local-work]]` Retain guest/command limits and action budget through the open invocation path; verify exhaustion, malformed output, timeout and output overflow.
- [x] 1.11 `[[agent-os-operation/immutable-artifact-evidence]]` Retain immutable verified CAS behind the new contracts; verify corruption rejection and exact guest evidence scope.
- [x] 1.12 `[[agent-os-operation/filtered-knowledge-context]]` Retain expiry and separate confirmed/draft collections through requested knowledge access; verify no unconditional context injection and expiry equality.
- [x] 1.13 `[design-only]` Update README, architecture and verification claims to the implemented mandatory boundaries; verify fresh build/run/inspect instructions and unchanged original design checksum.

## 2. Verification and Publication

- [x] 2.1 `[[agent-os-operation/replaceable-business-progression]]` Run the updated prototype tests and real Component/provider examples; record results without claiming complete Stage 1.
- [x] 2.2 `[[agent-os-operation/durable-action-outcomes]]` Run transaction, uncertainty, lease contention and stale-completion scenarios; retain legacy Go regression evidence.
- [ ] 2.3 `[design-only]` Publish the approved reader-facing Conceptual Model from model.md into the new owning main spec during apply; validate all main specs, leaving Requirement merging to the official archive command.
- [x] 2.4 `[design-only]` Run full Markdown/OpenSpec lint after implementation and documentation settle; review all strengthened requirements against implementation evidence.
- [ ] 2.5 `[design-only]` When the change is ready to archive, use `openspec archive add-agent-os-prototype --yes` and `npm run lint:openspec`; do not mark publication complete before both succeed.

Publication note: [specification rules section 8](../../specs/README.md#8-delta-and-publication) reserves Requirement merging for the official archive command. A new model-only main spec fails strict validation because a spec must have at least one Requirement. The complete approved Conceptual Model therefore remains in model.md until publication can accompany archival; no invalid main spec or premature Requirement copy is retained.

Current evidence is recorded in [Agent OS prototype verification](../../../docs/verification/agent-os-prototype.md). The real Component CLI suite passed all 17 cases without failures or skips. The change remains open; publication and merge are not requested by this work.
