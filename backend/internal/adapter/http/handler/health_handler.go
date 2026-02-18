// DesignDocument/7_BackendDesign.md 7.5.1 GET /health
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	presenter.RespondOK(c, gin.H{"status": "ok"})
}
