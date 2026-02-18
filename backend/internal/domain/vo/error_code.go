// DesignDocument/7_BackendDesign.md 7.7 エラー設計
package vo

type ErrorCode string

const (
	ErrorCodeUnauthorized         ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden            ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound             ErrorCode = "NOT_FOUND"
	ErrorCodeConflict             ErrorCode = "CONFLICT"
	ErrorCodeDuplicateEvent       ErrorCode = "DUPLICATE_EVENT"
	ErrorCodeValidationError      ErrorCode = "VALIDATION_ERROR"
	ErrorCodeRateLimited          ErrorCode = "RATE_LIMITED"
	ErrorCodeInsufficientCredit   ErrorCode = "INSUFFICIENT_CREDITS"
	ErrorCodeInternalError        ErrorCode = "INTERNAL_ERROR"
	ErrorCodeOpenAIError          ErrorCode = "OPENAI_ERROR"
	ErrorCodeStripeError          ErrorCode = "STRIPE_ERROR"
	ErrorCodeInvalidPackage       ErrorCode = "INVALID_PACKAGE"
	ErrorCodeFeatureDisabled      ErrorCode = "FEATURE_DISABLED"
	ErrorCodeOutputSchemaInvalid  ErrorCode = "OUTPUT_SCHEMA_INVALID"
	ErrorCodeOutputRetryExhausted ErrorCode = "OUTPUT_RETRY_EXHAUSTED"
)

func (e ErrorCode) String() string {
	return string(e)
}
