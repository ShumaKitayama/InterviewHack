// DesignDocument/3_UIUXDesign.md, 8_DBDesign.dbml - UI定数

import type { AssetType } from "./supabase/database.types";

export const ASSET_TYPE_OPTIONS: {
  value: AssetType;
  label: string;
}[] = [
  { value: "episode", label: "ガクチカ" },
  { value: "work", label: "職務経歴" },
  { value: "strength", label: "強み" },
  { value: "weakness", label: "弱み" },
  { value: "skill", label: "スキル" },
  { value: "academic", label: "学歴" },
  { value: "project", label: "プロジェクト" },
  { value: "other", label: "その他" },
];

export const ASSET_TYPE_LABEL: Record<AssetType, string> = {
  episode: "ガクチカ",
  work: "職務経歴",
  strength: "強み",
  weakness: "弱み",
  skill: "スキル",
  academic: "学歴",
  project: "プロジェクト",
  other: "その他",
};

export const SELECTION_PHASE_OPTIONS = [
  { value: "researching", label: "リサーチ中" },
  { value: "es_submitted", label: "ES提出" },
  { value: "first_interview", label: "一次面接" },
  { value: "second_interview", label: "二次面接" },
  { value: "final_interview", label: "最終面接" },
  { value: "offer", label: "内定" },
  { value: "rejected", label: "不合格" },
] as const;

export const SELECTION_PHASE_LABEL: Record<string, string> = Object.fromEntries(
  SELECTION_PHASE_OPTIONS.map((o) => [o.value, o.label]),
);

export const INTERVIEW_PHASE_OPTIONS = [
  { value: "first", label: "一次面接" },
  { value: "second", label: "二次面接" },
  { value: "final", label: "最終面接" },
  { value: "casual", label: "カジュアル" },
  { value: "group", label: "グループ" },
] as const;

export const INTERVIEW_RESULT_OPTIONS = [
  { value: "pending", label: "結果待ち" },
  { value: "pass", label: "合格" },
  { value: "fail", label: "不合格" },
] as const;

export const ES_STATUS_LABEL: Record<string, string> = {
  draft: "作成中",
  in_review: "推敲中",
  completed: "完了",
  submitted: "提出済",
};

export const QA_REVIEW_STATUS_LABEL: Record<string, string> = {
  new: "未復習",
  reviewed: "復習済み",
  needs_review: "要再確認",
};
