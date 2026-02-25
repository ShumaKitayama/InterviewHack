# Google OAuth 認証の設定手順

InterviewHack では **Supabase Auth** で Google OAuth を利用します（設計書 6_AuthAccountChargingDesign.md）。  
フロントの「Googleで続行」ボタンとコールバック処理は実装済みです。以下で **Google Cloud Console** と **Supabase Dashboard** を設定すると、Google ログインが動作します。

---

## 1. Google Cloud Console での設定

### 1.1 プロジェクトと OAuth 同意画面

1. [Google Cloud Console](https://console.cloud.google.com/) にログイン
2. プロジェクトを選択（または新規作成）
3. **API とサービス** → **OAuth 同意画面**
   - ユーザータイプ: **外部**（一般ユーザー向け）または **内部**（テスト用）
   - アプリ名・ユーザーサポートメール・デベロッパー連絡先を入力
   - **スコープ**: デフォルトの `email`, `profile`, `openid` で可（Supabase が利用）

### 1.2 OAuth 2.0 クライアント ID の作成

1. **API とサービス** → **認証情報** → **認証情報を作成** → **OAuth クライアント ID**
2. アプリケーションの種類: **ウェブアプリケーション**
3. 名前: 任意（例: `InterviewHack Web`）
4. **承認済みの JavaScript 生成元**
   - ローカル: `http://localhost:5173`
   - 本番: `https://あなたのドメイン`
5. **承認済みのリダイレクト URI**
   - **Supabase のコールバック URL** を 1 件追加する必要があります。
   - Supabase Dashboard で確認（次のセクション 2.1 で表示される URL をコピー）
   - 形式例: `https://xxxxxxxxxxxx.supabase.co/auth/v1/callback`
6. **作成** をクリックし、**クライアント ID** と **クライアントシークレット** を控える

---

## 2. Supabase Dashboard での設定

### 2.1 Google プロバイダーを有効化

1. [Supabase Dashboard](https://supabase.com/dashboard) でプロジェクトを開く
2. **Authentication** → **Providers** → **Google** を開く
3. **Enable Sign in with Google** を ON
4. **Client ID**: Google Cloud で発行したクライアント ID を貼り付け
5. **Client Secret**: Google Cloud で発行したクライアントシークレットを貼り付け
6. **Save** をクリック

ここに表示されている **Callback URL**（例: `https://xxx.supabase.co/auth/v1/callback`）を、上記 1.2 の「承認済みのリダイレクト URI」に必ず追加してください。

### 2.2 リダイレクト URL の許可

1. **Authentication** → **URL Configuration**
2. **Redirect URLs** に、アプリ側のコールバック URL を追加:
   - ローカル: `http://localhost:5173/auth/callback`
   - 本番: `https://あなたのドメイン/auth/callback`

これにより、Google 認証後に Supabase から上記 URL へリダイレクトされます。

---

## 3. フロントの環境変数

`.env` に以下が設定されていることを確認してください。

```env
VITE_SUPABASE_URL=https://xxxxxxxxxxxx.supabase.co
VITE_SUPABASE_ANON_KEY=eyJ...
```

オプションで、コールバックのベース URL を明示する場合:

```env
VITE_APP_ORIGIN=http://localhost:5173
```

（未設定の場合は `window.location.origin` が使われます。）

---

## 4. 動作確認

1. フロントを起動: `npm run dev`
2. ブラウザで `http://localhost:5173/login` を開く
3. **Googleで続行** をクリック
4. Google の認証画面 → 許可後、`/auth/callback` を経由して `/` に遷移すれば成功

エラーになる場合の確認ポイント:

- Google Cloud の「承認済みのリダイレクト URI」に **Supabase の Callback URL** が 1 件あるか
- Supabase の「Redirect URLs」に **`http://localhost:5173/auth/callback`** が含まれるか
- ブラウザのコンソール・ネットワークタブでエラー内容を確認

---

## 5. 本番・ステージング時

- **承認済みの JavaScript 生成元** と **承認済みのリダイレクト URI** に本番 URL を追加
- Supabase の **Redirect URLs** に本番の `https://あなたのドメイン/auth/callback` を追加
- 必要に応じて Google の「アプリの公開」と審査（外部ユーザー向けの場合）

以上で Google OAuth 認証の設定は完了です。
