## 1. Construction Gate

- [x] 1.1 Record explicit human approval of the high-risk DesignDoc after all Blocking and High review findings are resolved.

## 2. Provider-Independent Plan

- [x] 2.1 Write public table-driven tests for Plan text, Unicode, line-break, Target Date, zero-value, typed-error, cardinality, uniqueness, declared-order, snapshot, and defensive-copy behavior from PLN-1 through PLN-5.
- [x] 2.2 Implement immutable Plan local values and typed validation results with the minimum behavior required by the failing tests.
- [x] 2.3 Implement the Plan aggregate invariants and defensive ownership with the minimum behavior required by the failing tests.
- [x] 2.4 Refactor Plan validation so local and aggregate invariants each have one owner and the package has no Provider dependency.

## 3. GitHub Creation Dry-Run

- [x] 3.1 Write public tests for Repository validation, explicit representation, canonical Markdown byte concatenation, API version, and zero/error behavior from GPCD-1, GPCD-2, GPCD-6, and GPCD-7.
- [x] 3.2 Implement the GitHub Repository, representation, validation result, and private canonical narrative behavior required by the failing tests.
- [x] 3.3 Write public Milestone-representation tests covering request values, optional fields, declared Task order, Target Date encoding, and typed Milestone-number references from GPCD-3.
- [x] 3.4 Implement the Milestone creation Request Plan with the minimum request and reference values required by the failing tests.
- [x] 3.5 Write public Issue-representation tests covering request values, optional fields, declared Task order, parent-number and child-ID references, and the 100/101 Task boundary from GPCD-4.
- [x] 3.6 Implement the Issue creation Request Plan with the minimum request and reference values required by the failing tests.
- [x] 3.7 Write the public Request-reference property test across the specified Task-count boundaries and defensive-copy tests from GPCD-5.
- [x] 3.8 Refactor private representation mapping and Request Plan graph validation so Provider mapping decisions and graph invariants have separate owners without adding a public Strategy, client, executor, or generic Provider Port.
- [x] 3.9 Add the approved Plan-owned `IsValid()` contract with a failing public test, then remove Provider-side reconstruction of Plan invariants.
- [x] 3.10 Refactor request creation through private Request Plan construction so representation mappings do not calculate and reinterpret graph references independently; move the 100-Task constraint and target-date narrative completion to the Issue representation owner.
- [x] 3.11 Re-author incomplete tests with the required `go-test-authoring` workflow so literal public observations, every typed validation result, independent text/Repository boundaries, both-representation repeatability, and complete zero results are executable specifications.

## 4. Canonical Documents and Verification

- [x] 4.1 Align `PRODUCT.md` and `ARCHITECTURE.md` with Provider-native Milestones, Host-facing concrete Provider previews, and the unchanged Plan Change Target and Authorization path.
- [x] 4.2 Run `openspec validate plan-github-creation-dry-run --strict`, `go mod tidy -diff`, `go test -v -race ./...`, and `golangci-lint run` and resolve every failure after the approved review corrections.
- [x] 4.3 Capture dependency and public-surface evidence with `go list` and `go doc` and verify the allowlists and prohibited contracts in the DesignDoc after the approved review corrections.
- [x] 4.4 Perform final model, SOLID, architecture, interface, test, procedural-code, and code-quality reviews; reconcile every Blocking or High finding before completion.
