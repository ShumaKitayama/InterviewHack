// Integration test harness: real router, use cases and PostgreSQL; the Firebase
// verifier is a stub (11.3). Set TEST_DATABASE_URL to run, e.g.
// TEST_DATABASE_URL=postgres://postgres@localhost:5432/interviewhack_test go test ./...
// DesignDocument/11_TestDesign.md 11.0 / 11.3
package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/middleware"
	"github.com/interviewhack/backend/internal/app"
	"github.com/interviewhack/backend/internal/infra/postgres"
	"github.com/interviewhack/backend/internal/infra/system"
	"github.com/interviewhack/backend/internal/port/external"
)

// stubVerifier accepts tokens of the form "uid:<uid>" and rejects anything else
// (expired / tampered / wrong aud are covered by infra/firebase unit tests).
type stubVerifier struct{}

func (stubVerifier) VerifyToken(_ context.Context, token string) (external.Claims, error) {
	uid, ok := strings.CutPrefix(token, "uid:")
	if !ok || uid == "" {
		return external.Claims{}, errors.New("invalid token")
	}
	email := uid + "@example.com"
	return external.Claims{UID: uid, Email: &email}, nil
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var (
	dbOnce sync.Once
	testDB *postgres.DB
	dbErr  error
)

func openDB(t *testing.T) *postgres.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}
	dbOnce.Do(func() {
		ctx := context.Background()
		testDB, dbErr = postgres.Open(ctx, url)
		if dbErr != nil {
			return
		}
		dbErr = applyMigrations(ctx, testDB)
		if dbErr == nil {
			// IT-DB-008: applying the migrations twice must not fail.
			dbErr = applyMigrations(ctx, testDB)
		}
	})
	if dbErr != nil {
		t.Fatalf("database: %v", dbErr)
	}
	return testDB
}

func applyMigrations(ctx context.Context, db *postgres.DB) error {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil || len(files) == 0 {
		return errors.New("no migrations found")
	}
	sort.Strings(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Pool.Exec(ctx, string(sql)); err != nil {
			return errors.New(filepath.Base(f) + ": " + err.Error())
		}
	}
	return nil
}

const allTables = `users, assets, asset_tags, asset_tag_relations, companies, research_materials, es_projects, es_sections,
  es_section_versions, es_project_asset_relations, es_project_research_material_relations, interviews, interview_logs,
  qa, qa_asset_relations, prompt_sessions, dashboard_summaries, share_links`

type env struct {
	t      *testing.T
	db     *postgres.DB
	router *gin.Engine
	clock  *fakeClock
}

type envOptions struct {
	shareViewOff, shareCreateOff bool
	commitDisabled               []string
	shareLimit                   int
}

func newEnv(t *testing.T, opts ...envOptions) *env {
	t.Helper()
	db := openDB(t)
	if _, err := db.Pool.Exec(context.Background(), `TRUNCATE `+allTables+` CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var o envOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	clock := &fakeClock{now: time.Now().UTC().Truncate(time.Second)}
	limit := o.shareLimit
	if limit == 0 {
		limit = 1000
	}
	r := app.NewRouter(app.Options{
		DB: db, Verifier: stubVerifier{}, Tokens: system.TokenGenerator{}, Clock: clock, IDs: system.IDGenerator{},
		ShareBaseURL: "https://api.example.test", ShareViewEnabled: !o.shareViewOff, ShareCreateEnabled: !o.shareCreateOff,
		CommitDisabledUseCases: o.commitDisabled,
		UserLimiter:            middleware.NewLimiter(100000, time.Minute),
		ShareLimiter:           middleware.NewLimiter(limit, time.Minute),
	})
	return &env{t: t, db: db, router: r, clock: clock}
}

type resp struct {
	status int
	header http.Header
	body   []byte
}

func (r resp) json() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(r.body, &m)
	return m
}

func (r resp) code() string {
	s, _ := r.json()["code"].(string)
	return s
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			e.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = "203.0.113.10:1234"
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{status: w.Code, header: w.Header(), body: w.Body.Bytes()}
}

// must performs a request and fails unless the status matches.
func (e *env) must(status int, method, path, token string, body any) map[string]any {
	e.t.Helper()
	r := e.do(method, path, token, body)
	if r.status != status {
		e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, r.status, status, r.body)
	}
	return r.json()
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.db.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("count %q: %v", sql, err)
	}
	return n
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func items(m map[string]any) []any {
	v, _ := m["items"].([]any)
	return v
}

// seed is the fixed data set of 11.0 for one user.
type seed struct {
	token, companyID, asset1, asset2, materialID, projectID, sectionID, interviewID, logID string
}

func (e *env) seed(uid string) seed {
	tok := "uid:" + uid
	s := seed{token: tok}
	s.companyID = str(e.must(201, "POST", "/companies", tok, map[string]any{
		"name": "株式会社" + uid, "jobTitle": "バックエンドエンジニア", "selectionPhase": "一次面接", "motivationMemo": "開発者体験",
	}), "id")
	s.asset1 = str(e.must(201, "POST", "/assets", tok, map[string]any{"type": "episode", "title": uid + "の学園祭", "content": uid + "の決済システムを構築"}), "id")
	s.asset2 = str(e.must(201, "POST", "/assets", tok, map[string]any{"type": "strength", "title": uid + "の粘り強さ", "content": "障害を特定"}), "id")
	s.materialID = str(e.must(201, "POST", "/research-materials", tok, map[string]any{
		"companyId": s.companyID, "title": "採用ページ", "type": "url", "sourceUrl": "https://example.com", "contentRaw": uid + " は挑戦を歓迎する文化",
	}), "id")
	s.projectID = str(e.must(201, "POST", "/es-projects", tok, map[string]any{"companyId": s.companyID}), "id")
	s.sectionID = str(e.must(201, "POST", "/es-projects/"+s.projectID+"/sections", tok, map[string]any{"title": "志望動機", "characterLimit": 400}), "id")
	s.interviewID = str(e.must(201, "POST", "/interviews", tok, map[string]any{"companyId": s.companyID, "interviewDate": e.clock.Now().AddDate(0, 0, 3).Format("2006-01-02")}), "id")
	s.logID = str(e.must(201, "POST", "/interview-logs", tok, map[string]any{"interviewId": s.interviewID, "questionText": "ガクチカは？", "answerText": "学園祭です"}), "id")
	return s
}
