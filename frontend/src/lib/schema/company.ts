// DesignDocument/5_FrontendDesign.md 5.9 / 8_DBDesign.dbml - 企業フォームスキーマ
import { z } from "zod";

export const companySchema = z.object({
  name: z
    .string()
    .min(1, "企業名を入力してください")
    .max(200, "企業名は200文字以内です"),
  job_title: z.string().max(200, "業界は200文字以内です"),
  selection_phase: z.string(),
  interview_date: z.string(),
  motivation_memo: z.string().max(5000, "志望動機は5000文字以内です"),
  reference_url: z.string().max(5000, "メモは5000文字以内です"),
});

export type CompanyInput = z.infer<typeof companySchema>;
