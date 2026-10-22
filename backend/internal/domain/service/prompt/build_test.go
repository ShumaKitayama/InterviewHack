// DesignDocument/11_TestDesign.md 11.2.1 テンプレート生成 / 11.7 プロンプト回帰テスト
package prompt

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

var update = flag.Bool("update", false, "update golden files")

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

// fixture is the fixed input set of 11.0 (U1 / C1 / A1-A2 / R1 / E1 / L1).
func fixture(uc vo.UseCase) BuildInput {
	company := &entity.Company{ID: "c1", Name: "株式会社テスト", JobTitle: sp("バックエンドエンジニア"), SelectionPhase: sp("一次面接"), MotivationMemo: sp("開発者体験への投資に惹かれた")}
	assets := []entity.Asset{
		{ID: "a1", Type: "episode", Title: "学園祭の決済システム", Content: "来場者 3,000 人の決済を 2 週間で構築した", TagNames: []string{"ガクチカ", "リーダー"}},
		{ID: "a2", Type: "strength", Title: "粘り強さ", Content: "原因不明の障害を 3 日かけて特定した【重要】"},
	}
	materials := []entity.ResearchMaterial{{ID: "r1", Title: "採用ページ", ContentRaw: sp("挑戦を歓迎する文化。年 2 回のハッカソンを開催。")}}
	in := BuildInput{UseCase: uc, Company: company, Assets: assets, Materials: materials}
	switch uc {
	case vo.UseCaseAnswerReview:
		in.Materials = nil
		in.InterviewLog = &entity.InterviewLog{QuestionText: "学生時代に力を入れたことは？", AnswerText: "学園祭の決済システムを作りました。", SelfEvaluation: sp("結論が弱かった")}
	case vo.UseCaseSummaryGenerate:
		in = BuildInput{UseCase: uc, FeedbackLogs: []entity.InterviewLog{
			{QuestionText: "志望動機は？", AIFeedbackText: sp("具体性：数字が不足している")},
			{QuestionText: "弱みは？", AIFeedbackText: sp("構成：結論が最後にある")},
		}}
	case vo.UseCaseResearchSummarize:
		in.Assets = nil
	case vo.UseCaseESGenerate:
		in.Section = &entity.ESSection{Title: "志望動機", CharacterLimit: ip(400)}
	case vo.UseCaseESRefine:
		in.Section = &entity.ESSection{Title: "志望動機", CharacterLimit: ip(400), ContentDraft: sp("私は貴社の開発者体験への投資に惹かれました。")}
		in.Options.Instructions = "もっと具体的に"
		in.Materials = []entity.ResearchMaterial{{ID: "r1", Title: "採用ページ", Summary: sp("求める人物像：挑戦する人")}}
	}
	return in
}

// UT-TPL-001 / 002 / 007
func TestBuildAllUseCases(t *testing.T) {
	for _, uc := range vo.AllUseCases {
		t.Run(string(uc), func(t *testing.T) {
			p, err := Build(fixture(uc))
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			want, _ := LatestVersion(uc)
			if p.TemplateVersion != want {
				t.Fatalf("version = %s, want %s", p.TemplateVersion, want)
			}
			if !strings.Contains(p.Text, "【形式:"+want+"】") {
				t.Fatalf("prompt lacks the format line instruction for %s", want)
			}
			// 出力形式の指示は必ず最後に置く (4.3.1)
			if strings.LastIndex(p.Text, "# 出力形式") < strings.LastIndex(p.Text, "# ") {
				t.Fatalf("output format section is not the last section")
			}
			golden := filepath.Join("testdata", "golden", want+".txt")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(p.Text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			b, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if string(b) != p.Text {
				t.Fatalf("prompt differs from golden file %s (run go test -run TestBuildAllUseCases ./internal/domain/service/prompt -update after review)", golden)
			}
		})
	}
}

// UT-TPL-003: only the selected assets/materials are included; user data markers are sanitized.
func TestBuildIncludesOnlySelectedData(t *testing.T) {
	in := fixture(vo.UseCaseQAGenerate)
	in.Assets = in.Assets[:1]
	p, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Text, "学園祭の決済システム") || strings.Contains(p.Text, "粘り強さ") {
		t.Fatalf("selected/unselected assets not respected")
	}
	in = fixture(vo.UseCaseQAGenerate)
	p, _ = Build(in)
	if strings.Contains(p.Text, "【重要】") || !strings.Contains(p.Text, "〔重要〕") {
		t.Fatalf("marker brackets in user data must be replaced (4.3.2)")
	}
}

// UT-TPL-004: options are reflected.
func TestBuildOptions(t *testing.T) {
	in := fixture(vo.UseCaseQAGenerate)
	in.Options.QuestionCount = 5
	p, _ := Build(in)
	if !strings.Contains(p.Text, "質問を5問") {
		t.Fatalf("questionCount not reflected")
	}
	in.Options.QuestionCount = 99
	if _, err := Build(in); !domainerror.Is(err, vo.ErrorCodeUnprocessable) {
		t.Fatalf("questionCount out of range must be 422, got %v", err)
	}
	es := fixture(vo.UseCaseESGenerate)
	p, _ = Build(es)
	if !strings.Contains(p.Text, "400字の±10%以内") {
		t.Fatalf("character limit not reflected")
	}
	rv := fixture(vo.UseCaseAnswerReview)
	rv.Options.Strictness = "hard"
	p, _ = Build(rv)
	if !strings.Contains(p.Text, "特に厳しく") {
		t.Fatalf("strictness=hard not reflected")
	}
}

// UT-TPL-005
func TestBuildUnknown(t *testing.T) {
	if _, err := Build(BuildInput{UseCase: "unknown"}); !domainerror.Is(err, vo.ErrorCodeValidationError) {
		t.Fatalf("unknown use case must be VALIDATION_ERROR, got %v", err)
	}
	if _, err := BuildVersion(fixture(vo.UseCaseQAGenerate), "qa-v9"); !domainerror.Is(err, vo.ErrorCodeValidationError) {
		t.Fatalf("unknown version must be VALIDATION_ERROR, got %v", err)
	}
	if _, err := BuildVersion(fixture(vo.UseCaseQAGenerate), VersionReviewV1); !domainerror.Is(err, vo.ErrorCodeValidationError) {
		t.Fatalf("version of another use case must be rejected")
	}
}

// UT-TPL-006
func TestBuildDeterministic(t *testing.T) {
	for _, uc := range vo.AllUseCases {
		a, _ := Build(fixture(uc))
		b, _ := Build(fixture(uc))
		if a.Text != b.Text {
			t.Fatalf("%s: prompt is not deterministic", uc)
		}
	}
}

func TestBuildRequiredInputs(t *testing.T) {
	cases := []struct {
		name string
		in   BuildInput
		code vo.ErrorCode
	}{
		{"qa without company", BuildInput{UseCase: vo.UseCaseQAGenerate}, vo.ErrorCodeValidationError},
		{"review without log", BuildInput{UseCase: vo.UseCaseAnswerReview}, vo.ErrorCodeValidationError},
		{"summary without feedback", BuildInput{UseCase: vo.UseCaseSummaryGenerate}, vo.ErrorCodeUnprocessable},
		{"research with two materials", BuildInput{UseCase: vo.UseCaseResearchSummarize, Materials: make([]entity.ResearchMaterial, 2)}, vo.ErrorCodeValidationError},
		{"research with empty body", BuildInput{UseCase: vo.UseCaseResearchSummarize, Materials: []entity.ResearchMaterial{{Title: "x"}}}, vo.ErrorCodeUnprocessable},
		{"es without section", BuildInput{UseCase: vo.UseCaseESGenerate}, vo.ErrorCodeValidationError},
		{"refine without draft", BuildInput{UseCase: vo.UseCaseESRefine, Section: &entity.ESSection{Title: "x"}}, vo.ErrorCodeUnprocessable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Build(tc.in); !domainerror.Is(err, tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
}

// 4.3.5: long content_raw is truncated per material; long prompts warn.
func TestBuildLengthRules(t *testing.T) {
	in := fixture(vo.UseCaseQAGenerate)
	long := strings.Repeat("あ", MaxMaterialChars+500)
	in.Materials = []entity.ResearchMaterial{{Title: "長い資料", ContentRaw: &long}}
	p, _ := Build(in)
	if strings.Contains(p.Text, strings.Repeat("あ", MaxMaterialChars+1)) || !strings.Contains(p.Text, "先頭 3,000 字まで") {
		t.Fatalf("content_raw must be truncated to 3,000 chars with a note")
	}
	for i := 0; i < 3; i++ {
		in.Materials = append(in.Materials, entity.ResearchMaterial{Title: "資料", ContentRaw: &long})
	}
	p, _ = Build(in)
	if len(p.Warnings) == 0 || p.Warnings[0] != WarningPromptTooLong {
		t.Fatalf("prompt over 8,000 chars must warn")
	}
	summary := "要約済み"
	in.Materials = []entity.ResearchMaterial{{Title: "資料", ContentRaw: &long, Summary: &summary}}
	p, _ = Build(in)
	if !strings.Contains(p.Text, "要約済み") || strings.Contains(p.Text, "ああああ") {
		t.Fatalf("summary must be preferred over content_raw")
	}
}
