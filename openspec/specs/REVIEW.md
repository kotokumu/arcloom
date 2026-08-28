# Specification Review

仕様の意味品質を確認するチェックリストである。OpenSpecの構文検証は`openspec validate`が担う。

## 1. 優先度

| Priority | 判定 | 承認条件 |
|---|---|---|
| P0 | 誤った規範、矛盾、未決事項の規範化、SSOT境界違反、公開時の情報欠落 | 必ず修正する |
| P1 | 実装・検証の再現性を損なう欠落または曖昧さ | 原則として修正する |
| P2 | 可読性、保守性、将来の誤解リスク | 修正または理由を記録する |

---

## 2. ProblemとScope

- proposalの問題、期待成果、成功条件が対応しているか。
- In ScopeとOut of Scopeがconsumerから見た境界を定めているか。
- 技術方式を成果または要求として固定していないか。
- capabilityの新設理由を既存capabilityとの差で説明できるか。
- nested capability pathの各segmentがkebab-caseか。

---

## 3. Conceptual Model

- Requirementが使う重要な対象、状態、分類、値、単位、関係を事前に定義しているか。
- 対象の同定と一意性が必要な箇所で明確か。
- 状態空間と遷移条件を混同していないか。
- 同じ概念を複数capabilityで再定義していないか。
- 物理DB、API payload、DTO、class、file構造を概念モデルとして転記していないか。
- 同義語、同語多義、未定量な形容が残っていないか。
- modelの`Unresolved Decisions`に仕様を変える事項が残っていないか。
- main specを変える場合、完全な置換後本文がmodelにあるか。
- テンプレートに欄があることだけを理由に概念や図を追加していないか。

---

## 4. Requirement

- 1 Requirementが1つの独立して変更・検証できる保証を持つか。
- 規範の核心がMUSTを含み、条件と保証を明確にしているか。
- ブロックの項目が同じ分類軸に揃っているか。
- 受理条件、状態遷移、出力、副作用、失敗時の保証に必要な欠落がないか。
- concurrency、idempotency、permission、compatibilityが関係する箇所で契約化されているか。
- 非機能要求が対象、条件、測定方法、閾値を持つか。
- 実装詳細、現行実装の偶然、不具合、暫定注記、未決事項が混ざっていないか。
- nested capabilityを含むRequirement IDが正しいpathを使うか。

---

## 5. Scenario

- 各Requirementに主要正常系があるか。
- 保証に関係するerror、boundary、permission、concurrency、idempotency、compatibilityを扱うか。
- GIVENがconsumerと事前状態、WHENがinteractionまたはevent、THENが観察可能な結果になっているか。
- Scenarioから入力と期待結果を一意に組み立てられるか。
- ScenarioがRequirementにない規範や未定義語を導入していないか。
- プロジェクトに存在しない観点をテンプレートに合わせて追加していないか。

---

## 6. SSOTとinterface

- API、data、UI、message、external contractの既存SSOTを重複していないか。
- Requirementの`参照`から関連Requirementと別SSOTを追跡できるか。
- interfaceの詳細と、そのinterfaceを通じた観察可能な保証を混同していないか。
- machine-readable interfaceが正本の場合、fieldやtypeをmain specへ転記していないか。
- Conceptual Modelと実装構造の対応はdesignが所有しているか。

---

## 7. Workflowと公開

- proposalのcapabilityとRequirement影響がdelta specsへ反映されているか。
- modelのRequirement Candidatesと実際のRequirementsに説明できない差がないか。
- designが全delta specsの後に作成され、WHATを再定義していないか。
- delta specのlevel-two sectionがPurposeと標準Requirement operationだけか。
- Conceptual Model置換を持つcapabilityに同じpathのRequirement deltaがあるか。
- apply中にmain specのConceptual Modelを先行更新していないか。
- tasksがRequirement IDと検証方法を持つか。
- 公開に`node tools/archive-change.mjs <change-name>`を使うか。

---

## 8. 最終確認

- [ ] P0/P1指摘が0件である。
- [ ] `openspec validate <change-name> --strict`が成功する。
- [ ] main specとdelta specの概念、用語、Requirement IDが矛盾しない。
- [ ] 実装者と検証者が追加の仕様判断なしで着手できる。
- [ ] 公開後の`openspec validate --specs --strict`が成功する。
