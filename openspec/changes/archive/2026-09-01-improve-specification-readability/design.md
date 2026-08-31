## Context

This is a behavior-preserving specification refactor across the Quality Workflow and all eleven accepted main specs. Product concepts, responsibilities, state ownership, public Go contracts, and Architecture boundaries remain unchanged.

The latest `~/openspec-template` supplies the representation-selection semantics. Arcloom retains its project context and established Japanese Requirement block labels.

## Goals / Non-Goals

### Goals

- Keep one representation-selection rule across methodology, schema instructions, project rules, templates, and review criteria.
- Make every main spec readable from its Conceptual Model and Requirement rules before Scenarios.
- Preserve all 75 observable guarantees and assign stable kebab-case IDs to the 36 legacy numbered Requirements.
- Use a maintained Markdown linter for general Markdown syntax and style checks.

### Non-Goals

- Change product behavior, implementation code, tests, public contracts, Components, or dependency direction.
- Build an OpenSpec-specific Markdown parser.
- Add a custom publication protocol for removing Scenarios.
- Mechanically decide whether an author selected the semantically correct normative representation.

## Design

### Documentation ownership

| Concern | Owner | Representation |
|---|---|---|
| Representation selection | OpenSpec authoring methodology | `README.md`, `MODELING.md`, and `REVIEW.md` |
| Artifact-generation guidance | Quality Workflow | `openspec/config.yaml`, schema instructions, and templates |
| Canonical blocks and Scenario tags | Specification methodology | `requirement-structure.json` |
| General Markdown defects | remark-lint | Repository configuration and CI invocation |
| OpenSpec semantics | Requirement authors and reviewers | Strict OpenSpec validation plus the documented review criteria |
| Accepted guarantees | Existing capabilities | Eleven main `spec.md` files |

### Representation selection

Rules select their representation from their structure:

| Rule structure | Normative representation |
|---|---|
| Continuous or ordered domain partitions | Partition Table |
| Condition combinations with distinct results | Decision Table |
| Lifecycle triggers and guards | State Transition Table |
| Conditions that always hold | Invariant |
| Representative concrete behavior | Scenario |

Conceptual Models define vocabulary, classifications, state meanings, relationships, and structural invariants. Requirements define acceptance, decisions, transitions, operational invariants, outputs, side effects, and failures. Scenarios demonstrate rules and introduce no new normative meaning.

### Standard Markdown lint

The repository uses `remark-cli`, `remark-gfm`, and `remark-preset-lint-recommended` with versions pinned by `package-lock.json`. GFM parsing makes tables and other repository syntax explicit mdast nodes before the maintained lint rules inspect them.

The selection was based on a corpus trial rather than CLI convenience:

| Candidate | Corpus result | Decision |
|---|---|---|
| markdownlint rules through `markdownlint-cli2` | 2,814 findings in 61 files, dominated by table-column style conflicts | Rejected because useful enforcement would require broad style exceptions across the GFM-heavy specification corpus. |
| remark recommended preset with `remark-gfm` | 685 initial findings; 684 came from one identifiable OpenSpec syntax conflict and the remaining final-newline defect was valid | Selected because one documented rule exception preserves the maintained correctness-oriented preset without custom parsing or rules. |
| mdast alone | Defines the syntax tree but performs no validation | Used through remark, not treated as a standalone lint tool. |

`markdownlint-cli2` was only the runner used to evaluate the markdownlint rule family; its glob and configuration conveniences do not determine rule suitability.

The recommended preset's `no-undefined-references` rule is disabled declaratively because OpenSpec intentionally uses bracketed Scenario tags such as `[happy]`, double-bracket cross-references, and `[related]` block labels. No repository-specific parser, plugin, or lint rule is introduced.

Markdown lint does not validate Requirement block vocabulary, block order, Scenario tags, conceptual completeness, or semantic equivalence. Those concerns remain in the OpenSpec methodology and review process.

### Corpus migration

Each existing Scenario outcome is mapped to a Requirement rule, table row, Invariant, or retained Scenario before removal. Legacy numbered Requirement headings are renamed explicitly to stable kebab-case IDs. The two control-loop Conceptual Models are replaced because their prior overviews do not provide the state and result distinctions needed to read their Requirements.

## Responsibility Check

| Responsibility | Owner | Boundary |
|---|---|---|
| Select the semantically correct specification representation | OpenSpec methodology and reviewer | Cannot be inferred reliably from Markdown syntax. |
| Detect general Markdown defects | remark-lint | Uses the maintained mdast parser and rule set. |
| Validate OpenSpec artifact syntax | OpenSpec CLI | Uses the repository schema and strict validation. |
| Preserve product guarantees during rewriting | Existing capability spec and reviewer | Every old outcome must remain derivable from accepted normative text. |

No product Concept, Component, Package, Port, or public Go contract changes.

## Verification

- `npm run lint:markdown`
- `openspec schema validate quality-spec`
- `openspec validate --specs --strict`
- `openspec validate --changes --strict`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `git diff --check`

The corpus review confirms 75 Requirements remain under stable IDs and every retained Scenario is a representative example derived from its Requirement.

## Risks / Trade-offs

- Markdown lint cannot enforce OpenSpec-specific meaning. The methodology and review checklist remain authoritative for semantic structure.
- A concise rewrite can omit a rare guarantee. The guarantee-preservation review compares every removed Scenario outcome with the resulting normative text.
- Broad default lint rules can conflict with intentional OpenSpec formatting. Configuration exceptions are limited to documented structural conflicts rather than implemented as custom rules.
- Upstream template wording can diverge from Arcloom terminology. Updates require semantic comparison instead of byte-copying.

## Migration / Rollback

No product or data migration exists. Reverting the documentation commit restores the former presentation without affecting implementation or runtime state.

## Open Questions

None.
