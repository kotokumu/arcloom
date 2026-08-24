## Purpose

Defines how a read-only GitHub Planning Provider reconstructs a provider-independent Plan observation from one bound GitHub Milestone or parent Issue without owning or mutating the external facts.

## ADDED Requirements

### Requirement: GHPO-1 Explicit target binding
A GitHub Plan representation observation SHALL be bound for its lifetime to one locally valid Repository, one explicitly selected Milestone or Issue representation, and one valid GitHub `ResourceNumber`. `ResourceNumber` SHALL be a distinct immutable value representing a positive GitHub Milestone or Issue `number`, not a GitHub REST `id`; construction from a non-positive integer SHALL fail with the stable `invalid_resource_number` category and `ResourceNumberField`.

Separate Milestone-observer and Issue-observer constructors SHALL make representation selection explicit without a representation flag. Observer construction SHALL reject a missing HTTP client, a client with a non-nil cookie Jar, an invalid Repository, or a zero or otherwise invalid `ResourceNumber` before any GitHub request. Failure SHALL return a nil observation boundary and the existing typed GitHub-planning validation error extended with `InvalidClient`, `InvalidResourceNumber`, `ClientField = 5`, and `ResourceNumberField = 6`; existing exported category and field values SHALL remain unchanged. When simultaneous Observer inputs are invalid, precedence SHALL be client, Repository, then ResourceNumber. The zero Repository SHALL produce `InvalidRepository` with `RepositoryOwnerField`; owner/name validation and their precedence remain the existing `NewRepository` contract rather than an Observer-constructor decision. Construction SHALL NOT verify remote existence, access, permissions, or lifecycle.

The Provider SHALL use an internal copy of the Host-supplied client, SHALL set `CheckRedirect` to return `http.ErrUseLastResponse` without invoking the Host callback, and SHALL close every received response body. It SHALL NOT mutate the supplied client. Rejecting a non-nil cookie Jar SHALL prevent observation from mutating Host-owned cookie state. The Host SHALL keep a custom Transport and its reachable collaborators safe for concurrent requests and SHALL NOT mutate them after construction. Provider-native target identity SHALL remain inside the GitHub Provider and SHALL NOT appear in `planrepresentation.Observation`.

#### Scenario: Milestone target is bound
- **WHEN** a valid GitHub client, Repository, and ResourceNumber are supplied to the Milestone-observer constructor
- **THEN** the resulting observation boundary can observe only that Milestone representation

#### Scenario: Issue target is bound
- **WHEN** a valid GitHub client, Repository, and ResourceNumber are supplied to the Issue-observer constructor
- **THEN** the resulting observation boundary can observe only that parent Issue representation

#### Scenario: Binding is structurally invalid
- **WHEN** the client is missing or has a cookie Jar, the Repository is invalid, or the ResourceNumber is zero or otherwise invalid
- **THEN** construction fails with the first applicable stable validation category and field, performs no GitHub request, and returns a nil observation boundary

#### Scenario: Resource number is distinguished from REST identity
- **WHEN** a caller selects a root resource for observation
- **THEN** it must construct a ResourceNumber and cannot pass a raw GitHub REST `id` to an observer constructor

### Requirement: GHPO-2 Versioned payload decoding
The Provider SHALL inspect only a machine-readable block at the beginning of the Milestone description or parent Issue body and SHALL NOT parse the human-readable narrative. It SHALL recognize the exact `<!-- arcloom-plan:v1\n` prefix and `\n-->` suffix defined by GPCD-2. The encoded text SHALL use the URL-safe base64 alphabet without padding and SHALL be canonical by exact decode-and-re-encode comparison. Decoded bytes SHALL be valid UTF-8 and contain exactly one JSON object, optionally surrounded or separated by JSON whitespace, with no trailing non-whitespace data and no duplicate top-level payload-object member names. Duplicate-name detection SHALL NOT inspect nested objects inside an ignored unknown value or a non-string acceptance-condition member; those values remain governed by the local ignore or incomplete-collection rules below. The decoder SHALL accept any valid JSON member order and JSON string escape form; canonical producer encoding is not an acceptance requirement. Unknown version-one members SHALL be ignored. A Milestone payload SHALL provide exactly one string `goal` and one array `acceptance_conditions`; an Issue payload SHALL additionally provide `target_date` as either a string or `null`.

Payload failures SHALL map as follows:

| Failure | Goal | Acceptance conditions | Issue target date |
|---|---|---|---|
| Block absent, prefix/version unsupported, suffix absent, noncanonical or invalid base64url, invalid UTF-8, non-object JSON, malformed JSON, duplicate top-level payload-object member name, or trailing non-whitespace data | Unavailable | Incomplete with no known members | Unavailable |
| `goal` missing or not a string | Unavailable | Decode independently | Decode independently |
| `acceptance_conditions` missing or not an array | Decode independently | Incomplete with no known members | Decode independently |
| A condition array member is not a string | Decode independently | Preserve every string member and mark the collection incomplete | Decode independently |
| `target_date` missing or neither a string nor `null` | Decode independently | Decode independently | Unavailable |

The envelope-level row takes precedence because individual members cannot be established from an unusable envelope. Otherwise each member row applies independently, and simultaneous member faults produce the union of their localized outcomes while preserving every independently decoded field. A top-level member name containing an unpaired UTF-16 surrogate escape or another sequence that cannot be recovered as an exact Unicode scalar sequence makes the envelope unusable, including when that name would otherwise be unknown. An unknown member with a valid name SHALL be ignored without interpreting its value, including a value string containing an unpaired surrogate. In a required known string value, an unpaired surrogate makes that field unavailable. In an acceptance-condition member, it excludes that member and makes the collection incomplete.

A successfully decoded scalar or string collection member that violates a Plan text or date invariant SHALL remain known as the applicable Plan violation rather than unavailable. A successfully decoded empty acceptance-condition array SHALL be a known complete empty collection; under PRR-4 it produces missing-member differences against an expected Plan and SHALL NOT invent the aggregate-only `MissingAcceptanceCondition` violation.

#### Scenario: Version-one payload is valid
- **WHEN** the bound resource begins with a valid payload for its representation
- **THEN** the exact Goal, acceptance-condition membership, and applicable Issue target date are available for the observation

#### Scenario: Human-readable narrative changes
- **WHEN** the human-readable Markdown after a valid machine-readable block is edited without changing the block
- **THEN** the reconstructed Plan observation is unchanged

#### Scenario: Payload syntax is unusable
- **WHEN** the block is absent, has an unsupported version, or cannot be decoded as the required JSON shape
- **THEN** Goal and acceptance-condition membership are unavailable and an Issue target date is unavailable while independently observable native facts remain usable

#### Scenario: One payload member is unusable
- **WHEN** one required payload member is missing or has the wrong JSON type while the other required members decode independently
- **THEN** only the Plan location backed by that member is unavailable

#### Scenario: Unknown member contains unusable text
- **WHEN** an unknown member has a valid name and its ignored value contains an unpaired surrogate
- **THEN** every required version-one member is decoded independently and remains unaffected

#### Scenario: Member name is not recoverable
- **WHEN** any top-level member name contains an unpaired surrogate
- **THEN** the payload envelope is unusable and every payload-backed location is unavailable

#### Scenario: Decoded Plan value is invalid
- **WHEN** a decoded Goal, acceptance condition, or Issue target date violates a Plan invariant
- **THEN** the affected location contains the applicable known Plan violation and is not classified as unavailable solely because of that violation

#### Scenario: Acceptance-condition array is empty
- **WHEN** a valid payload contains an empty acceptance-condition array
- **THEN** acceptance-condition membership is known complete and empty without an aggregate-only missing-condition violation

### Requirement: GHPO-3 Milestone representation observation
For a bound Milestone representation, the Provider SHALL obtain the Milestone and map its native title to Plan name, its versioned payload to Goal and acceptance conditions, its `due_on` value to the optional target date, and the titles of all non-Pull-Request Issues assigned to that Milestone to Tasks. It SHALL include open and closed Issues and consume every available page. An absent `due_on` SHALL mean a known absent target date. A `due_on` value SHALL map to a known target date only when it is exactly a valid Plan date followed by `T00:00:00Z`; any other present value SHALL be a known invalid-target-date violation. Pull Requests assigned to the Milestone SHALL NOT be Plan Tasks.

#### Scenario: Complete Milestone representation is observed
- **WHEN** the Milestone, its valid payload, and every page of assigned Issues are available
- **THEN** the observation contains the exact title, Goal, acceptance conditions, non-Pull-Request Issue titles, and optional target date

#### Scenario: Closed Task Issue is assigned
- **WHEN** a closed Issue remains assigned to the bound Milestone
- **THEN** its exact title remains a member of the observed Task collection

#### Scenario: Pull Request is assigned
- **WHEN** a Pull Request is assigned to the bound Milestone
- **THEN** it does not become an observed Task

#### Scenario: Milestone due time is noncanonical
- **WHEN** a present `due_on` value is not exactly midnight UTC in the reversible form
- **THEN** the target-date location contains a known invalid-target-date violation

### Requirement: GHPO-4 Issue representation observation
For a bound Issue representation, the Provider SHALL obtain the parent Issue and map its native title to Plan name, its versioned payload to Goal, acceptance conditions, and optional target date, and the titles of all Sub-issues to Tasks. It SHALL consume every available Sub-issue page. A resource returned as a Pull Request SHALL not satisfy the parent Issue representation and SHALL make the Plan root unavailable.

#### Scenario: Complete Issue representation is observed
- **WHEN** the parent Issue, its valid payload, and every Sub-issue page are available
- **THEN** the observation contains the exact parent title, Goal, acceptance conditions, Sub-issue titles, and optional target date

#### Scenario: Parent resource is a Pull Request
- **WHEN** the bound Issue number identifies a Pull Request
- **THEN** the observation reports the Plan root unavailable and contains no descendant state

#### Scenario: Issue has no Sub-issues
- **WHEN** the Sub-issue collection is completely observed and empty
- **THEN** the Task collection is known complete and empty

### Requirement: GHPO-5 Observation of GitHub failures
The Provider SHALL use caller cancellation or deadline expiration as the only errors returned through the observation Port. It SHALL translate access denial, redirect responses, hidden or ambiguous not-found responses, gone responses, rate limiting, GitHub service failures, transport failures, unsupported response shapes, and other inability to establish GitHub facts into unavailable information at the least Plan location covering the affected facts. It SHALL NOT follow `3xx` responses or change the bound Repository or resource number. GitHub `404` and `410` responses SHALL make the affected root or collection unavailable and SHALL NOT be treated as authoritative Plan absence. This GitHub Provider SHALL never return an authoritative-absence observation because the selected API contracts provide no unambiguous absence signal. Provider errors, response bodies, request identifiers, authentication details, rate-limit details, redirect targets, and GitHub target identity SHALL NOT cross the observation Port.

If the returned Observer is invoked directly with a nil context, it SHALL perform no request and return a valid root-unavailable Observation with no error. The normal Controller call path remains governed by PRR-7 and rejects nil context before invoking any Observer.

Required response facts and their localization SHALL be:

| Unusable or contradictory response fact | Observation outcome |
|---|---|
| Root response cannot be obtained, decoded as one object, or matched by positive resource number to the immutable binding | Plan root unavailable |
| Bound Issue root contains the Pull Request marker | Plan root unavailable |
| Root title is absent, null, or not a string | Plan name unavailable; other root facts decode independently |
| Root description/body is absent, null, or not a string | Apply the all-payload-backed-fields row of GHPO-2 |
| Milestone `due_on` is absent or null | Target date known absent |
| Milestone `due_on` is present but not a string | Target date unavailable |
| Milestone `due_on` is a string that is not the exact reversible midnight-UTC form | Known invalid-target-date violation |
| Collection response cannot be obtained or decoded as an array | Preserve previously coherent Task members and mark the Task collection incomplete |
| Collection item identity is absent, non-positive, or not an integer | Exclude that item and mark the Task collection incomplete |
| Collection item title is absent, null, or not a string | Exclude that item and mark the Task collection incomplete |
| Milestone collection item is a Pull Request | Exclude it without making the Task collection incomplete |
| Issue Sub-issue item unexpectedly contains a Pull Request marker | Exclude it and mark the Task collection incomplete |
| Pagination metadata is malformed, repeats a page, or points outside the same bound endpoint and Repository | Do not follow it; preserve coherent members and mark the Task collection incomplete |

#### Scenario: Root request is inaccessible
- **WHEN** GitHub denies or cannot complete the request for the bound Milestone or parent Issue
- **THEN** the observation reports only the Plan root unavailable and returns no GitHub error through the observation Port

#### Scenario: GitHub returns not found
- **WHEN** the bound-resource request returns `404`
- **THEN** the observation reports the Plan root unavailable rather than authoritative absence

#### Scenario: GitHub redirects or reports gone
- **WHEN** a root or collection request returns a redirect or `410`
- **THEN** the Provider does not follow the redirect or rebind the target and marks the affected root or Task collection unavailable

#### Scenario: Caller cancels observation
- **WHEN** the supplied context is cancelled or its deadline expires before observation returns
- **THEN** the observation boundary returns that context error and no usable Observation; the consumer ignores any Observation value accompanying an error

#### Scenario: Observer is called directly with nil context
- **WHEN** a caller outside the normal Controller path invokes the GitHub Observer with nil context
- **THEN** it performs no request and returns a root-unavailable Observation with no error

#### Scenario: Payload-backed facts are unusable
- **WHEN** the root resource is available but its machine-readable payload cannot establish some payload-backed facts
- **THEN** native facts remain available and unavailability is limited to Goal, acceptance-condition membership, and the Issue target date as applicable

### Requirement: GHPO-6 Paginated collection integrity
The Provider SHALL request all collection pages with the GitHub REST maximum supported page size. It SHALL identify an Issue or Sub-issue across pages only by its positive integer GitHub REST `id` field inside the Provider boundary; Repository-local `number` and GraphQL `node_id` SHALL NOT determine identity. Repeated appearances of the same GitHub resource with the same exact title SHALL contribute one observed member and SHALL make collection membership incomplete. Repeated appearances of one identity with different titles SHALL contribute neither conflicting title and SHALL make collection membership incomplete. Distinct GitHub resources with equal exact titles SHALL remain distinct observations so that the Plan observation reports the duplicate-member violation. Failure to obtain a later page SHALL preserve coherent members already obtained and SHALL mark the affected collection incomplete. Collection response order SHALL NOT determine Plan correspondence.

#### Scenario: Collection spans multiple pages
- **WHEN** GitHub reports another page for assigned Issues or Sub-issues
- **THEN** the Provider reads every page before reporting complete collection membership

#### Scenario: Later page fails
- **WHEN** at least one coherent member is obtained before a later collection page cannot be obtained
- **THEN** the observation preserves that member and marks the Task collection incomplete

#### Scenario: Same resource repeats across pages
- **WHEN** the same GitHub resource identity appears more than once during one paginated read
- **THEN** equal titles contribute one Task member, conflicting titles contribute neither title, and the Task collection is marked incomplete

#### Scenario: Distinct resources share a title
- **WHEN** two different GitHub Issue identities have the same exact title
- **THEN** the observation reports the applicable duplicate-Task violation rather than deduplicating them by title

### Requirement: GHPO-7 GitHub API contract
The Provider SHALL target GitHub.com REST API version `2022-11-28`, request raw Markdown bodies, and use the get-Milestone, get-Issue, list-repository-Issues, and list-Sub-issues contracts applicable to the selected representation. It SHALL request both open and closed repository Issues for Milestone Tasks and SHALL request the maximum supported page size of 100 for repository Issues and Sub-issues. Authentication material and client configuration SHALL be supplied by the Host and SHALL remain outside Plan and Plan Representation contracts. GitHub Enterprise Server SHALL remain outside this contract. GitHub has announced March 10, 2028 as the end of support for API version `2022-11-28`; migration to another version requires a separate contract change, and an unsupported-version response under this contract maps to unavailability at the affected root or collection.

#### Scenario: Milestone requests are inspected
- **WHEN** the Milestone representation is observed
- **THEN** the requests identify the bound Repository and Milestone number, use the fixed API version, and request all Issue states without requesting a mutation

#### Scenario: Issue requests are inspected
- **WHEN** the Issue representation is observed
- **THEN** the requests identify the bound Repository and parent Issue number, use the fixed API version, request the raw body and all Sub-issue pages, and request no mutation

### Requirement: GHPO-8 Read-only stateless observation
Each observation SHALL reconstruct current state from new GitHub reads and SHALL require no prior observation, result, cache, or Arcloom-owned durable state. The same configured observation boundary SHALL support concurrent calls without sharing per-call response, pagination, payload, member, or error state. Observation SHALL perform no external mutation, Change Authorization, persistence, Provider-assigned identifier generation, or local credential storage.

#### Scenario: GitHub facts change between observations
- **WHEN** two calls receive different current GitHub facts
- **THEN** each returned observation reflects only the facts read by that call

#### Scenario: Observations run concurrently
- **WHEN** independent callers use the same configured observation boundary concurrently
- **THEN** each call returns an isolated observation without mixed pages, payloads, members, errors, or a data race

#### Scenario: Runtime state is discarded
- **WHEN** all prior runtime state is lost before a later observation
- **THEN** the Provider can reconstruct the observation from the binding, Host-supplied client, and new GitHub reads
