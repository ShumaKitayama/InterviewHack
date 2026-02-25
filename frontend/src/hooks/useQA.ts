// DesignDocument/5_FrontendDesign.md 5.6.1, 5.8 - Q&A hooks
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../features/auth/AuthContext";
import { supabase } from "../lib/supabase/client";
import type { QA } from "../lib/supabase/database.types";

export function useQAList(filters?: {
  companyId?: string;
  reviewStatus?: string;
  favoriteOnly?: boolean;
}) {
  const { user } = useAuth();

  return useQuery<QA[]>({
    queryKey: ["qa", filters],
    queryFn: async () => {
      let query = supabase
        .from("qa")
        .select("*, company:companies(*)")
        .order("created_at", { ascending: false });

      if (filters?.companyId) {
        query = query.eq("company_id", filters.companyId);
      }
      if (filters?.reviewStatus) {
        query = query.eq("review_status", filters.reviewStatus);
      }
      if (filters?.favoriteOnly) {
        query = query.eq("is_favorite", true);
      }

      const { data, error } = await query;
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useQADetail(id: string | undefined) {
  const { user } = useAuth();

  return useQuery<QA | null>({
    queryKey: ["qa", id],
    queryFn: async () => {
      if (!id) return null;
      const { data, error } = await supabase
        .from("qa")
        .select("*, company:companies(*)")
        .eq("id", id)
        .single();
      if (error) throw error;
      return data;
    },
    enabled: !!user && !!id,
  });
}

export function useToggleFavorite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      isFavorite,
    }: {
      id: string;
      isFavorite: boolean;
    }) => {
      const { error } = await supabase
        .from("qa")
        .update({
          is_favorite: isFavorite,
          updated_at: new Date().toISOString(),
        })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["qa"] });
    },
  });
}

export function useUpdateReviewStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: string }) => {
      const { error } = await supabase
        .from("qa")
        .update({ review_status: status, updated_at: new Date().toISOString() })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["qa"] });
    },
  });
}

export function useCreateQA() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      company_id?: string;
      question_text: string;
      answer_text: string;
      source_type: string;
    }) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("qa")
        .insert({
          user_id: user.id,
          company_id: input.company_id || null,
          question_text: input.question_text,
          answer_text: input.answer_text,
          source_type: input.source_type,
          is_favorite: false,
          review_status: "new",
        })
        .select()
        .single();
      if (error) throw error;
      return data as QA;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["qa"] });
    },
  });
}
