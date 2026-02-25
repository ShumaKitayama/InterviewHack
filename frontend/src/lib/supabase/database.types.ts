// DesignDocument/8_DBDesign.dbml - Supabase テーブル型定義

export type AssetType =
  | "academic"
  | "work"
  | "project"
  | "skill"
  | "strength"
  | "weakness"
  | "episode"
  | "other";

export interface Asset {
  id: string;
  user_id: string;
  type: AssetType;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface AssetTag {
  id: string;
  user_id: string;
  name: string;
}

export interface AssetTagRelation {
  asset_id: string;
  tag_id: string;
}

export interface Company {
  id: string;
  user_id: string;
  name: string;
  job_title: string | null;
  selection_phase: string | null;
  motivation_memo: string | null;
  reference_url: string | null;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface ResearchMaterial {
  id: string;
  user_id: string;
  company_id: string;
  title: string;
  type: "url" | "pdf" | "note" | "image";
  source_url: string | null;
  content_raw: string | null;
  summary: string | null;
  created_at: string;
  updated_at: string;
}

export type ESProjectStatus = "draft" | "in_review" | "completed" | "submitted";

export interface ESProject {
  id: string;
  user_id: string;
  company_id: string;
  status: ESProjectStatus;
  deadline: string | null;
  created_at: string;
  updated_at: string;
  company?: Company;
}

export interface ESSection {
  id: string;
  es_project_id: string;
  user_id: string;
  title: string;
  content_draft: string | null;
  content_final: string | null;
  character_limit: number | null;
  version: number;
  created_at: string;
  updated_at: string;
}

export type InterviewResult = "pass" | "fail" | "pending";

export interface Interview {
  id: string;
  user_id: string;
  company_id: string | null;
  interview_date: string;
  phase: string | null;
  result: InterviewResult | null;
  memo: string | null;
  created_at: string;
  updated_at: string;
  company?: Company;
}

export interface InterviewLog {
  id: string;
  interview_id: string;
  user_id: string;
  question_text: string;
  answer_text: string;
  self_evaluation: string | null;
  ai_feedback_text: string | null;
  ai_improved_answer_text: string | null;
  score: number | null;
  created_at: string;
  updated_at: string;
}

export type QASourceType = "pre_generated" | "review_improved";
export type QAReviewStatus = "new" | "reviewed" | "needs_review";

export interface QA {
  id: string;
  user_id: string;
  company_id: string | null;
  question_text: string;
  answer_text: string;
  source_type: QASourceType;
  is_favorite: boolean;
  review_status: QAReviewStatus;
  created_at: string;
  updated_at: string;
  company?: Company;
}

export interface CreditBalance {
  user_id: string;
  balance: number;
  updated_at: string;
}

export interface CreditTransaction {
  id: string;
  user_id: string;
  type: "purchase" | "consume" | "bonus" | "adjustment";
  amount: number;
  status: "pending" | "committed" | "reversed";
  reason: string;
  external_ref: string;
  created_at: string;
}

export interface CreditPackage {
  id: string;
  name: string;
  credits: number;
  stripe_price_id: string;
  is_active: boolean;
}

export interface CreditCost {
  feature: string;
  cost: number;
  is_active: boolean;
}

export interface Profile {
  user_id: string;
  display_name: string | null;
  created_at: string;
  updated_at: string;
}
