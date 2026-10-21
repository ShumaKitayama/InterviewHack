// DesignDocument/5_FrontendDesign.md 5.6.1, 5.8 - 面接 CRUD hooks
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../features/auth/useAuth";
import type {
  InterviewInput,
  InterviewLogInput,
} from "../lib/schema/interview";
import { supabase } from "../lib/supabase/client";
import type { Interview, InterviewLog } from "../lib/supabase/database.types";

export function useInterviews() {
  const { user } = useAuth();

  return useQuery<Interview[]>({
    queryKey: ["interviews"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("interviews")
        .select("*, company:companies(*)")
        .order("interview_date", { ascending: false });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useInterview(id: string | undefined) {
  const { user } = useAuth();

  return useQuery<Interview | null>({
    queryKey: ["interview", id],
    queryFn: async () => {
      if (!id) return null;
      const { data, error } = await supabase
        .from("interviews")
        .select("*, company:companies(*)")
        .eq("id", id)
        .single();
      if (error) throw error;
      return data;
    },
    enabled: !!user && !!id,
  });
}

export function useInterviewLogs(interviewId: string | undefined) {
  const { user } = useAuth();

  return useQuery<InterviewLog[]>({
    queryKey: ["interviewLogs", interviewId],
    queryFn: async () => {
      if (!interviewId) return [];
      const { data, error } = await supabase
        .from("interview_logs")
        .select("*")
        .eq("interview_id", interviewId)
        .order("created_at", { ascending: true });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user && !!interviewId,
  });
}

export function useCreateInterview() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: InterviewInput) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("interviews")
        .insert({
          user_id: user.id,
          company_id: input.company_id,
          interview_date: input.interview_date,
          phase: input.phase || null,
          memo: input.memo || null,
          result: "pending",
        })
        .select("*, company:companies(*)")
        .single();
      if (error) throw error;
      return data as Interview;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["interviews"] });
    },
  });
}

export function useCreateInterviewLog() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      interviewId,
      input,
    }: {
      interviewId: string;
      input: InterviewLogInput;
    }) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("interview_logs")
        .insert({
          interview_id: interviewId,
          user_id: user.id,
          question_text: input.question_text,
          answer_text: input.answer_text,
          self_evaluation: input.self_evaluation || null,
        })
        .select()
        .single();
      if (error) throw error;
      return data as InterviewLog;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["interviewLogs", variables.interviewId],
      });
    },
  });
}

export function useUpdateInterviewResult() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, result }: { id: string; result: string }) => {
      const { error } = await supabase
        .from("interviews")
        .update({ result, updated_at: new Date().toISOString() })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["interviews"] });
      queryClient.invalidateQueries({ queryKey: ["interview", variables.id] });
    },
  });
}
