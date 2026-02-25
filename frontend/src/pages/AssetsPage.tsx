// DesignDocument/5_FrontendDesign.md 5.4.1, 3_UIUXDesign.md 3.3.2 - 自己資産一覧
import { useNavigate } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useAssets } from "../hooks/useAssets";
import { ASSET_TYPE_LABEL } from "../lib/constants";
import type { AssetType } from "../lib/supabase/database.types";

const typeColorMap: Record<AssetType, string> = {
  episode: "bg-orange-100 text-orange-700",
  work: "bg-blue-100 text-blue-700",
  strength: "bg-green-100 text-green-700",
  weakness: "bg-red-100 text-red-700",
  skill: "bg-purple-100 text-purple-700",
  academic: "bg-cyan-100 text-cyan-700",
  project: "bg-indigo-100 text-indigo-700",
  other: "bg-gray-100 text-gray-700",
};

export function AssetsPage() {
  const navigate = useNavigate();
  const { data: assets, isLoading, error, refetch } = useAssets();

  return (
    <div className="animate-fade-in">
      <div className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
        <div className="flex items-center justify-between px-4 h-14">
          <h1 className="text-lg font-bold text-gray-900">自己資産</h1>
          <button
            onClick={() => navigate("/assets/new")}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-orange-500 text-white hover:bg-orange-600 transition-colors"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={2.5}
            >
              <path d="M12 4v16m8-8H4" />
            </svg>
          </button>
        </div>
      </div>

      {isLoading && <LoadingSpinner />}
      {error && <ErrorDisplay onRetry={refetch} />}

      {!isLoading && !error && assets?.length === 0 && (
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          }
          title="自己資産がまだありません"
          description="学歴・職歴・強み・エピソードなどを登録して、AI面接対策の精度を高めましょう"
          actionLabel="最初の資産を追加"
          actionTo="/assets/new"
        />
      )}

      {assets && assets.length > 0 && (
        <div className="p-4 space-y-3">
          {assets.map((asset) => (
            <button
              key={asset.id}
              onClick={() => navigate(`/assets/${asset.id}`)}
              className="w-full bg-white rounded-xl p-4 text-left active:bg-gray-50 transition-colors shadow-sm"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <span
                    className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium mb-2 ${typeColorMap[asset.type as AssetType] ?? typeColorMap.other}`}
                  >
                    {ASSET_TYPE_LABEL[asset.type as AssetType] ?? asset.type}
                  </span>
                  <p className="text-sm font-semibold text-gray-900 truncate">
                    {asset.title}
                  </p>
                  <p className="text-xs text-gray-500 mt-1 line-clamp-2">
                    {asset.content}
                  </p>
                </div>
                <svg
                  className="w-4 h-4 text-gray-300 ml-2 mt-1 flex-shrink-0"
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
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
