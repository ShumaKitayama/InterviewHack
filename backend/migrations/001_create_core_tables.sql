-- DesignDocument/8_DBDesign.md 8.2-8.4 / DesignDocument/8_DBDesign.dbml
-- コアドメインテーブル: profiles, assets, companies, interviews, qa, es

-- 8.2 profiles
CREATE TABLE IF NOT EXISTS profiles (
  user_id uuid PRIMARY KEY REFERENCES auth.users(id),
  display_name text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- 8.3 assets
CREATE TABLE IF NOT EXISTS assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  type text NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_assets_user_id ON assets(user_id);
CREATE INDEX IF NOT EXISTS idx_assets_user_created_desc ON assets(user_id, created_at DESC);

-- 8.3 asset_tags
CREATE TABLE IF NOT EXISTS asset_tags (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  name text NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_asset_tags_user_name ON asset_tags(user_id, name);

-- 8.3 asset_tag_relations
CREATE TABLE IF NOT EXISTS asset_tag_relations (
  asset_id uuid REFERENCES assets(id) ON DELETE CASCADE,
  tag_id uuid REFERENCES asset_tags(id) ON DELETE CASCADE,
  PRIMARY KEY (asset_id, tag_id)
);

-- 8.3 companies
CREATE TABLE IF NOT EXISTS companies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  name text NOT NULL,
  job_title text,
  selection_phase text,
  motivation_memo text,
  reference_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_companies_user_id ON companies(user_id);
CREATE INDEX IF NOT EXISTS idx_companies_user_name ON companies(user_id, name);

-- 8.3 research_materials
CREATE TABLE IF NOT EXISTS research_materials (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  company_id uuid NOT NULL REFERENCES companies(id),
  title text NOT NULL,
  type text NOT NULL,
  source_url text,
  content_raw text,
  summary text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_research_materials_user_id ON research_materials(user_id);
CREATE INDEX IF NOT EXISTS idx_research_materials_company_id ON research_materials(company_id);

-- 8.3 es_projects
CREATE TABLE IF NOT EXISTS es_projects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  company_id uuid NOT NULL REFERENCES companies(id),
  status text NOT NULL DEFAULT 'draft',
  deadline date,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_es_projects_user_id ON es_projects(user_id);
CREATE INDEX IF NOT EXISTS idx_es_projects_company_id ON es_projects(company_id);
CREATE INDEX IF NOT EXISTS idx_es_projects_user_status ON es_projects(user_id, status);

-- 8.3 es_sections
CREATE TABLE IF NOT EXISTS es_sections (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  es_project_id uuid NOT NULL REFERENCES es_projects(id),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  title text NOT NULL,
  content_draft text,
  content_final text,
  character_limit int,
  version int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_es_sections_project ON es_sections(es_project_id);
CREATE INDEX IF NOT EXISTS idx_es_sections_user ON es_sections(user_id);

-- 8.3 es_project_asset_relations
CREATE TABLE IF NOT EXISTS es_project_asset_relations (
  es_project_id uuid REFERENCES es_projects(id) ON DELETE CASCADE,
  asset_id uuid REFERENCES assets(id) ON DELETE CASCADE,
  PRIMARY KEY (es_project_id, asset_id)
);

-- 8.3 es_project_research_material_relations
CREATE TABLE IF NOT EXISTS es_project_research_material_relations (
  es_project_id uuid REFERENCES es_projects(id) ON DELETE CASCADE,
  research_material_id uuid REFERENCES research_materials(id) ON DELETE CASCADE,
  PRIMARY KEY (es_project_id, research_material_id)
);

-- 8.3 interviews
CREATE TABLE IF NOT EXISTS interviews (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  company_id uuid REFERENCES companies(id),
  interview_date date NOT NULL,
  phase text,
  result text,
  memo text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_interviews_user_id ON interviews(user_id);
CREATE INDEX IF NOT EXISTS idx_interviews_company_id ON interviews(company_id);
CREATE INDEX IF NOT EXISTS idx_interviews_user_date_desc ON interviews(user_id, interview_date DESC);

-- 8.3 interview_logs
CREATE TABLE IF NOT EXISTS interview_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  interview_id uuid NOT NULL REFERENCES interviews(id),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  question_text text NOT NULL,
  answer_text text NOT NULL,
  self_evaluation text,
  ai_feedback_text text,
  ai_improved_answer_text text,
  score int,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_interview_logs_user_id ON interview_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_interview_logs_interview ON interview_logs(interview_id);
CREATE INDEX IF NOT EXISTS idx_interview_logs_user_created_desc ON interview_logs(user_id, created_at DESC);

-- 8.3 qa
CREATE TABLE IF NOT EXISTS qa (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  company_id uuid REFERENCES companies(id),
  question_text text NOT NULL,
  answer_text text NOT NULL,
  source_type text NOT NULL,
  is_favorite boolean NOT NULL DEFAULT false,
  review_status text NOT NULL DEFAULT 'new',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_qa_user_id ON qa(user_id);
CREATE INDEX IF NOT EXISTS idx_qa_user_created_desc ON qa(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_qa_company_id ON qa(company_id);

-- 8.3 qa_asset_relations
CREATE TABLE IF NOT EXISTS qa_asset_relations (
  qa_id uuid REFERENCES qa(id) ON DELETE CASCADE,
  asset_id uuid REFERENCES assets(id) ON DELETE CASCADE,
  PRIMARY KEY (qa_id, asset_id)
);
