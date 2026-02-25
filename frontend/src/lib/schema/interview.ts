// DesignDocument/5_FrontendDesign.md 5.9 / 8_DBDesign.dbml - 面接フォームスキーマ
import { z } from "zod";

export const interviewSchema = z.object({
  company_id: z.string().min(1, "企業を選択してください"),
  interview_date: z.string().min(1, "面接日を入力してください"),
  phase: z.string(),
  memo: z.string().max(2000, "メモは2000文字以内です"),
});

export type InterviewInput = z.infer<typeof interviewSchema>;

export const interviewLogSchema = z.object({
  question_text: z
    .string()
    .min(1, "質問を入力してください")
    .max(1000, "質問は1000文字以内です"),
  answer_text: z
    .string()
    .min(1, "回答を入力してください")
    .max(5000, "回答は5000文字以内です"),
  self_evaluation: z.string().max(2000, "自己評価は2000文字以内です"),
});

export type InterviewLogInput = z.infer<typeof interviewLogSchema>;
