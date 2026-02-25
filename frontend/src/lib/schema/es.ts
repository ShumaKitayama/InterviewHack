// DesignDocument/5_FrontendDesign.md 5.9 / 8_DBDesign.dbml - ESフォームスキーマ
import { z } from "zod";

export const esProjectSchema = z.object({
  company_id: z.string().min(1, "企業を選択してください"),
  deadline: z.string(),
});

export type ESProjectInput = z.infer<typeof esProjectSchema>;

export const esSectionSchema = z.object({
  title: z
    .string()
    .min(1, "タイトルを入力してください")
    .max(200, "タイトルは200文字以内です"),
  content_draft: z.string().max(10000, "内容は10000文字以内です"),
  character_limit: z.number().int().positive().optional(),
});

export type ESSectionInput = z.infer<typeof esSectionSchema>;
