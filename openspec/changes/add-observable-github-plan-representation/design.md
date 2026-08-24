# Observable GitHub Plan Representation

## 0. Document Scope

| Information | Normative document |
|---|---|
| Product concept, scope, capabilities, and principles | `PRODUCT.md` |
| Component responsibilities, external authority, dependency rules, and Port ownership | `ARCHITECTURE.md` |
| Accepted Plan, dry-run, and Plan Representation Controller behavior | `openspec/specs/plan/spec.md`, `openspec/specs/github-plan-creation-dry-run/spec.md`, and `openspec/specs/plan-representation-reconciliation/spec.md` |
| Proposed observable behavior | The two delta specs in this Change |
| Structure, contracts, decisions, and verification design for this Change | This DesignDoc |
| Implementation workflow and review gates | `docs/DEVELOPMENT.md` |

## 1. Purpose / Out of Scope

### 1.1 Purpose

The GitHub Planning Provider supplies the existing Plan Representation Controller with a current, provider-independent Observation of one bound GitHub Milestone or parent Issue. GitHub remains authoritative for the represented resources. Arcloom reconstructs a Disposable Projection for each call and stores no copy.

The same Provider boundary also owns one reversible payload protocol shared by the existing creation dry-run and the new observer. The payload carries only Plan meaning that GitHub does not represent natively; native GitHub titles and relationships remain independently observable facts.

### 1.2 Out of Scope

- Applying, authorizing, or scheduling a GitHub mutation
- Discovering which Repository or resource represents a Plan
- Treating `404`, `410`, or another GitHub response as authoritative absence
- Parsing the human-readable Markdown narrative
- Persisting observations, credentials, provider identifiers, response bodies, or reconciliation results
- GitHub Enterprise Server, GitHub Projects, Linear, or another Provider
- Retry, cache, ETag, historical observation, or snapshot-token behavior
- Changing the provider-independent Plan or Observer Port

## 2. Behavior Design

### 2.1 Functional Requirements

#### Target binding and observation

| ID | Rule | Source |
|---|---|---|
| FR-1 | ResourceNumber and the named Milestone/Issue Observer constructors validate local configuration, return a nil Port on Observer-construction failure, and perform no request during construction. | GHPO-1 |
| FR-2 | A successful boundary remains bound to one Repository, representation, and positive resource number and does not expose them through the Observer Port. | GHPO-1, PRR-1 |
| FR-3 | Each call obtains new GitHub facts, returns one valid Observation or the caller's context error, and shares no per-call state. | GHPO-5, GHPO-8, PRR-2, PRR-7 |
| FR-4 | The Provider never mutates GitHub and never constructs an authoritative-absence Observation. | GHPO-5, GHPO-8 |

#### Reversible payload protocol

| ID | Rule | Source |
|---|---|---|
| FR-5 | The dry-run prefixes the existing narrative with the one canonical version-one payload byte sequence. | GPCD-2, GPCD-5 |
| FR-6 | The observer decodes the accepted version-one grammar, ignores the narrative and unknown v1 members, and localizes unusable members exactly as specified. | GHPO-2 |
| FR-7 | The codec preserves every valid decoded Plan string exactly and rejects duplicate top-level payload-object member names and noncanonical base64url. | GPCD-2, GHPO-2, PLN-2 |
| FR-8 | Plan-owned scalar and collection classifiers, rather than GitHub code, decide Plan invariant violations. | GHPO-2, PRR-2, PLN-2 through PLN-5 |

#### GitHub-native representation

| ID | Rule | Source |
|---|---|---|
| FR-9 | Milestone observation maps the root title, payload, `due_on`, and every open or closed assigned non-PR Issue into Plan locations. | GHPO-3 |
| FR-10 | Issue observation maps the parent title, payload, and every Sub-issue into Plan locations; a Pull Request root is unavailable. | GHPO-4 |
| FR-11 | Response-shape, status, transport, and pagination failures preserve independently known facts and mark the least affected Plan location unavailable. | GHPO-5, GHPO-6, PRR-2 |
| FR-12 | Provider identity resolves page repetition before titles cross the Port; equal-title distinct resources remain distinct so Plan rules can detect duplication. | GHPO-6 |
| FR-13 | All requests use GitHub.com REST `2022-11-28`, raw Markdown media type, the exact bound target, and a page size of 100 without following redirects. | GHPO-1, GHPO-7 |

### 2.2 Non-functional Requirements

| ID | Rule | Verification |
|---|---|---|
| NFR-1 | The returned Observer is safe for concurrent calls when the Host satisfies the `http.Client` precondition. | Race-enabled concurrent behavior test |
| NFR-2 | Runtime-state loss does not affect correctness because every Observation is reconstructed. | Independent-call behavior test with changed responses |
| NFR-3 | No Provider error, response body, credential, target identifier, or rate-limit detail crosses the Observer Port. | Failure-boundary tests and public API inspection |

No latency, throughput, availability, or retry target is introduced.

## 3. Structure Design

### 3.1 Conceptual Model

| Concept | Meaning | State | Authority / lifecycle owner | Behavior / decision | Constraint / invariant |
|---|---|---|---|---|---|
| Plan | Provider-independent expected planning meaning | Immutable valid value | Plan Component | Owns text, collection, and target-date invariants | Contains no GitHub identifier or resource type |
| GitHub Plan Representation Scheme | Provider-owned rule for mapping either Milestone-based or Issue-based GitHub facts to Plan locations | One explicit closed variant | GitHub Planning Provider | Determines native versus payload-backed locations, root kind, Task relationship, and valid Payload v1 shape | Scheme is explicit and never inferred; it owns no GitHub resource lifecycle |
| GitHub Repository | Provider-specific immutable owner/name locator | Exact locally valid path segments | GitHub owns the addressed external namespace; GitHub Planning Provider owns local form validity | Scopes every root number and REST request | It identifies no root without Scheme and ResourceNumber |
| GitHub ResourceNumber | Provider-specific positive `number` intended to address a Milestone or Issue root | Immutable positive integer value | GitHub owns assigned external numbers; GitHub Planning Provider owns local value validity | Marks root-number meaning distinctly from REST `id`, GraphQL `node_id`, and request-plan result kinds | It proves positivity and intended role, not the integer's external provenance; it identifies no root alone |
| GitHub Representation Binding | Composite immutable root address | Repository, selected Scheme, and typed ResourceNumber captured for the Observer lifetime | Host selects target; GitHub Planning Provider owns local structural validity; GitHub owns remote resource lifecycle | Keeps every call on one composite address | No component independently identifies a root; remote existence, access, and HTTP client lifecycle are not Binding validity |
| Arcloom Plan Payload v1 | Versioned reversible machine protocol at the beginning of native content | Canonical produced bytes or accepted decoded meaning | GitHub Planning Provider owns schema and interpretation; GitHub owns current stored bytes and their lifecycle | Encodes/decodes payload-backed Plan meaning | It is not an Arcloom record or recovery source; absence/corruption makes facts unavailable and no Arcloom backup exists |
| Generated Human Plan Narrative | Deterministic Markdown presentation after the generated payload | Bytes derived from Plan values in a creation RequestPlan | GitHub Planning Provider owns rendering | Presents proposed Plan meaning to people | Passive artifact; not a reconstruction source; a later GitHub-edited suffix is uninterpreted external content |
| GitHub Resource Facts | Individual Provider-native root, related-resource, identity, status, and pagination facts | External resources and responses | GitHub | Supply authoritative external facts | Never become an Arcloom persistence record |
| GitHub Observation Fact Set | Facts accumulated during one bounded observation window | One Binding, root facts, coherent Task resources, seen REST `id` and page keys, and completeness | GitHub owns source facts; GitHub Planning Provider owns only this transient assembly lifecycle | Protects per-call identity coherence and page isolation before projection | Disposable at return; not atomic beyond detected contradictions and never shared across calls |
| GitHub Observation Interpretation Policy | Provider-specific rule for deciding what Plan knowledge can be established from a Fact Set | Stateless decision over one call's Fact Set and payload outcomes | GitHub Planning Provider | Classifies statuses, shapes, payload fields, pagination, and identity contradictions as known or locally unavailable/incomplete | Establishes provider-independent knowledge, never external facts; does not own payload syntax, Plan invariants, or Observation algebra |
| Plan Representation Observation | Immutable provider-independent Disposable Projection | Root and Plan-location knowledge established during one bounded observation window | Plan Representation Controller owns its valid state algebra | Represents known values, violations, and localized unavailability | It is not an atomic GitHub snapshot and contains no GitHub vocabulary or errors |
| GitHub Creation Request Plan | Existing immutable passive proposed-mutation artifact | Target Repository/Scheme, topologically ordered request values, and typed references | GitHub Planning Provider | Protects dependency topology, compatible result references, immutability, and target consistency | Does not execute, mutate, or claim success |

#### Relationships

| Source | Relationship | Target | Multiplicity / consistency |
|---|---|---|---|
| GitHub Representation Binding | combines | GitHub Repository, Representation Scheme, and ResourceNumber | Exactly one of each; none identifies a root independently |
| GitHub Plan Representation Scheme | maps | GitHub Resource Facts to Plan locations | Mapping may yield known, invalid, or unavailable Plan-shaped meaning; external facts do not constitute a valid Plan |
| Native root title | represents | Plan name | Exact text; missing or unusable title localizes to Plan name |
| Milestone assigned non-PR Issues / Issue Sub-issues | represent | Tasks | Zero or more; GitHub identity is used only before mapping titles |
| Representation Scheme | determines the valid shape of | Arcloom Plan Payload v1 | Milestone requires Goal/conditions; Issue additionally requires target date |
| Native content prefix | contains | Arcloom Plan Payload v1 | Zero or one usable block; failure affects only backed locations |
| Creation Request Plan | contains | Generated Human Plan Narrative | Deterministic proposed bytes; current GitHub suffix may later differ and is ignored |
| GitHub Observation Fact Set | accumulates from | GitHub Resource Facts | One bounded observation window; discarded at return |
| GitHub Observation Interpretation Policy | interprets | GitHub Observation Fact Set and Payload field outcomes | Establishes only coherent Plan knowledge and is the sole owner that assigns GitHub uncertainty to affected Plan locations |
| GitHub Observation Fact Set | projects through the Interpretation Policy into | Plan Representation Observation | Best established knowledge for the window, not an atomic snapshot |
| Plan Representation Controller | consumes through its Observer Port | Plan Representation Observation | Existing provider-independent contract remains unchanged |

#### Concept minimality

| Candidate | Decision | Reason |
|---|---|---|
| Milestone representation and Issue representation classes | Do not introduce | They are closed variants of one representation and have no independent Arcloom lifecycle. |
| Public Payload type or codec Port | Do not introduce | Only one Provider package owns both consumers; private functions protect one protocol decision without exposing a new contract. |
| Pagination manager, parser, decoder, mapper, or observer service | Do not introduce | These are procedural stages without independent identity, lifecycle, or authority. |
| Public Binding type | Do not introduce | The meaningful immutable binding is represented by the closure returned as the existing Observer Port; consumers need no Provider identity value. |
| GitHub ResourceNumber | Keep as a public immutable value | Removing it permits raw REST `id` and root `number` to be confused at the Host-facing constructor; a single value serves both root variants without importing either into the Observer Port. |
| Observation Fact Set | Keep as a transient concept, not a public type | Removing it hides page/identity/coherence state; a private per-call value protects those invariants and is discarded at return. |
| Observation Interpretation Policy class or interface | Do not introduce | The Policy is a meaningful decision owner, but it has no state, identity, lifecycle, or alternate implementation; private functions and tests are sufficient. |
| GitHub SDK abstraction | Do not introduce | The concrete Provider consumes only `net/http`; a second implementation or consumer constraint has not been established. |
| Cache, repository, observation history, or retry policy | Do not introduce | They are not required for current correctness and could obscure external authority. |
| Authoritative GitHub absence | Do not introduce | The selected GitHub contracts do not provide an unambiguous fact for it. |

### 3.2 Responsibility and Package Design

#### Responsibility assignment

| Responsibility / decision | Owner | Information and authority used | Invariant protected | Not owner / reason |
|---|---|---|---|---|
| Plan element validity and duplicate classification | Plan Component | Provider-independent Plan rules | One Plan validity policy | GitHub Provider must not copy these rules. |
| Observation state combinations, Location validity, and provider-independent covering/suppression algebra | Plan Representation Controller | Plan locations and Observation algebra | Only valid provider-independent observations cross its Port | It does not decide which GitHub fact affects which Plan location. |
| Select credentials, client, Repository, representation, and number | Arcloom Host | Invocation/deployment context | Correct target association for the applicable Change | Composition Root only wires; Controller has no GitHub identity vocabulary. |
| Validate and preserve the local target binding | GitHub Planning Provider | Repository, representation selected by constructor, typed ResourceNumber, and GitHub target syntax | One immutable target; no construction-time remote claim | HTTP client ownership and access behavior are separate REST-boundary concerns. |
| Validate positive root-number form and distinguish it from REST identity | GitHub ResourceNumber value | Supplied integer and GitHub identifier semantics | Invalid or ambiguous raw integers do not cross the Host-facing Observer boundary | Binding consumes the value; request-plan ResultKind and collection identity have different meanings. |
| Validate and copy the Host client, refuse redirects, and issue reads | GitHub REST boundary inside the Planning Provider | Host-supplied client and approved REST contract | External access remains read-only and target-contained | Binding does not own transport/authentication lifecycle; Host does not own Provider request rules. |
| Own external resource identity, permissions, and lifecycle | GitHub | Native system authority | Durable external facts | Arcloom never persists or re-owns them. |
| Define Payload v1 canonical producer and accepted decoder grammar | GitHub Planning Provider payload protocol | GPCD-2 and GHPO-2 | One reversible protocol decision and private syntactic/typed field outcomes | It does not assign Plan locations or construct Observation child values. |
| Render the human narrative | GitHub Planning Provider creation preview | Valid Plan and presentation specification | Deterministic human projection | Observer ignores the narrative. |
| Accumulate one bounded set of coherent root/page facts | GitHub Observation Fact Set | Immutable Binding, raw Provider facts, seen positive REST `id` values and pages | Per-call coherence, completeness, and concurrent-call isolation | Observer coordinates its lifecycle but does not own its identity/coherence invariants. |
| Obtain GitHub responses and coordinate one Observation | GitHub Planning Provider observer implementation | Immutable Binding, REST boundary, Observation Fact Set, Interpretation Policy, and consumer Observation constructors | One read-only logical Observation per call | It owns no status, payload, localization, identity-coherence, or Plan-validity decision. |
| Decide what Plan knowledge is established and which Plan location uncertainty affects | GitHub Observation Interpretation Policy | HTTP status, response shape, typed payload field outcomes, Observation Fact Set, positive REST `id`, and pagination facts | Unknown facts are not invented as values, violations, or absence | Payload protocol knows syntax but not Plan locations; Controller knows location algebra but not GitHub semantics; observer only coordinates. |
| Validate Interpretation Policy outputs and apply provider-independent covering/suppression | Plan Representation Observation values and constructors | Affected Plan locations and classified values supplied by the Provider | Valid immutable Observation combinations | It does not reinterpret GitHub status, shapes, identities, or payload fields. |
| Derive semantic differences and determination | Plan Representation Controller | Expected Plan and returned Observation | Provider-independent reconciliation meaning | Observer never decides satisfaction. |

#### Package responsibilities

| Package | Responsibilities implemented | Public contracts | Hidden implementation | Permitted dependencies |
|---|---|---|---|---|
| `plan` | Existing Plan concepts and invariants | Existing values and classifiers | Validation details | Go standard library |
| `planrepresentation` | Existing Controller, Observation algebra, and consumer-owned Observer Port | Existing `Observer`, Observation constructors, Controller and result contracts | Correspondence and evidence construction | `plan`, `reconciliation`, Go standard library |
| `githubplanning` | GitHub Planning Provider creation preview, binding, payload protocol, REST access, Observation Fact Set, Interpretation Policy, and Observer implementation | Existing preview contracts; typed ResourceNumber; extended validation categories/fields; Milestone and Issue Observer constructors | Payload codec outcomes, binding, request/response representations, per-call identities/pages, and interpretation decisions | `plan`, `planrepresentation`, Go standard library including `net/http` |

No new package is added. Payload production and observation belong together because they change with the same GitHub representation protocol. Inside `githubplanning`, package-level files and private functions separate payload protocol, creation projection, target binding, REST access, per-call Observation Fact Set, Interpretation Policy, and Observer coordination. These are responsibility boundaries, not processing-stage classes or public subpackages.

#### Package dependencies

| Source | Target | Public contract used | Reason | Must not cross |
|---|---|---|---|---|
| `planrepresentation` | `plan` | Plan values and validation results | Express provider-independent observation meaning | Provider identity and HTTP details |
| `githubplanning` | `plan` | Plan values used by the creation preview | Produce Provider representation from accepted Plan meaning | GitHub rules into Plan |
| `githubplanning` | `planrepresentation` | `Observer` and Observation constructors | Implement the consumer-owned observation Port | GitHub DTOs, identifiers, errors, statuses, pagination |
| Arcloom Host | `githubplanning` | Concrete Provider configuration and preview APIs | Select a Provider and supply external configuration | Business decisions into Composition Root |

#### Independent evolution scenario impact

| Scenario / confidence | Primary owner | Expected propagation | Verdict |
|---|---|---|---|
| Human Markdown changes independently / committed | Narrative renderer | `githubplanning` renderer and presentation tests; decoder unchanged | Pass |
| Payload v2 coexists with v1 / plausible | Payload protocol | `githubplanning` codec and conformance tests; Plan and Controller unchanged | Pass; no strategy is added before v2 exists. |
| GitHub title, date, relationship, access, or pagination facts change / committed | GitHub observation responsibility | Provider mapping and boundary tests | Pass |
| GitHub repeats an `id`, changes a title, or returns cross-Repository Sub-issues / committed | Observation Fact Set and Interpretation Policy | Provider identity/coherence code and tests only; `number` and `node_id` remain irrelevant | Pass |
| REST `2022-11-28` retires / committed | GitHub REST boundary | Request contracts, response representations, and conformance tests | Pass; separate contract Change required. |
| HTTP implementation or authentication wrapper changes / plausible | Host and GitHub REST boundary | Host-supplied client and Provider HTTP tests | Pass |
| A Linear Provider is added / plausible | New Linear Provider Module | New package implements existing Observer Port; no `githubplanning` dependency | Pass |
| Authorized mutation is added / plausible | Plan Change Target plus Change Authorization | Separate mutation Port and Provider implementation | Pass; current observer remains read-only. |
| Quota or latency pressure motivates caching / plausible | Product and Architecture decision | Separate design; any cache must be a Disposable Projection and cannot be required for correctness | Pass; no speculative cache Port. |

#### SOLID and procedural-risk assessment

| Principle | Risk | Mitigation |
|---|---|---|
| SRP | A single observer function could own binding, payload, paging, validation, and observation decisions. | Keep private responsibilities aligned to binding, payload protocol, GitHub facts, and Observation construction; do not split them by execution step. |
| OCP | Payload/API version strategies could be added for hypothetical versions. | Support only payload v1 and REST `2022-11-28`; record later versions as new contract Changes. |
| LSP | The Provider could leak a GitHub error or invalid Observation through the Port. | Convert every non-caller failure to a valid localized Observation and contract-test the Port. |
| ISP | A broad GitHub service could combine preview, reads, and future mutation. | Publish one constructor for the existing read-only Observer Port; keep preview concrete and mutation absent. |
| DIP | Core Packages could depend on HTTP or GitHub DTOs. | Only `githubplanning` imports `net/http`; it returns consumer-owned Observation values. |

The implementation must not introduce `Manager`, `Processor`, `Handler`, `Client` wrapper, or generic Provider interface merely to mirror fetch/decode/map stages.

### 3.3 Interface Design

#### Extended validation contract

```go
package githubplanning

const (
	InvalidClient         ViolationCode = "invalid_client"
	InvalidResourceNumber ViolationCode = "invalid_resource_number"
)

const (
	ClientField         Field = 5
	ResourceNumberField Field = 6
)
```

Existing validation categories and fields keep their values and meaning. Resource-number and Observer constructors return `*ValidationError`, inspectable with `errors.As`, for local configuration failures. They do not wrap a Provider error. Simultaneous Observer-input precedence is client, Repository, then ResourceNumber. A zero Repository returns `InvalidRepository` with `RepositoryOwnerField`; the existing `NewRepository` contract separately owns owner/name validation and precedence.

#### Concrete Provider construction and consumer Port

```go
package githubplanning

// ResourceNumber is a positive GitHub Milestone or Issue number. It is not a
// GitHub REST id, GraphQL node_id, or request-plan ResultKind.
type ResourceNumber struct { /* immutable */ }

func NewResourceNumber(value int64) (ResourceNumber, error)
func (n ResourceNumber) Int64() int64

func NewMilestoneObserver(
	client *http.Client,
	repository Repository,
	number ResourceNumber,
) (planrepresentation.Observer, error)

func NewIssueObserver(
	client *http.Client,
	repository Repository,
	number ResourceNumber,
) (planrepresentation.Observer, error)
```

| Contract dimension | Definition |
|---|---|
| Consumer | Arcloom Host configures the concrete Provider at the Composition Root boundary; Plan Representation Controller invokes the returned Port. |
| Preconditions | Non-nil client with nil cookie Jar; locally valid Repository; constructed ResourceNumber. Host does not mutate a custom Transport or its collaborators after construction and supplies concurrent-safe transport behavior. |
| Success | Returns a non-nil `planrepresentation.Observer` permanently bound to the supplied target. No request occurs during construction. |
| Failure | Returns a nil Observer and typed validation error; no request or external state change. |
| Side effects | Observation performs only GitHub.com REST reads. It does not mutate, authorize, persist, log response content, or follow redirects. |
| Hidden details | Client copy, target URL, headers, Provider identities, page state, response representations, payload bytes, statuses, and errors. |
| Protected constraint | GitHub configuration remains outside the consumer Port while one immutable binding and Provider error containment are enforced. |

`ResourceNumber` and the two named constructors prevent direct accidental mixing of a raw integer with the root-number parameter and avoid a representation flag. Construction proves positivity and intended role, not whether the caller originally obtained that integer from REST `number` rather than `id`. This concrete API is a Composition Root boundary used by the Arcloom Host. Client, Repository, and ResourceNumber terminate in `githubplanning`; none crosses the returned `planrepresentation.Observer`.

The Provider rejects a non-nil cookie Jar, copies the supplied `http.Client` value, replaces `CheckRedirect` without calling the Host callback, and returns `http.ErrUseLastResponse`. It never changes the caller's client. It closes every received response body after extracting the required facts. A custom Transport and its reachable collaborators remain Host-owned, shared external-access mechanisms under the stated immutability and concurrency precondition; the Provider does not inspect or copy credential material from them.

#### Observer postcondition

The returned function preserves the existing `planrepresentation.Observer` contract:

```go
func(ctx context.Context) (planrepresentation.Observation, error)
```

- A direct call with nil context performs no request and returns a valid root-unavailable Observation with nil error. The supported Controller path still rejects nil before invoking the Observer.
- When a request or the final completion check observes cancellation or deadline expiration, the function returns the matching `ctx.Err()`; under the existing Port contract the consumer ignores the accompanying Observation value, whose representation is not part of this contract.
- Every other HTTP, GitHub, decoding, response-shape, or coherence failure returns a valid Observation with localized unavailability and a nil error.
- Success returns exactly one valid present Observation and a nil error.
- No return contains Provider target identity or error detail.

#### Private implementation representations

The following are implementation representations, not new public contracts:

| Representation | Minimum form | Protected rule |
|---|---|---|
| Binding | Immutable private value captured by the Observer closure | Repository, representation selected by constructor, typed ResourceNumber, and target immutability |
| REST access configuration | Copied `http.Client` captured separately by the Observer closure | Redirect refusal and read-only external access without changing Binding meaning |
| Payload meaning | Private struct/value plus encode/decode functions | Shared v1 member vocabulary, canonical output, tolerant accepted grammar, and syntactic/typed field outcomes only |
| Root facts | Private Provider response value | Independent root field classification before Observation construction |
| Task resource | Private positive REST `id` and title pair | Identity-based page coherence before title-only projection; `number` and `node_id` are ignored |
| Observation Fact Set | Private per-call value containing root facts, Task resources, seen identities/pages, and completeness | Same-identity conflict handling, bounded observation-window meaning, and discard-at-return isolation |

#### Example composition

```go
repository, err := githubplanning.NewRepository("owner", "repository")
if err != nil { /* inspect *githubplanning.ValidationError */ }

number, err := githubplanning.NewResourceNumber(42)
if err != nil { /* inspect *githubplanning.ValidationError */ }

// authenticatedClient is dedicated to this integration, has no cookie Jar,
// and its Transport remains immutable and concurrent-safe after construction.
observer, err := githubplanning.NewMilestoneObserver(authenticatedClient, repository, number)
if err != nil {
	var validation *githubplanning.ValidationError
	if errors.As(err, &validation) { /* use Code and Field */ }
}

controller, err := planrepresentation.NewController(observer)
if err != nil { /* invalid Observer configuration */ }

result, err := controller.Reconcile(ctx, expectedPlan)
```

The Host performs this composition. The Composition Root only chooses and wires the concrete Provider; it does not inspect GitHub facts or own reconciliation decisions.

### 3.4 External REST Boundary

| Representation | Root request | Task request |
|---|---|---|
| Milestone | `GET /repos/{owner}/{repo}/milestones/{number}` | `GET /repos/{owner}/{repo}/issues?milestone={number}&state=all&per_page=100&page={n}` |
| Issue | `GET /repos/{owner}/{repo}/issues/{number}` | `GET /repos/{owner}/{repo}/issues/{number}/sub_issues?per_page=100&page={n}` |

Every request uses `https://api.github.com`, `X-GitHub-Api-Version: 2022-11-28`, `Accept: application/vnd.github.raw+json`, and an Arcloom User-Agent. Repository path segments and query values are encoded as URL components rather than concatenated as untrusted syntax.

The observer follows only an unambiguous `rel="next"` URL that remains HTTPS on `api.github.com`, has the exact bound endpoint path and immutable selection query, and names an unseen positive page. Any malformed, repeated, redirected, or escaping next relation stops traversal and makes the collection incomplete. Per-call maps of identity and visited page are discarded at return.

## 4. Decisions

### 4.1 Keep observation in `githubplanning`

The existing package already owns Repository, representation selection, request contracts, and the GitHub representation rules. The observer changes for the same GitHub schema and payload protocol. A second package would duplicate those decisions or require a premature public Provider abstraction.

Alternative rejected: a separate `githubobservation` package. It would either depend on `githubplanning` as a utility package or copy Repository, representation, and payload rules.

### 4.2 Reuse the existing consumer-owned Observer Port

The Plan Representation Controller already defines the minimum provider-independent function contract it consumes. The concrete constructor returns that function and keeps the binding in a closure. No public GitHub observer class or broad Provider interface is needed.

Alternative rejected: add target arguments to `Observer`. That would leak GitHub identity into the consumer Port and weaken lifetime binding.

### 4.3 Use a Host-supplied `*http.Client` and direct REST representations

The Provider needs four read contracts, exact headers, raw bodies, cancellation, response status, and redirect control. The Go standard library supplies those constraints directly. A Provider SDK is not required and might obscure endpoint/version behavior or lag Sub-issue support.

Alternative rejected: define an interface matching GitHub endpoints. It would be an implementation-shaped abstraction created only for tests. External-boundary tests instead use an `httptest` transport through the public client contract.

### 4.4 Separate canonical production from tolerant v1 decoding

The dry-run emits one deterministic byte representation. Observation accepts semantically valid v1 JSON variations but requires canonical unpadded base64url and rejects duplicate top-level payload-object member names. Nested objects inside ignored or locally unusable values are not interpreted. This lets externally stored JSON survive harmless member-order or escape changes without making request-plan equality ambiguous.

Alternative rejected: accept only producer-canonical JSON. It would turn representational details with the same v1 meaning into unavailable information.

### 4.5 Keep native facts outside the payload

Plan name and Tasks remain GitHub titles and relationships. The payload carries only Goal, acceptance conditions, and the Issue target date. Editing a native title or relationship therefore creates an observable Plan difference rather than a contradiction between duplicated facts.

Alternative rejected: serialize the complete Plan into the payload. It would make GitHub-native edits ambiguous and reduce the representation to an opaque blob.

### 4.6 Localize uncertainty conservatively

Unusable root identity makes the root unavailable. Unusable payload fields and collection members affect only their owned Plan locations. Same-identity conflicting titles are not asserted as known. The Provider never converts an ambiguous `404` or `410` into absence.

Alternative rejected: fail the whole call for Provider problems. That would violate the Observer Port and discard independently known facts.

## 5. Risks, Migration, and Rollback

### 5.1 Risks / Trade-offs

- **Existing GitHub representations have no payload** → Native title, Tasks, and Milestone date remain observable; payload-backed locations are unavailable. This Change performs no backfill.
- **REST `2022-11-28` support ends March 10, 2028** → Keep the version explicit and map rejection to unavailability; approve a separate contract Change before migration.
- **Multiple HTTP reads cannot guarantee an atomic GitHub snapshot** → Detect only specified contradictions and pagination instability. Undetectable concurrent changes remain outside PRR-2.
- **Host mutates a shared client or non-concurrent Transport** → Make immutability and concurrent safety a constructor precondition; race tests cover the Provider's own state.
- **Custom canonical string encoding is more code than default JSON encoding** → Keep it private, table-test every escape class, and decode through an independent accepted-grammar path.
- **A read-only observer can consume API quota** → Use page size 100 and no hidden retry. Do not introduce a correctness cache.

### 5.2 Migration Plan

1. Change the dry-run output to prefix all newly planned descriptions and bodies with Payload v1.
2. Add the observer constructor and read-only GitHub mapping.
3. Existing stored GitHub resources remain untouched. They reconcile with unavailable payload-backed locations until changed by a separately authorized mechanism outside this Change.
4. Rollback removes the observer and restores the previous preview output. Any already published payload remains an inert HTML comment and does not alter the human narrative. No Arcloom data migration or recovery is required.

## 6. Test Specification

Go test construction SHALL follow the repository's Go test-authoring workflow: generate the applicable table-driven scaffold first, then add one behavior at a time through Red–Green–Refactor. Tests use external package boundaries where practical and do not inspect private layout.

Semantic Observer tests SHALL pass the returned Port into `planrepresentation.Controller` and assert only public Result determination, literal Difference payloads, and literal unavailable Locations. They SHALL NOT use `cmp.AllowUnexported`, same-package access to Observation state, or the production payload encoder/decoder as an oracle. Direct Observer tests are limited to public error, context, request, and resource-lifecycle behavior.

### 6.1 Requirement Coverage

| Requirement | Observable behavior | Verification method | Owner | Required evidence |
|---|---|---|---|---|
| GPCD-2 | Canonical payload bytes plus unchanged human narrative | Black-box Go tests through `NewCreationRequestPlan` with independent literals | `githubplanning` preview tests | Exact complete description/body bytes for both Schemes and escape boundaries |
| GPCD-6 | Preview performs no read/mutation and Observer performs only approved reads | Recording Transport tests plus dependency/diff inspection | `githubplanning` boundary tests and conformance review | Zero preview requests; only declared `GET` requests; no mutation dependency or code path |
| GHPO-1 | Typed number, named constructors, validation, binding, and client ownership | Compile-time API inspection plus black-box Go tests | `githubplanning` constructor tests | Public signatures; literal error code/field; zero construction requests; client/Jar/redirect/body lifecycle assertions |
| GHPO-2 | Producer/decoder grammar and localized payload outcomes | Preview byte tests plus Controller-mediated observer tests | `githubplanning` payload boundary tests | Literal payloads and public reconciliation evidence for every grammar/localization class |
| GHPO-3 | Milestone facts map to Plan locations | Controller-mediated black-box Go tests | `githubplanning` Milestone tests | Literal Result evidence for title, payload, date, open/closed Issues, PR exclusion, and boundaries |
| GHPO-4 | Issue facts map to Plan locations | Controller-mediated black-box Go tests | `githubplanning` Issue tests | Literal Result evidence for title, payload/date, Sub-issues, parent PR, and boundaries |
| GHPO-5 | Failures remain localized; only caller context errors cross | Direct error tests plus Controller-mediated availability tests | `githubplanning` failure tests | Exact context errors with ignored Observation values, literal unavailable Locations, no Provider detail |
| GHPO-6 | Every page, REST `id` coherence, completeness, and safe next-link containment | Controller-mediated boundary/property tables | `githubplanning` pagination tests | Order-independent literal Result evidence and request log proving no escaping follow |
| GHPO-7 | Fixed GitHub.com REST request contract | Recording Transport inspection | `githubplanning` request tests | Literal method, URL, query, media type, version, page size, and User-Agent |
| GHPO-8 | Read-only, stateless, concurrent reconstruction | Repeated-call and barrier-based race tests plus diff inspection | `githubplanning` lifecycle tests and conformance review | Different current Results per call, isolated concurrent Results, race pass, no persistence/credential-storage code |

`ResourceNumber` preventing direct raw-integer use is compile-time contract evidence; it cannot prove where a caller obtained an integer before explicit construction. No-authoritative-store, no credential copy, and no mutation beyond HTTP reads are confirmed by public dependency and diff review in addition to behavioral tests.

### 6.2 Payload producer and decoder

| Behavior | Test evidence |
|---|---|
| Canonical Milestone and Issue payloads | Literal expected complete description/body bytes, including independently calculated base64url |
| Every JSON escape class and exact Unicode preservation | Literal encoded bytes plus Controller-mediated public Result determination, Difference payload, and unavailable Location |
| Valid alternate JSON member order, whitespace, and escapes | Hand-authored payloads accepted with literal semantic values |
| Noncanonical base64url, padding, invalid UTF-8/JSON, duplicate top-level names, nested duplicate names inside ignored/localized values, unknown version, and unpaired surrogates in known/unknown names and values | Literal Plan-location availability outcomes, including ignored unknown values, locally incomplete non-string members, and envelope-level unusable names |
| Missing/wrong-type fields and mixed-type condition array | Independent field-local values and incomplete collection assertions |
| Empty acceptance array | Complete empty collection leading to member absence, not aggregate invalid evidence |
| Human narrative edits | Identical Controller-mediated public Result evidence from different suffix bytes |

Required combination cases prevent first-failure short-circuiting:

- missing Goal together with a mixed acceptance array and invalid Issue target date preserves every independent outcome;
- a hand-authored blank Goal becomes `InvalidObservedCategory` with `plan.InvalidText` at Goal while independently valid facts remain available;
- valid strings before and after non-string, blank, and duplicate acceptance members retain valid members, aggregate applicable Plan violations, and keep membership incomplete;
- unpaired high and low surrogates, one valid surrogate pair, escaped solidus, upper/lower `\u` hex digits, the four RFC JSON whitespace bytes, and a representative non-JSON whitespace byte use hand-authored payload literals;
- top-level duplicate names make the envelope unusable, while nested duplicate names inside an ignored unknown value or localized non-string acceptance member do not;
- padded base64url, standard `+/` alphabet, invalid encoded length, and non-zero unused trailing bits are separate literal cases whose expected meaning is calculated without production code.

### 6.3 Constructor and immutable binding

| Boundary | Cases |
|---|---|
| Client | nil, valid, and non-nil cookie Jar; literal validation category and field |
| Repository | zero/invalid owner, invalid name, unusual preserved valid values |
| Named representation constructor | Milestone and Issue each bind only their declared root kind; no representation flag exists |
| ResourceNumber | negative, zero, one, large positive; raw REST `id` cannot satisfy the typed parameter |
| Simultaneous invalid values | Client, zero Repository, and zero ResourceNumber precedence through public constructors; owner/name precedence remains `NewRepository` coverage |
| Construction side effects | Transport records zero requests |
| Redirect isolation | Original callback is not invoked; Provider uses `http.ErrUseLastResponse`; redirect is localized; original client policy remains unchanged |
| Response-body lifecycle | Instrumented bodies are closed on success, GitHub error, decode failure, and redirect |

After construction, tests mutate only copied `http.Client` value fields on the original client, such as `CheckRedirect` and `Timeout`, and prove the existing Observer is unchanged. Tests do not mutate the shared custom Transport because its immutability is a Host precondition. Body lifecycle cases include successful decode, HTTP error, body-read error, JSON-decode error, redirect, and later-page failure.

### 6.4 Milestone and Issue mapping

| Representation | Required cases |
|---|---|
| Milestone | title/payload/date, nil date, noncanonical date, open and closed Issues, assigned PR exclusion, zero/one/100 first-page Task boundaries |
| Issue | title/payload/date, zero/one/100 first-page Sub-issue boundaries, parent PR rejection, unexpected Sub-issue PR marker |
| Exactness | Whitespace, Unicode, distinct resources with equal titles, invalid titles, duplicate valid titles |

Expected observations and reconciliation results use literal Plan locations, violation codes, values, completeness, and determination rather than the production encoder or mapper as the test oracle.

### 6.5 Failure and response-shape localization

Each GHPO-5 matrix row expands into literal response-shape tables rather than one representative. Root documents cover null, array, malformed JSON, and object plus trailing value. Root number and collection `id` cover missing, null, zero, negative, fraction, overflow, and binding mismatch where applicable. Title, content, and date cover absent, null, and wrong JSON type independently. Status coverage includes representative `3xx`, `401`, `403`, `404`, `410`, `422`, `429`, and `5xx` responses. Every case asserts exact unaffected public Result evidence and the exact unavailable Location.

### 6.6 Pagination coherence

| Case | Required assertion |
|---|---|
| Multiple complete pages | Every distinct resource contributes in order-independent correspondence and collection is complete |
| 101 resources | The first full page and one later-page resource are both represented and membership is complete |
| Later page failure | Earlier coherent members remain; collection is incomplete |
| Same identity and title repeats | One member remains; collection is incomplete |
| Same identity with conflicting titles | Neither title remains; collection is incomplete |
| Different identities with equal titles | Duplicate-Task violation is observed |
| Repeated/escaping/malformed next relation | No unsafe follow; coherent members remain; collection is incomplete |

Same-identity conflicts use the fact-set permutations `A,A,B`, `A,B,A`, and reversed page order and produce identical public Result evidence with no title retained for the conflicting identity. Distinct identities with an equal title always produce the Plan-owned duplicate-Task violation.

Next-link tables include one valid GitHub form and separately reject duplicate `rel="next"`, changed scheme, host, port, userinfo, path, immutable `milestone`/`state`/`per_page` query, an additional selection query, zero/negative page, and a revisited page. Rejection performs no request to the candidate URL, retains coherent members, and makes Tasks incomplete.

### 6.7 Context, concurrency, and statelessness

- Cancellation and deadline tests use a synchronizable Transport rather than sleeps. Cancellation and deadline are each observed before the root read, during the root read, after the root succeeds, during a later page, and at the final completion check; every applicable case returns the exact `ctx.Err()`, the accompanying Observation is ignored, and no Provider error crosses. A direct nil-context call returns root unavailable with nil error and performs no request.
- Concurrent calls carry a test-only correlation value in each request context. A concurrent-safe recording Transport uses that value to return independent fact/page sets behind a barrier; assertions use each call's public Result rather than request order and run under `go test -race`.
- Consecutive calls with changed GitHub facts prove new reconstruction and absence of a correctness cache.
- Request inspection proves only `GET` methods, exact GitHub.com paths/query, page size 100, raw media type, fixed API version, and no redirect follow. Redirect tests assert no second request, no Host callback, correct localization, and body close; `http.ErrUseLastResponse` remains a design choice rather than a test oracle.

### 6.8 Verification commands

```sh
go test -v -race ./...
go vet ./...
golangci-lint run
go mod tidy -diff
openspec validate add-observable-github-plan-representation --strict
```

## 7. Open Questions

N/A. Deferred capabilities such as another payload version, API-version migration, retries, caching, mutation, and additional Providers require separate product Changes rather than implementation choices inside this one.
