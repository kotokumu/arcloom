# Real Plan Proof Record

This disposable, operator-owned record captures evidence for the `control-real-github-plan` real Plan proof. It is not authoritative external state and does not establish causality between an application request and later GitHub facts.

## Status

- Initial GitHub Plan Snapshot: established
- Initial Codex assessment: `Revise`
- Exact Revision authorization: `Authorized`
- External Actor receipt: `ReceiptAcknowledged`
- Later GitHub Plan Snapshot: established
- Final Codex assessment: `Complete`

## Stable Target Reference

| Field | Exact value |
|---|---|
| Context | `github:kotokumu/arcloom` |
| Identity | `milestone:1` |
| Native target | <https://github.com/kotokumu/arcloom/milestone/1> |

## Initial GitHub Plan Snapshot

The public `githubplan` Snapshot Observer established this Snapshot from one fresh GitHub observation at `2026-08-29T13:20:20Z`.

| Plan field | Exact value |
|---|---|
| Name | `Arcloom Real Plan Proof` |
| Goal | `Demonstrate one real GitHub Plan can be controlled to completion through Arcloom's public contracts.` |
| Acceptance condition 1 | `An initial current Plan and progress are established from a fresh observation of this Milestone.` |
| Acceptance condition 2 | `A later fresh observation establishes the resulting current Plan independently of the application request.` |
| Acceptance condition 3 | `Named Goal and acceptance-condition evidence supports a Complete Codex assessment.` |
| Task 1 | `Define the exact authorized revision` |
| Target date | absent |

Progress was `open`; membership was complete; Task 1 was `open`.

## Initial Codex Assessment

The public `codexplancontrol` adapter invoked the public `codexappserver.Client` contract with Codex app-server `0.149.1`, model `gpt-5.6-sol`, reasoning effort `high`, and the exact current Plan above.

The caller-owned observation material was:

```json
{
  "target": {
    "url": "https://github.com/kotokumu/arcloom/milestone/1"
  },
  "representationProgress": {
    "overall": "open",
    "membershipComplete": true,
    "tasks": [
      {
        "name": "Define the exact authorized revision",
        "state": "open"
      }
    ]
  },
  "proofRequirementsNotRepresentedAsTasks": [
    "Obtain ReceiptAcknowledged from one currently authorized external Actor interaction for the exact Revision.",
    "Establish the resulting current Plan through a later fresh GitHub observation.",
    "Record named Goal and acceptance-condition evidence and obtain a Complete assessment."
  ]
}
```

The assessment outcome was `Revise`. The proposed Plan was:

| Plan field | Exact value |
|---|---|
| Name | `Arcloom Real Plan Proof` |
| Goal | `Demonstrate one real GitHub Plan can be controlled to completion through Arcloom's public contracts.` |
| Acceptance condition 1 | `An initial current Plan and progress are established from a fresh observation of this Milestone.` |
| Acceptance condition 2 | `A later fresh observation establishes the resulting current Plan independently of the application request.` |
| Acceptance condition 3 | `Named Goal and acceptance-condition evidence supports a Complete Codex assessment.` |
| Task 1 | `Define the exact authorized revision.` |
| Task 2 | `Obtain ReceiptAcknowledged from one currently authorized external Actor interaction for the exact Revision.` |
| Task 3 | `Establish the resulting current Plan through a later fresh GitHub observation.` |
| Task 4 | `Record named Goal and acceptance-condition evidence and obtain a Complete assessment.` |
| Target date | absent |

## Exact Revision Authorization

The exact Revision is the Stable Target Reference, Initial GitHub Plan Snapshot, and proposed Plan recorded above. The operator authorized that complete subject with the exact user message `承認します`.

| Authorization field | Evidence |
|---|---|
| Reference | `codex-task:user-message:2026-08-29T13:32:06Z:承認します` |
| Current at evaluation | yes |
| Subject binding | exact Target Reference, current Plan, and proposed Plan |
| Policy decision | `Authorized` |

The public `authorization.Policy` recalculated this evidence for the exact Revision immediately before the application request. A different Revision would have been denied.

## External Actor Receipt

The public `planapplication.RequestApplication` contract invoked one external Actor once for the Authorized Revision. The Actor freshly checked the current GitHub Plan and its native Task mapping before mutation.

| Receipt field | Evidence |
|---|---|
| Actor invocations | `1` |
| Kind | `ReceiptAcknowledged` |
| Acknowledged at | `2026-08-29T13:34:08Z` |
| Native reference 1 | <https://github.com/kotokumu/arcloom/issues/25> |
| Native reference 2 | <https://github.com/kotokumu/arcloom/issues/26> |
| Native reference 3 | <https://github.com/kotokumu/arcloom/issues/27> |
| Native reference 4 | <https://github.com/kotokumu/arcloom/issues/28> |

This receipt establishes only that the Actor acknowledged the exact request. It does not establish mutation, current GitHub state, or Plan completion.

## Later GitHub Plan Snapshot

The public `githubplan` Snapshot Observer established this Snapshot through a new GitHub observation at `2026-08-29T13:38:40Z`. The observation was independent of the application request and receipt.

| Plan field | Exact observed value |
|---|---|
| Name | `Arcloom Real Plan Proof` |
| Goal | `Demonstrate one real GitHub Plan can be controlled to completion through Arcloom's public contracts.` |
| Acceptance condition 1 | `An initial current Plan and progress are established from a fresh observation of this Milestone.` |
| Acceptance condition 2 | `A later fresh observation establishes the resulting current Plan independently of the application request.` |
| Acceptance condition 3 | `Named Goal and acceptance-condition evidence supports a Complete Codex assessment.` |
| Task 1 | `Define the exact authorized revision.` |
| Task 2 | `Obtain ReceiptAcknowledged from one currently authorized external Actor interaction for the exact Revision.` |
| Task 3 | `Establish the resulting current Plan through a later fresh GitHub observation.` |
| Task 4 | `Record named Goal and acceptance-condition evidence and obtain a Complete assessment.` |
| Target date | absent |

The observed Plan was semantically equal to the exact authorized proposed Plan. Progress was `open`; membership was complete; all four observed Tasks were `open`. Progress is observation material and does not establish the Plan Control outcome.

## Named Goal and Acceptance Evidence

| Subject | Evidence |
|---|---|
| Goal | The initial Snapshot established a current Plan; Codex assessed its exact material as `Revise`; current authorization was bound to that exact Revision; one Actor invocation returned `ReceiptAcknowledged`; and a later independent Snapshot established the exact proposed Plan. |
| Acceptance condition 1 | The Initial GitHub Plan Snapshot section records the fresh observation timestamp, exact current Plan, overall progress, complete membership, and Task progress. |
| Acceptance condition 2 | The Later GitHub Plan Snapshot section records a new observation timestamp and exact resulting Plan without using the application result as state evidence. |
| Acceptance condition 3 | This section names the Goal and binds evidence separately to every acceptance condition for the final Codex assessment. |

## Final Codex Assessment

The public `codexplancontrol` adapter invoked the public `codexappserver.Client` contract with Codex app-server `0.149.1`, model `gpt-5.6-sol`, reasoning effort `high`, the exact later current Plan, and the evidence above. The assessment outcome was `Complete`.

## Completion Criteria

| Criterion | Recorded evidence |
|---|---|
| CA-1 | Stable Target Reference and Initial GitHub Plan Snapshot |
| CA-2 | Initial Codex Assessment with exact current Plan and caller-owned observation material |
| CA-3 | Exact Revision Authorization and External Actor Receipt |
| CA-4 | Later GitHub Plan Snapshot, independently observed and equal to the proposed Plan |
| CA-5 | Named Goal and Acceptance Evidence plus Final Codex Assessment |
| CA-6 | Both Snapshot and assessment references, exact Revision, authorization reference, and receipt references in this disposable record |

The timestamps record this proof composition but do not assert causality, authority, or a mandatory Product operation order.
