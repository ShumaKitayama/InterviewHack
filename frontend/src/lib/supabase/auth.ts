// DesignDocument/5_FrontendDesign.md 5.5 認証設計
import { supabase } from "./client";

const appOrigin = import.meta.env.VITE_APP_ORIGIN || window.location.origin;

export async function signInWithGoogle() {
  return supabase.auth.signInWithOAuth({
    provider: "google",
    options: { redirectTo: `${appOrigin}/auth/callback` },
  });
}

export async function signInWithEmail(email: string, password: string) {
  return supabase.auth.signInWithPassword({ email, password });
}

export async function signUpWithEmail(email: string, password: string) {
  return supabase.auth.signUp({ email, password });
}

export async function resetPassword(email: string) {
  return supabase.auth.resetPasswordForEmail(email, {
    redirectTo: `${appOrigin}/reset-password`,
  });
}

export async function signOut() {
  return supabase.auth.signOut();
}
