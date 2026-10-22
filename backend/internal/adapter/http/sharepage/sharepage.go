// Package sharepage renders the public ES share page with html/template
// (contextual auto-escaping, IT-SH-010).
// DesignDocument/7_BackendDesign.md 7.5.6 / 9_SecurityDesign.md 共有リンクの安全性
package sharepage

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/domain/entity"
)

//go:embed templates/*.html
var files embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"paragraphs": func(s string) []string { return strings.Split(strings.TrimSpace(s), "\n") },
}).ParseFS(files, "templates/*.html"))

// CSP is the Content-Security-Policy of the share page (DirectionChange 2.10.8).
const CSP = "default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// SetHeaders sets the security headers required on every share-page response.
func SetHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("X-Robots-Tag", "noindex")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Content-Security-Policy", CSP)
	h.Set("X-Content-Type-Options", "nosniff")
}

type pageData struct {
	Title       string
	Message     string
	CompanyName *string
	Sections    []entity.ShareSnapshotSection
	Year        int
}

func render(c *gin.Context, status int, name string, data pageData) {
	data.Year = time.Now().Year()
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		slog.Error("render share page", "request_id", c.GetString("request_id"), "error", err.Error())
		status = http.StatusInternalServerError
		buf.Reset()
		buf.WriteString("<!doctype html><meta charset=\"utf-8\"><title>エラー</title><p>エラーが発生しました。</p>")
	}
	SetHeaders(c)
	c.Status(status)
	_, _ = c.Writer.Write(buf.Bytes())
	c.Abort()
}

// RenderSnapshot renders a valid link (200).
func RenderSnapshot(c *gin.Context, s entity.ShareSnapshot) {
	render(c, http.StatusOK, "share.html", pageData{Title: "ES の共有", CompanyName: s.CompanyName, Sections: s.Sections})
}

// RenderGone renders an expired / revoked link (410).
func RenderGone(c *gin.Context) {
	render(c, http.StatusGone, "message.html", pageData{Title: "リンクは無効です", Message: "このリンクは無効です。共有した人に新しいリンクを依頼してください。"})
}

// RenderNotFound renders an unknown link (404), indistinguishable from 410 for viewers.
func RenderNotFound(c *gin.Context) {
	render(c, http.StatusNotFound, "message.html", pageData{Title: "リンクは無効です", Message: "このリンクは無効です。共有した人に新しいリンクを依頼してください。"})
}

// RenderPaused renders the feature-flag pause page (503).
func RenderPaused(c *gin.Context) {
	render(c, http.StatusServiceUnavailable, "message.html", pageData{Title: "一時停止中", Message: "共有ページは一時停止中です。しばらくしてから再度お試しください。"})
}

// RenderRateLimited renders 429.
func RenderRateLimited(c *gin.Context) {
	render(c, http.StatusTooManyRequests, "message.html", pageData{Title: "アクセスが集中しています", Message: "短時間にアクセスが集中しています。しばらくしてから再度お試しください。"})
}

// RenderError renders 500.
func RenderError(c *gin.Context) {
	render(c, http.StatusInternalServerError, "message.html", pageData{Title: "エラー", Message: "エラーが発生しました。しばらくしてから再度お試しください。"})
}
