# arcloom

既存の AI エージェントを外部プロセスとして使い、差し替え可能な業務アプリと実行・評価ドライバで動かす Agent OS のプロトタイプです。[設計原文 v0.1](docs/design/agent-os-v0.1.original.md) を保存し、その Stage 1 の一部を実装しています。Stage 1 全体の完成ではなく、既存の Go 実装も維持しています。

ビルド条件、実行方法、状態の確認方法と未実装の範囲は [Agent OS README](agent-os/README.md) を参照してください。ローカルデモはリポジトリのルートから実行できます。

```bash
bash agent-os/examples/demo.sh
```
