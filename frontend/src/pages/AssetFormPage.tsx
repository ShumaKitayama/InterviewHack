// DesignDocument/5_FrontendDesign.md 5.9, 3_UIUXDesign.md 3.3.2 - 自己資産作成/編集
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { ChipSelector } from "../components/ui/ChipSelector";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import {
  useAsset,
  useAssetTagRelations,
  useAssetTags,
  useCreateAsset,
  useUpdateAsset,
} from "../hooks/useAssets";
import { ASSET_TYPE_OPTIONS } from "../lib/constants";
import { assetSchema, type AssetInput } from "../lib/schema/asset";
import type { AssetType } from "../lib/supabase/database.types";

export function AssetFormPage() {
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id;
  const navigate = useNavigate();

  const { data: existingAsset, isLoading: loadingAsset } = useAsset(id);
  const { data: allTags } = useAssetTags();
  const { data: existingTagIds } = useAssetTagRelations(id);

  const createAsset = useCreateAsset();
  const updateAsset = useUpdateAsset();

  const [tagInput, setTagInput] = useState("");
  // 未編集の間はサーバー上のタグをそのまま表示する。編集後だけローカルの下書きを使う。
  const [tagDraft, setTagDraft] = useState<{
    scope: string;
    tags: string[];
  } | null>(null);

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<AssetInput>({
    resolver: zodResolver(assetSchema),
    defaultValues: {
      type: "episode",
      title: "",
      content: "",
      tags: [],
    },
  });

  const selectedType = useWatch({ control, name: "type" });

  const serverTags = useMemo(() => {
    if (!existingTagIds || !allTags) return [];
    return existingTagIds
      .map((tagId) => allTags.find((tag) => tag.id === tagId)?.name)
      .filter((name): name is string => Boolean(name));
  }, [existingTagIds, allTags]);

  const tagScope = `${id ?? "new"}:${existingTagIds?.join("\0") ?? ""}`;
  const tags = tagDraft?.scope === tagScope ? tagDraft.tags : serverTags;

  useEffect(() => {
    if (existingAsset) {
      setValue("type", existingAsset.type as AssetType);
      setValue("title", existingAsset.title);
      setValue("content", existingAsset.content);
    }
  }, [existingAsset, setValue]);

  const updateTags = (next: string[]) => {
    setTagDraft({ scope: tagScope, tags: next });
  };

  const addTag = () => {
    const trimmed = tagInput.trim();
    if (trimmed && !tags.includes(trimmed)) {
      updateTags([...tags, trimmed]);
    }
    setTagInput("");
  };

  const removeTag = (tagName: string) => {
    updateTags(tags.filter((tag) => tag !== tagName));
  };

  const onSubmit = async (data: AssetInput) => {
    const input = { ...data, tags };
    try {
      if (isEdit && id) {
        await updateAsset.mutateAsync({ id, input });
      } else {
        await createAsset.mutateAsync(input);
      }
      navigate("/assets", { replace: true });
    } catch {
      // mutation error is handled by TanStack Query
    }
  };

  if (isEdit && loadingAsset) {
    return <LoadingSpinner />;
  }

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader
        title={isEdit ? "自己資産を編集" : "自己資産を追加"}
        showBack
        rightAction={{
          label: "保存",
          onClick: handleSubmit(onSubmit),
          disabled: isSubmitting,
        }}
      />

      <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-6">
        {/* カテゴリ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            カテゴリ
          </label>
          <ChipSelector
            options={ASSET_TYPE_OPTIONS.slice(0, 5)}
            value={selectedType as AssetType}
            onChange={(v) => setValue("type", v)}
          />
        </div>

        {/* タイトル */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            タイトル
          </label>
          <input
            {...register("title")}
            placeholder="例: チームリーダーとしてプロジェクトを成功に..."
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
          {errors.title && (
            <p className="text-xs text-red-500 mt-1">{errors.title.message}</p>
          )}
        </div>

        {/* 内容 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            内容
          </label>
          <textarea
            {...register("content")}
            placeholder="具体的なエピソード、数値、成果などを含めて記述してください"
            rows={6}
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
          {errors.content && (
            <p className="text-xs text-red-500 mt-1">
              {errors.content.message}
            </p>
          )}
        </div>

        {/* タグ */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            タグ
          </label>
          {tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 px-3 py-1 bg-orange-50 text-orange-700 rounded-full text-xs font-medium"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() => removeTag(tag)}
                    className="ml-0.5 hover:text-orange-900"
                  >
                    <svg
                      className="w-3.5 h-3.5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </span>
              ))}
            </div>
          )}
          <div className="flex gap-2">
            <input
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addTag();
                }
              }}
              placeholder="タグを追加"
              className="flex-1 px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
            />
            <button
              type="button"
              onClick={addTag}
              className="w-11 h-11 flex items-center justify-center bg-orange-500 text-white rounded-xl hover:bg-orange-600 transition-colors flex-shrink-0"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                strokeWidth={2.5}
              >
                <path d="M12 4v16m8-8H4" />
              </svg>
            </button>
          </div>
        </div>
      </form>

      <BottomAction
        label="保存する"
        onClick={handleSubmit(onSubmit)}
        loading={isSubmitting}
        disabled={isSubmitting}
      />
    </div>
  );
}
