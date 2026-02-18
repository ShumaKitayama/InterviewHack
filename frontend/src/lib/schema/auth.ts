// DesignDocument/5_FrontendDesign.md 5.9 フォーム設計（Zod）
import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().email("有効なメールアドレスを入力してください"),
  password: z.string().min(6, "パスワードは6文字以上です"),
});

export type LoginInput = z.infer<typeof loginSchema>;
