// DesignDocument/5_FrontendDesign.md 5.5.1 /auth/callback
// Supabase OAuth コールバック: PKCE の code 交換、または implicit 時のセッション反映後にリダイレクト
import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { supabase } from "../lib/supabase/client";

export function AuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function handleCallback() {
      const code = searchParams.get("code");
      const hashParams = new URLSearchParams(
        window.location.hash?.replace(/^#/, "") || "",
      );
      const errorParam = searchParams.get("error") ?? hashParams.get("error");
      const errorDescription =
        searchParams.get("error_description") ??
        hashParams.get("error_description");

      if (errorParam) {
        setError(errorDescription || errorParam);
        setTimeout(() => navigate("/login", { replace: true }), 3000);
        return;
      }

      if (code) {
        const { error: exchangeError } =
          await supabase.auth.exchangeCodeForSession(code);
        if (cancelled) return;
        if (exchangeError) {
          setError(exchangeError.message);
          setTimeout(() => navigate("/login", { replace: true }), 3000);
          return;
        }
      }

      const {
        data: { session },
      } = await supabase.auth.getSession();
      if (cancelled) return;

      if (session) {
        navigate("/", { replace: true });
      } else {
        setError("セッションの取得に失敗しました");
        setTimeout(() => navigate("/login", { replace: true }), 3000);
      }
    }

    handleCallback();
    return () => {
      cancelled = true;
    };
  }, [navigate, searchParams]);

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4 px-4">
        <p className="text-red-600 text-sm text-center">{error}</p>
        <p className="text-gray-500 text-xs">ログイン画面に戻ります...</p>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
    </div>
  );
}
