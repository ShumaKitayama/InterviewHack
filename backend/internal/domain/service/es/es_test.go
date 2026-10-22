// DesignDocument/2_DomainDesign.md 2.2.5 / 2.2.6.1, DirectionChange_20261022.md 2.10.8
package es

import (
	"testing"

	"github.com/interviewhack/backend/internal/domain/vo"
)

func TestStatusAfterContentSaved(t *testing.T) {
	cases := []struct {
		from vo.ESStatus
		body string
		want vo.ESStatus
	}{
		{vo.ESNotStarted, "本文", vo.ESDraft},
		{vo.ESNotStarted, "  ", vo.ESNotStarted},
		{vo.ESInReview, "本文", vo.ESInReview},
		{vo.ESSubmitted, "本文", vo.ESSubmitted},
		{vo.ESDraft, "本文", vo.ESDraft},
	}
	for _, tc := range cases {
		if got := StatusAfterContentSaved(tc.from, tc.body); got != tc.want {
			t.Fatalf("%s + %q = %s, want %s", tc.from, tc.body, got, tc.want)
		}
	}
}

func TestShouldKeepVersion(t *testing.T) {
	empty, blank, text := "", " \n", "本文"
	if ShouldKeepVersion(nil) || ShouldKeepVersion(&empty) || ShouldKeepVersion(&blank) || !ShouldKeepVersion(&text) {
		t.Fatal("only non-empty bodies are versioned")
	}
	if SourceFor(vo.UseCaseESRefine) != vo.VersionSourceESRefine || SourceFor(vo.UseCaseESGenerate) != vo.VersionSourceESGenerate {
		t.Fatal("unexpected version source")
	}
}
