// DesignDocument/5_FrontendDesign.md 5.4.1, 3_UIUXDesign.md 3.5.9 - Q&Aバンク一覧
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useQAList, useToggleFavorite } from "../hooks/useQA";
import { QA_REVIEW_STATUS_LABEL } from "../lib/constants";

const statusColor: Record<string, string> = {
  new: "bg-blue-100 text-blue-700",
  reviewed: "bg-green-100 text-green-700",
  needs_review: "bg-yellow-100 text-yellow-700",
};

type FilterTab = "all" | "favorites" | "needs_review";

export function QAPage() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<FilterTab>("all");

  const filters =
    activeTab === "favorites"
      ? { favoriteOnly: true }
      : activeTab === "needs_review"
        ? { reviewStatus: "needs_review" }
        : undefined;

  const { data: qaList, isLoading, error, refetch } = useQAList(filters);
  const toggleFavorite = useToggleFavorite();

  const tabs: { key: FilterTab; label: string }[] = [
    { key: "all", label: "すべて" },
    { key: "favorites", label: "お気に入り" },
    { key: "needs_review", label: "要復習" },
  ];

  return (
    <div className="animate-fade-in">
      <div className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
        <div className="flex items-center justify-between px-4 h-14">
          <h1 className="text-lg font-bold text-gray-900">Q&Aバンク</h1>
          <button
            onClick={() => navigate("/review")}
            className="text-sm font-medium text-orange-500 hover:underline"
          >
            復習する
          </button>
        </div>
        <div className="flex px-4 gap-1 pb-2">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`px-4 py-1.5 rounded-full text-xs font-medium transition-all ${
                activeTab === tab.key
                  ? "bg-gray-900 text-white"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading && <LoadingSpinner />}
      {error && <ErrorDisplay onRetry={refetch} />}

      {!isLoading && !error && qaList?.length === 0 && (
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          title="Q&Aがまだありません"
          description="面接対策で生成したQ&Aや振り返りの改善版がここに蓄積されます"
          actionLabel="面接対策を始める"
          actionTo="/prep"
        />
      )}

      {qaList && qaList.length > 0 && (
        <div className="p-4 space-y-3">
          {qaList.map((qa) => (
            <div
              key={qa.id}
              className="bg-white rounded-xl shadow-sm overflow-hidden"
            >
              <button
                onClick={() => navigate(`/qa/${qa.id}`)}
                className="w-full p-4 text-left active:bg-gray-50 transition-colors"
              >
                <div className="flex items-start gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-gray-900 line-clamp-2">
                      {qa.question_text}
                    </p>
                    <p className="text-xs text-gray-500 mt-1 line-clamp-1">
                      {qa.answer_text}
                    </p>
                    <div className="flex items-center gap-2 mt-2">
                      {qa.company?.name && (
                        <span className="text-xs text-gray-400">
                          {qa.company.name}
                        </span>
                      )}
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${statusColor[qa.review_status] ?? statusColor.new}`}
                      >
                        {QA_REVIEW_STATUS_LABEL[qa.review_status] ??
                          qa.review_status}
                      </span>
                    </div>
                  </div>
                </div>
              </button>
              <div className="border-t border-gray-50 px-4 py-2 flex justify-end">
                <button
                  onClick={() =>
                    toggleFavorite.mutate({
                      id: qa.id,
                      isFavorite: !qa.is_favorite,
                    })
                  }
                  className="p-1"
                >
                  <svg
                    className={`w-5 h-5 ${qa.is_favorite ? "text-orange-500 fill-orange-500" : "text-gray-300"}`}
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                    fill={qa.is_favorite ? "currentColor" : "none"}
                  >
                    <path d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
