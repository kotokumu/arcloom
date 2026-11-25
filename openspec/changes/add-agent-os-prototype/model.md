# Specification Analysis: Agent OS prototype

## 1. Boundary

| Item | Decision | Evidence |
|---|---|---|
| Product capability | New agent-os-operation | Operator starts, inspects and resumes application-controlled work |
| Change classification | New product behavior | Existing Plan operation and reconciliation control retain disposable state |
| Included behavior | Limited local Stage 1 subset with one owner | Reviewed design and original v0.1 |
| Excluded behavior | Full Stage 1, hostile native-driver isolation, full syscall surface, UI and semantic GC | Explicit prototype non-goals |

## 2. Consumers and Observable Events

| Consumer / actor | Trigger / prior state | Interaction or event | Observable result |
|---|---|---|---|
| Operator | Configured application and drivers | Start or resume work | Bound run or explicit rejection |
| Business application | Run signal and prior state | Propose next transition | Persisted state and authorized action, wait or completion |
| Operator | Interrupted external work | Restart | Uncertain action without automatic replay |
| Signal consumer | Unacknowledged matching events | Receive and acknowledge | Independent durable delivery progress |
| Operator | Evidence and knowledge available | Inspect artifacts or assemble context | Verified evidence and nonexpired, separated context |
| Resource holder | Resource free or held | Acquire or release lease | Exclusive local ownership or refusal |

## 3. Conceptual Model

The following subsection is complete reader-facing Conceptual Model text for the new main spec. Publication is a separate pending task; Requirement publication remains owned by the official archive command.

### Concepts and Authority

`agent-os-operation` owns the following prototype meanings. They do not redefine the existing Go Plan or reconciliation capabilities.

| Concept | Meaning and relevant identity | Authority |
|---|---|---|
| Intent | Immutable operator-provided requirements snapshot bound to a run | Operator; applications cannot overwrite it |
| Task | Identified local run, bound to one application and configuration, with durable state and action budget | OS owns operational lifecycle; application owns business meaning |
| AgentProcess | Identified external action belonging to one run and generation | OS owns recorded lifecycle; external program owns external effects |
| Artifact | Immutable evidence or result identified by content digest | Owner scope stores bytes; external facts retain their source authority |
| KnowledgeNode | Operator-ingested interpretation with evidence references, status, trust and optional expiry | Operator owns ingestion; applications have no write syscall in this increment |
| Signal | Immutable event occurrence with run/action correlation where applicable | OS owns event preservation, not the truth of external business claims |
| Lease | Temporary ownership of a resource key by a holder with an expiry and increasing fencing token | OS owns local exclusion; no external fencing guarantee |
| Mailbox | Consumer-specific subscription, delivery mode and acknowledgement position | Consumer owns progress independently of other consumers |
| Business application | Replaceable program deciding business state and next action | Application owns workflow and success criteria |
| Driver capability | Named execution or feedback ability offered by a selected external provider | Provider owns external interaction; OS owns dispatch permission |
| Owner scope | One operator's state directory and its records | Operator; application context never crosses directories |

### States and Classifications

| Classification | Meaning |
|---|---|
| Pending action | Dispatch intent is recorded but no completion is recorded |
| Completed action | Its result and correlated signal are recorded |
| Uncertain action | An interruption prevents establishing the external outcome; terminal for automatic dispatch |
| Application wait | No new external action requested by the current transition |
| Application complete | Application declares its business work complete |
| budget_hit | Terminal run condition preventing further external actions |
| Queue delivery | Matching occurrences remain distinct in first-in, first-out order |
| Coalescing delivery | Explicitly selected replaceable notifications may collapse to the latest matching occurrence |
| Human/stable knowledge | Confirmed context, kept separate from provisional drafts |
| Draft knowledge | Provisional context with finite expiry; may be overturned |

Action budget counts accepted external actions. Knowledge and lease expiry use the operation's current time; an expiry equal to the current time is expired. The configured deadline bounds command duration; the configured output limit bounds accepted output bytes.

### Relationships and Structural Invariants

- A run binds one Intent snapshot, one application content identity and one configuration identity. A recorded external action belongs to exactly one run.
- Signal occurrences are immutable; each consumer's acknowledgement is independent.
- Artifact identity commits to its bytes. Knowledge references evidence without replacing it as the source of truth.
- One operator scope may contain multiple runs. Applications receive their own run signals and owner-scoped filtered context; they have no arbitrary artifact or knowledge access.
- The original seven public object categories remain distinct from Mailbox, which is a delivery mechanism.

## 4. Requirement Candidates

| Requirement slug | Actor and event | Guarantee | Concepts used | Normative representations | Important scenario classes |
|---|---|---|---|---|---|
| replaceable-business-progression | Operator selects app and drivers | Business workflow and providers are independently replaceable | Application, driver, Task | Invariants | Happy, error |
| explicit-dispatch-authority | App requests external effect | Exact capability grant required; one owner scope | Scope, capability, Intent | Decision table, invariants | Happy, permission |
| durable-action-outcomes | Operator resumes after interruption | Preserve transitions and uncertainty without replay | Task, AgentProcess, Signal | State transition table | Happy, idempotency, compatibility, concurrency |
| bounded-local-work | App or driver exceeds configured limits | Visible bounded failure and finite action budget | Task, AgentProcess | Partition table | Happy, boundary, error |
| independent-signal-delivery | Consumers receive and acknowledge | Durable independent queue/coalescing behavior | Mailbox, Signal | Decision table, invariants | Happy, concurrency, idempotency |
| immutable-artifact-evidence | Operator reads stored evidence | Preserve and verify content identity | Artifact | Invariant | Happy, error |
| filtered-knowledge-context | Operator ingests and uses knowledge | Expiry and physical separation of confirmed/draft context | KnowledgeNode, Artifact, Scope | Partition table, invariants | Happy, boundary, permission |
| local-resource-leases | Holder acquires/releases ownership | Expiry and stale-token rejection | Lease | Decision table | Happy, concurrency, boundary |

## 5. Unresolved Decisions

None.

## 6. Sources

- [Original Agent OS v0.1](../../../docs/design/agent-os-v0.1.original.md).
- [Reviewed prototype design](design.md), including explicit scope decisions and independent scenario review.
- Existing `plan-feedback-loop-operation` and `reconciliation-control-loop` main specs: disposable Plan operations are not durable Agent OS work.
- [Development guidelines](../../../docs/DEVELOPMENT.md) and [specification rules](../../specs/README.md).
