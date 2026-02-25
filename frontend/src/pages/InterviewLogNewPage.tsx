// DesignDocument/3_UIUXDesign.md 3.5.7 - 面接ログ入力
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { PageHeader } from "../components/ui/PageHeader";
import { useCreateInterviewLog } from "../hooks/useInterviews";
import {
  interviewLogSchema,
  type InterviewLogInput,
} from "../lib/schema/interview";

interface LogEntry {
  question_text: string;
  answer_text: string;
  self_evaluation: string;
}

export function InterviewLogNewPage() {
  const { id: interviewId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const createLog = useCreateInterviewLog();

  const [savedLogs, setSavedLogs] = useState<LogEntry[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<InterviewLogInput>({
    resolver: zodResolver(interviewLogSchema),
    defaultValues: {
      question_text: "",
      answer_text: "",
      self_evaluation: "",
    },
  });

  const addLog = (data: InterviewLogInput) => {
    setSavedLogs((prev) => [...prev, data]);
    reset();
  };

  const submitAll = async () => {
    if (!interviewId) return;
    setSubmitting(true);
    try {
      for (const log of savedLogs) {
        await createLog.mutateAsync({
          interviewId,
          input: log,
        });
      }
      navigate(`/interviews/${interviewId}`, { replace: true });
    } catch {
      // handled by TanStack Query
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader title="ログ入力" showBack />

      <div className="p-4 space-y-4">
        {/* 追加済みログ */}
        {savedLogs.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-gray-500">
              追加済み ({savedLogs.length}件)
            </p>
            {savedLogs.map((log, i) => (
              <div
                key={i}
                className="bg-green-50 rounded-xl p-3 flex items-start gap-2"
              >
                <span className="text-xs font-bold text-green-600 mt-0.5">
                  Q{i + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-900 truncate">
                    {log.question_text}
                  </p>
                </div>
                <button
                  onClick={() =>
                    setSavedLogs((prev) => prev.filter((_, idx) => idx !== i))
                  }
                  className="text-gray-400 hover:text-red-500 flex-shrink-0"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>
            ))}
          </div>
        )}

        {/* 入力フォーム */}
        <form onSubmit={handleSubmit(addLog)} className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              質問 <span className="text-red-500">*</span>
            </label>
            <textarea
              {...register("question_text")}
              placeholder="面接で聞かれた質問"
              rows={2}
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
            />
            {errors.question_text && (
              <p className="text-xs text-red-500 mt-1">
                {errors.question_text.message}
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              回答 <span className="text-red-500">*</span>
            </label>
            <textarea
              {...register("answer_text")}
              placeholder="自分がした回答"
              rows={4}
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
            />
            {errors.answer_text && (
              <p className="text-xs text-red-500 mt-1">
                {errors.answer_text.message}
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              自己評価（任意）
            </label>
            <textarea
              {...register("self_evaluation")}
              placeholder="うまく言えた点、反省点など"
              rows={2}
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
            />
          </div>

          <button
            type="submit"
            className="w-full py-3 border-2 border-dashed border-gray-200 rounded-xl text-sm font-medium text-gray-500 hover:border-orange-300 hover:text-orange-500 transition-colors"
          >
            + この質問を追加してもう1問入力
          </button>
        </form>
      </div>

      <BottomAction
        label={`${savedLogs.length > 0 ? savedLogs.length + "件を" : ""}保存する`}
        onClick={submitAll}
        loading={submitting}
        disabled={savedLogs.length === 0 || submitting}
      />
    </div>
  );
}
