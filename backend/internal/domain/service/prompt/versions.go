// DesignDocument/4_LLMDesign.md 4.3.6 テンプレートのバージョニング
package prompt

import "github.com/interviewhack/backend/internal/domain/vo"

// Template versions (DirectionChange_20261022.md 2.10.1).
const (
	VersionQAV1       = "qa-v1"
	VersionReviewV1   = "review-v1"
	VersionSummaryV1  = "summary-v1"
	VersionResearchV1 = "research-v1"
	VersionESDraftV1  = "esdraft-v1"
	VersionESRefineV1 = "esrefine-v1"
)

// latest is the version new prompts are generated with.
var latest = map[vo.UseCase]string{
	vo.UseCaseQAGenerate:        VersionQAV1,
	vo.UseCaseAnswerReview:      VersionReviewV1,
	vo.UseCaseSummaryGenerate:   VersionSummaryV1,
	vo.UseCaseResearchSummarize: VersionResearchV1,
	vo.UseCaseESGenerate:        VersionESDraftV1,
	vo.UseCaseESRefine:          VersionESRefineV1,
}

// owners maps every known version (past and present) to its use case.
// Parsers must keep supporting every version listed here (4.3.6).
var owners = map[string]vo.UseCase{
	VersionQAV1:       vo.UseCaseQAGenerate,
	VersionReviewV1:   vo.UseCaseAnswerReview,
	VersionSummaryV1:  vo.UseCaseSummaryGenerate,
	VersionResearchV1: vo.UseCaseResearchSummarize,
	VersionESDraftV1:  vo.UseCaseESGenerate,
	VersionESRefineV1: vo.UseCaseESRefine,
}

// LatestVersion returns the current template version of a use case.
func LatestVersion(uc vo.UseCase) (string, bool) {
	v, ok := latest[uc]
	return v, ok
}

// UseCaseOf returns the use case a template version belongs to.
func UseCaseOf(version string) (vo.UseCase, bool) {
	uc, ok := owners[version]
	return uc, ok
}
