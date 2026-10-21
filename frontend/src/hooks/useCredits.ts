// DesignDocument/5_FrontendDesign.md 5.6.1, 5.6.2, 5.8 - クレジット hooks
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../features/auth/useAuth";
import { supabase } from "../lib/supabase/client";
import type {
  CreditBalance,
  CreditPackage,
  CreditTransaction,
} from "../lib/supabase/database.types";

export function useCreditBalance() {
  const { user } = useAuth();

  return useQuery<CreditBalance | null>({
    queryKey: ["creditBalance"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("credit_balances")
        .select("*")
        .single();
      if (error && error.code !== "PGRST116") throw error;
      return data ?? null;
    },
    enabled: !!user,
    staleTime: 10 * 1000,
  });
}

export function useCreditTransactions() {
  const { user } = useAuth();

  return useQuery<CreditTransaction[]>({
    queryKey: ["creditTransactions"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("credit_transactions")
        .select("*")
        .order("created_at", { ascending: false })
        .limit(50);
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useCreditPackages() {
  return useQuery<CreditPackage[]>({
    queryKey: ["creditPackages"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("credit_packages")
        .select("*")
        .eq("is_active", true)
        .order("credits", { ascending: true });
      if (error) throw error;
      return data ?? [];
    },
  });
}
