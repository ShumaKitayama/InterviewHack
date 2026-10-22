## 概要

<!-- 変更内容を簡潔に記述 -->

## 変更レイヤー

- [ ] Domain
- [ ] Usecase
- [ ] Adapter
- [ ] Infrastructure
- [ ] iOS（InterviewHackKit / App）
- [ ] CI/CD
- [ ] DesignDocument

## 関連設計書

<!-- 対応する設計書の章/節番号を記載（AGENTS.md ルール） -->

- DesignDocument/

## 設計準拠チェックリスト（14.5 共通ゲート）

- [ ] 設計書を確認し、実装が一致している（差分がある場合は設計を先に更新した）
- [ ] Domain 層に外部 SDK・ネットワーク・DB 依存を導入していない（7.2.1、`arch_check.sh`）
- [ ] iOS アプリは BFF 経由でのみデータに触れている（5、7）
- [ ] すべてのクエリを検証済みトークンの `user_id` でスコープしている（8、9）
- [ ] `prompt_sessions` の状態遷移（created → pasted → saved / abandoned）を守っている（2、7）
- [ ] commit は条件付き UPDATE で冪等。共有トークンはハッシュのみ保存（7、8、9）
- [ ] BFF から LLM API・任意 URL への外部呼び出しをしていない（4、7、9）
- [ ] `X-Request-Id` を伝搬している（5、7）
- [ ] 本文・プロンプト・貼り戻し回答・トークン・メールアドレスをログに出していない（9）
- [ ] シークレットをリポジトリにコミットしていない（9、14）

## テスト

- [ ] ユニットテスト追加/更新済み
- [ ] 統合テスト（`TEST_DATABASE_URL`）実行済み
- [ ] `swift test`（ios/ 変更時）実行済み
- [ ] 手動テスト実施済み
