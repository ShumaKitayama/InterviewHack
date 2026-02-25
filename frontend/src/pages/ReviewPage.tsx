// DesignDocument/3_UIUXDesign.md 3.4.4, 3.5.9 - Q&A復習UI（カード）
import { useState } from "react";
import { EmptyState } from "../components/ui/EmptyState";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useQAList, useUpdateReviewStatus } from "../hooks/useQA";

export function ReviewPage() {
  const { data: qaList, isLoading } = useQAList();
  const updateStatus = useUpdateReviewStatus();
  const [currentIndex, setCurrentIndex] = useState(0);
  const [showAnswer, setShowAnswer] = useState(false);

  if (isLoading) return <LoadingSpinner />;

  if (!qaList || qaList.length === 0) {
    return (
      <div>
        <PageHeader title="復習" showBack />
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          }
          title="復習するQ&Aがありません"
          description="面接対策でQ&Aを生成してから復習しましょう"
          actionLabel="面接対策へ"
          actionTo="/prep"
        />
      </div>
    );
  }

  const currentQA = qaList[currentIndex];
  const isLast = currentIndex === qaList.length - 1;
  const isFirst = currentIndex === 0;

  const goNext = () => {
    if (!isLast) {
      setCurrentIndex((prev) => prev + 1);
      setShowAnswer(false);
    }
  };

  const goPrev = () => {
    if (!isFirst) {
      setCurrentIndex((prev) => prev - 1);
      setShowAnswer(false);
    }
  };

  const markReviewed = () => {
    updateStatus.mutate({ id: currentQA.id, status: "reviewed" });
    goNext();
  };

  const markNeedsReview = () => {
    updateStatus.mutate({ id: currentQA.id, status: "needs_review" });
    goNext();
  };

  return (
    <div className="animate-fade-in min-h-screen bg-gray-50">
      <PageHeader title="復習" showBack />

      <div className="p-4">
        {/* 進捗 */}
        <div className="flex items-center justify-between mb-4">
          <span className="text-xs text-gray-500">
            {currentIndex + 1} / {qaList.length}
          </span>
          {currentQA.company?.name && (
            <span className="text-xs text-gray-400">
              {currentQA.company.name}
            </span>
          )}
        </div>

        {/* プログレスバー */}
        <div className="w-full h-1 bg-gray-200 rounded-full mb-6 overflow-hidden">
          <div
            className="h-full bg-orange-500 rounded-full transition-all duration-300"
            style={{
              width: `${((currentIndex + 1) / qaList.length) * 100}%`,
            }}
          />
        </div>

        {/* カード */}
        <div
          className="bg-white rounded-2xl shadow-lg min-h-[320px] flex flex-col cursor-pointer"
          onClick={() => setShowAnswer(!showAnswer)}
        >
          {/* 質問 */}
          <div className="flex-1 flex items-center justify-center p-6">
            <div className="text-center">
              <p className="text-xs font-medium text-orange-500 mb-3">質問</p>
              <p className="text-lg font-semibold text-gray-900 leading-relaxed">
                {currentQA.question_text}
              </p>
            </div>
          </div>

          {/* 回答（タップで表示） */}
          {showAnswer ? (
            <div className="border-t border-gray-100 p-6 bg-gray-50 rounded-b-2xl">
              <p className="text-xs font-medium text-green-600 mb-2">回答</p>
              <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">
                {currentQA.answer_text}
              </p>
            </div>
          ) : (
            <div className="border-t border-gray-100 p-4 text-center">
              <p className="text-xs text-gray-400">タップして回答を表示</p>
            </div>
          )}
        </div>

        {/* アクション */}
        {showAnswer && (
          <div className="flex gap-3 mt-6 animate-fade-in">
            <button
              onClick={markNeedsReview}
              className="flex-1 py-3 bg-yellow-50 text-yellow-700 rounded-xl text-sm font-medium hover:bg-yellow-100 transition-colors"
            >
              要再確認
            </button>
            <button
              onClick={markReviewed}
              className="flex-1 py-3 bg-green-50 text-green-700 rounded-xl text-sm font-medium hover:bg-green-100 transition-colors"
            >
              復習済み
            </button>
          </div>
        )}

        {/* ナビゲーション */}
        <div className="flex justify-between mt-6">
          <button
            onClick={goPrev}
            disabled={isFirst}
            className="px-4 py-2 text-sm text-gray-500 disabled:opacity-30"
          >
            ← 前へ
          </button>
          <button
            onClick={goNext}
            disabled={isLast}
            className="px-4 py-2 text-sm text-gray-500 disabled:opacity-30"
          >
            次へ →
          </button>
        </div>
      </div>
    </div>
  );
}
