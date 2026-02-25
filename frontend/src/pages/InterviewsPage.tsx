// DesignDocument/5_FrontendDesign.md 5.4.1, 3_UIUXDesign.md 3.5.7 - 面接一覧
import { useNavigate } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useInterviews } from "../hooks/useInterviews";

const resultBadge: Record<string, { label: string; className: string }> = {
  pending: { label: "結果待ち", className: "bg-yellow-100 text-yellow-700" },
  pass: { label: "合格", className: "bg-green-100 text-green-700" },
  fail: { label: "不合格", className: "bg-red-100 text-red-700" },
};

export function InterviewsPage() {
  const navigate = useNavigate();
  const { data: interviews, isLoading, error, refetch } = useInterviews();

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return `${d.getFullYear()}/${String(d.getMonth() + 1).padStart(2, "0")}/${String(d.getDate()).padStart(2, "0")}`;
  };

  return (
    <div className="animate-fade-in">
      <div className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
        <div className="flex items-center justify-between px-4 h-14">
          <h1 className="text-lg font-bold text-gray-900">面接記録</h1>
          <button
            onClick={() => navigate("/interviews/new")}
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

      {!isLoading && !error && interviews?.length === 0 && (
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          }
          title="面接記録がありません"
          description="面接を記録して、AIに回答を添削してもらいましょう"
          actionLabel="面接を記録"
          actionTo="/interviews/new"
        />
      )}

      {interviews && interviews.length > 0 && (
        <div className="p-4 space-y-3">
          {interviews.map((interview) => (
            <button
              key={interview.id}
              onClick={() => navigate(`/interviews/${interview.id}`)}
              className="w-full bg-white rounded-xl p-4 text-left active:bg-gray-50 transition-colors shadow-sm"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900">
                    {interview.company?.name ?? "企業未設定"}
                  </p>
                  <div className="flex items-center gap-2 mt-1.5">
                    <span className="text-xs text-gray-500">
                      {formatDate(interview.interview_date)}
                    </span>
                    {interview.phase && (
                      <span className="text-xs text-gray-400">
                        {interview.phase}
                      </span>
                    )}
                  </div>
                  {interview.result && resultBadge[interview.result] && (
                    <span
                      className={`inline-block mt-2 px-2.5 py-0.5 rounded-full text-xs font-medium ${resultBadge[interview.result].className}`}
                    >
                      {resultBadge[interview.result].label}
                    </span>
                  )}
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
