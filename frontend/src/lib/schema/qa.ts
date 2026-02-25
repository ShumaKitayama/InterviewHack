// DesignDocument/5_FrontendDesign.md 5.9 / 8_DBDesign.dbml - Q&Aフォームスキーマ
import { z } from "zod";

export const qaSchema = z.object({
  company_id: z.string(),
  question_text: z
    .string()
    .min(1, "質問を入力してください")
    .max(1000, "質問は1000文字以内です"),
  answer_text: z
    .string()
    .min(1, "回答を入力してください")
    .max(5000, "回答は5000文字以内です"),
});

export type QAInput = z.infer<typeof qaSchema>;
