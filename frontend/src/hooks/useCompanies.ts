// DesignDocument/5_FrontendDesign.md 5.6.1, 5.8 - 企業 CRUD hooks
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../features/auth/useAuth";
import type { CompanyInput } from "../lib/schema/company";
import { supabase } from "../lib/supabase/client";
import type { Company } from "../lib/supabase/database.types";

export function useCompanies() {
  const { user } = useAuth();

  return useQuery<Company[]>({
    queryKey: ["companies"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("companies")
        .select("*")
        .is("deleted_at", null)
        .order("created_at", { ascending: false });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useCompany(id: string | undefined) {
  const { user } = useAuth();

  return useQuery<Company | null>({
    queryKey: ["company", id],
    queryFn: async () => {
      if (!id) return null;
      const { data, error } = await supabase
        .from("companies")
        .select("*")
        .eq("id", id)
        .is("deleted_at", null)
        .single();
      if (error) throw error;
      return data;
    },
    enabled: !!user && !!id,
  });
}

export function useCreateCompany() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CompanyInput) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("companies")
        .insert({
          user_id: user.id,
          name: input.name,
          job_title: input.job_title || null,
          selection_phase: input.selection_phase || null,
          motivation_memo: input.motivation_memo || null,
          reference_url: input.reference_url || null,
        })
        .select()
        .single();
      if (error) throw error;
      return data as Company;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
    },
  });
}

export function useUpdateCompany() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: CompanyInput }) => {
      const { data, error } = await supabase
        .from("companies")
        .update({
          name: input.name,
          job_title: input.job_title || null,
          selection_phase: input.selection_phase || null,
          motivation_memo: input.motivation_memo || null,
          reference_url: input.reference_url || null,
          updated_at: new Date().toISOString(),
        })
        .eq("id", id)
        .select()
        .single();
      if (error) throw error;
      return data as Company;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
      queryClient.invalidateQueries({ queryKey: ["company", variables.id] });
    },
  });
}

export function useDeleteCompany() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await supabase
        .from("companies")
        .update({ deleted_at: new Date().toISOString() })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
    },
  });
}
