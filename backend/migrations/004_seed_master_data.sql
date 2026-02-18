-- DesignDocument/6_AuthAccountCahrgingDesign.md 6.4
-- Seed data for credit_packages and credit_costs

INSERT INTO credit_packages (id, name, credits, stripe_price_id, is_active) VALUES
  (gen_random_uuid(), '100 Credits', 100, 'price_placeholder_100', true),
  (gen_random_uuid(), '500 Credits', 500, 'price_placeholder_500', true),
  (gen_random_uuid(), '1000 Credits', 1000, 'price_placeholder_1000', true)
ON CONFLICT DO NOTHING;

INSERT INTO credit_costs (feature, cost, is_active) VALUES
  ('qa_generate', 10, true),
  ('answer_review', 5, true),
  ('summary_generate', 2, true),
  ('research_summarize', 5, true),
  ('es_generate', 10, true),
  ('es_refine', 5, true),
  ('regenerate', 10, true)
ON CONFLICT (feature) DO NOTHING;
