// DesignDocument/5_FrontendDesign.md 5.5.3 セッション管理
import { useContext } from "react";
import { AuthContext } from "./auth-context";

export function useAuth() {
  return useContext(AuthContext);
}
