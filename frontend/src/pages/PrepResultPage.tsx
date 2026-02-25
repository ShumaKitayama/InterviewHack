// DesignDocument/3_UIUXDesign.md 3.5.3 - 生成結果表示
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useCreateQA, useQAList } from "../hooks/useQA";

interface GeneratedQA {
  question: string;
  answer: string;
}

export function PrepResultPage() {
  const { generationId } = useParams<{ generationId: string }>();
  const navigate = useNavigate();
  const createQA = useCreateQA();

  // In MVP, generation results would come from the generation_jobs table or API
  // For now, we show a placeholder that integrates with the QA list
  const { data: qaList, isLoading } = useQAList();

  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);

  // Placeholder generated Q&As (would come from API response in production)
  const [generatedQAs] = useState<GeneratedQA[]>([]);

  const handleSaveAll = async () => {
    setSaving(true);
    try {
      for (const qa of generatedQAs) {
        await createQA.mutateAsync({
          question_text: qa.question,
          answer_text: qa.answer,
          source_type: "pre_generated",
        });
      }
      navigate("/qa", { replace: true });
    } catch {
      // handled by TanStack Query
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) return <LoadingSpinner message="結果を読み込み中..." />;

  return (
    <div className="animate-fade-in pb-24">
      <PageHeader
        title="生成結果"
        showBack
        rightAction={
          generatedQAs.length > 0
            ? {
                label: "すべて保存",
                onClick: handleSaveAll,
                disabled: saving,
              }
            : undefined
        }
      />

      <div className="p-4 space-y-4">
        {generatedQAs.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 rounded-full bg-orange-50 flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-8 h-8 text-orange-500"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
              >
                <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
              </svg>
            </div>
            <h3 className="text-base font-semibold text-gray-900 mb-2">
              生成中です
            </h3>
            <p className="text-sm text-gray-500 mb-6">生成ID: {generationId}</p>
            <p className="text-xs text-gray-400">
              BFFからの生成結果を待機中...バックエンドのAI生成APIが接続されると、ここに想定Q&Aが表示されます。
            </p>
          </div>
        ) : (
          <>
            <p className="text-sm text-gray-600">
              {generatedQAs.length}問の想定Q&Aが生成されました
            </p>
            {generatedQAs.map((qa, index) => (
              <div
                key={index}
                className="bg-white rounded-xl shadow-sm overflow-hidden"
              >
                <button
                  onClick={() =>
                    setExpandedIndex(expandedIndex === index ? null : index)
                  }
                  className="w-full p-4 text-left flex items-start gap-3"
                >
                  <span className="text-xs font-bold text-orange-500 mt-0.5">
                    Q{index + 1}
                  </span>
                  <p className="flex-1 text-sm font-medium text-gray-900">
                    {qa.question}
                  </p>
                  <svg
                    className={`w-4 h-4 text-gray-400 transition-transform ${expandedIndex === index ? "rotate-180" : ""}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 9l-7 7-7-7"
                    />
                  </svg>
                </button>
                {expandedIndex === index && (
                  <div className="px-4 pb-4 pt-0 border-t border-gray-50">
                    <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap mt-3">
                      {qa.answer}
                    </p>
                  </div>
                )}
              </div>
            ))}
          </>
        )}

        {/* 既存Q&A表示 */}
        {qaList && qaList.length > 0 && (
          <div className="mt-8">
            <h3 className="text-base font-semibold text-gray-900 mb-3">
              保存済みQ&A
            </h3>
            <div className="space-y-2">
              {qaList.slice(0, 5).map((qa) => (
                <button
                  key={qa.id}
                  onClick={() => navigate(`/qa/${qa.id}`)}
                  className="w-full bg-white rounded-xl p-3 text-left text-sm text-gray-700 active:bg-gray-50 truncate"
                >
                  {qa.question_text}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
