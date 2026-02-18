-- DesignDocument/8_DBDesign.md 8.3-8.6 RLS Policies
-- Applied via Supabase MCP apply_migration

-- Enable RLS on all tables
ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag_relations ENABLE ROW LEVEL SECURITY;
ALTER TABLE companies ENABLE ROW LEVEL SECURITY;
ALTER TABLE research_materials ENABLE ROW LEVEL SECURITY;
ALTER TABLE es_projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE es_sections ENABLE ROW LEVEL SECURITY;
ALTER TABLE es_project_asset_relations ENABLE ROW LEVEL SECURITY;
ALTER TABLE es_project_research_material_relations ENABLE ROW LEVEL SECURITY;
ALTER TABLE interviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE interview_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE qa ENABLE ROW LEVEL SECURITY;
ALTER TABLE qa_asset_relations ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_balances ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE stripe_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_packages ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_costs ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_jobs ENABLE ROW LEVEL SECURITY;

-- Core domain: auth.uid() = user_id for all operations
-- Billing: SELECT for user, write for service_role only
-- Packages/costs: public read
-- See full policy list in Supabase migration history
