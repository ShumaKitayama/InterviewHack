# GoogleService-Info（コミットしない）

DesignDocument/5_FrontendDesign.md 5.12.2 に従い、Firebase の設定ファイルはリポジトリに入れません。
Firebase Console から dev / prod の 2 つのプロジェクトの plist をダウンロードし、このディレクトリに次の名前で置いてください。

- `GoogleService-Info-Dev.plist`（Debug-Dev 構成で使う）
- `GoogleService-Info-Prod.plist`（Release-Prod 構成で使う）

ビルド時に Run Script が構成に応じたファイルを `GoogleService-Info.plist` としてアプリにコピーします。
