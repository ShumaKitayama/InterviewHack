// DesignDocument/5_FrontendDesign.md 5.5.3 セッション管理
// コンテキスト定義だけを置く。コンポーネントと同居させると
// react-refresh/only-export-components に抵触する。
import type { Session, User } from "@supabase/supabase-js";
import { createContext } from "react";

export interface AuthContextType {
  session: Session | null;
  user: User | null;
  loading: boolean;
}

export const AuthContext = createContext<AuthContextType>({
  session: null,
  user: null,
  loading: true,
});
