-- DesignDocument/8_DBDesign.md 8.5 / 8.6（AI連携・共有リンク）
-- 冪等：何度適用してもエラーにならない。

CREATE TABLE IF NOT EXISTS prompt_sessions (
  id                uuid PRIMARY KEY,
  user_id           uuid NOT NULL REFERENCES users(id),
  use_case          text NOT NULL CHECK (use_case IN ('qa_generate','answer_review','summary_generate','research_summarize','es_generate','es_refine')),
  template_version  text NOT NULL,
  -- 対象が削除されても集計用に行は残す（8.8）
  company_id        uuid NULL REFERENCES companies(id) ON DELETE SET NULL,
  es_project_id     uuid NULL REFERENCES es_projects(id) ON DELETE SET NULL,
  es_section_id     uuid NULL REFERENCES es_sections(id) ON DELETE SET NULL,
  interview_id      uuid NULL REFERENCES interviews(id) ON DELETE SET NULL,
  interview_log_id  uuid NULL REFERENCES interview_logs(id) ON DELETE SET NULL,
  input_refs        jsonb NOT NULL DEFAULT '{}'::jsonb,
  status            text NOT NULL DEFAULT 'created' CHECK (status IN ('created','pasted','saved','abandoned')),
  parse_result      text NULL CHECK (parse_result IS NULL OR parse_result IN ('success','partial','failed')),
  created_at        timestamptz NOT NULL DEFAULT now(),
  pasted_at         timestamptz NULL,
  completed_at      timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_prompt_sessions_user_status_created ON prompt_sessions (user_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_prompt_sessions_usecase_version ON prompt_sessions (use_case, template_version);

CREATE TABLE IF NOT EXISTS dashboard_summaries (
  id                   uuid PRIMARY KEY,
  user_id              uuid NOT NULL REFERENCES users(id),
  content              text NOT NULL,
  source_interview_id  uuid NULL REFERENCES interviews(id) ON DELETE SET NULL,
  created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dashboard_summaries_user_created_desc ON dashboard_summaries (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS share_links (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  es_project_id      uuid NOT NULL REFERENCES es_projects(id) ON DELETE CASCADE,
  token_hash         text NOT NULL UNIQUE,
  snapshot           jsonb NOT NULL,
  show_company_name  boolean NOT NULL DEFAULT false,
  expires_at         timestamptz NOT NULL,
  revoked_at         timestamptz NULL,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_share_links_project ON share_links (es_project_id);
CREATE INDEX IF NOT EXISTS idx_share_links_expires_at ON share_links (expires_at);
