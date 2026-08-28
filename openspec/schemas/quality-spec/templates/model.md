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

<!-- Requirement の解釈に必要な概念だけを残す。 -->

| Concept | Meaning | Identity / relevant values | Owner capability and rationale |
|---|---|---|---|
| <!-- canonical term --> | <!-- specification meaning --> | <!-- only what affects requirements --> | <!-- single owner + reason --> |

### 3-1. Supporting Models（任意）

<!--
概念一覧だけでは Requirement を一意に解釈できない場合に限り、必要な表現を追加する。
候補: 状態・分類・値域の表、関係図、不変条件、ライフサイクル、入力制約。
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

| Requirement slug | Actor and event | Guarantee | Concepts used | Important scenario classes |
|---|---|---|---|---|
| <!-- kebab-case --> | <!-- actor + trigger/action --> | <!-- observable contract --> | <!-- canonical concepts --> | <!-- happy/error/boundary/etc. --> |

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
