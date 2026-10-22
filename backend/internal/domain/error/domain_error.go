// DesignDocument/7_BackendDesign.md 7.7 エラー設計
package domainerror

import (
	"errors"
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

// DomainError carries an application error code and a user-facing message.
type DomainError struct {
	Code    vo.ErrorCode
	Message string
	Detail  map[string]any
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a DomainError.
func New(code vo.ErrorCode, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

// As extracts a DomainError from err.
func As(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

// Is reports whether err is a DomainError with the given code.
func Is(err error, code vo.ErrorCode) bool {
	de, ok := As(err)
	return ok && de.Code == code
}

// ErrValidation is a 400 error for malformed input (JSON, uuid, missing fields).
func ErrValidation(message string) *DomainError {
	return New(vo.ErrorCodeValidationError, message)
}

// ErrUnprocessable is a 422 error for business-rule violations (ranges, limits).
func ErrUnprocessable(message string) *DomainError {
	return New(vo.ErrorCodeUnprocessable, message)
}

func ErrUnauthorized() *DomainError {
	return New(vo.ErrorCodeUnauthorized, "認証が必要です")
}

func ErrNotFound(resource string) *DomainError {
	return New(vo.ErrorCodeNotFound, fmt.Sprintf("%s が見つかりません", resource))
}

func ErrSessionNotFound() *DomainError {
	return New(vo.ErrorCodeSessionNotFound, "AI連携のセッションが見つかりません")
}

func ErrSessionStateConflict(status string) *DomainError {
	return &DomainError{
		Code:    vo.ErrorCodeSessionStateConflict,
		Message: "このセッションは現在の状態では操作できません",
		Detail:  map[string]any{"status": status},
	}
}

func ErrConflict(message string) *DomainError {
	return New(vo.ErrorCodeConflict, message)
}

func ErrShareLinkGone() *DomainError {
	return New(vo.ErrorCodeShareLinkGone, "このリンクは無効です")
}

func ErrFeatureDisabled(feature string) *DomainError {
	return &DomainError{
		Code:    vo.ErrorCodeFeatureDisabled,
		Message: "この機能は一時停止中です",
		Detail:  map[string]any{"feature": feature},
	}
}
