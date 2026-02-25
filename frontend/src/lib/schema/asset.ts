// DesignDocument/5_FrontendDesign.md 5.9 / 8_DBDesign.dbml - 自己資産フォームスキーマ
import { z } from "zod";

export const assetSchema = z.object({
  type: z.enum([
    "academic",
    "work",
    "project",
    "skill",
    "strength",
    "weakness",
    "episode",
    "other",
  ]),
  title: z
    .string()
    .min(1, "タイトルを入力してください")
    .max(200, "タイトルは200文字以内です"),
  content: z
    .string()
    .min(1, "内容を入力してください")
    .max(5000, "内容は5000文字以内です"),
  tags: z.array(z.string()),
});

export type AssetInput = z.infer<typeof assetSchema>;
