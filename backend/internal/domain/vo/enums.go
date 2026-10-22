// DesignDocument/2_DomainDesign.md 2.2 / DirectionChange_20261022.md 2.10.1
package vo

// UseCase identifies an AI連携 use case (2.10.1).
type UseCase string

const (
	UseCaseQAGenerate        UseCase = "qa_generate"
	UseCaseAnswerReview      UseCase = "answer_review"
	UseCaseSummaryGenerate   UseCase = "summary_generate"
	UseCaseResearchSummarize UseCase = "research_summarize"
	UseCaseESGenerate        UseCase = "es_generate"
	UseCaseESRefine          UseCase = "es_refine"
)

// AllUseCases lists every supported use case.
var AllUseCases = []UseCase{
	UseCaseQAGenerate, UseCaseAnswerReview, UseCaseSummaryGenerate,
	UseCaseResearchSummarize, UseCaseESGenerate, UseCaseESRefine,
}

// Valid reports whether u is a known use case.
func (u UseCase) Valid() bool {
	for _, k := range AllUseCases {
		if u == k {
			return true
		}
	}
	return false
}

// Parsed reports whether the response for this use case is structurally parsed
// (true) or taken in as full text (false). 4.2 の表.
func (u UseCase) Parsed() bool {
	switch u {
	case UseCaseQAGenerate, UseCaseAnswerReview, UseCaseResearchSummarize:
		return true
	default:
		return false
	}
}

// SessionStatus is the prompt_sessions state (7.6.1).
type SessionStatus string

const (
	SessionCreated   SessionStatus = "created"
	SessionPasted    SessionStatus = "pasted"
	SessionSaved     SessionStatus = "saved"
	SessionAbandoned SessionStatus = "abandoned"
)

// Valid reports whether s is a known status.
func (s SessionStatus) Valid() bool {
	switch s {
	case SessionCreated, SessionPasted, SessionSaved, SessionAbandoned:
		return true
	}
	return false
}

// ParseResult is the outcome of parsing a pasted response (4.3.7).
type ParseResult string

const (
	ParseSuccess ParseResult = "success"
	ParsePartial ParseResult = "partial"
	ParseFailed  ParseResult = "failed"
)

// ESStatus is the ES project status (2.2.5, DirectionChange 2.10.8).
type ESStatus string

const (
	ESNotStarted ESStatus = "not_started"
	ESDraft      ESStatus = "draft"
	ESInReview   ESStatus = "in_review"
	ESCompleted  ESStatus = "completed"
	ESSubmitted  ESStatus = "submitted"
)

// Valid reports whether s is a known ES status.
func (s ESStatus) Valid() bool {
	switch s {
	case ESNotStarted, ESDraft, ESInReview, ESCompleted, ESSubmitted:
		return true
	}
	return false
}

// QASourceType is qa.source_type.
type QASourceType string

const (
	QASourcePreGenerated   QASourceType = "pre_generated"
	QASourceReviewImproved QASourceType = "review_improved"
)

// ReviewStatus is qa.review_status.
type ReviewStatus string

const (
	ReviewNew         ReviewStatus = "new"
	ReviewReviewed    ReviewStatus = "reviewed"
	ReviewNeedsReview ReviewStatus = "needs_review"
)

// Valid reports whether r is a known review status.
func (r ReviewStatus) Valid() bool {
	switch r {
	case ReviewNew, ReviewReviewed, ReviewNeedsReview:
		return true
	}
	return false
}

// VersionSource is es_section_versions.source (2.2.6.1).
type VersionSource string

const (
	VersionSourceManual     VersionSource = "manual"
	VersionSourceESGenerate VersionSource = "es_generate"
	VersionSourceESRefine   VersionSource = "es_refine"
)

// AssetType is assets.type.
type AssetType string

// ValidAssetType reports whether t is one of the asset kinds in 8_DBDesign.dbml.
func ValidAssetType(t string) bool {
	switch t {
	case "academic", "work", "project", "skill", "strength", "weakness", "episode", "other":
		return true
	}
	return false
}

// ResearchMaterialType is research_materials.type.
func ValidResearchMaterialType(t string) bool {
	switch t {
	case "url", "pdf", "note", "image":
		return true
	}
	return false
}
