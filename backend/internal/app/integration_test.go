// DesignDocument/11_TestDesign.md 11.3 Integration テスト
package app_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// 11.3.1 認証
func TestAuth(t *testing.T) {
	e := newEnv(t)
	// IT-AUTH-002 / 003 / 004 (invalid tokens are rejected by the verifier)
	for _, tok := range []string{"", "garbage", "uid:"} {
		if r := e.do("GET", "/me", tok, nil); r.status != 401 || r.code() != "UNAUTHORIZED" {
			t.Fatalf("token %q: status %d code %s", tok, r.status, r.code())
		}
	}
	// IT-AUTH-001 / 006
	me1 := e.must(200, "GET", "/me", "uid:u1", nil)
	me2 := e.must(200, "GET", "/me", "uid:u1", nil)
	if str(me1, "id") == "" || str(me1, "id") != str(me2, "id") || str(me1, "email") != "u1@example.com" {
		t.Fatalf("user not stable: %v %v", me1, me2)
	}
	if n := e.count(`SELECT count(*) FROM users WHERE firebase_uid = 'u1'`); n != 1 {
		t.Fatalf("users rows = %d", n)
	}
	// IT-AUTH-005: a user id in the body is ignored.
	u2 := e.must(200, "GET", "/me", "uid:u2", nil)
	a := e.must(201, "POST", "/assets", "uid:u1", map[string]any{"type": "skill", "title": "Go", "content": "3年", "userId": str(u2, "id")})
	if n := e.count(`SELECT count(*) FROM assets WHERE id = $1 AND user_id = $2`, str(a, "id"), str(me1, "id")); n != 1 {
		t.Fatal("asset must belong to the token's user")
	}
	// X-Request-Id is propagated (7.7.3).
	r := e.do("GET", "/health", "", nil)
	if r.status != 200 || r.header.Get("X-Request-Id") == "" {
		t.Fatalf("health: %d %v", r.status, r.header)
	}
	// IT-SH-008: other endpoints need auth.
	if r := e.do("GET", "/dashboard", "", nil); r.status != 401 {
		t.Fatalf("dashboard without auth: %d", r.status)
	}
}

// 11.3.2 user_id 越境防止 (IT-AZ-001〜007)
func TestAuthorization(t *testing.T) {
	e := newEnv(t)
	u1, u2 := e.seed("u1"), e.seed("u2")
	qa := e.must(201, "POST", "/qa", u2.token, map[string]any{"questionText": "Q", "answerText": "A"})

	// Every single-resource endpoint of U2 is 404 for U1 (IT-AZ-001 / 002 / 007).
	for _, p := range []string{
		"/assets/" + u2.asset1, "/companies/" + u2.companyID, "/research-materials/" + u2.materialID,
		"/es-projects/" + u2.projectID, "/es-sections/" + u2.sectionID, "/interviews/" + u2.interviewID,
		"/interview-logs/" + u2.logID, "/qa/" + str(qa, "id"), "/es-projects/" + u2.projectID + "/sections",
		"/es-sections/" + u2.sectionID + "/versions", "/es-projects/" + u2.projectID + "/share-links",
	} {
		if r := e.do("GET", p, u1.token, nil); r.status != 404 {
			t.Fatalf("GET %s by U1: %d", p, r.status)
		}
	}
	if r := e.do("PATCH", "/qa/"+str(qa, "id"), u1.token, map[string]any{"answerText": "x"}); r.status != 404 {
		t.Fatalf("PATCH U2 qa: %d", r.status)
	}
	if r := e.do("DELETE", "/qa/"+str(qa, "id"), u1.token, nil); r.status != 404 {
		t.Fatalf("DELETE U2 qa: %d", r.status)
	}
	// Lists never include U2's rows.
	for _, p := range []string{"/assets", "/companies", "/es-projects", "/interviews", "/qa"} {
		for _, it := range items(e.must(200, "GET", p, u1.token, nil)) {
			id := str(it.(map[string]any), "id")
			for _, other := range []string{u2.asset1, u2.asset2, u2.companyID, u2.projectID, u2.interviewID, str(qa, "id")} {
				if id == other {
					t.Fatalf("%s leaked U2 row %s", p, id)
				}
			}
		}
	}
	if r := e.do("GET", "/research-materials?companyId="+u2.companyID, u1.token, nil); r.status != 404 {
		t.Fatalf("list U2 materials: %d", r.status)
	}
	// Relations to U2's data are rejected.
	if r := e.do("POST", "/qa", u1.token, map[string]any{"questionText": "Q", "answerText": "A", "assetIds": []string{u2.asset1}}); r.status != 404 {
		t.Fatalf("relation to U2 asset: %d", r.status)
	}
	// IT-AZ-006: U2's ids are rejected when creating a prompt; nothing of U2 enters it.
	for _, body := range []map[string]any{
		{"useCase": "qa_generate", "companyId": u2.companyID},
		{"useCase": "qa_generate", "companyId": u1.companyID, "assetIds": []string{u1.asset1, u2.asset1}},
		{"useCase": "qa_generate", "companyId": u1.companyID, "researchMaterialIds": []string{u2.materialID}},
		{"useCase": "answer_review", "interviewLogId": u2.logID},
		{"useCase": "es_generate", "esProjectId": u1.projectID, "esSectionId": u2.sectionID},
	} {
		if r := e.do("POST", "/prompt-sessions", u1.token, body); r.status != 404 {
			t.Fatalf("prompt with U2 data %v: %d %s", body, r.status, r.body)
		}
	}
	// IT-AZ-003: U2's session cannot be touched by U1.
	s := e.must(201, "POST", "/prompt-sessions", u2.token, map[string]any{"useCase": "qa_generate", "companyId": u2.companyID})
	sid := str(s, "sessionId")
	for p, body := range map[string]any{
		"/parse":   map[string]any{"responseText": "【Q1】x\n【A1】y"},
		"/commit":  map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}},
		"/abandon": nil,
	} {
		r := e.do("POST", "/prompt-sessions/"+sid+p, u1.token, body)
		if r.status != 404 || r.code() != "SESSION_NOT_FOUND" {
			t.Fatalf("U1 %s on U2 session: %d %s", p, r.status, r.code())
		}
	}
	// IT-AZ-004 / 005
	if r := e.do("POST", "/es-projects/"+u2.projectID+"/share-links", u1.token, map[string]any{"sectionIds": []string{u2.sectionID}, "expiresInDays": 7}); r.status != 404 {
		t.Fatalf("share U2 project: %d", r.status)
	}
	e.must(200, "PATCH", "/es-sections/"+u2.sectionID, u2.token, map[string]any{"contentDraft": "本文"})
	link := e.must(201, "POST", "/es-projects/"+u2.projectID+"/share-links", u2.token, map[string]any{"sectionIds": []string{u2.sectionID}, "expiresInDays": 7})
	if r := e.do("DELETE", "/share-links/"+str(link, "id"), u1.token, nil); r.status != 404 {
		t.Fatalf("revoke U2 link: %d", r.status)
	}
}

// 11.2.5 入力バリデーション (UT-VAL-001〜006)
func TestValidation(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	cases := []struct {
		id, method, path string
		body             any
		status           int
	}{
		{"UT-VAL-001 path uuid", "GET", "/companies/not-a-uuid", nil, 400},
		{"UT-VAL-001 body uuid", "POST", "/research-materials", map[string]any{"companyId": "bad", "title": "x", "type": "note"}, 400},
		{"malformed json", "POST", "/assets", "{", 400},
		{"UT-VAL-002 empty question", "POST", "/interview-logs", map[string]any{"interviewId": s.interviewID, "questionText": "", "answerText": "a"}, 422},
		{"UT-VAL-003 empty answer", "POST", "/interview-logs", map[string]any{"interviewId": s.interviewID, "questionText": "q", "answerText": ""}, 422},
		{"UT-VAL-005 too long", "POST", "/assets", map[string]any{"type": "skill", "title": "t", "content": strings.Repeat("あ", 20001)}, 422},
		{"bad asset type", "POST", "/assets", map[string]any{"type": "unknown", "title": "t", "content": "c"}, 422},
		{"bad es status", "PATCH", "/es-projects/" + s.projectID, map[string]any{"status": "done"}, 422},
		{"unknown use case", "POST", "/prompt-sessions", map[string]any{"useCase": "chat"}, 400},
		{"missing target", "POST", "/prompt-sessions", map[string]any{"useCase": "qa_generate"}, 400},
		{"bad cursor", "GET", "/assets?cursor=bad*cursor", nil, 400},
	}
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
	}
	cases = append(cases, struct {
		id, method, path string
		body             any
		status           int
	}{"UT-VAL-004 too many ids", "POST", "/qa", map[string]any{"questionText": "q", "answerText": "a", "assetIds": ids}, 422})
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			r := e.do(tc.method, tc.path, s.token, tc.body)
			if r.status != tc.status {
				t.Fatalf("status %d, want %d: %s", r.status, tc.status, r.body)
			}
			if r.code() != "VALIDATION_ERROR" {
				t.Fatalf("code %s", r.code())
			}
		})
	}
	// UT-VAL-006: commit of the wrong shape.
	sess := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	sid := str(sess, "sessionId")
	e.must(200, "POST", "/prompt-sessions/"+sid+"/parse", s.token, map[string]any{"responseText": "【形式:qa-v1】\n【Q1】q\n【A1】a"})
	if r := e.do("POST", "/prompt-sessions/"+sid+"/commit", s.token, map[string]any{"feedbackText": "x", "improvedAnswerText": "y"}); r.status != 422 {
		t.Fatalf("wrong commit shape: %d %s", r.status, r.body)
	}
	if r := e.do("POST", "/prompt-sessions/"+sid+"/commit", s.token, map[string]any{"items": []any{}, "unexpected": 1}); r.status != 400 {
		t.Fatalf("unknown commit field: %d", r.status)
	}
}

// 11.3.3 AI連携
func TestPromptSessions(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	path := func(id, suffix string) string { return "/prompt-sessions/" + id + suffix }

	// IT-PS-001 / 003 / UT-PS-008
	created := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{
		"useCase": "qa_generate", "companyId": s.companyID, "assetIds": []string{s.asset1}, "researchMaterialIds": []string{s.materialID},
		"options": map[string]any{"questionCount": 2},
	})
	sid := str(created, "sessionId")
	prompt := str(created, "promptText")
	if str(created, "templateVersion") != "qa-v1" || !strings.Contains(prompt, "u1の学園祭") || strings.Contains(prompt, "u1の粘り強さ") || !strings.Contains(prompt, "挑戦を歓迎") {
		t.Fatalf("prompt = %s", prompt)
	}
	var refs string
	_ = e.db.Pool.QueryRow(t.Context(), `SELECT input_refs::text FROM prompt_sessions WHERE id = $1`, sid).Scan(&refs)
	if strings.Contains(refs, "決済") || !strings.Contains(refs, s.asset1) || !strings.Contains(refs, "questionCount") {
		t.Fatalf("input_refs must hold ids/options only: %s", refs)
	}
	if again := e.must(200, "GET", path(sid, "/prompt"), s.token, nil); str(again, "promptText") != prompt {
		t.Fatal("IT-PS-003: regenerated prompt differs")
	}
	// IT-PS-015: commit before parse.
	if r := e.do("POST", path(sid, "/commit"), s.token, map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}); r.status != 409 || r.code() != "SESSION_STATE_CONFLICT" {
		t.Fatalf("commit before parse: %d %s", r.status, r.code())
	}
	// IT-PS-004
	response := "はい！\n【形式:qa-v1】\n【Q1】ガクチカは？\n【A1】学園祭の決済です。\n【意図1】主体性\n【Q2】弱みは？\n【A2】慎重さです。\n【意図2】自己理解"
	parsed := e.must(200, "POST", path(sid, "/parse"), s.token, map[string]any{"responseText": response})
	preview := parsed["preview"].(map[string]any)
	if str(parsed, "parseResult") != "success" || len(preview["items"].([]any)) != 2 {
		t.Fatalf("parse = %v", parsed)
	}
	if e.count(`SELECT count(*) FROM qa`) != 0 {
		t.Fatal("parse must not save anything")
	}
	// IT-PS-002: unfinished sessions are listed.
	list := e.must(200, "GET", "/prompt-sessions?status=created,pasted", s.token, nil)
	if len(items(list)) != 1 || str(items(list)[0].(map[string]any), "status") != "pasted" || str(items(list)[0].(map[string]any), "targetLabel") != "株式会社u1" {
		t.Fatalf("pending list = %v", list)
	}
	// IT-PS-005 / 016
	commit := map[string]any{"items": []any{
		map[string]any{"questionText": "ガクチカは？", "answerText": "学園祭の決済です。（修正）"},
		map[string]any{"questionText": "弱みは？", "answerText": "慎重さです。"},
	}}
	res := e.must(200, "POST", path(sid, "/commit"), s.token, commit)
	if str(res, "status") != "saved" || e.count(`SELECT count(*) FROM qa WHERE source_type = 'pre_generated' AND company_id = $1`, s.companyID) != 2 {
		t.Fatalf("commit = %v", res)
	}
	if e.count(`SELECT count(*) FROM qa_asset_relations WHERE asset_id = $1`, s.asset1) != 2 {
		t.Fatal("assets used for the prompt must be linked to the saved Q&A")
	}
	if e.count(`SELECT count(*) FROM qa WHERE question_text LIKE '%主体性%' OR answer_text LIKE '%主体性%'`) != 0 {
		t.Fatal("IT-PS-016: intent must not be saved")
	}
	// IT-PS-010: second commit conflicts and saves nothing.
	if r := e.do("POST", path(sid, "/commit"), s.token, commit); r.status != 409 || r.code() != "SESSION_STATE_CONFLICT" {
		t.Fatalf("second commit: %d", r.status)
	}
	if e.count(`SELECT count(*) FROM qa`) != 2 {
		t.Fatal("second commit must not duplicate")
	}
	// After a 409 the client can read the session and see saved (7.6.1).
	if got := e.must(200, "GET", path(sid, ""), s.token, nil); str(got, "status") != "saved" {
		t.Fatalf("session = %v", got)
	}
	// IT-PS-014: neither the prompt nor the raw response is stored anywhere.
	if e.count(`SELECT count(*) FROM prompt_sessions WHERE input_refs::text LIKE '%学園祭%' OR input_refs::text LIKE '%慎重さ%'`) != 0 {
		t.Fatal("raw text stored in prompt_sessions")
	}

	// IT-PS-006: answer_review
	rv := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "answer_review", "interviewLogId": s.logID, "options": map[string]any{"strictness": "hard"}})
	rid := str(rv, "sessionId")
	p := e.must(200, "POST", path(rid, "/parse"), s.token, map[string]any{"responseText": "【形式:review-v1】\n【指摘】\n- 具体性：数字がない\n【改善案】\n- 数字を入れる"})
	if str(p, "parseResult") != "partial" || len(p["warnings"].([]any)) == 0 {
		t.Fatalf("partial review = %v", p)
	}
	e.must(200, "POST", path(rid, "/commit"), s.token, map[string]any{"feedbackText": "具体性：数字がない", "improvedAnswerText": "3,000人の決済を構築", "saveToQa": true})
	if e.count(`SELECT count(*) FROM interview_logs WHERE id = $1 AND ai_feedback_text = '具体性：数字がない' AND ai_improved_answer_text = '3,000人の決済を構築'`, s.logID) != 1 ||
		e.count(`SELECT count(*) FROM qa WHERE source_type = 'review_improved'`) != 1 {
		t.Fatal("answer_review commit not saved")
	}

	// IT-PS-007: research_summarize
	rs := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "research_summarize", "companyId": s.companyID, "researchMaterialIds": []string{s.materialID}})
	rsid := str(rs, "sessionId")
	rp := e.must(200, "POST", path(rsid, "/parse"), s.token, map[string]any{"responseText": "【形式:research-v1】\n【魅力・特徴】\n- 挑戦\n【求める人物像】\n主体的\n【最近のトピック】\n- 新サービス"})
	summary := str(rp["preview"].(map[string]any), "summaryText")
	e.must(200, "POST", path(rsid, "/commit"), s.token, map[string]any{"summaryText": summary})
	if e.count(`SELECT count(*) FROM research_materials WHERE id = $1 AND summary LIKE '%求める人物像%'`, s.materialID) != 1 {
		t.Fatal("research summary not saved")
	}

	// IT-PS-008: summary_generate → dashboard
	sg := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "summary_generate"})
	sgid := str(sg, "sessionId")
	e.must(200, "POST", path(sgid, "/parse"), s.token, map[string]any{"responseText": "【形式:summary-v1】\n- 結論から話す"})
	e.must(200, "POST", path(sgid, "/commit"), s.token, map[string]any{"text": "- 結論から話す"})
	dash := e.must(200, "GET", "/dashboard", s.token, nil)
	latest, _ := dash["latestSummary"].(map[string]any)
	if latest == nil || str(latest, "content") != "- 結論から話す" || len(dash["upcomingInterviews"].([]any)) != 1 {
		t.Fatalf("dashboard = %v", dash)
	}

	// IT-PS-009 / 017 / IT-VER-001: es_generate then es_refine
	if st := str(e.must(200, "GET", "/es-projects/"+s.projectID, s.token, nil), "status"); st != "not_started" {
		t.Fatalf("IT-PS-018: initial status %s", st)
	}
	eg := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "es_generate", "esProjectId": s.projectID, "esSectionId": s.sectionID, "assetIds": []string{s.asset2}})
	egid := str(eg, "sessionId")
	e.must(200, "POST", path(egid, "/parse"), s.token, map[string]any{"responseText": "【形式:esdraft-v1】\n最初のドラフト"})
	e.must(200, "POST", path(egid, "/commit"), s.token, map[string]any{"text": "最初のドラフト"})
	sec := e.must(200, "GET", "/es-sections/"+s.sectionID, s.token, nil)
	proj := e.must(200, "GET", "/es-projects/"+s.projectID, s.token, nil)
	if str(sec, "contentDraft") != "最初のドラフト" || str(proj, "status") != "draft" || e.count(`SELECT count(*) FROM es_section_versions`) != 0 {
		t.Fatalf("es_generate: section %v project %v", sec, proj)
	}
	if ids, _ := proj["assetIds"].([]any); len(ids) != 1 || ids[0] != s.asset2 {
		t.Fatalf("draft assets must be linked to the project: %v", proj["assetIds"])
	}
	e.must(200, "PATCH", "/es-projects/"+s.projectID, s.token, map[string]any{"status": "in_review"})
	er := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "es_refine", "esProjectId": s.projectID, "esSectionId": s.sectionID, "options": map[string]any{"instructions": "具体的に"}})
	if !strings.Contains(str(er, "promptText"), "最初のドラフト") {
		t.Fatal("refine prompt must include the current draft")
	}
	erid := str(er, "sessionId")
	e.must(200, "POST", path(erid, "/parse"), s.token, map[string]any{"responseText": "【形式:esrefine-v1】\n推敲後"})
	e.must(200, "POST", path(erid, "/commit"), s.token, map[string]any{"text": "推敲後"})
	if e.count(`SELECT count(*) FROM es_section_versions WHERE es_section_id = $1 AND source = 'es_refine' AND content = '最初のドラフト'`, s.sectionID) != 1 {
		t.Fatal("IT-VER-001: previous draft must be kept as an es_refine version")
	}
	if st := str(e.must(200, "GET", "/es-projects/"+s.projectID, s.token, nil), "status"); st != "in_review" {
		t.Fatalf("IT-PS-017: status must stay in_review, got %s", st)
	}
	// IT-PS-018: any status may be set.
	e.must(200, "PATCH", "/es-projects/"+s.projectID, s.token, map[string]any{"status": "submitted"})
	if st := str(e.must(200, "PATCH", "/es-projects/"+s.projectID, s.token, map[string]any{"status": "draft"}), "status"); st != "draft" {
		t.Fatalf("status back to draft: %s", st)
	}

	// IT-PS-012 / 013
	ab := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	abid := str(ab, "sessionId")
	e.must(200, "POST", path(abid, "/abandon"), s.token, nil)
	for _, req := range []struct {
		method, suffix string
		body           any
	}{
		{"POST", "/parse", map[string]any{"responseText": "x"}},
		{"POST", "/commit", map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}},
		{"POST", "/abandon", nil},
		{"GET", "/prompt", nil},
	} {
		if r := e.do(req.method, path(abid, req.suffix), s.token, req.body); r.status != 409 {
			t.Fatalf("%s after abandon: %d", req.suffix, r.status)
		}
	}
	if r := e.do("POST", path("00000000-0000-0000-0000-000000000000", "/parse"), s.token, map[string]any{"responseText": "x"}); r.status != 404 || r.code() != "SESSION_NOT_FOUND" {
		t.Fatalf("unknown session: %d %s", r.status, r.code())
	}
	// IT-PS-002: sessions older than 7 days are abandoned when listed.
	old := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	if _, err := e.db.Pool.Exec(t.Context(), `UPDATE prompt_sessions SET created_at = $2 WHERE id = $1`, str(old, "sessionId"), e.clock.Now().Add(-7*24*time.Hour-time.Minute)); err != nil {
		t.Fatal(err)
	}
	fresh := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	pending := items(e.must(200, "GET", "/prompt-sessions?status=created,pasted", s.token, nil))
	if len(pending) != 1 || str(pending[0].(map[string]any), "sessionId") != str(fresh, "sessionId") {
		t.Fatalf("pending = %v", pending)
	}
	if e.count(`SELECT count(*) FROM prompt_sessions WHERE id = $1 AND status = 'abandoned' AND completed_at IS NOT NULL`, str(old, "sessionId")) != 1 {
		t.Fatal("old session must be abandoned")
	}
}

// IT-PS-011: concurrent commits save exactly once.
func TestConcurrentCommit(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	sess := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	sid := str(sess, "sessionId")
	e.must(200, "POST", "/prompt-sessions/"+sid+"/parse", s.token, map[string]any{"responseText": "【Q1】q\n【A1】a"})
	var wg sync.WaitGroup
	statuses := make([]int, 8)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = e.do("POST", "/prompt-sessions/"+sid+"/commit", s.token, map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}).status
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, st := range statuses {
		switch st {
		case 200:
			ok++
		case 409:
		default:
			t.Fatalf("unexpected status %d", st)
		}
	}
	if ok != 1 || e.count(`SELECT count(*) FROM qa`) != 1 {
		t.Fatalf("successes = %d, qa rows = %d", ok, e.count(`SELECT count(*) FROM qa`))
	}
}

// IT-PS-019 / IT-SH-012: feature flags.
func TestFeatureFlags(t *testing.T) {
	e := newEnv(t, envOptions{commitDisabled: []string{"qa_generate"}, shareCreateOff: true})
	s := e.seed("u1")
	sess := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
	sid := str(sess, "sessionId")
	e.must(200, "POST", "/prompt-sessions/"+sid+"/parse", s.token, map[string]any{"responseText": "【Q1】q\n【A1】a"})
	if r := e.do("POST", "/prompt-sessions/"+sid+"/commit", s.token, map[string]any{"items": []any{map[string]any{"questionText": "q", "answerText": "a"}}}); r.status != 503 || r.code() != "FEATURE_DISABLED" {
		t.Fatalf("disabled commit: %d %s", r.status, r.code())
	}
	if e.count(`SELECT count(*) FROM prompt_sessions WHERE id = $1 AND status = 'pasted'`, sid) != 1 {
		t.Fatal("session must stay pasted")
	}
	eg := e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "es_generate", "esProjectId": s.projectID, "esSectionId": s.sectionID})
	e.must(200, "POST", "/prompt-sessions/"+str(eg, "sessionId")+"/parse", s.token, map[string]any{"responseText": "本文"})
	e.must(200, "POST", "/prompt-sessions/"+str(eg, "sessionId")+"/commit", s.token, map[string]any{"text": "本文"})
	if r := e.do("POST", "/es-projects/"+s.projectID+"/share-links", s.token, map[string]any{"sectionIds": []string{s.sectionID}, "expiresInDays": 7}); r.status != 503 || r.code() != "FEATURE_DISABLED" {
		t.Fatalf("disabled share create: %d %s", r.status, r.code())
	}

	v := newEnv(t, envOptions{shareViewOff: true})
	r := v.do("GET", "/s/anything", "", nil)
	if r.status != 503 || !strings.Contains(string(r.body), "一時停止中") || r.header.Get("X-Robots-Tag") != "noindex" {
		t.Fatalf("paused share page: %d %s", r.status, r.body)
	}
}

func shareToken(t *testing.T, url string) string {
	t.Helper()
	i := strings.Index(url, "/s/")
	if i < 0 {
		t.Fatalf("url %s", url)
	}
	return url[i+3:]
}

func checkShareHeaders(t *testing.T, r resp) {
	t.Helper()
	want := map[string]string{
		"X-Robots-Tag":            "noindex",
		"Cache-Control":           "no-store",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
		"Content-Type":            "text/html; charset=utf-8",
	}
	for k, v := range want {
		if got := r.header.Get(k); got != v {
			t.Fatalf("status %d header %s = %q, want %q", r.status, k, got, v)
		}
	}
	if r.header.Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id missing on share page")
	}
}

// 11.3.4 共有リンク・共有ページ
func TestShareLinks(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	e.must(200, "PATCH", "/me", s.token, map[string]any{"displayName": "山田太郎"})
	e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": "<script>alert(1)</script>\n二行目", "contentFinal": ""})
	hidden := str(e.must(201, "POST", "/es-projects/"+s.projectID+"/sections", s.token, map[string]any{"title": "非公開の自己PR", "contentDraft": "見せない"}), "id")

	// UT-SH-003 via HTTP
	if r := e.do("POST", "/es-projects/"+s.projectID+"/share-links", s.token, map[string]any{"sectionIds": []string{s.sectionID}, "expiresInDays": 2}); r.status != 422 {
		t.Fatalf("expiresInDays=2: %d", r.status)
	}
	// IT-SH-001
	created := e.must(201, "POST", "/es-projects/"+s.projectID+"/share-links", s.token, map[string]any{"sectionIds": []string{s.sectionID}, "expiresInDays": 7})
	url := str(created, "url")
	tok := shareToken(t, url)
	if !strings.HasPrefix(url, "https://api.example.test/s/") || e.count(`SELECT count(*) FROM share_links WHERE token_hash = $1`, tok) != 0 ||
		e.count(`SELECT count(*) FROM share_links WHERE show_company_name = false`) != 1 {
		t.Fatal("only the token hash may be stored; showCompanyName defaults to false")
	}
	// IT-SH-002 / 006 / 010
	r := e.do("GET", "/s/"+tok, "", nil)
	if r.status != 200 {
		t.Fatalf("share page: %d", r.status)
	}
	checkShareHeaders(t, r)
	page := string(r.body)
	if strings.Contains(page, "<script>") || !strings.Contains(page, "&lt;script&gt;") {
		t.Fatal("IT-SH-010: body must be escaped")
	}
	if strings.Contains(page, "山田太郎") || strings.Contains(page, "株式会社u1") || strings.Contains(page, "見せない") || !strings.Contains(page, `name="robots" content="noindex"`) {
		t.Fatal("IT-SH-006: user name / company / unselected section must not appear")
	}
	// IT-SH-007: later edits do not change the snapshot.
	e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": "編集後"})
	if strings.Contains(string(e.do("GET", "/s/"+tok, "", nil).body), "編集後") {
		t.Fatal("snapshot must not change")
	}
	// Company name only when requested; content_final wins.
	e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentFinal": "完成版"})
	withCo := e.must(201, "POST", "/es-projects/"+s.projectID+"/share-links", s.token, map[string]any{"sectionIds": []string{s.sectionID, hidden}, "expiresInDays": 1, "showCompanyName": true})
	coPage := string(e.do("GET", "/s/"+shareToken(t, str(withCo, "url")), "", nil).body)
	if !strings.Contains(coPage, "株式会社u1") || !strings.Contains(coPage, "完成版") || !strings.Contains(coPage, "見せない") {
		t.Fatal("company name / content_final missing")
	}
	// List does not expose URLs.
	list := e.must(200, "GET", "/es-projects/"+s.projectID+"/share-links", s.token, nil)
	if len(items(list)) != 2 || strings.Contains(fmt.Sprint(list), "/s/") {
		t.Fatalf("list = %v", list)
	}
	// IT-SH-005
	nf := e.do("GET", "/s/unknown-token", "", nil)
	if nf.status != 404 {
		t.Fatalf("unknown token: %d", nf.status)
	}
	checkShareHeaders(t, nf)
	// IT-SH-004: revoked
	e.must(204, "DELETE", "/share-links/"+str(created, "id"), s.token, nil)
	e.must(204, "DELETE", "/share-links/"+str(created, "id"), s.token, nil)
	gone := e.do("GET", "/s/"+tok, "", nil)
	if gone.status != http.StatusGone {
		t.Fatalf("revoked: %d", gone.status)
	}
	checkShareHeaders(t, gone)
	// IT-SH-003: expired
	e.clock.Advance(24*time.Hour + time.Second)
	if r := e.do("GET", "/s/"+shareToken(t, str(withCo, "url")), "", nil); r.status != http.StatusGone {
		t.Fatalf("expired: %d", r.status)
	}
	// IT-SH-011: deleting the project deletes its links.
	e.must(204, "DELETE", "/es-projects/"+s.projectID, s.token, nil)
	if e.count(`SELECT count(*) FROM share_links`) != 0 {
		t.Fatal("share links must be deleted with the project")
	}
	if r := e.do("GET", "/s/"+tok, "", nil); r.status != 404 {
		t.Fatalf("after project deletion: %d", r.status)
	}
}

// IT-SH-009: per-IP rate limit on the share page.
func TestShareRateLimit(t *testing.T) {
	e := newEnv(t, envOptions{shareLimit: 3})
	for i := 0; i < 3; i++ {
		if r := e.do("GET", "/s/none", "", nil); r.status != 404 {
			t.Fatalf("request %d: %d", i, r.status)
		}
	}
	r := e.do("GET", "/s/none", "", nil)
	if r.status != http.StatusTooManyRequests || r.header.Get("Retry-After") == "" {
		t.Fatalf("4th request: %d", r.status)
	}
	checkShareHeaders(t, r)
}

// 11.3.6.1 ESセクションの版
func TestSectionVersions(t *testing.T) {
	e := newEnv(t)
	s, other := e.seed("u1"), e.seed("u2")
	v := "/es-sections/" + s.sectionID + "/versions"
	// IT-VER-003: empty body cannot be versioned.
	if r := e.do("POST", v, s.token, nil); r.status != 422 {
		t.Fatalf("empty version: %d", r.status)
	}
	// IT-VER-002: PATCH does not create versions but bumps version.
	sec := e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": "版1"})
	if sec["version"].(float64) != 2 || e.count(`SELECT count(*) FROM es_section_versions`) != 0 {
		t.Fatalf("patch: %v", sec)
	}
	if st := str(e.must(200, "GET", "/es-projects/"+s.projectID, s.token, nil), "status"); st != "draft" {
		t.Fatalf("manual body save must move not_started → draft, got %s", st)
	}
	saved := e.must(201, "POST", v, s.token, nil)
	if str(saved, "source") != "manual" || str(saved, "content") != "版1" {
		t.Fatalf("saved = %v", saved)
	}
	// IT-VER-004: restore keeps the current body and bumps version.
	e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": "版2"})
	restored := e.must(200, "POST", v+"/"+str(saved, "id")+"/restore", s.token, nil)
	if str(restored, "contentDraft") != "版1" || restored["version"].(float64) != 4 {
		t.Fatalf("restored = %v", restored)
	}
	list := items(e.must(200, "GET", v, s.token, nil))
	if len(list) != 2 || str(list[0].(map[string]any), "content") != "版2" || str(list[0].(map[string]any), "source") != "manual" {
		t.Fatalf("versions = %v", list)
	}
	// IT-VER-005: at most 20 versions are kept.
	for i := 0; i < 25; i++ {
		e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": fmt.Sprintf("本文%02d", i)})
		e.must(201, "POST", v, s.token, nil)
	}
	list = items(e.must(200, "GET", v, s.token, nil))
	if len(list) != 20 || str(list[0].(map[string]any), "content") != "本文24" || str(list[19].(map[string]any), "content") != "本文05" {
		t.Fatalf("versions after 27 saves: %d first=%v", len(list), list[0])
	}
	// IT-VER-006
	for _, req := range []struct{ m, p string }{{"GET", v}, {"POST", v}, {"POST", v + "/" + str(saved, "id") + "/restore"}} {
		if r := e.do(req.m, req.p, other.token, nil); r.status != 404 {
			t.Fatalf("%s %s by other user: %d", req.m, req.p, r.status)
		}
	}
	// IT-DB-008: deleting the section deletes its versions.
	e.must(204, "DELETE", "/es-sections/"+s.sectionID, s.token, nil)
	if e.count(`SELECT count(*) FROM es_section_versions WHERE es_section_id = $1`, s.sectionID) != 0 {
		t.Fatal("versions must be deleted with the section")
	}
}

// 11.3.5 退会
func TestAccountDeletion(t *testing.T) {
	e := newEnv(t)
	u1, u2 := e.seed("u1"), e.seed("u2")
	for _, s := range []seed{u1, u2} {
		e.must(200, "PATCH", "/es-sections/"+s.sectionID, s.token, map[string]any{"contentDraft": "本文"})
		e.must(201, "POST", "/es-sections/"+s.sectionID+"/versions", s.token, nil)
		tag := e.must(201, "POST", "/asset-tags", s.token, map[string]any{"name": "タグ"})
		e.must(200, "PATCH", "/assets/"+s.asset1, s.token, map[string]any{"tagIds": []string{str(tag, "id")}})
		e.must(201, "POST", "/qa", s.token, map[string]any{"questionText": "q", "answerText": "a", "assetIds": []string{s.asset1}})
		e.must(201, "POST", "/prompt-sessions", s.token, map[string]any{"useCase": "qa_generate", "companyId": s.companyID})
		e.must(200, "PATCH", "/es-projects/"+s.projectID, s.token, map[string]any{"assetIds": []string{s.asset1}, "researchMaterialIds": []string{s.materialID}})
	}
	link := e.must(201, "POST", "/es-projects/"+u1.projectID+"/share-links", u1.token, map[string]any{"sectionIds": []string{u1.sectionID}, "expiresInDays": 7})
	e.must(200, "PATCH", "/me", u1.token, map[string]any{"displayName": "退会する人"})
	var u1ID string
	_ = e.db.Pool.QueryRow(t.Context(), `SELECT id FROM users WHERE firebase_uid = 'u1'`).Scan(&u1ID)

	// IT-ACC-001
	e.must(204, "DELETE", "/me", u1.token, nil)
	for _, table := range []string{"assets", "asset_tags", "companies", "research_materials", "es_projects", "es_sections", "es_section_versions", "interviews", "interview_logs", "qa", "prompt_sessions", "share_links", "dashboard_summaries"} {
		if n := e.count(`SELECT count(*) FROM `+table+` WHERE user_id = $1`, u1ID); n != 0 {
			t.Fatalf("%s still has %d rows of U1", table, n)
		}
	}
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1 AND deleted_at IS NOT NULL AND email IS NULL AND display_name IS NULL`, u1ID); n != 1 {
		t.Fatal("users row must remain with PII cleared")
	}
	// IT-ACC-002
	if r := e.do("GET", "/s/"+shareToken(t, str(link, "url")), "", nil); r.status != 404 {
		t.Fatalf("share link after deletion: %d", r.status)
	}
	// IT-ACC-004 / IT-AUTH-008
	if r := e.do("GET", "/me", u1.token, nil); r.status != 401 {
		t.Fatalf("old token after deletion: %d", r.status)
	}
	if n := e.count(`SELECT count(*) FROM users WHERE firebase_uid = 'u1'`); n != 1 {
		t.Fatal("no new users row may be created")
	}
	if n := e.count(`SELECT count(*) FROM users WHERE firebase_uid = 'u1' AND email IS NULL`); n != 1 {
		t.Fatal("email must not be re-synced for a deleted user")
	}
	// IT-ACC-003
	for _, p := range []string{"/assets/" + u2.asset1, "/companies/" + u2.companyID, "/es-sections/" + u2.sectionID} {
		e.must(200, "GET", p, u2.token, nil)
	}
	if e.count(`SELECT count(*) FROM es_section_versions`) != 1 || e.count(`SELECT count(*) FROM qa_asset_relations`) != 1 {
		t.Fatal("U2 data must be untouched")
	}
}

// IT-ACC-005: a failure in the middle of deletion rolls everything back.
func TestAccountDeletionIsAtomic(t *testing.T) {
	e := newEnv(t)
	u1 := e.seed("u1")
	var u1ID string
	_ = e.db.Pool.QueryRow(t.Context(), `SELECT id FROM users WHERE firebase_uid = 'u1'`).Scan(&u1ID)
	// Make the final users UPDATE fail with a trigger, after the deletes ran.
	ctx := t.Context()
	if _, err := e.db.Pool.Exec(ctx, `CREATE OR REPLACE FUNCTION fail_user_update() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'boom'; END $$ LANGUAGE plpgsql;
		CREATE TRIGGER fail_user_update BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION fail_user_update();`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := e.db.Pool.Exec(ctx, `DROP TRIGGER IF EXISTS fail_user_update ON users; DROP FUNCTION IF EXISTS fail_user_update();`); err != nil {
			t.Error(err)
		}
	}()
	if r := e.do("DELETE", "/me", u1.token, nil); r.status != 500 || r.code() != "INTERNAL_ERROR" || strings.Contains(string(r.body), "boom") {
		t.Fatalf("failing deletion: %d %s", r.status, r.body)
	}
	if e.count(`SELECT count(*) FROM assets WHERE user_id = $1`, u1ID) != 2 || e.count(`SELECT count(*) FROM es_sections WHERE user_id = $1`, u1ID) != 1 {
		t.Fatal("nothing may be deleted when deletion fails")
	}
}

// 11.3.6 DB整合性
func TestDBIntegrity(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	var uid string
	_ = e.db.Pool.QueryRow(t.Context(), `SELECT id FROM users WHERE firebase_uid = 'u1'`).Scan(&uid)
	missing := "00000000-0000-0000-0000-000000000001"
	stmts := map[string]string{
		"IT-DB-001": `INSERT INTO interviews (id, user_id, company_id, interview_date) VALUES (gen_random_uuid(), '` + uid + `', '` + missing + `', now())`,
		"IT-DB-002": `INSERT INTO interview_logs (id, interview_id, user_id, question_text, answer_text) VALUES (gen_random_uuid(), '` + missing + `', '` + uid + `', 'q', 'a')`,
		"IT-DB-003": `INSERT INTO qa_asset_relations (qa_id, asset_id) VALUES ('` + missing + `', '` + s.asset1 + `')`,
		"IT-DB-004": `INSERT INTO es_sections (id, es_project_id, user_id, title) VALUES (gen_random_uuid(), '` + missing + `', '` + uid + `', 't')`,
		"IT-DB-005": `INSERT INTO es_project_asset_relations (es_project_id, asset_id) VALUES ('` + s.projectID + `', '` + missing + `')`,
		"IT-DB-006": `INSERT INTO es_project_research_material_relations (es_project_id, research_material_id) VALUES ('` + missing + `', '` + s.materialID + `')`,
		"IT-DB-007": `INSERT INTO share_links (id, user_id, es_project_id, token_hash, snapshot, expires_at) VALUES (gen_random_uuid(), '` + uid + `', '` + missing + `', 'h', '{}', now())`,
	}
	for id, sql := range stmts {
		if _, err := e.db.Pool.Exec(t.Context(), sql); err == nil {
			t.Fatalf("%s: insert with a missing reference must fail", id)
		}
	}
	// Soft-deleted assets/companies disappear from lists and lookups.
	e.must(204, "DELETE", "/assets/"+s.asset1, s.token, nil)
	if r := e.do("GET", "/assets/"+s.asset1, s.token, nil); r.status != 404 {
		t.Fatalf("soft-deleted asset: %d", r.status)
	}
	if e.count(`SELECT count(*) FROM assets WHERE id = $1 AND deleted_at IS NOT NULL`, s.asset1) != 1 {
		t.Fatal("asset must be soft-deleted")
	}
	// Tag names are unique per user (409).
	e.must(201, "POST", "/asset-tags", s.token, map[string]any{"name": "Go"})
	if r := e.do("POST", "/asset-tags", s.token, map[string]any{"name": "Go"}); r.status != 409 || r.code() != "CONFLICT" {
		t.Fatalf("duplicate tag: %d", r.status)
	}
	// Pagination returns every row exactly once.
	for i := 0; i < 5; i++ {
		e.must(201, "POST", "/qa", s.token, map[string]any{"questionText": fmt.Sprintf("q%d", i), "answerText": "a", "isFavorite": i == 3})
	}
	seen := map[string]bool{}
	cursor := ""
	first := ""
	for page := 0; page < 10; page++ {
		p := "/qa?limit=2"
		if cursor != "" {
			p += "&cursor=" + cursor
		}
		res := e.must(200, "GET", p, s.token, nil)
		for _, it := range items(res) {
			m := it.(map[string]any)
			if first == "" {
				first = str(m, "questionText")
			}
			if seen[str(m, "id")] {
				t.Fatal("duplicate row across pages")
			}
			seen[str(m, "id")] = true
		}
		next, _ := res["nextCursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 5 || first != "q3" {
		t.Fatalf("pagination saw %d rows, first %s (favorites first)", len(seen), first)
	}
}
