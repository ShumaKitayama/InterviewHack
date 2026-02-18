## 概要

<!-- 変更内容を簡潔に記述 -->

## 変更レイヤー

- [ ] Domain
- [ ] Usecase
- [ ] Adapter
- [ ] Infrastructure
- [ ] Frontend

## 関連設計書

<!-- 対応する設計書の章/節番号を記載 (Agent.md ルール) -->

- DesignDocument/

## 設計準拠チェックリスト

- [ ] 設計書を確認し、実装が一致している
- [ ] Domain層に外部SDK依存を導入していない (7_BackendDesign.md)
- [ ] BFF専用テーブルにフロントから直接書き込みしていない (8_DBDesign.md)
- [ ] クレジットは Reserve/Commit/Reversed の状態遷移に従っている (6, 7, 9)
- [ ] Webhook処理は冪等である (9_SecurityDesign.md)
- [ ] X-Request-Id を伝搬している (7_BackendDesign.md)
- [ ] RLSが有効なテーブルのポリシーを変更していない (8_DBDesign.md)
- [ ] シークレットをリポジトリにコミットしていない

## テスト

- [ ] ユニットテスト追加/更新済み
- [ ] 手動テスト実施済み
