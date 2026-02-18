// DesignDocument/7_BackendDesign.md 7.7 エラー設計
package domainerror

import (
	"fmt"

	"github.com/interviewhack/backend/internal/domain/vo"
)

type DomainError struct {
	Code    vo.ErrorCode
	Message string
	Detail  map[string]any
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code vo.ErrorCode, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

func NewWithDetail(code vo.ErrorCode, message string, detail map[string]any) *DomainError {
	return &DomainError{Code: code, Message: message, Detail: detail}
}

func ErrInsufficientCredits(balance, cost int) *DomainError {
	return NewWithDetail(
		vo.ErrorCodeInsufficientCredit,
		"クレジット残高が不足しています",
		map[string]any{"balance": balance, "cost": cost},
	)
}

func ErrInvalidPackage(packageID string) *DomainError {
	return New(vo.ErrorCodeInvalidPackage, fmt.Sprintf("無効なパッケージです: %s", packageID))
}

func ErrDuplicateEvent(eventID string) *DomainError {
	return New(vo.ErrorCodeDuplicateEvent, fmt.Sprintf("処理済みのイベントです: %s", eventID))
}

func ErrFeatureDisabled(feature string) *DomainError {
	return New(vo.ErrorCodeFeatureDisabled, fmt.Sprintf("この機能は無効です: %s", feature))
}

func ErrNotFound(resource string) *DomainError {
	return New(vo.ErrorCodeNotFound, fmt.Sprintf("%s が見つかりません", resource))
}

func ErrValidation(message string) *DomainError {
	return New(vo.ErrorCodeValidationError, message)
}

func ErrUnauthorized() *DomainError {
	return New(vo.ErrorCodeUnauthorized, "認証が必要です")
}

func ErrForbidden() *DomainError {
	return New(vo.ErrorCodeForbidden, "アクセスが許可されていません")
}

func ErrInvalidStatusTransition(from, to string) *DomainError {
	return New(vo.ErrorCodeConflict, fmt.Sprintf("無効な状態遷移です: %s -> %s", from, to))
}
