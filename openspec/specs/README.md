# Product Specification Rules

`openspec/specs/`は、consumerから観察・検証できる保証と、その解釈に必要な概念定義のSSOTである。
実装構造、調査メモ、未決事項、不具合の記録は置かない。

## 1. Artifactの責務

| Artifact | 答える問い | 書かないこと |
|---|---|---|
| proposal | なぜ変えるか。どの成果と範囲を扱うか | 技術方式、詳細な保証 |
| model | どの概念、状態、関係、制約が保証を支えるか | 確定していない規範、実装設計 |
| spec | 対象は何を保証するか | 実装方法、現行実装の偶然、未決事項 |
| design | 仕様をどう実現するか | 要求の再定義 |
| tasks | 何をどの順序で実装・検証するか | 新しい要求、設計判断 |

main specは次の順序で構成する。Conceptual Modelは分析用artifactの残骸ではなく、仕様を読むための概要である。
capability固有の用語、状態、分類、値域、関係、単位、同定規則、不変条件がある場合は省略しない。

```text
Purpose → Conceptual Model（必要な場合）→ Requirements
```

Requirement固有の外部SSOTは、そのRequirement内の`参照`に置く。Conceptual Modelの根拠は、同section内の
`### References`に置ける。archiveで維持できない独立した`## References`は使わない。

---

## 2. Capabilityの境界とpath

capabilityは、consumerから見た一貫した責務で切る。consumerには利用者、呼出し元、外部システム、
別component、自動処理を含む。新設前に既存capabilityを検索する。

| 対象 | 判定 | 置き場所 |
|---|---|---|
| 観察可能な機能 | consumer、契機、結果を説明できる | その保証を所有するcapability |
| 再利用される製品規則 | 複数capabilityが同じ意味へ依存する | 意味を最も強く規定するcapability |
| 実装だけの構造 | consumerが依存する契約を持たない | designまたはcode |

技術レイヤー、データ構造、framework、shared utilityを、その存在だけでcapabilityにしない。

capability IDは`openspec/specs/`から`spec.md`の親ディレクトリまでの相対pathである。各segmentは
kebab-caseとする。ネストしたpathを使用できる。

```text
openspec/specs/platform/search/spec.md
→ platform/search
```

---

## 3. Conceptual Model

Conceptual Model（概念モデル）は、Requirement群の解釈に必要な概念、状態、分類、関係、制約を定義する。
用語集や物理データ定義を転記する場所ではない。

次のいずれかを満たす概念だけを定義する。

- 複数のRequirementが参照する。
- 状態、分類、値集合、単位、同定規則を持つ。
- 概念間の関係や不変条件が振る舞いの意味を変える。
- 同語多義または同義語がRequirementの解釈を変える。
- Scenarioから逆算しなければ意味を一意に読めない。

テーブル、カラム、DTO、API payload、class、packageをConceptual Modelにしない。概念と物理構造の対応は
designが所有する。

| 層 | 定義するもの | 定義しないもの |
|---|---|---|
| Conceptual Model | 意味、同定、状態空間、分類値、関係、構造的不変条件、値域、単位 | 遷移契機、操作手順、副作用 |
| Requirement | 適用条件、入力の受理、判定、状態遷移、操作上の不変条件、結果、副作用、失敗時の保証 | 概念や分類値の初出定義 |
| Scenario | 具体的な前提と行為に対する観察可能な結果 | Requirementにない規範 |

各概念の定義元は1 capabilityに限定する。意味、不変条件、ライフサイクルを最も強く規定するcapabilityが
所有する。他のcapabilityは再定義せずRequirement IDまたはConceptual ModelのReferencesから参照する。

---

## 4. Requirement

Requirement IDは`<capability-path>/<requirement-slug>`である。cross-referenceは
`[[<capability-path>/<requirement-slug>]]`と書く。slugとpathの各segmentはkebab-caseとする。

```markdown
### Requirement: <stable-kebab-case-slug>

対象は、<規範の核心>する（MUST）。

- **前提条件**: <適用できる事前状態、権限、参照対象の存在条件>
- **入力と受理**: <入力、許容範囲、既定、受理・拒否条件>
- **振る舞いの規則**: <判定、計算、状態遷移、観察可能な出力>
- **不変条件**: <各許容結果の前後で常に維持する条件>
- **副作用**: <関連状態、履歴、通知への変化と変化させないもの>
- **排他・冪等**: <同時実行、再送、重複、原子性>
- **失敗の扱い**: <consumerが観察できる失敗結果>
- **参照**: <Requirementが依存する別SSOT>

#### Scenario: <concrete behavior> [happy]

- **GIVEN** <consumerと事前状態>
- **WHEN** <interactionまたはevent>
- **THEN** <観察可能な結果>
```

上のコードは記述形式を示す書き下ろし例である。ブロックは必要なものだけを使い、
`_schema/requirement-structure.json`の順序を守る。同じブロック内の箇条書きは同じ分類軸に揃える。

独立して変更・検証する保証が異なる場合はRequirementを分ける。入口や技術経路だけが異なり、
consumerから見た保証が同じ場合は分けない。

非機能要求も、対象、条件、測定方法、閾値、失敗時の保証が確定している場合はRequirementとして書く。
「高速」「安全」「大量」などの未定量な形容だけを規範にしない。

---

## 5. 仕様表現の選択

規則の構造から表現を選ぶ。すべてを散文またはScenarioへ展開しない。1つのRequirementに複数の構造が
含まれる場合は、必要な表現を併用する。該当する構造がない規則は簡潔な規範文で書く。

| 規則の構造 | 表現 | 規範を置く場所 |
|---|---|---|
| 数値、日時、version、件数など、連続または順序を持つdomainで範囲により結果が異なる | Partition Table | 受理規則は`入力と受理`、結果規則は`振る舞いの規則` |
| 到達可能な条件の組み合わせにより結果が異なる | Decision Table | `振る舞いの規則` |
| 概念がtriggerとguardによりlifecycle stateを移る | State Transition Table | `振る舞いの規則`。状態名と意味はConceptual Model |
| 常に維持する条件がある | Invariant | 概念の妥当性はConceptual Model、操作が維持する保証はRequirementの`不変条件` |
| 規則を具体的に例示または検証する | Scenario | 完全なRequirementの後 |

Partition Tableは次の列を使う。境界の包含・除外を明記し、意図しないgapやoverlapを残さない。値の意味、
単位、精度、時刻基準が自明でない場合はConceptual Modelで定義する。

| Partition | Condition or range | Acceptance or result |
|---|---|---|

Decision Tableは次の列を使う。異なる保証を生む到達可能な組み合わせを網羅し、既定結果があれば明記する。
Conceptual Modelが禁止する組み合わせを作らない。

| Rule | Preconditions or state | Input or event condition | Output or response | Side Effects |
|---|---|---|---|---|

State Transition Tableは次の列を使う。各状態の意味はConceptual Modelで一度だけ定義する。表にない遷移を
拒否、無視、またはcontract外のいずれとするかを明記する。

| Current state | Trigger or event | Guard | Next state | Output or Side Effects |
|---|---|---|---|---|

Invariantは成功経路や入力検証ではない。状態を変えるRequirementは、どの操作がInvariantを維持し、
維持できないときconsumerが何を観察するかを書く。

---

## 6. Scenario

各Requirementは主要な正常系を少なくとも1つ持つ。保証が変わる場合に限り、次の観点を追加する。

| Tag | 対象 |
|---|---|
| `happy` | 主要な正常結果 |
| `error` | 入力拒否、前提不成立、依存先の失敗 |
| `boundary` | 空、0、上限、期限境界、全件、部分件 |
| `permission` | 未認証、権限不足、許可範囲外 |
| `concurrency` | 同時更新、競合、順序逆転 |
| `idempotency` | 再送、重複、retry |
| `compatibility` | 既存consumer、既存data、contract互換性 |

GIVENは実行前から存在する状態、WHENはconsumerの行為またはevent、THENは外部から観察できる結果を書く。
Scenarioは完全なRequirementから導出される具体例であり、partition、decision rule、transition、Invariantの
唯一の置き場所にしない。内部関数の呼出しや特定テーブルへの書込みだけを期待結果にしない。

---

## 7. 外部contractと実装SSOT

| 情報 | SSOT | specの扱い |
|---|---|---|
| 製品規則、状態遷移、不変条件 | main spec | Conceptual ModelまたはRequirementに書く |
| APIの型、status、error code | OpenAPI、IDL | Requirementの`参照`から参照する |
| dataの型、制約、index | migration、schema | 概念上の意味だけを書き、物理定義を参照する |
| UIの配置、component、visual state | design system、UI artifact | consumerが観察できる操作と結果だけを書く |
| file、message、eventの項目contract | machine-readable schema | 既存SSOTを参照する |
| 不具合、実装乖離、調査メモ | issue tracker、audit record | specに書かない |

machine-readableなinterface SSOTがない場合は、`openspec/templates/interface-contract.md`をOpenSpec外の
管理場所へコピーして正本にできる。この文書はOpenSpec capabilityではない。interfaceを通じた観察可能な
保証は、それを所有するcapabilityのRequirementに置く。

参照型は`[related]`、`[interface]`、`[api]`、`[data]`、`[code]`、`[decision]`、`[policy]`、
`[external]`から選ぶ。参照は規範の代わりではなく、規範と別SSOTの接続を担う。

---

## 8. Deltaと公開

delta specで使用できるlevel-two sectionは次だけである。

- `## Purpose`: 新規capabilityだけに使用する。
- `## ADDED Requirements`
- `## MODIFIED Requirements`
- `## REMOVED Requirements`
- `## RENAMED Requirements`

`MODIFIED`は更新後のRequirement全体を載せる。独自sectionはOpenSpec 1.10.0のarchiveでmain specへ
反映されない。`## References`やConceptual Modelのdeltaを追加しない。

Conceptual Modelの変更は`model.md`の`Main Spec Conceptual Model Replacements`へ完全な置換後本文として書く。
公開には次を使う。

```bash
node tools/archive-change.mjs <change-name>
```

公開コマンドは未知のdelta sectionを拒否し、Conceptual ModelをstageしてからRequirement deltaをarchiveし、
公開後の全main specsをstrict validationする。directな`openspec archive`はConceptual Modelを反映しない。

---

## 9. 完成条件

- modelに仕様を変える未決事項が残っていない。
- capability境界と概念所有が既存main specsと矛盾しない。
- Conceptual Modelに未定義の重要語、分類値、単位、関係がない。
- 各Requirementが1つの独立した保証を持つ。
- 連続・順序domain、条件の組み合わせ、lifecycle、常時成立条件に対応する規範表現がConceptual ModelまたはRequirementにある。
- 各Requirementに検証可能なScenarioがある。
- ScenarioがRequirementにない規範を追加せず、規範表現の代わりになっていない。
- 実装詳細、現行実装の偶然、不具合、未決事項が規範に混ざっていない。
- Requirementの`参照`から別SSOTと関連Requirementを追跡できる。
- `openspec validate <change-name> --strict`が成功する。
- `REVIEW.md`のP0/P1指摘が解消されている。
- Quality Workflowでは`archive-change.mjs`による公開後strict validationが成功する。
