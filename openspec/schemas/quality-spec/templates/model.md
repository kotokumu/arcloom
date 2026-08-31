# Specification Analysis: <!-- change name -->

<!--
Requirement の理解と境界判断に必要な表現だけを使う。
該当しない表・節を埋めるための概念を作らず、不要な任意節は削除する。
-->

## 1. Boundary

| Item | Decision | Evidence |
|---|---|---|
| Product capability | <!-- existing/new capability --> | <!-- existing spec or source --> |
| Change classification | <!-- behavior change / pure implementation --> | <!-- reason --> |
| Included behavior | <!-- scope --> | <!-- proposal/source --> |
| Excluded behavior | <!-- non-goal --> | <!-- proposal/source --> |

---

## 2. Consumers and Observable Events

<!-- 仕様変更がある場合に記載する。pure implementation では「該当なし」と理由を書く。 -->

| Consumer / actor | Trigger / prior state | Interaction or event | Observable result |
|---|---|---|---|
| <!-- caller/user/system/process --> | <!-- trigger/state --> | <!-- action/event --> | <!-- result --> |

---

## 3. Conceptual Model

<!--
Requirementを読む前に必要な概念、状態、分類、関係、制約だけを残し、公開する簡潔な概要を選ぶ。
Scenarioから用語や状態空間を逆算させない。
-->

| Concept | Meaning | Identity / relevant values | Owner capability and rationale |
|---|---|---|---|
| <!-- canonical term --> | <!-- specification meaning --> | <!-- only what affects requirements --> | <!-- single owner + reason --> |

### 3-1. Supporting Models（任意）

<!--
概念一覧だけでは Requirement を一意に解釈できない場合に限り、必要な表現を追加する。
候補: 状態定義、分類、値域の表、関係図、構造的不変条件。
受理partition、条件の組み合わせ、transition rule、操作上のInvariant、出力、副作用はRequirementへ置く。
物理テーブル、DTO、API payload、class の図は作らない。
-->

---

## 4. Main Spec Conceptual Model Replacements

<!--
変更がなければ `None.` と書く。
変更があるcapabilityごとに、pathをcode表記したlevel-three見出しを置く。その直後の
markdown fenceへ`## Conceptual Model`見出しを除いた完全な置換後本文を書く。
差分や省略記号は書かない。各 capability には同じpathのRequirement deltaも必要である。
Conceptual Model節全体を削除する場合は、markdown fenceの代わりに`REMOVE`と書く。
`tools/archive-change.mjs` がこの内容をmain specへ反映し、Requirement deltaと一緒にarchiveする。
-->

None.

---

## 5. Requirement Candidates

| Requirement slug | Actor and event | Guarantee | Concepts used | Normative representations | Important scenario classes |
|---|---|---|---|---|---|
| <!-- kebab-case --> | <!-- actor + trigger/action --> | <!-- observable contract --> | <!-- canonical concepts --> | <!-- Partition / Decision / State Transition / Invariant / prose --> | <!-- happy/error/boundary/etc. --> |

---

## 6. Unresolved Decisions

<!--
現在の未決状態のSSOT。proposalのDecisions Requiredをすべて解決するか、ここへ引き継ぐ。
specs作成前に解決し、公開前は正確に`None.`と書く。推奨案を決定として扱わない。
-->

None.

---

## 7. Sources

- <!-- authoritative source used for the model -->
