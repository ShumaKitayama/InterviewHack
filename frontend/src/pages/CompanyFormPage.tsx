// DesignDocument/5_FrontendDesign.md 5.9, 3_UIUXDesign.md 3.3.2 - 企業作成/編集
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { ChipSelector } from "../components/ui/ChipSelector";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import {
  useCompany,
  useCreateCompany,
  useUpdateCompany,
} from "../hooks/useCompanies";
import { SELECTION_PHASE_OPTIONS } from "../lib/constants";
import { companySchema, type CompanyInput } from "../lib/schema/company";

export function CompanyFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id;
  const navigate = useNavigate();

  const { data: existingCompany, isLoading } = useCompany(id);
  const createCompany = useCreateCompany();
  const updateCompany = useUpdateCompany();

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<CompanyInput>({
    resolver: zodResolver(companySchema),
    defaultValues: {
      name: "",
      job_title: "",
      selection_phase: "researching",
      interview_date: "",
      motivation_memo: "",
      reference_url: "",
    },
  });

  const selectedPhase = useWatch({ control, name: "selection_phase" });

  useEffect(() => {
    if (existingCompany) {
      setValue("name", existingCompany.name);
      setValue("job_title", existingCompany.job_title ?? "");
      setValue(
        "selection_phase",
        existingCompany.selection_phase ?? "researching",
      );
      setValue("motivation_memo", existingCompany.motivation_memo ?? "");
      setValue("reference_url", existingCompany.reference_url ?? "");
    }
  }, [existingCompany, setValue]);

  const onSubmit = async (data: CompanyInput) => {
    try {
      if (isEdit && id) {
        await updateCompany.mutateAsync({ id, input: data });
      } else {
        await createCompany.mutateAsync(data);
      }
      navigate("/companies", { replace: true });
    } catch {
      // mutation error handled by TanStack Query
    }
  };

  if (isEdit && isLoading) {
    return <LoadingSpinner />;
  }

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader
        title={isEdit ? "企業を編集" : "企業を追加"}
        showBack
        rightAction={{
          label: "保存",
          onClick: handleSubmit(onSubmit),
          disabled: isSubmitting,
        }}
      />

      <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-6">
        {/* 企業名 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            企業名 <span className="text-red-500">*</span>
          </label>
          <input
            {...register("name")}
            placeholder="例: 株式会社〇〇"
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
          {errors.name && (
            <p className="text-xs text-red-500 mt-1">{errors.name.message}</p>
          )}
        </div>

        {/* 業界 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            業界
          </label>
          <input
            {...register("job_title")}
            placeholder="例: IT / コンサルティング / 金融"
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>

        {/* 選考フェーズ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            選考フェーズ
          </label>
          <ChipSelector
            options={SELECTION_PHASE_OPTIONS}
            value={selectedPhase ?? "researching"}
            onChange={(v) => setValue("selection_phase", v)}
          />
        </div>

        {/* 面接予定日 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            面接予定日
          </label>
          <div className="relative">
            <input
              {...register("interview_date")}
              type="date"
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
            />
          </div>
        </div>

        {/* 志望動機 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            志望動機
          </label>
          <textarea
            {...register("motivation_memo")}
            placeholder="この企業を志望する理由を記載"
            rows={4}
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>

        {/* メモ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            メモ
          </label>
          <textarea
            {...register("reference_url")}
            placeholder="企業の特徴、面接のポイントなど"
            rows={3}
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>
      </form>

      <BottomAction
        label="保存する"
        onClick={handleSubmit(onSubmit)}
        loading={isSubmitting}
        disabled={isSubmitting}
      />
    </div>
  );
}
