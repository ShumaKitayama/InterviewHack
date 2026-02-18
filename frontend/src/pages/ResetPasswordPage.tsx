// DesignDocument/5_FrontendDesign.md 5.5 / 6_AuthAccountCahrgingDesign.md 6.2.5
import { useState } from "react";
import { resetPassword } from "../lib/supabase/auth";

export function ResetPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const { error } = await resetPassword(email);
    if (error) {
      setError(error.message);
    } else {
      setSent(true);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="max-w-md w-full space-y-6">
        <h1 className="text-2xl font-bold text-center text-gray-900">
          パスワードリセット
        </h1>
        {sent ? (
          <p className="text-center text-green-600">
            リセットメールを送信しました。メールを確認してください。
          </p>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="メールアドレス"
              required
              className="w-full px-4 py-3 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
            />
            {error && <p className="text-sm text-red-600">{error}</p>}
            <button
              type="submit"
              className="w-full py-3 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
            >
              リセットメールを送信
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
