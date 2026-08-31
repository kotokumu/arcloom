## Purpose

<!-- New capability only. Write at least 50 concrete characters. Delete this section for an existing capability delta. -->

## ADDED Requirements

### Requirement: <!-- stable-kebab-case-slug -->

対象は、<!-- 規範の核心 -->する（MUST）。

- **前提条件**: <!-- 操作を適用できる事前状態・権限・存在条件 -->
- **入力と受理**: <!-- 入力、許容範囲、拒否条件 -->
- **振る舞いの規則**: <!-- 判定、計算、状態遷移、出力 -->
- **不変条件**: <!-- 各許容結果の前後で維持する条件 -->
- **副作用**: <!-- 関連状態・履歴・通知への変化と、変化させないもの -->
- **排他・冪等**: <!-- 同時実行、再送、原子性 -->
- **失敗の扱い**: <!-- 利用者またはconsumerが観察できる失敗結果 -->
- **参照**: <!-- optional: [interface]/[api]/[data]/[policy]/[external] SSOT path -->

<!--
規則の構造に応じて、該当block内で次の表現を使う。
- 連続または順序domain: Partition Table（Partition | Condition or range | Acceptance or result）
- 条件の組み合わせ: Decision Table（Rule | Preconditions or state | Input or event condition | Output or response | Side Effects）
- lifecycle: State Transition Table（Current state | Trigger or event | Guard | Next state | Output or Side Effects）
- 常時成立する操作保証: `不変条件`内の規範的Invariant
該当する表現だけを使う。規範表またはInvariantをmodel.mdやScenarioだけに残さない。
-->

#### Scenario: <!-- concrete behavior --> [happy]

- **GIVEN** <!-- actor and prior state -->
- **WHEN** <!-- actor action or external event -->
- **THEN** <!-- observable outcome -->

## MODIFIED Requirements

<!-- 既存 Requirement の見出しから全 Scenario までをコピーし、完全な更新後の内容を書く。 -->

## REMOVED Requirements

### Requirement: <!-- removed-slug -->

- **Reason**: <!-- removal reason -->
- **Migration**: <!-- consumer/data migration or none -->

## RENAMED Requirements

- FROM: `### Requirement: <old-slug>`
- TO: `### Requirement: <new-slug>`
