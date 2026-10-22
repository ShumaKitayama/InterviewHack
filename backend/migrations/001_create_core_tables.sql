-- DesignDocument/8_DBDesign.md / 8_DBDesign.dbml（Neon / PostgreSQL）
-- 冪等：何度適用してもエラーにならない（8.9 / IT-DB-008）。RLS は使わない（D6）。

CREATE TABLE IF NOT EXISTS users (
  id            uuid PRIMARY KEY,
  firebase_uid  text NOT NULL UNIQUE,
  email         text NULL,
  display_name  text NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz NULL
);

CREATE TABLE IF NOT EXISTS assets (
  id          uuid PRIMARY KEY,
  user_id     uuid NOT NULL REFERENCES users(id),
  type        text NOT NULL CHECK (type IN ('academic','work','project','skill','strength','weakness','episode','other')),
  title       text NOT NULL,
  content     text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_assets_user ON assets (user_id);
CREATE INDEX IF NOT EXISTS idx_assets_user_created_desc ON assets (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS asset_tags (
  id       uuid PRIMARY KEY,
  user_id  uuid NOT NULL REFERENCES users(id),
  name     text NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_asset_tags_user_name ON asset_tags (user_id, name);

CREATE TABLE IF NOT EXISTS asset_tag_relations (
  asset_id  uuid NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  tag_id    uuid NOT NULL REFERENCES asset_tags(id) ON DELETE CASCADE,
  PRIMARY KEY (asset_id, tag_id)
);

CREATE TABLE IF NOT EXISTS companies (
  id               uuid PRIMARY KEY,
  user_id          uuid NOT NULL REFERENCES users(id),
  name             text NOT NULL,
  job_title        text NULL,
  selection_phase  text NULL,
  motivation_memo  text NULL,
  reference_url    text NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  deleted_at       timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_companies_user ON companies (user_id);
CREATE INDEX IF NOT EXISTS idx_companies_user_name ON companies (user_id, name);

CREATE TABLE IF NOT EXISTS research_materials (
  id           uuid PRIMARY KEY,
  user_id      uuid NOT NULL REFERENCES users(id),
  company_id   uuid NOT NULL REFERENCES companies(id),
  title        text NOT NULL,
  type         text NOT NULL CHECK (type IN ('url','pdf','note','image')),
  source_url   text NULL,
  content_raw  text NULL,
  summary      text NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_research_materials_user ON research_materials (user_id);
CREATE INDEX IF NOT EXISTS idx_research_materials_company ON research_materials (company_id);

CREATE TABLE IF NOT EXISTS es_projects (
  id          uuid PRIMARY KEY,
  user_id     uuid NOT NULL REFERENCES users(id),
  company_id  uuid NOT NULL REFERENCES companies(id),
  status      text NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started','draft','in_review','completed','submitted')),
  deadline    date NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_es_projects_user ON es_projects (user_id);
CREATE INDEX IF NOT EXISTS idx_es_projects_company ON es_projects (company_id);
CREATE INDEX IF NOT EXISTS idx_es_projects_user_status ON es_projects (user_id, status);

CREATE TABLE IF NOT EXISTS es_sections (
  id               uuid PRIMARY KEY,
  es_project_id    uuid NOT NULL REFERENCES es_projects(id) ON DELETE CASCADE,
  user_id          uuid NOT NULL REFERENCES users(id),
  title            text NOT NULL,
  content_draft    text NULL,
  content_final    text NULL,
  character_limit  int NULL,
  version          int NOT NULL DEFAULT 1,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_es_sections_project ON es_sections (es_project_id);
CREATE INDEX IF NOT EXISTS idx_es_sections_user ON es_sections (user_id);

CREATE TABLE IF NOT EXISTS es_section_versions (
  id             uuid PRIMARY KEY,
  es_section_id  uuid NOT NULL REFERENCES es_sections(id) ON DELETE CASCADE,
  user_id        uuid NOT NULL REFERENCES users(id),
  version        int NOT NULL,
  content        text NOT NULL,
  source         text NOT NULL CHECK (source IN ('manual','es_generate','es_refine')),
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_es_section_versions_section_created_desc ON es_section_versions (es_section_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_es_section_versions_user ON es_section_versions (user_id);

CREATE TABLE IF NOT EXISTS es_project_asset_relations (
  es_project_id  uuid NOT NULL REFERENCES es_projects(id) ON DELETE CASCADE,
  asset_id       uuid NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  PRIMARY KEY (es_project_id, asset_id)
);

CREATE TABLE IF NOT EXISTS es_project_research_material_relations (
  es_project_id         uuid NOT NULL REFERENCES es_projects(id) ON DELETE CASCADE,
  research_material_id  uuid NOT NULL REFERENCES research_materials(id) ON DELETE CASCADE,
  PRIMARY KEY (es_project_id, research_material_id)
);

CREATE TABLE IF NOT EXISTS interviews (
  id              uuid PRIMARY KEY,
  user_id         uuid NOT NULL REFERENCES users(id),
  company_id      uuid NULL REFERENCES companies(id),
  interview_date  date NOT NULL,
  phase           text NULL,
  result          text NULL CHECK (result IS NULL OR result IN ('pass','fail','pending','')),
  memo            text NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_interviews_user ON interviews (user_id);
CREATE INDEX IF NOT EXISTS idx_interviews_company ON interviews (company_id);
CREATE INDEX IF NOT EXISTS idx_interviews_user_date_desc ON interviews (user_id, interview_date DESC);

CREATE TABLE IF NOT EXISTS interview_logs (
  id                       uuid PRIMARY KEY,
  interview_id             uuid NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
  user_id                  uuid NOT NULL REFERENCES users(id),
  question_text            text NOT NULL,
  answer_text              text NOT NULL,
  self_evaluation          text NULL,
  ai_feedback_text         text NULL,
  ai_improved_answer_text  text NULL,
  score                    int NULL,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_interview_logs_user ON interview_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_interview_logs_interview ON interview_logs (interview_id);
CREATE INDEX IF NOT EXISTS idx_interview_logs_user_created_desc ON interview_logs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS qa (
  id             uuid PRIMARY KEY,
  user_id        uuid NOT NULL REFERENCES users(id),
  company_id     uuid NULL REFERENCES companies(id),
  question_text  text NOT NULL,
  answer_text    text NOT NULL,
  source_type    text NOT NULL CHECK (source_type IN ('pre_generated','review_improved')),
  is_favorite    boolean NOT NULL DEFAULT false,
  review_status  text NOT NULL DEFAULT 'new' CHECK (review_status IN ('new','reviewed','needs_review')),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_qa_user ON qa (user_id);
CREATE INDEX IF NOT EXISTS idx_qa_user_created_desc ON qa (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_qa_company ON qa (company_id);

CREATE TABLE IF NOT EXISTS qa_asset_relations (
  qa_id     uuid NOT NULL REFERENCES qa(id) ON DELETE CASCADE,
  asset_id  uuid NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  PRIMARY KEY (qa_id, asset_id)
);
