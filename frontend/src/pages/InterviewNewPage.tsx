// DesignDocument/3_UIUXDesign.md 3.4.3, 3.5.7 - 面接作成
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { ChipSelector } from "../components/ui/ChipSelector";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useCompanies } from "../hooks/useCompanies";
import { useCreateInterview } from "../hooks/useInterviews";
import { INTERVIEW_PHASE_OPTIONS } from "../lib/constants";
import { interviewSchema, type InterviewInput } from "../lib/schema/interview";

export function InterviewNewPage() {
  const navigate = useNavigate();
  const { data: companies, isLoading: loadingCompanies } = useCompanies();
  const createInterview = useCreateInterview();

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<InterviewInput>({
    resolver: zodResolver(interviewSchema),
    defaultValues: {
      company_id: "",
      interview_date: new Date().toISOString().split("T")[0],
      phase: "",
      memo: "",
    },
  });

  const selectedPhase = watch("phase");

  const onSubmit = async (data: InterviewInput) => {
    try {
      const result = await createInterview.mutateAsync(data);
      navigate(`/interviews/${result.id}`, { replace: true });
    } catch {
      // handled by TanStack Query
    }
  };

  if (loadingCompanies) return <LoadingSpinner />;

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader title="面接を記録" showBack />

      <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-6">
        {/* 企業選択 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            企業 <span className="text-red-500">*</span>
          </label>
          {companies && companies.length > 0 ? (
            <select
              {...register("company_id")}
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all appearance-none"
            >
              <option value="">企業を選択</option>
              {companies.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          ) : (
            <div className="text-center py-4">
              <p className="text-sm text-gray-500 mb-2">
                まず企業を登録してください
              </p>
              <button
                type="button"
                onClick={() => navigate("/companies/new")}
                className="text-sm text-orange-500 font-medium hover:underline"
              >
                企業を追加
              </button>
            </div>
          )}
          {errors.company_id && (
            <p className="text-xs text-red-500 mt-1">
              {errors.company_id.message}
            </p>
          )}
        </div>

        {/* 面接日 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            面接日 <span className="text-red-500">*</span>
          </label>
          <input
            {...register("interview_date")}
            type="date"
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
          {errors.interview_date && (
            <p className="text-xs text-red-500 mt-1">
              {errors.interview_date.message}
            </p>
          )}
        </div>

        {/* 面接フェーズ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            面接フェーズ
          </label>
          <ChipSelector
            options={INTERVIEW_PHASE_OPTIONS}
            value={selectedPhase ?? ""}
            onChange={(v) => setValue("phase", v)}
          />
        </div>

        {/* メモ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            メモ
          </label>
          <textarea
            {...register("memo")}
            placeholder="面接の感想や気づきなど"
            rows={3}
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>
      </form>

      <BottomAction
        label="面接を記録する"
        onClick={handleSubmit(onSubmit)}
        loading={isSubmitting}
        disabled={isSubmitting || !companies?.length}
      />
    </div>
  );
}
