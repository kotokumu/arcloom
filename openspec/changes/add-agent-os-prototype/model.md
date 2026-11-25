# Specification Analysis: Agent OS prototype

## 1. Boundary

| Item | Decision | Evidence |
|---|---|---|
| Product capability | New agent-os-operation | Operator starts, inspects and resumes application-controlled work |
| Change classification | New product behavior | Existing Plan operation and reconciliation control retain disposable state |
| Included behavior | Local Stage 1 subset with open capabilities, requested guest services and integrated Mailbox/Lease dispatch; one owner | Reviewed design and original v0.1 |
| Excluded behavior | Full Stage 1, hostile native-driver isolation, full syscall surface, UI and semantic GC | Explicit prototype non-goals |

## 2. Consumers and Observable Events

| Consumer / actor | Trigger / prior state | Interaction or event | Observable result |
|---|---|---|---|
| Operator | Configured application and drivers | Start or resume work | Bound run or explicit rejection |
| Business application | Run signal and prior state | Request authorized evidence and stage a transition | Typed result and staged invocation, wait or completion; no effect on trap |
| Capability provider | Declared dependency scope | Invoke a required ability | Only explicitly granted dependency is available |
| Host | Kernel submitted durable action | Collect pending work outside the kernel | Exact correlated completion or protected uncertainty |
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
| Lease | Resource reservation with holder, expiry, fencing token and active/reconciling disposition | OS owns dispatch exclusion; no external fencing guarantee |
| Mailbox | Consumer-specific subscription, delivery mode and acknowledgement position | Consumer owns progress independently of other consumers |
| Business application | Replaceable program deciding business state and next action | Application owns workflow and success criteria |
| Capability invocation | Request for a named ability and opaque business input, independent of implementation kind | Application owns request meaning; OS owns authority and dispatch |
| Provider dependency | Named ability required by a provider, resolved to one implementation within its declared scope | Provider owns its needs; OS preserves explicit grants |
| Staged guest resource | Callback-local lease/ticket representing a proposal, not committed ownership or dispatched work | Guest boundary owns lifetime and validity |
| Pending operation | Submitted external work whose result the host may collect later | Driver owns transport lifecycle; kernel owns durable action correlation |
| Owner scope | One operator's state directory and its records | Operator; application context never crosses directories |

### States and Classifications

| Classification | Meaning |
|---|---|
| Pending action | Dispatch intent is recorded but no completion is recorded |
| Completed action | Its result and correlated signal are recorded |
| Uncertain action | An interruption prevents establishing the external outcome; terminal for automatic dispatch |
| Application wait | No new external action requested by the current transition |
| Application complete | Application declares its business work complete |
| Application invocation | Application returns a valid staged ticket for an open capability request |
| Active reservation | Current exclusive resource ownership |
| Reconciling reservation | Possible external effects require explicit reconciliation before reuse |
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
- One operator scope may contain multiple runs. Applications receive their own run signals and request evidence through granted services. Knowledge stays in the owner scope; artifact reads are limited to the current Signal artifact. No implicit context accompanies every invocation.
- The original seven public object categories remain distinct from Mailbox, which is a delivery mechanism.
- Application roots and each provider dependency have separate callable scopes; grants are explicit at every edge.
- Each provider may declare a resource key; the default is the invoked capability. Every dependency edge in one composed invocation shares that effective key, so one reservation protects the entire composition. Multiple-resource atomic reservations are not included.
- A staged guest resource is valid only for its callback. Durable action identity and its Lease survive that callback.
- Reconciling reservations cannot be treated as expired reusable resources while external outcomes remain unresolved.

## 4. Requirement Candidates

| Requirement slug | Actor and event | Guarantee | Concepts used | Normative representations | Important scenario classes |
|---|---|---|---|---|---|
| replaceable-business-progression | Operator selects app and drivers | Business workflow and providers are independently replaceable | Application, driver, Task | Invariants | Happy, error |
| explicit-dispatch-authority | App requests external effect | Exact root and dependency grants required; one owner scope | Scope, capability, Intent | Decision table, invariants | Happy, permission |
| scoped-capability-composition | Operator selects capability graph | Complete acyclic resolution with dependency-local authority | Capability invocation, provider dependency | Decision table, invariants | Happy, error, permission |
| requested-guest-services | Guest reads or stages work | Typed authorized requests; no implicit context or early effect | Signal, KnowledgeNode, Artifact, staged resource | Decision table, invariants | Happy, permission, error |
| pending-invocation-handoff | Kernel submits; host collects | No transport wait in kernel; exact correlated completion | Pending operation, AgentProcess, Lease | State transition table | Happy, concurrency |
| durable-action-outcomes | Operator resumes after interruption | Preserve transitions and uncertainty without replay | Task, AgentProcess, Signal | State transition table | Happy, idempotency, compatibility, concurrency |
| bounded-local-work | App or driver exceeds configured limits | Visible bounded failure and finite action budget | Task, AgentProcess | Partition table | Happy, boundary, error |
| independent-signal-delivery | Consumers receive and acknowledge | Durable independent queue/coalescing behavior | Mailbox, Signal | Decision table, invariants | Happy, concurrency, idempotency |
| immutable-artifact-evidence | Operator reads stored evidence | Preserve and verify content identity | Artifact | Invariant | Happy, error |
| filtered-knowledge-context | Operator ingests and uses knowledge | Expiry and physical separation of confirmed/draft context | KnowledgeNode, Artifact, Scope | Partition table, invariants | Happy, boundary, permission |
| local-resource-leases | Dispatch reserves/completes/reconciles | Atomic exclusion, protected uncertainty and stale-token rejection | Lease, AgentProcess, Mailbox | Decision table | Happy, concurrency, boundary |

## 5. Unresolved Decisions

None.

## 6. Sources

- [Original Agent OS v0.1](../../../docs/design/agent-os-v0.1.original.md).
- [Reviewed prototype design](design.md), including explicit scope decisions and independent scenario review.
- Existing `plan-feedback-loop-operation` and `reconciliation-control-loop` main specs: disposable Plan operations are not durable Agent OS work.
- [Development guidelines](../../../docs/DEVELOPMENT.md) and [specification rules](../../specs/README.md).
