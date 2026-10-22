// DesignDocument/7_BackendDesign.md 7.7 エラー設計（API共通）
package presenter

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
	"github.com/interviewhack/backend/internal/port/repository"
)

// ErrorBody is the unified error response (7.7.1).
type ErrorBody struct {
	Status    string         `json:"status"`
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Detail    map[string]any `json:"detail,omitempty"`
	RequestID string         `json:"requestId,omitempty"`
}

// ErrorCodeKey is the gin context key the logger reads error_code from.
const ErrorCodeKey = "error_code"

func requestID(c *gin.Context) string {
	v, _ := c.Get("request_id")
	s, _ := v.(string)
	return s
}

// Status maps an error code to an HTTP status (7.7.2).
func Status(code vo.ErrorCode) int {
	switch code {
	case vo.ErrorCodeValidationError:
		return http.StatusBadRequest
	case vo.ErrorCodeUnprocessable:
		return http.StatusUnprocessableEntity
	case vo.ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case vo.ErrorCodeForbidden:
		return http.StatusForbidden
	case vo.ErrorCodeNotFound, vo.ErrorCodeSessionNotFound:
		return http.StatusNotFound
	case vo.ErrorCodeConflict, vo.ErrorCodeSessionStateConflict:
		return http.StatusConflict
	case vo.ErrorCodeShareLinkGone:
		return http.StatusGone
	case vo.ErrorCodeRateLimited:
		return http.StatusTooManyRequests
	case vo.ErrorCodeFeatureDisabled:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// RespondError writes the error. Unknown errors become 500 without internal detail.
func RespondError(c *gin.Context, err error) {
	de, ok := domainerror.As(err)
	if !ok {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			de = domainerror.ErrNotFound("リソース")
		case errors.Is(err, repository.ErrInvalidCursor):
			de = domainerror.ErrValidation("cursor が不正です")
		case errors.Is(err, repository.ErrConflict):
			de = domainerror.ErrConflict("リソースが重複しています")
		default:
			slog.Error("internal error", "request_id", requestID(c), "error", err.Error())
			de = domainerror.New(vo.ErrorCodeInternalError, "内部エラーが発生しました")
		}
	}
	c.Set(ErrorCodeKey, de.Code.String())
	c.AbortWithStatusJSON(Status(de.Code), ErrorBody{
		Status:    "error",
		Code:      de.Code.String(),
		Message:   de.Message,
		Detail:    de.Detail,
		RequestID: requestID(c),
	})
}
