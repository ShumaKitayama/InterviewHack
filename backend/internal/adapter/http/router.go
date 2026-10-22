// DesignDocument/7_BackendDesign.md 7.5 API一覧
package http

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/handler"
	"github.com/interviewhack/backend/internal/adapter/http/middleware"
	"github.com/interviewhack/backend/internal/adapter/http/sharepage"
	"github.com/interviewhack/backend/internal/port/external"
)

// RouterDeps are the dependencies of the router.
type RouterDeps struct {
	Handler      *handler.Handler
	AuthVerifier external.AuthVerifier
	Users        middleware.UserResolver
	// Optional limiters (tests inject their own).
	UserLimiter  *middleware.Limiter
	ShareLimiter *middleware.Limiter
}

// NewRouter builds the Gin engine.
func NewRouter(d *RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Cloud Run sets X-Forwarded-For; trust only its front end for ClientIP.
	r.TrustedPlatform = gin.PlatformGoogleAppEngine
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.Logger(), middleware.BodyLimit())
	h := d.Handler

	if d.UserLimiter == nil {
		d.UserLimiter = middleware.NewLimiter(middleware.UserRequestsPerMinute, time.Minute)
	}
	if d.ShareLimiter == nil {
		d.ShareLimiter = middleware.NewLimiter(middleware.IPSharePerMinute, time.Minute)
	}

	r.GET("/health", h.Health)
	r.GET("/s/:token", middleware.RateLimitIP(d.ShareLimiter, func(c *gin.Context, _ time.Duration) {
		sharepage.RenderRateLimited(c)
	}), h.SharePage)

	a := r.Group("")
	a.Use(middleware.Auth(d.AuthVerifier, d.Users), middleware.RateLimitUser(d.UserLimiter))

	a.GET("/me", h.GetMe)
	a.PATCH("/me", h.PatchMe)
	a.DELETE("/me", h.DeleteMe)

	a.GET("/assets", h.ListAssets)
	a.POST("/assets", h.CreateAsset)
	a.GET("/assets/:id", h.GetAsset)
	a.PATCH("/assets/:id", h.PatchAsset)
	a.DELETE("/assets/:id", h.DeleteAsset)

	a.GET("/asset-tags", h.ListTags)
	a.POST("/asset-tags", h.CreateTag)
	a.GET("/asset-tags/:id", h.GetTag)
	a.PATCH("/asset-tags/:id", h.PatchTag)
	a.DELETE("/asset-tags/:id", h.DeleteTag)

	a.GET("/companies", h.ListCompanies)
	a.POST("/companies", h.CreateCompany)
	a.GET("/companies/:id", h.GetCompany)
	a.PATCH("/companies/:id", h.PatchCompany)
	a.DELETE("/companies/:id", h.DeleteCompany)

	a.GET("/research-materials", h.ListResearch)
	a.POST("/research-materials", h.CreateResearch)
	a.GET("/research-materials/:id", h.GetResearch)
	a.PATCH("/research-materials/:id", h.PatchResearch)
	a.DELETE("/research-materials/:id", h.DeleteResearch)

	a.GET("/es-projects", h.ListESProjects)
	a.POST("/es-projects", h.CreateESProject)
	a.GET("/es-projects/:id", h.GetESProject)
	a.PATCH("/es-projects/:id", h.PatchESProject)
	a.DELETE("/es-projects/:id", h.DeleteESProject)
	a.GET("/es-projects/:id/sections", h.ListESSections)
	a.POST("/es-projects/:id/sections", h.CreateESSection)
	a.GET("/es-projects/:id/share-links", h.ListShareLinks)
	a.POST("/es-projects/:id/share-links", h.CreateShareLink)

	a.GET("/es-sections/:id", h.GetESSection)
	a.PATCH("/es-sections/:id", h.PatchESSection)
	a.DELETE("/es-sections/:id", h.DeleteESSection)
	a.GET("/es-sections/:id/versions", h.ListVersions)
	a.POST("/es-sections/:id/versions", h.SaveVersion)
	a.POST("/es-sections/:id/versions/:versionId/restore", h.RestoreVersion)

	a.DELETE("/share-links/:id", h.RevokeShareLink)

	a.GET("/interviews", h.ListInterviews)
	a.POST("/interviews", h.CreateInterview)
	a.GET("/interviews/:id", h.GetInterview)
	a.PATCH("/interviews/:id", h.PatchInterview)
	a.DELETE("/interviews/:id", h.DeleteInterview)

	a.GET("/interview-logs", h.ListInterviewLogs)
	a.POST("/interview-logs", h.CreateInterviewLog)
	a.GET("/interview-logs/:id", h.GetInterviewLog)
	a.PATCH("/interview-logs/:id", h.PatchInterviewLog)
	a.DELETE("/interview-logs/:id", h.DeleteInterviewLog)

	a.GET("/qa", h.ListQA)
	a.POST("/qa", h.CreateQA)
	a.GET("/qa/:id", h.GetQA)
	a.PATCH("/qa/:id", h.PatchQA)
	a.DELETE("/qa/:id", h.DeleteQA)

	a.GET("/dashboard", h.GetDashboard)

	a.POST("/prompt-sessions", h.CreatePromptSession)
	a.GET("/prompt-sessions", h.ListPromptSessions)
	a.GET("/prompt-sessions/:id", h.GetPromptSession)
	a.GET("/prompt-sessions/:id/prompt", h.GetPrompt)
	a.POST("/prompt-sessions/:id/parse", h.ParseResponse)
	a.POST("/prompt-sessions/:id/commit", h.CommitSession)
	a.POST("/prompt-sessions/:id/abandon", h.AbandonSession)

	return r
}
