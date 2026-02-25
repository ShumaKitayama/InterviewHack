// DesignDocument/3_UIUXDesign.md 3.5.1, 5_FrontendDesign.md 5.4.1 - ホーム（ダッシュボード）
import { useNavigate } from "react-router-dom";
import { useDashboardStats } from "../hooks/useStats";

function StatItem({ value, label }: { value: number; label: string }) {
  return (
    <div className="flex-1 text-center">
      <p className="text-2xl font-bold text-white">{value}</p>
      <p className="text-xs text-gray-300 mt-0.5">{label}</p>
    </div>
  );
}

const quickActions = [
  {
    to: "/prep",
    label: "面接対策",
    sub: "AI質問生成",
    bgClass: "bg-blue-50",
    icon: (
      <div className="w-10 h-10 rounded-xl bg-orange-100 flex items-center justify-center">
        <svg
          className="w-5 h-5 text-orange-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          strokeWidth={2}
        >
          <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
        </svg>
      </div>
    ),
  },
  {
    to: "/interviews",
    label: "振り返り",
    sub: "AI添削",
    bgClass: "bg-green-50",
    icon: (
      <div className="w-10 h-10 rounded-xl bg-green-100 flex items-center justify-center">
        <svg
          className="w-5 h-5 text-green-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          strokeWidth={2}
        >
          <path d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
        </svg>
      </div>
    ),
  },
  {
    to: "/companies/new",
    label: "企業追加",
    sub: "リサーチ",
    bgClass: "bg-sky-50",
    icon: (
      <div className="w-10 h-10 rounded-xl bg-sky-100 flex items-center justify-center">
        <svg
          className="w-5 h-5 text-sky-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          strokeWidth={2}
        >
          <path d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
        </svg>
      </div>
    ),
  },
  {
    to: "/assets/new",
    label: "資産追加",
    sub: "自己分析",
    bgClass: "bg-amber-50",
    icon: (
      <div className="w-10 h-10 rounded-xl bg-amber-100 flex items-center justify-center">
        <svg
          className="w-5 h-5 text-orange-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          strokeWidth={2}
        >
          <path d="M12 4v16m8-8H4" />
        </svg>
      </div>
    ),
  },
];

export function HomePage() {
  const navigate = useNavigate();
  const { data: stats } = useDashboardStats();

  return (
    <div className="animate-fade-in">
      {/* ヘッダー */}
      <div className="px-4 pt-4 pb-2">
        <h1 className="text-xl font-bold text-center text-gray-900">
          Interview Hack
        </h1>
      </div>

      {/* ダッシュボードカード */}
      <div className="px-4 pt-2 pb-4">
        <div className="bg-gradient-to-br from-slate-700 to-slate-900 rounded-2xl p-5 text-white shadow-lg">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-7 h-7 rounded-lg bg-white/15 flex items-center justify-center">
              <svg
                className="w-4 h-4 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                strokeWidth={2}
              >
                <path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            </div>
            <h2 className="text-lg font-bold">就活ダッシュボード</h2>
          </div>
          <p className="text-sm text-gray-300 mb-5">
            準備 → 面接 → 振り返りのループで内定に近づこう
          </p>
          <div className="flex border-t border-white/10 pt-4">
            <StatItem value={stats?.activeCompanies ?? 0} label="選考中" />
            <div className="w-px bg-white/10" />
            <StatItem value={stats?.offers ?? 0} label="内定" />
            <div className="w-px bg-white/10" />
            <StatItem value={stats?.qaCount ?? 0} label="Q&A" />
            <div className="w-px bg-white/10" />
            <StatItem value={stats?.assetCount ?? 0} label="資産" />
          </div>
        </div>
      </div>

      {/* クイックアクション */}
      <div className="px-4">
        <h3 className="text-base font-semibold text-gray-900 mb-3">
          クイックアクション
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {quickActions.map((action) => (
            <button
              key={action.to}
              onClick={() => navigate(action.to)}
              className={`${action.bgClass} rounded-2xl p-4 text-left active:scale-[0.97] transition-transform`}
            >
              {action.icon}
              <p className="text-sm font-semibold text-gray-900 mt-3">
                {action.label}
              </p>
              <p className="text-xs text-gray-500 mt-0.5">{action.sub}</p>
            </button>
          ))}
        </div>
      </div>

      {/* 追加メニュー */}
      <div className="px-4 mt-6 mb-4">
        <h3 className="text-base font-semibold text-gray-900 mb-3">
          その他の機能
        </h3>
        <div className="space-y-2">
          {[
            {
              to: "/es",
              label: "ES作成",
              desc: "エントリーシートの作成・管理",
              icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z",
            },
            {
              to: "/review",
              label: "復習",
              desc: "Q&Aカードで面接対策を復習",
              icon: "M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15",
            },
            {
              to: "/billing",
              label: "クレジット",
              desc: "AIクレジットの購入・管理",
              icon: "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z",
            },
            {
              to: "/settings",
              label: "設定",
              desc: "アカウント・プラン管理",
              icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
            },
          ].map((item) => (
            <button
              key={item.to}
              onClick={() => navigate(item.to)}
              className="w-full flex items-center gap-3 bg-white rounded-xl px-4 py-3.5 active:bg-gray-50 transition-colors text-left"
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
      </div>
    </div>
  );
}
