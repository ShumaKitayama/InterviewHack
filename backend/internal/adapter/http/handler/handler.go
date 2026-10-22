// Package handler implements the Gin handlers of 7.5. Handlers only decode,
// validate the format (400), call use cases and present results.
// DesignDocument/7_BackendDesign.md 7.2.1 / 7.5
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/middleware"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/account"
	"github.com/interviewhack/backend/internal/usecase/asset"
	"github.com/interviewhack/backend/internal/usecase/company"
	"github.com/interviewhack/backend/internal/usecase/dashboard"
	"github.com/interviewhack/backend/internal/usecase/es"
	"github.com/interviewhack/backend/internal/usecase/interview"
	"github.com/interviewhack/backend/internal/usecase/prompt"
	"github.com/interviewhack/backend/internal/usecase/qa"
	"github.com/interviewhack/backend/internal/usecase/research"
	"github.com/interviewhack/backend/internal/usecase/share"
)

// Handler holds the use cases.
type Handler struct {
	Account      *account.Service
	Assets       *asset.Service
	Companies    *company.Service
	Research     *research.Service
	ES           *es.Service
	Interviews   *interview.Service
	QA           *qa.Service
	Dashboard    *dashboard.Service
	Prompts      *prompt.Service
	Share        *share.Service
	ShareBaseURL string
}

func userID(c *gin.Context) string { return c.GetString(middleware.UserIDKey) }

// bind decodes a JSON body. Malformed JSON is 400. Unknown fields such as a
// client-supplied userId are ignored (IT-AUTH-005).
func bind(c *gin.Context, v any) bool { return decode(c, v, false) }

// bindStrict also rejects unknown fields, so a commit payload of the wrong
// use case is detected (UT-VAL-006).
func bindStrict(c *gin.Context, v any) bool { return decode(c, v, true) }

func decode(c *gin.Context, v any, strict bool) bool {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			presenter.RespondError(c, domainerror.ErrUnprocessable("リクエストが大きすぎます"))
			return false
		}
		presenter.RespondError(c, domainerror.ErrValidation("リクエストを読み込めません"))
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(v); err != nil {
		presenter.RespondError(c, domainerror.ErrValidation("JSON の形式が不正です: "+err.Error()))
		return false
	}
	return true
}

// pathID validates a uuid path parameter (400 on malformed ids, 7.5).
func pathID(c *gin.Context, name string) (string, bool) {
	id := c.Param(name)
	if !dto.IsUUID(id) {
		presenter.RespondError(c, domainerror.ErrValidation(name+" の形式が不正です"))
		return "", false
	}
	return id, true
}

func uuidField(c *gin.Context, field string, v *string) bool {
	if v != nil && !dto.IsUUID(*v) {
		presenter.RespondError(c, domainerror.ErrValidation(field+" の形式が不正です"))
		return false
	}
	return true
}

func uuidOpt(c *gin.Context, field string, o dto.Opt[string]) bool {
	return !o.Set || uuidField(c, field, o.Val)
}

func uuidList(c *gin.Context, field string, ids []string) bool {
	if !dto.AllUUID(ids) {
		presenter.RespondError(c, domainerror.ErrValidation(field+" に不正な ID が含まれています"))
		return false
	}
	return true
}

func uuidQuery(c *gin.Context, name string) (string, bool) {
	v := c.Query(name)
	if v != "" && !dto.IsUUID(v) {
		presenter.RespondError(c, domainerror.ErrValidation(name+" の形式が不正です"))
		return "", false
	}
	return v, true
}

func pageParams(c *gin.Context) (repository.PageParams, bool) {
	p := repository.PageParams{Cursor: c.Query("cursor")}
	if s := c.Query("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			presenter.RespondError(c, domainerror.ErrValidation("limit が不正です"))
			return p, false
		}
		p.Limit = n
	}
	return p, true
}

func respondList[T, U any](c *gin.Context, page repository.Page[T], err error, f func(T) U) {
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, dto.List[U]{Items: dto.Map(page.Items, f), NextCursor: page.NextCursor})
}

func respond[T, U any](c *gin.Context, v T, err error, f func(T) U, created bool) {
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	if created {
		presenter.RespondCreated(c, f(v))
		return
	}
	presenter.RespondOK(c, f(v))
}

func respondNoContent(c *gin.Context, err error) {
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondNoContent(c)
}

func splitCSV(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Health is GET /health.
func (h *Handler) Health(c *gin.Context) {
	presenter.RespondOK(c, gin.H{"status": "ok"})
}
