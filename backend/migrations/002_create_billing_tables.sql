-- DesignDocument/8_DBDesign.md 8.5-8.6 / DesignDocument/8_DBDesign.dbml
-- 課金・AI系テーブル: credit_balances, credit_transactions, stripe_events, credit_packages, credit_costs, generation_jobs

-- 8.5 credit_balances
CREATE TABLE IF NOT EXISTS credit_balances (
  user_id uuid PRIMARY KEY REFERENCES auth.users(id),
  balance int NOT NULL DEFAULT 0 CHECK (balance >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- 8.5 credit_transactions
CREATE TABLE IF NOT EXISTS credit_transactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  type text NOT NULL,
  amount int NOT NULL,
  status text NOT NULL,
  reason text NOT NULL,
  external_ref text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_credit_tx_user_created_desc ON credit_transactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_tx_external_ref ON credit_transactions(external_ref);
CREATE UNIQUE INDEX IF NOT EXISTS uq_credit_tx_user_ref_type ON credit_transactions(user_id, external_ref, type);

-- 8.5 stripe_events
CREATE TABLE IF NOT EXISTS stripe_events (
  id text PRIMARY KEY,
  received_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_stripe_events_received_desc ON stripe_events(received_at DESC);

-- 8.5 credit_packages
CREATE TABLE IF NOT EXISTS credit_packages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  credits int NOT NULL,
  stripe_price_id text NOT NULL,
  is_active boolean NOT NULL DEFAULT true
);

-- 8.5 credit_costs
CREATE TABLE IF NOT EXISTS credit_costs (
  feature text PRIMARY KEY,
  cost int NOT NULL,
  is_active boolean NOT NULL DEFAULT true
);

-- 8.4 generation_jobs
CREATE TABLE IF NOT EXISTS generation_jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  external_ref text NOT NULL,
  user_id uuid REFERENCES auth.users(id),
  job_type text NOT NULL,
  input_snapshot jsonb NOT NULL,
  model_name text NOT NULL,
  token_used int,
  cost_estimate numeric,
  status text NOT NULL,
  error_code text,
  error_message text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_user_created_desc ON generation_jobs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_job_type ON generation_jobs(job_type);
