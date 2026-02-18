// DesignDocument/7_BackendDesign.md 7.7 エラー設計（API共通）
package presenter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

type ErrorBody struct {
	Status    string         `json:"status"`
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Detail    map[string]any `json:"detail,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

func RespondError(c *gin.Context, err error) {
	requestID, _ := c.Get("request_id")
	reqIDStr, _ := requestID.(string)

	if domErr, ok := err.(*domainerror.DomainError); ok {
		status := mapCodeToHTTPStatus(domErr.Code)
		c.JSON(status, ErrorBody{
			Status:    "error",
			Code:      domErr.Code.String(),
			Message:   domErr.Message,
			Detail:    domErr.Detail,
			RequestID: reqIDStr,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorBody{
		Status:    "error",
		Code:      vo.ErrorCodeInternalError.String(),
		Message:   "内部エラーが発生しました",
		RequestID: reqIDStr,
	})
}

// DesignDocument/7_BackendDesign.md 7.7.2 HTTPステータスとエラーコード対応表
func mapCodeToHTTPStatus(code vo.ErrorCode) int {
	switch code {
	case vo.ErrorCodeValidationError, vo.ErrorCodeInvalidPackage:
		return http.StatusBadRequest
	case vo.ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case vo.ErrorCodeForbidden:
		return http.StatusForbidden
	case vo.ErrorCodeNotFound:
		return http.StatusNotFound
	case vo.ErrorCodeConflict, vo.ErrorCodeDuplicateEvent:
		return http.StatusConflict
	case vo.ErrorCodeRateLimited:
		return http.StatusTooManyRequests
	case vo.ErrorCodeInsufficientCredit:
		return http.StatusPaymentRequired
	case vo.ErrorCodeOpenAIError, vo.ErrorCodeStripeError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
