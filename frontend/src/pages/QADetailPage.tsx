// DesignDocument/3_UIUXDesign.md 3.5.9 - Q&A詳細
import { useParams } from "react-router-dom";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import {
  useQADetail,
  useToggleFavorite,
  useUpdateReviewStatus,
} from "../hooks/useQA";
import { QA_REVIEW_STATUS_LABEL } from "../lib/constants";

export function QADetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: qa, isLoading } = useQADetail(id);
  const toggleFavorite = useToggleFavorite();
  const updateReviewStatus = useUpdateReviewStatus();

  if (isLoading) return <LoadingSpinner />;
  if (!qa) return <ErrorDisplay message="Q&Aが見つかりません" />;

  const statusOptions = [
    { value: "new", label: "未復習" },
    { value: "reviewed", label: "復習済み" },
    { value: "needs_review", label: "要再確認" },
  ];

  return (
    <div className="animate-fade-in pb-8">
      <PageHeader
        title="Q&A詳細"
        showBack
        rightAction={{
          label: qa.is_favorite ? "★" : "☆",
          onClick: () =>
            toggleFavorite.mutate({
              id: qa.id,
              isFavorite: !qa.is_favorite,
            }),
        }}
      />

      <div className="p-4 space-y-4">
        {/* 企業名 */}
        {qa.company?.name && (
          <div className="flex items-center gap-2">
            <span className="px-3 py-1 bg-gray-100 rounded-full text-xs font-medium text-gray-600">
              {qa.company.name}
            </span>
            <span className="px-3 py-1 bg-blue-50 rounded-full text-xs font-medium text-blue-600">
              {QA_REVIEW_STATUS_LABEL[qa.review_status] ?? qa.review_status}
            </span>
          </div>
        )}

        {/* 質問 */}
        <div className="bg-white rounded-xl p-5 shadow-sm">
          <p className="text-xs font-semibold text-orange-500 mb-2">質問</p>
          <p className="text-base text-gray-900 leading-relaxed">
            {qa.question_text}
          </p>
        </div>

        {/* 回答 */}
        <div className="bg-white rounded-xl p-5 shadow-sm">
          <p className="text-xs font-semibold text-green-600 mb-2">回答</p>
          <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">
            {qa.answer_text}
          </p>
        </div>

        {/* 復習ステータス変更 */}
        <div>
          <p className="text-sm font-semibold text-gray-900 mb-2">
            復習ステータス
          </p>
          <div className="flex gap-2">
            {statusOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() =>
                  updateReviewStatus.mutate({
                    id: qa.id,
                    status: opt.value,
                  })
                }
                className={`px-4 py-2 rounded-full text-xs font-medium transition-all ${
                  qa.review_status === opt.value
                    ? "bg-gray-900 text-white"
                    : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
