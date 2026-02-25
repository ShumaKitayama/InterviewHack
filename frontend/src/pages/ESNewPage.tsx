// DesignDocument/3_UIUXDesign.md 3.4.2 - ESプロジェクト作成
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useCompanies } from "../hooks/useCompanies";
import { useCreateESProject } from "../hooks/useES";
import { esProjectSchema, type ESProjectInput } from "../lib/schema/es";

export function ESNewPage() {
  const navigate = useNavigate();
  const { data: companies, isLoading } = useCompanies();
  const createProject = useCreateESProject();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ESProjectInput>({
    resolver: zodResolver(esProjectSchema),
    defaultValues: {
      company_id: "",
      deadline: "",
    },
  });

  const onSubmit = async (data: ESProjectInput) => {
    try {
      const result = await createProject.mutateAsync(data);
      navigate(`/es/${result.id}`, { replace: true });
    } catch {
      // handled by TanStack Query
    }
  };

  if (isLoading) return <LoadingSpinner />;

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader title="ES作成" showBack />

      <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-6">
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
            <div className="text-center py-4 bg-gray-50 rounded-xl">
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

        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            提出期限
          </label>
          <input
            {...register("deadline")}
            type="date"
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>
      </form>

      <BottomAction
        label="ES作成を開始"
        onClick={handleSubmit(onSubmit)}
        loading={isSubmitting}
        disabled={isSubmitting || !companies?.length}
      />
    </div>
  );
}
