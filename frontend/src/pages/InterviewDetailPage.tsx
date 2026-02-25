// DesignDocument/3_UIUXDesign.md 3.5.7, 3.5.8 - 面接詳細・ログ一覧
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useInterview, useInterviewLogs } from "../hooks/useInterviews";
import { api } from "../lib/api/client";

export function InterviewDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: interview, isLoading } = useInterview(id);
  const {
    data: logs,
    isLoading: loadingLogs,
    refetch: refetchLogs,
  } = useInterviewLogs(id);
  const [reviewing, setReviewing] = useState<string | null>(null);

  const handleReview = async (logId: string) => {
    const log = logs?.find((l) => l.id === logId);
    if (!log) return;
    setReviewing(logId);
    try {
      await api.post("/api/ai/answer-review", {
        interview_log_id: logId,
        question: log.question_text,
        answer: log.answer_text,
      });
      await refetchLogs();
    } catch {
      // API error handled by client
    } finally {
      setReviewing(null);
    }
  };

  if (isLoading) return <LoadingSpinner />;
  if (!interview) return <ErrorDisplay message="面接が見つかりません" />;

  return (
    <div className="animate-fade-in pb-24">
      <PageHeader title={interview.company?.name ?? "面接詳細"} showBack />

      <div className="p-4 space-y-4">
        {/* 面接情報 */}
        <div className="bg-white rounded-xl p-4 shadow-sm">
          <div className="flex items-center gap-4 text-sm text-gray-600">
            <span>{interview.interview_date}</span>
            {interview.phase && <span>{interview.phase}</span>}
          </div>
          {interview.memo && (
            <p className="text-sm text-gray-500 mt-2">{interview.memo}</p>
          )}
        </div>

        {/* ログ一覧 */}
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-gray-900">
            質問・回答 ({logs?.length ?? 0})
          </h2>
          <button
            onClick={() => navigate(`/interviews/${id}/logs/new`)}
            className="text-sm font-medium text-orange-500 hover:underline"
          >
            + 追加
          </button>
        </div>

        {loadingLogs && <LoadingSpinner />}

        {logs?.length === 0 && !loadingLogs && (
          <div className="text-center py-8">
            <p className="text-sm text-gray-500 mb-3">
              質問と回答を記録しましょう
            </p>
            <button
              onClick={() => navigate(`/interviews/${id}/logs/new`)}
              className="px-5 py-2.5 bg-orange-500 text-white text-sm font-medium rounded-full hover:bg-orange-600 transition-colors"
            >
              ログを入力
            </button>
          </div>
        )}

        {logs?.map((log, index) => (
          <div
            key={log.id}
            className="bg-white rounded-xl p-4 shadow-sm space-y-3"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-gray-400">
                Q{index + 1}
              </span>
              {!log.ai_feedback_text && (
                <button
                  onClick={() => handleReview(log.id)}
                  disabled={reviewing === log.id}
                  className="text-xs font-medium text-orange-500 hover:underline disabled:opacity-50"
                >
                  {reviewing === log.id ? "添削中..." : "AI添削"}
                </button>
              )}
            </div>

            <div>
              <p className="text-xs font-medium text-gray-400 mb-1">質問</p>
              <p className="text-sm text-gray-900">{log.question_text}</p>
            </div>

            <div>
              <p className="text-xs font-medium text-gray-400 mb-1">回答</p>
              <p className="text-sm text-gray-700">{log.answer_text}</p>
            </div>

            {log.ai_feedback_text && (
              <div className="border-t border-gray-100 pt-3">
                <p className="text-xs font-medium text-orange-500 mb-1">
                  AIフィードバック
                </p>
                <p className="text-sm text-gray-700">{log.ai_feedback_text}</p>
              </div>
            )}

            {log.ai_improved_answer_text && (
              <div className="bg-green-50 rounded-lg p-3">
                <p className="text-xs font-medium text-green-700 mb-1">
                  改善版回答
                </p>
                <p className="text-sm text-gray-700">
                  {log.ai_improved_answer_text}
                </p>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
