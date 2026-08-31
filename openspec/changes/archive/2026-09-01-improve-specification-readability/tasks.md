## 1. Methodology

- [x] 1.1 Adapt the latest `~/openspec-template` rules for reader-facing Conceptual Models, representation selection, Invariants, and example-only Scenarios into Arcloom's methodology.
- [x] 1.2 Apply the same guidance to Quality Workflow instructions and templates.
- [x] 1.3 Upgrade `requirement-structure.json` while retaining the established Japanese labels.

## 2. Corpus conformance

- [x] 2.1 Rewrite all eleven capabilities with rule-appropriate normative representations.
- [x] 2.2 Rename all 36 legacy numbered Requirement titles to stable kebab-case IDs.
- [x] 2.3 Normalize unsupported Requirement blocks and add allowed tags to every retained Scenario.
- [x] 2.4 Remove redundant examples only after their outcomes are represented by a rule, table row, Invariant, or retained Scenario.

## 3. Equivalence review

- [x] 3.1 Review every `reconciliation-control-loop` guarantee, including publication and cancellation races.
- [x] 3.2 Review every `plan-reconciliation-loop` guarantee, including binding, fresh observation, classification, and read-only behavior.
- [x] 3.3 Review the other nine capabilities under the same guarantee-preservation rule.
- [x] 3.4 Review both Conceptual Model replacements for vocabulary, state completeness, relationships, and invariants.

## 4. Standard validation

- [x] 4.1 Configure and run remark-lint with GFM parsing and maintained rules, without custom Markdown rules.
- [x] 4.2 Run strict Quality Workflow schema, main-spec, and active-Change validation.
- [x] 4.3 Run the Go test suite, race detector, vet, and diff checks.
- [x] 4.4 Confirm no P0 or P1 review issue remains.
