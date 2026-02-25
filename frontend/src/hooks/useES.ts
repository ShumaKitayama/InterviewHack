// DesignDocument/5_FrontendDesign.md 5.6.1, 5.8 - ES hooks
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../features/auth/AuthContext";
import type { ESProjectInput, ESSectionInput } from "../lib/schema/es";
import { supabase } from "../lib/supabase/client";
import type { ESProject, ESSection } from "../lib/supabase/database.types";

export function useESProjects() {
  const { user } = useAuth();

  return useQuery<ESProject[]>({
    queryKey: ["es_projects"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("es_projects")
        .select("*, company:companies(*)")
        .order("created_at", { ascending: false });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useESProject(id: string | undefined) {
  const { user } = useAuth();

  return useQuery<ESProject | null>({
    queryKey: ["es_project", id],
    queryFn: async () => {
      if (!id) return null;
      const { data, error } = await supabase
        .from("es_projects")
        .select("*, company:companies(*)")
        .eq("id", id)
        .single();
      if (error) throw error;
      return data;
    },
    enabled: !!user && !!id,
  });
}

export function useESSections(projectId: string | undefined) {
  const { user } = useAuth();

  return useQuery<ESSection[]>({
    queryKey: ["es_sections", projectId],
    queryFn: async () => {
      if (!projectId) return [];
      const { data, error } = await supabase
        .from("es_sections")
        .select("*")
        .eq("es_project_id", projectId)
        .order("created_at", { ascending: true });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user && !!projectId,
  });
}

export function useCreateESProject() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: ESProjectInput) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("es_projects")
        .insert({
          user_id: user.id,
          company_id: input.company_id,
          status: "draft",
          deadline: input.deadline || null,
        })
        .select("*, company:companies(*)")
        .single();
      if (error) throw error;
      return data as ESProject;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["es_projects"] });
    },
  });
}

export function useCreateESSection() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      projectId,
      input,
    }: {
      projectId: string;
      input: ESSectionInput;
    }) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("es_sections")
        .insert({
          es_project_id: projectId,
          user_id: user.id,
          title: input.title,
          content_draft: input.content_draft || null,
          character_limit: input.character_limit ?? null,
        })
        .select()
        .single();
      if (error) throw error;
      return data as ESSection;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["es_sections", variables.projectId],
      });
    },
  });
}

export function useUpdateESSection() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      content,
    }: {
      id: string;
      projectId: string;
      content: string;
    }) => {
      const { error } = await supabase
        .from("es_sections")
        .update({
          content_draft: content,
          updated_at: new Date().toISOString(),
        })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["es_sections", variables.projectId],
      });
    },
  });
}
