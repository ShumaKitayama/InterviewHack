// Contract fixtures: real API responses written for the iOS client tests, so
// the Swift models are checked against what the BFF actually returns (7.5).
// Run with CONTRACT_OUT=<dir> to (re)write the fixtures.
package app_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContractFixtures(t *testing.T) {
	e := newEnv(t)
	s := e.seed("contract")
	tok := s.token
	out := map[string][]byte{}
	save := func(name string, r resp, want int) map[string]any {
		t.Helper()
		if r.status != want {
			t.Fatalf("%s: status %d, want %d: %s", name, r.status, want, r.body)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, r.body, "", "  "); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out[name] = pretty.Bytes()
		return r.json()
	}

	e.must(200, "PATCH", "/me", tok, map[string]any{"displayName": "テスト"})
	save("me", e.do("GET", "/me", tok, nil), 200)
	tag := e.must(201, "POST", "/asset-tags", tok, map[string]any{"name": "ガクチカ"})
	save("asset_tags", e.do("GET", "/asset-tags", tok, nil), 200)
	save("asset", e.do("PATCH", "/assets/"+s.asset1, tok, map[string]any{"tagIds": []string{str(tag, "id")}}), 200)
	save("assets", e.do("GET", "/assets?limit=1", tok, nil), 200)
	save("company", e.do("GET", "/companies/"+s.companyID, tok, nil), 200)
	save("research_material", e.do("GET", "/research-materials/"+s.materialID, tok, nil), 200)
	save("es_project", e.do("PATCH", "/es-projects/"+s.projectID, tok, map[string]any{"deadline": "2026-12-01", "assetIds": []string{s.asset1}}), 200)
	save("es_sections", e.do("GET", "/es-projects/"+s.projectID+"/sections", tok, nil), 200)
	save("es_section", e.do("PATCH", "/es-sections/"+s.sectionID, tok, map[string]any{"contentDraft": "本文"}), 200)
	save("es_section_version", e.do("POST", "/es-sections/"+s.sectionID+"/versions", tok, nil), 201)
	save("es_section_versions", e.do("GET", "/es-sections/"+s.sectionID+"/versions", tok, nil), 200)
	save("interview", e.do("PATCH", "/interviews/"+s.interviewID, tok, map[string]any{"result": "pending"}), 200)
	save("interview_logs", e.do("GET", "/interview-logs?interviewId="+s.interviewID, tok, nil), 200)
	save("qa", e.do("POST", "/qa", tok, map[string]any{"questionText": "Q", "answerText": "A", "assetIds": []string{s.asset1}}), 201)

	created := save("prompt_created", e.do("POST", "/prompt-sessions", tok, map[string]any{"useCase": "qa_generate", "companyId": s.companyID, "assetIds": []string{s.asset1}}), 201)
	sid := str(created, "sessionId")
	save("prompt_text", e.do("GET", "/prompt-sessions/"+sid+"/prompt", tok, nil), 200)
	save("parse_qa", e.do("POST", "/prompt-sessions/"+sid+"/parse", tok, map[string]any{"responseText": "【形式:qa-v1】\n【Q1】q\n【A1】a\n【意図1】i"}), 200)
	save("prompt_sessions", e.do("GET", "/prompt-sessions?status=created,pasted", tok, nil), 200)
	save("prompt_session", e.do("GET", "/prompt-sessions/"+sid, tok, nil), 200)
	save("commit", e.do("POST", "/prompt-sessions/"+sid+"/commit", tok, map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}), 200)
	save("error_conflict", e.do("POST", "/prompt-sessions/"+sid+"/commit", tok, map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}), 409)

	rv := e.must(201, "POST", "/prompt-sessions", tok, map[string]any{"useCase": "answer_review", "interviewLogId": s.logID})
	save("parse_review", e.do("POST", "/prompt-sessions/"+str(rv, "sessionId")+"/parse", tok, map[string]any{"responseText": "【指摘】\n- x\n【改善案】\n- y\n【改善後の回答】\nz"}), 200)
	e.must(200, "POST", "/prompt-sessions/"+str(rv, "sessionId")+"/commit", tok, map[string]any{"feedbackText": "x", "improvedAnswerText": "z"})
	rs := e.must(201, "POST", "/prompt-sessions", tok, map[string]any{"useCase": "research_summarize", "companyId": s.companyID, "researchMaterialIds": []string{s.materialID}})
	save("parse_research", e.do("POST", "/prompt-sessions/"+str(rs, "sessionId")+"/parse", tok, map[string]any{"responseText": "【魅力・特徴】\n- a\n【求める人物像】\nb\n【最近のトピック】\n- c"}), 200)
	eg := e.must(201, "POST", "/prompt-sessions", tok, map[string]any{"useCase": "es_generate", "esProjectId": s.projectID, "esSectionId": s.sectionID})
	save("parse_text", e.do("POST", "/prompt-sessions/"+str(eg, "sessionId")+"/parse", tok, map[string]any{"responseText": "自由な文章"}), 200)
	save("error_bad_request", e.do("GET", "/companies/not-a-uuid", tok, nil), 400)
	save("abandon", e.do("POST", "/prompt-sessions/"+str(eg, "sessionId")+"/abandon", tok, nil), 200)
	sg := e.must(201, "POST", "/prompt-sessions", tok, map[string]any{"useCase": "summary_generate"})
	e.must(200, "POST", "/prompt-sessions/"+str(sg, "sessionId")+"/parse", tok, map[string]any{"responseText": "- 結論から"})
	e.must(200, "POST", "/prompt-sessions/"+str(sg, "sessionId")+"/commit", tok, map[string]any{"text": "- 結論から"})
	save("dashboard", e.do("GET", "/dashboard", tok, nil), 200)

	save("share_created", e.do("POST", "/es-projects/"+s.projectID+"/share-links", tok, map[string]any{"sectionIds": []string{s.sectionID}, "expiresInDays": 7}), 201)
	save("share_links", e.do("GET", "/es-projects/"+s.projectID+"/share-links", tok, nil), 200)
	save("error_validation", e.do("POST", "/assets", tok, map[string]any{"type": "bad", "title": "t", "content": "c"}), 422)
	save("error_unauthorized", e.do("GET", "/me", "", nil), 401)

	dir := os.Getenv("CONTRACT_OUT")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, b := range out {
		if err := os.WriteFile(filepath.Join(dir, name+".json"), append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
