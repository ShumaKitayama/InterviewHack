// DesignDocument/3_UIUXDesign.md 3.5.1 - ダッシュボード統計 hooks
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../features/auth/useAuth";
import { supabase } from "../lib/supabase/client";

interface DashboardStats {
  activeCompanies: number;
  offers: number;
  qaCount: number;
  assetCount: number;
}

export function useDashboardStats() {
  const { user } = useAuth();

  return useQuery<DashboardStats>({
    queryKey: ["dashboardStats"],
    queryFn: async () => {
      const [companiesRes, offersRes, qaRes, assetsRes] = await Promise.all([
        supabase
          .from("companies")
          .select("id", { count: "exact", head: true })
          .is("deleted_at", null)
          .not("selection_phase", "in", '("offer","rejected")'),
        supabase
          .from("companies")
          .select("id", { count: "exact", head: true })
          .is("deleted_at", null)
          .eq("selection_phase", "offer"),
        supabase.from("qa").select("id", { count: "exact", head: true }),
        supabase
          .from("assets")
          .select("id", { count: "exact", head: true })
          .is("deleted_at", null),
      ]);

      return {
        activeCompanies: companiesRes.count ?? 0,
        offers: offersRes.count ?? 0,
        qaCount: qaRes.count ?? 0,
        assetCount: assetsRes.count ?? 0,
      };
    },
    enabled: !!user,
    staleTime: 30 * 1000,
  });
}
