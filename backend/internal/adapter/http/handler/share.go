package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/adapter/http/sharepage"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/usecase/share"
)

// CreateShareLink is POST /es-projects/:id/share-links (7.5.6).
func (h *Handler) CreateShareLink(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.CreateShareRequest
	if !bind(c, &in) || !uuidList(c, "sectionIds", in.SectionIDs) {
		return
	}
	days := 0
	if in.ExpiresInDays != nil {
		days = *in.ExpiresInDays
	}
	res, err := h.Share.Create(c.Request.Context(), userID(c), id, share.CreateInput{SectionIDs: in.SectionIDs, ExpiresInDays: days, ShowCompanyName: in.ShowCompanyName})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	c.Set("share_link_id", res.Link.ID)
	presenter.RespondCreated(c, dto.ShareCreated{ID: res.Link.ID, URL: h.ShareBaseURL + "/s/" + res.Token, ExpiresAt: res.Link.ExpiresAt})
}

// ListShareLinks is GET /es-projects/:id/share-links.
func (h *Handler) ListShareLinks(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	links, err := h.Share.List(c.Request.Context(), userID(c), id)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	items := make([]dto.ShareLink, len(links))
	for i, l := range links {
		items[i] = dto.ShareLinkFrom(l, h.Share.IsActive(l))
	}
	presenter.RespondOK(c, gin.H{"items": items})
}

// RevokeShareLink is DELETE /share-links/:id.
func (h *Handler) RevokeShareLink(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Share.Revoke(c.Request.Context(), userID(c), id))
}

// SharePage is GET /s/:token. It renders HTML for every outcome; the token is
// never logged (the route pattern is logged instead).
func (h *Handler) SharePage(c *gin.Context) {
	l, err := h.Share.View(c.Request.Context(), c.Param("token"))
	if err == nil {
		c.Set("share_link_id", l.ID)
		c.Set("share_result", "ok")
		sharepage.RenderSnapshot(c, l.Snapshot)
		return
	}
	de, _ := domainerror.As(err)
	switch {
	case de != nil && de.Code == vo.ErrorCodeShareLinkGone:
		c.Set("share_link_id", l.ID)
		c.Set("share_result", "gone")
		c.Set(presenter.ErrorCodeKey, de.Code.String())
		sharepage.RenderGone(c)
	case de != nil && de.Code == vo.ErrorCodeFeatureDisabled:
		c.Set(presenter.ErrorCodeKey, de.Code.String())
		sharepage.RenderPaused(c)
	case de != nil && de.Code == vo.ErrorCodeNotFound:
		c.Set("share_result", "not_found")
		sharepage.RenderNotFound(c)
	default:
		c.Set(presenter.ErrorCodeKey, vo.ErrorCodeInternalError.String())
		sharepage.RenderError(c)
	}
}
