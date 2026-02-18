// DesignDocument/5_FrontendDesign.md 5.4.1 / ダッシュボード
import { useAuth } from "../features/auth/AuthContext";

export function HomePage() {
  const { user } = useAuth();
  return (
    <div className="p-4 space-y-4">
      <h1 className="text-2xl font-bold text-gray-900">ホーム</h1>
      <p className="text-gray-600">ようこそ、{user?.email ?? "ゲスト"}さん</p>
    </div>
  );
}
