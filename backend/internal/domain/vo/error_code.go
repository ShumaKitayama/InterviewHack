// DesignDocument/7_BackendDesign.md 7.7.2 HTTPステータスとエラーコード対応表
package vo

// ErrorCode is the machine-readable error kind returned in the `code` field.
type ErrorCode string

const (
	ErrorCodeValidationError      ErrorCode = "VALIDATION_ERROR"
	ErrorCodeUnprocessable        ErrorCode = "VALIDATION_ERROR_422"
	ErrorCodeUnauthorized         ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden            ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound             ErrorCode = "NOT_FOUND"
	ErrorCodeSessionNotFound      ErrorCode = "SESSION_NOT_FOUND"
	ErrorCodeConflict             ErrorCode = "CONFLICT"
	ErrorCodeSessionStateConflict ErrorCode = "SESSION_STATE_CONFLICT"
	ErrorCodeShareLinkGone        ErrorCode = "SHARE_LINK_GONE"
	ErrorCodeRateLimited          ErrorCode = "RATE_LIMITED"
	ErrorCodeInternalError        ErrorCode = "INTERNAL_ERROR"
	ErrorCodeFeatureDisabled      ErrorCode = "FEATURE_DISABLED"
)

// String returns the code as sent to clients. The 422 variant is reported as
// VALIDATION_ERROR (7.7.2: the same code is used for 400 and 422).
func (e ErrorCode) String() string {
	if e == ErrorCodeUnprocessable {
		return string(ErrorCodeValidationError)
	}
	return string(e)
}
