// DesignDocument/7_BackendDesign.md 7.5.3 ドメインデータの CRUD
package app_test

import (
	"testing"
)

func TestCRUD(t *testing.T) {
	e := newEnv(t)
	s := e.seed("u1")
	tok := s.token

	// Tags and assets with filters.
	tag := e.must(201, "POST", "/asset-tags", tok, map[string]any{"name": "ガクチカ"})
	tagID := str(tag, "id")
	if got := e.must(200, "GET", "/asset-tags/"+tagID, tok, nil); str(got, "name") != "ガクチカ" {
		t.Fatalf("tag = %v", got)
	}
	e.must(200, "PATCH", "/asset-tags/"+tagID, tok, map[string]any{"name": "学生時代"})
	a := e.must(200, "PATCH", "/assets/"+s.asset1, tok, map[string]any{"title": "学園祭（改）", "tagIds": []string{tagID}})
	if names := a["tagNames"].([]any); len(names) != 1 || names[0] != "学生時代" {
		t.Fatalf("asset tags = %v", a)
	}
	if n := len(items(e.must(200, "GET", "/assets?tagId="+tagID, tok, nil))); n != 1 {
		t.Fatalf("tag filter = %d", n)
	}
	if n := len(items(e.must(200, "GET", "/assets?type=strength", tok, nil))); n != 1 {
		t.Fatalf("type filter = %d", n)
	}
	if n := len(items(e.must(200, "GET", "/assets?q=改&sort=updated", tok, nil))); n != 1 {
		t.Fatalf("keyword filter = %d", n)
	}
	// Patching other fields keeps the tags.
	a = e.must(200, "PATCH", "/assets/"+s.asset1, tok, map[string]any{"content": "更新"})
	if len(a["tagIds"].([]any)) != 1 {
		t.Fatal("tags must be kept when tagIds is absent")
	}
	e.must(204, "DELETE", "/asset-tags/"+tagID, tok, nil)
	if len(e.must(200, "GET", "/assets/"+s.asset1, tok, nil)["tagIds"].([]any)) != 0 {
		t.Fatal("deleting a tag removes its relations")
	}
	if r := e.do("GET", "/asset-tags/"+tagID, tok, nil); r.status != 404 {
		t.Fatalf("deleted tag: %d", r.status)
	}

	// Companies.
	e.must(200, "PATCH", "/companies/"+s.companyID, tok, map[string]any{"selectionPhase": "最終面接", "referenceUrl": "https://example.com/recruit"})
	if n := len(items(e.must(200, "GET", "/companies?selectionPhase=最終面接&q=u1", tok, nil))); n != 1 {
		t.Fatalf("company filter = %d", n)
	}
	other := str(e.must(201, "POST", "/companies", tok, map[string]any{"name": "消す会社"}), "id")
	e.must(204, "DELETE", "/companies/"+other, tok, nil)
	if r := e.do("DELETE", "/companies/"+other, tok, nil); r.status != 404 {
		t.Fatalf("double delete: %d", r.status)
	}

	// Research materials.
	if n := len(items(e.must(200, "GET", "/research-materials?companyId="+s.companyID, tok, nil))); n != 1 {
		t.Fatalf("materials = %d", n)
	}
	m := e.must(200, "PATCH", "/research-materials/"+s.materialID, tok, map[string]any{"summary": "手で書いた要約", "type": "note"})
	if str(m, "summary") != "手で書いた要約" || str(m, "type") != "note" {
		t.Fatalf("material = %v", m)
	}
	if r := e.do("PATCH", "/research-materials/"+s.materialID, tok, map[string]any{"type": "video"}); r.status != 422 {
		t.Fatalf("bad material type: %d", r.status)
	}
	if r := e.do("GET", "/research-materials", tok, nil); r.status != 400 {
		t.Fatalf("companyId is required: %d", r.status)
	}

	// ES projects and sections.
	e.must(200, "PATCH", "/es-projects/"+s.projectID, tok, map[string]any{"deadline": "2026-12-01", "assetIds": []string{s.asset1}})
	p := e.must(200, "GET", "/es-projects/"+s.projectID, tok, nil)
	if str(p, "deadline") != "2026-12-01" || len(p["assetIds"].([]any)) != 1 {
		t.Fatalf("project = %v", p)
	}
	if n := len(items(e.must(200, "GET", "/es-projects?status=not_started&companyId="+s.companyID, tok, nil))); n != 1 {
		t.Fatalf("es filter = %d", n)
	}
	if r := e.do("PATCH", "/es-projects/"+s.projectID, tok, map[string]any{"deadline": "12/01"}); r.status != 400 {
		t.Fatalf("bad deadline: %d", r.status)
	}
	sec2 := str(e.must(201, "POST", "/es-projects/"+s.projectID+"/sections", tok, map[string]any{"title": "自己PR"}), "id")
	if n := len(items(e.must(200, "GET", "/es-projects/"+s.projectID+"/sections", tok, nil))); n != 2 {
		t.Fatalf("sections = %d", n)
	}
	e.must(200, "PATCH", "/es-sections/"+sec2, tok, map[string]any{"characterLimit": nil})
	e.must(204, "DELETE", "/es-sections/"+sec2, tok, nil)

	// Interviews and logs.
	e.must(200, "PATCH", "/interviews/"+s.interviewID, tok, map[string]any{"result": "pass", "memo": "手応えあり"})
	if r := e.do("PATCH", "/interviews/"+s.interviewID, tok, map[string]any{"result": "maybe"}); r.status != 422 {
		t.Fatalf("bad result: %d", r.status)
	}
	if n := len(items(e.must(200, "GET", "/interviews?companyId="+s.companyID+"&from=2000-01-01&to=2100-01-01", tok, nil))); n != 1 {
		t.Fatalf("interview filter = %d", n)
	}
	if n := len(items(e.must(200, "GET", "/interview-logs?interviewId="+s.interviewID, tok, nil))); n != 1 {
		t.Fatalf("logs = %d", n)
	}
	l := e.must(200, "PATCH", "/interview-logs/"+s.logID, tok, map[string]any{"selfEvaluation": "結論が弱い", "score": 60})
	if str(l, "selfEvaluation") != "結論が弱い" || l["score"].(float64) != 60 {
		t.Fatalf("log = %v", l)
	}
	e.must(200, "GET", "/interview-logs/"+s.logID, tok, nil)
	e.must(204, "DELETE", "/interview-logs/"+s.logID, tok, nil)
	e.must(204, "DELETE", "/interviews/"+s.interviewID, tok, nil)

	// Q&A review flow.
	q := e.must(201, "POST", "/qa", tok, map[string]any{"questionText": "志望動機は？", "answerText": "開発者体験", "companyId": s.companyID})
	qid := str(q, "id")
	if str(q, "reviewStatus") != "new" || str(q, "sourceType") != "pre_generated" {
		t.Fatalf("qa defaults = %v", q)
	}
	e.must(200, "PATCH", "/qa/"+qid, tok, map[string]any{"reviewStatus": "reviewed", "isFavorite": true})
	if n := len(items(e.must(200, "GET", "/qa?isFavorite=true&reviewStatus=reviewed&companyId="+s.companyID+"&q=志望", tok, nil))); n != 1 {
		t.Fatalf("qa filter = %d", n)
	}
	if r := e.do("PATCH", "/qa/"+qid, tok, map[string]any{"reviewStatus": "done"}); r.status != 422 {
		t.Fatalf("bad review status: %d", r.status)
	}
	if r := e.do("GET", "/qa?isFavorite=maybe", tok, nil); r.status != 400 {
		t.Fatalf("bad isFavorite: %d", r.status)
	}
	e.must(200, "GET", "/qa/"+qid, tok, nil)
	e.must(204, "DELETE", "/qa/"+qid, tok, nil)

	// Me and display name clearing.
	if me := e.must(200, "PATCH", "/me", tok, map[string]any{"displayName": ""}); me["displayName"] != nil {
		t.Fatalf("display name must be cleared: %v", me)
	}
	if r := e.do("GET", "/assets?limit=0", tok, nil); r.status != 400 {
		t.Fatalf("bad limit: %d", r.status)
	}
}
