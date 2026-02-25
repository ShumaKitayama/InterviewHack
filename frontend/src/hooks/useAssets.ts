// DesignDocument/5_FrontendDesign.md 5.6.1, 5.8 - 自己資産 CRUD hooks
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../features/auth/AuthContext";
import type { AssetInput } from "../lib/schema/asset";
import { supabase } from "../lib/supabase/client";
import type { Asset, AssetTag } from "../lib/supabase/database.types";

export function useAssets() {
  const { user } = useAuth();

  return useQuery<Asset[]>({
    queryKey: ["assets"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("assets")
        .select("*")
        .is("deleted_at", null)
        .order("created_at", { ascending: false });
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useAsset(id: string | undefined) {
  const { user } = useAuth();

  return useQuery<Asset | null>({
    queryKey: ["asset", id],
    queryFn: async () => {
      if (!id) return null;
      const { data, error } = await supabase
        .from("assets")
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

export function useAssetTags() {
  const { user } = useAuth();

  return useQuery<AssetTag[]>({
    queryKey: ["assetTags"],
    queryFn: async () => {
      const { data, error } = await supabase
        .from("asset_tags")
        .select("*")
        .order("name");
      if (error) throw error;
      return data ?? [];
    },
    enabled: !!user,
  });
}

export function useAssetTagRelations(assetId: string | undefined) {
  const { user } = useAuth();

  return useQuery<string[]>({
    queryKey: ["assetTagRelations", assetId],
    queryFn: async () => {
      if (!assetId) return [];
      const { data, error } = await supabase
        .from("asset_tag_relations")
        .select("tag_id")
        .eq("asset_id", assetId);
      if (error) throw error;
      return (data ?? []).map((r) => r.tag_id);
    },
    enabled: !!user && !!assetId,
  });
}

export function useCreateAsset() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: AssetInput) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("assets")
        .insert({
          user_id: user.id,
          type: input.type,
          title: input.title,
          content: input.content,
        })
        .select()
        .single();
      if (error) throw error;

      if (input.tags.length > 0) {
        for (const tagName of input.tags) {
          let tagId: string;
          const { data: existing } = await supabase
            .from("asset_tags")
            .select("id")
            .eq("user_id", user.id)
            .eq("name", tagName)
            .single();

          if (existing) {
            tagId = existing.id;
          } else {
            const { data: newTag, error: tagError } = await supabase
              .from("asset_tags")
              .insert({ user_id: user.id, name: tagName })
              .select()
              .single();
            if (tagError) throw tagError;
            tagId = newTag.id;
          }

          await supabase
            .from("asset_tag_relations")
            .insert({ asset_id: data.id, tag_id: tagId });
        }
      }

      return data as Asset;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["assets"] });
      queryClient.invalidateQueries({ queryKey: ["assetTags"] });
    },
  });
}

export function useUpdateAsset() {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: AssetInput }) => {
      if (!user) throw new Error("Not authenticated");

      const { data, error } = await supabase
        .from("assets")
        .update({
          type: input.type,
          title: input.title,
          content: input.content,
          updated_at: new Date().toISOString(),
        })
        .eq("id", id)
        .select()
        .single();
      if (error) throw error;

      await supabase.from("asset_tag_relations").delete().eq("asset_id", id);

      if (input.tags.length > 0) {
        for (const tagName of input.tags) {
          let tagId: string;
          const { data: existing } = await supabase
            .from("asset_tags")
            .select("id")
            .eq("user_id", user.id)
            .eq("name", tagName)
            .single();

          if (existing) {
            tagId = existing.id;
          } else {
            const { data: newTag, error: tagError } = await supabase
              .from("asset_tags")
              .insert({ user_id: user.id, name: tagName })
              .select()
              .single();
            if (tagError) throw tagError;
            tagId = newTag.id;
          }

          await supabase
            .from("asset_tag_relations")
            .insert({ asset_id: id, tag_id: tagId });
        }
      }

      return data as Asset;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["assets"] });
      queryClient.invalidateQueries({ queryKey: ["asset", variables.id] });
      queryClient.invalidateQueries({ queryKey: ["assetTags"] });
      queryClient.invalidateQueries({
        queryKey: ["assetTagRelations", variables.id],
      });
    },
  });
}

export function useDeleteAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await supabase
        .from("assets")
        .update({ deleted_at: new Date().toISOString() })
        .eq("id", id);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["assets"] });
    },
  });
}
