// DesignDocument/3_UIUXDesign.md 3.3.3, 5_FrontendDesign.md 5.4.1 - 設定
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { PageHeader } from "../components/ui/PageHeader";
import { useAuth } from "../features/auth/AuthContext";
import { signOut } from "../lib/supabase/auth";

export function SettingsPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [loggingOut, setLoggingOut] = useState(false);

  const handleLogout = async () => {
    setLoggingOut(true);
    await signOut();
    navigate("/login", { replace: true });
  };

  const menuItems = [
    {
      label: "クレジット管理",
      desc: "残高確認・購入",
      to: "/billing",
      icon: "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z",
    },
    {
      label: "面接対策",
      desc: "AI Q&A生成",
      to: "/prep",
      icon: "M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z",
    },
    {
      label: "ES作成",
      desc: "エントリーシート管理",
      to: "/es",
      icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z",
    },
  ];

  return (
    <div className="animate-fade-in pb-8">
      <PageHeader title="設定" showBack />

      <div className="p-4 space-y-6">
        {/* アカウント情報 */}
        <div className="bg-white rounded-xl p-4 shadow-sm">
          <p className="text-xs font-medium text-gray-400 mb-1">アカウント</p>
          <p className="text-sm font-semibold text-gray-900">
            {user?.email ?? "未設定"}
          </p>
          <p className="text-xs text-gray-400 mt-1">
            ID: {user?.id?.slice(0, 8)}...
          </p>
        </div>

        {/* メニュー */}
        <div className="space-y-2">
          {menuItems.map((item) => (
            <button
              key={item.to}
              onClick={() => navigate(item.to)}
              className="w-full flex items-center gap-3 bg-white rounded-xl px-4 py-3.5 active:bg-gray-50 transition-colors text-left shadow-sm"
            >
              <div className="w-10 h-10 rounded-xl bg-gray-100 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-5 h-5 text-gray-500"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  strokeWidth={1.8}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d={item.icon}
                  />
                </svg>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-gray-900">
                  {item.label}
                </p>
                <p className="text-xs text-gray-500">{item.desc}</p>
              </div>
              <svg
                className="w-4 h-4 text-gray-300 flex-shrink-0"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </button>
          ))}
        </div>

        {/* ログアウト */}
        <button
          onClick={handleLogout}
          disabled={loggingOut}
          className="w-full py-3 bg-red-50 text-red-600 rounded-xl text-sm font-medium hover:bg-red-100 transition-colors disabled:opacity-50"
        >
          {loggingOut ? "ログアウト中..." : "ログアウト"}
        </button>

        {/* バージョン情報 */}
        <p className="text-center text-xs text-gray-300">
          InterviewHack v0.1.0
        </p>
      </div>
    </div>
  );
}
