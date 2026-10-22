// Package es holds ES project/section rules.
// DesignDocument/2_DomainDesign.md 2.2.5 / 2.2.6.1, DirectionChange_20261022.md 2.10.8
package es

import (
	"strings"

	"github.com/interviewhack/backend/internal/domain/vo"
)

// MaxVersionsPerSection is how many past versions are kept per section.
const MaxVersionsPerSection = 20

// StatusAfterContentSaved returns the project status after a section body is saved.
// The only automatic transition is not_started → draft; other statuses are kept.
func StatusAfterContentSaved(current vo.ESStatus, body string) vo.ESStatus {
	if current == vo.ESNotStarted && strings.TrimSpace(body) != "" {
		return vo.ESDraft
	}
	return current
}

// ShouldKeepVersion reports whether the content being overwritten is worth saving
// as a version (empty bodies are not versioned).
func ShouldKeepVersion(previous *string) bool {
	return previous != nil && strings.TrimSpace(*previous) != ""
}

// SourceFor maps an AI連携 use case to the version source.
func SourceFor(uc vo.UseCase) vo.VersionSource {
	if uc == vo.UseCaseESRefine {
		return vo.VersionSourceESRefine
	}
	return vo.VersionSourceESGenerate
}
