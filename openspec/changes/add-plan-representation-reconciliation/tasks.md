## 1. Plan-Owned Reusable Validation

- [x] 1.1 Use the `go-test-authoring` workflow to generate failing table-driven tests for standalone Plan-name validation equivalence, including all text boundaries in DesignDoc section 6.8.
- [x] 1.2 Implement `plan.ValidateName` by reusing the Plan-owned name rule and preserve the existing typed validation contract.
- [x] 1.3 Use the `go-test-authoring` workflow to generate failing tests for acceptance-condition and Task collection validation, including nil, empty, zero values, exact duplicates, distinct Unicode, input indexes, and simultaneous independent violations.
- [x] 1.4 Implement `plan.ValidateAcceptanceConditions` and `plan.ValidateTasks`, and refactor `plan.New` to reuse them without changing aggregate completeness or violation precedence guarantees.
- [x] 1.5 Run the `plan` package tests and confirm existing Plan behavior remains unchanged.

## 2. Target-Independent Reconciliation Result

- [x] 2.1 Use the `go-test-authoring` workflow to generate failing tests for generic Core Result determination, zero-value behavior, valid empty evidence, input snapshots, and output snapshots.
- [x] 2.2 Implement the `reconciliation` package with `Determination` and generic `Result[D,U]` using collection snapshots and evidence-derived determination.
- [x] 2.3 Verify the Core Result does not inspect or depend on Plan-specific evidence types.

## 3. Plan Representation Observation Contract

- [x] 3.1 Use the `go-test-authoring` workflow to generate failing black-box tests for Location construction, member validation, zero Location behavior, and every least-unavailable-location boundary in DesignDoc section 6.3.
- [x] 3.2 Implement Plan Location values and the least-covering rule without Provider identifiers or resource vocabulary.
- [x] 3.3 Use the `go-test-authoring` workflow to generate failing public-constructor tests for root, scalar, target-date, collection, zero-child, duplicate-valid-member, and repeated-invalid-member Observation states.
- [x] 3.4 Implement immutable Observation values, typed `ObservationValidationError`, scalar/date classifiers, and complete/incomplete collection constructors using Plan-owned validators.
- [x] 3.5 Verify Observation constructor input slices are not retained and no public Observation accessor exposes private state or Provider details.

## 4. Semantic Correspondence and Evidence

- [x] 4.1 Use the `go-test-authoring` workflow to generate failing Controller behavior tests for exact scalar and collection correspondence, including case, whitespace, line endings, Unicode sequences, renamed Tasks, and collection order.
- [x] 4.2 Generate failing tests for target-date presence, value, invalid, and unavailable states.
- [x] 4.3 Generate failing tests for invalid observed Plan names, Goals, acceptance conditions, Tasks, target dates, valid duplicates, repeated invalid values, and violation aggregation.
- [x] 4.4 Generate failing tests for complete/incomplete membership, known evidence preservation, root absence/unavailability covering, and unavailable ordering.
- [x] 4.5 Generate the canonical Difference fixture from DesignDoc section 6.4 and assert the complete literal variant, location, payload, violation-code, and raw UTF-8 order sequence.
- [x] 4.6 Implement Semantic Correspondence as private stateless behavior in `planrepresentation`, keeping it separate from Controller call coordination.
- [x] 4.7 Implement closed immutable Meaning and Difference variants, unavailable evidence, Plan-specific covering/aggregation/ordering, and the Plan Representation Result over `reconciliation.Result`.
- [x] 4.8 Add Result and root-Plan-payload snapshot tests without inspecting private layouts.

## 5. Controller Port and Lifecycle

- [x] 5.1 Use the `go-test-authoring` workflow to generate failing tests for nil Observer construction, zero Controller, nil context, cancelled context, expired deadline, invalid Plan, invalid Observation, non-context Observer error, unrelated context error, and Observation-plus-error priority.
- [x] 5.2 Implement the named Observer function Port, `FailureError`, `NewController`, and `Reconcile` with the specified failure precedence and no Provider-error wrapping.
- [x] 5.3 Add channel-synchronized tests for cancellation immediately before successful return, cancellation before Observer failure return, and cancellation after Result return.
- [x] 5.4 Add sequential current-state reconstruction tests proving no prior Observation or Result is required.
- [x] 5.5 Add correlation-value and barrier-based concurrent reconciliation tests that do not assert call order and prove Result isolation.
- [x] 5.6 Run the concurrent suite with the Go race detector and remove all shared per-call state or races.

## 6. Conformance and Review

- [x] 6.1 Run `go test -v -race ./...` and the repository lint workflow locally.
- [x] 6.2 Run `go mod tidy -diff` and confirm the module remains tidy.
- [x] 6.3 Capture `go doc` evidence that target identity, Provider DTOs, Provider errors, and mutation contracts do not appear in the exported `reconciliation` or `planrepresentation` APIs.
- [x] 6.4 Review imports and `go list -deps` evidence that `reconciliation` depends only on the standard library and `planrepresentation` does not depend on `githubplanning` or another Provider Package.
- [x] 6.5 Verify the current Change contains no concrete Provider translation and record the normal and failure projection contract tests as obligations of each future Planning Provider Change.
- [x] 6.6 Run `openspec validate add-plan-representation-reconciliation --strict` and `git diff --check`.
- [x] 6.7 Obtain post-implementation architecture, SOLID, interface, procedural-code, and test-specification reviews; resolve all Blocking and High findings.
- [x] 6.8 Confirm implementation and tests trace every requirement and scenario without adding persistence, mutation, authorization, or target identity.
