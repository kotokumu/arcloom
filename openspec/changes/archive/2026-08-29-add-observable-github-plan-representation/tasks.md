## 1. Public Contract and Test Scaffolds

- [x] 1.1 Rename the existing `githubplanning` directory, package declaration, imports, and external-package tests to `githubplan` without changing behavior; run the existing race-enabled suite before adding contracts.
- [x] 1.2 Add the compile-only `githubplan` validation categories and fixed fields, typed ResourceNumber contract, and named Milestone/Issue Observer signatures defined by the DesignDoc without implementing observation behavior.
- [x] 1.3 Use the Go test-authoring workflow to generate applicable table-driven scaffolds for public functions and methods before hand-editing cases; test private payload and interpretation responsibilities only through public preview, Observer, Controller, and Result behavior.
- [x] 1.4 Add failing black-box tests for both named constructors, ResourceNumber boundaries, constructor success, nil output on failure, cookie-Jar rejection, stable simultaneous-input precedence, zero requests during construction, immutable target binding, and non-mutation of the supplied `http.Client`.

## 2. Payload Protocol

- [x] 2.1 Add failing behavior tests with independent literal oracles for canonical Milestone and Issue payload bytes, every JSON escape class, canonical unpadded base64url, and the unchanged human narrative.
- [x] 2.2 Implement the private shared Payload v1 producer so the existing creation request plan emits the canonical machine block without changing request topology or mutation boundaries.
- [x] 2.3 Add failing observer-boundary tests for accepted JSON member order, whitespace and escape variations; exact Unicode recovery; unknown members; unpaired surrogates in known/unknown names and values; malformed envelopes; duplicate top-level members; ignored nested duplicate names in unknown values; localized nested duplicate names in non-string acceptance members; noncanonical base64url; field-local faults; mixed-type and empty acceptance-condition arrays; and human-narrative independence.
- [x] 2.4 Implement the private Payload v1 decoder so it returns only syntactic and typed field outcomes; keep Plan-location assignment, Plan invariant classification, and Observation construction outside the codec.

## 3. Target Binding and GitHub REST Facts

- [x] 3.1 Add failing request and lifecycle tests for GitHub.com HTTPS URLs, escaped Repository segments, bound ResourceNumber, `GET` only, raw Markdown media type, `2022-11-28`, `state=all`, `per_page=100`, copied client-value stability, no second redirect request, no Host redirect callback, and body close on success, HTTP error, read error, decode error, and redirect.
- [x] 3.2 Implement the immutable Repository/Scheme/ResourceNumber Binding and separate REST access configuration using only `net/http`: reject cookie Jars, copy the client value, refuse redirects, close every received body, and perform no construction-time request.
- [x] 3.3 Add failing root and first-page response tests for Milestone and Issue using literal response bodies and public Result oracles.
- [x] 3.4 Implement root and first-page collection reads with private response representations, leaving multi-page coherence to section 5.

## 4. GitHub Representation Semantics

- [x] 4.1 Add failing Controller-mediated Milestone tests for native title, payload-backed fields including a blank Goal violation, absent/canonical/noncanonical `due_on`, open and closed assigned Issues, assigned Pull Request exclusion, exact titles, and zero/one/100 first-page Task boundaries.
- [x] 4.2 Implement the Milestone Representation Scheme's correspondence from typed GitHub fact and Payload outcomes to Plan locations, then construct one valid provider-independent Observation.
- [x] 4.3 Add failing Controller-mediated Issue tests for native title, payload-backed target date, zero/one/100 first-page Sub-issues, cross-Repository positive REST `id` identity, parent Pull Request rejection, and unexpected Sub-issue Pull Request shape.
- [x] 4.4 Implement the Issue Representation Scheme's correspondence from typed GitHub fact and Payload outcomes to Plan locations, then construct one valid provider-independent Observation.
- [x] 4.5 Add failing literal response-shape tables for every GHPO-5 localization row and simultaneous payload faults, including null/wrong/container/trailing shapes, integer boundaries, root identity mismatch, unusable title/content/date, unusable collection item `id`/title, and proof that `number` and `node_id` do not determine identity.
- [x] 4.6 Implement the private per-call GitHub Observation Fact Set as the sole owner of page admission, REST `id` coherence, same-identity conflict resolution, and completeness; keep GitHub-to-Plan-location correspondence exclusively in the selected Representation Scheme, then use Plan-owned classifiers and `planrepresentation` constructors to enforce Plan validity and Observation algebra; do not add an interpretation Policy, generic mapper, service, or Provider interface.

## 5. Pagination, Failure, and Lifecycle Behavior

- [x] 5.1 Add failing Controller-mediated pagination tables for 101-resource completion, later-page body close/failure, all same-`id` title permutations, distinct `id` values with equal titles, and each duplicate/malformed/repeated/escaping next-link component, asserting public Result evidence and no request to rejected links.
- [x] 5.2 Implement contained next-page traversal and per-call identity coherence with no shared mutable page or member state.
- [x] 5.3 Add failing boundary tests for redirects, `401`, `403`, `404`, `410`, `422`, `429`, `5xx`, malformed JSON, transport failure, and direct nil-context invocation, plus synchronized cancellation/deadline cases before root, during root, after root, during later page, and at final completion, asserting exact context or localized public outcome and no Provider leakage.
- [x] 5.4 Implement Provider-failure translation so only the supplied caller context error can cross the Observer Port and this GitHub Provider never produces authoritative absence.
- [x] 5.5 Add failing concurrent and repeated-call tests proving isolated facts, current-state reconstruction, no correctness cache, and no data race through one Observer.
- [x] 5.6 Make the Observer stateless and concurrent-safe under the declared Host client precondition.

## 6. Conformance and Review

- [x] 6.1 Run `go test -v -race ./...`, `go vet ./...`, `golangci-lint run`, `go mod tidy -diff`, and strict OpenSpec validation; resolve every failure.
- [x] 6.2 Review the implementation against the DesignDoc for Concept minimality, SOLID responsibility ownership, public-interface traceability, Provider boundary containment, and absence of procedure-centered abstractions; resolve every Blocking or High finding.
- [x] 6.3 Re-review the Go tests as behavioral specifications under the Go test-authoring workflow and resolve every Blocking or High finding.
- [x] 6.4 Confirm the diff contains no retained `githubplanning` package or non-archived import, external mutation, authorization path, persistence, credential storage, SDK dependency, GitHub Enterprise Server behavior, or unrelated change.
