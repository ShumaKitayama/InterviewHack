// DesignDocument/11_TestDesign.md 11.2.2 回答の解析
package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

const qaOK = `【形式:qa-v1】
【Q1】学生時代に力を入れたことは？
【A1】学園祭の決済システムを構築しました。
【意図1】主体性の確認
【Q2】弱みは？
【A2】慎重すぎる点です。
【意図2】自己理解`

const reviewOK = `【形式:review-v1】
【指摘】
- 具体性：数字がない
- 構成：結論が最後
【改善案】
- 冒頭で結論を述べる
【改善後の回答】
結論から言うと、来場者3,000人の決済を2週間で構築しました。`

const researchOK = `【形式:research-v1】
【魅力・特徴】
- 挑戦を歓迎する文化
- ハッカソン
- 開発者体験への投資
【求める人物像】
自ら課題を見つけて動ける人
【最近のトピック】
- 新サービスの公開`

func TestParse(t *testing.T) {
	cases := []struct {
		id      string
		uc      vo.UseCase
		version string
		text    string
		hints   ParseHints
		result  vo.ParseResult
		check   func(t *testing.T, o ParseOutput)
	}{
		{"UT-PRS-001", vo.UseCaseQAGenerate, VersionQAV1, qaOK, ParseHints{QuestionCount: 2}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if len(o.Preview.Items) != 2 || o.Preview.Items[1].QuestionText != "弱みは？" || o.Preview.Items[0].IntentText != "主体性の確認" {
				t.Fatalf("items = %+v", o.Preview.Items)
			}
			if len(o.Warnings) != 0 {
				t.Fatalf("unexpected warnings %v", o.Warnings)
			}
		}},
		{"UT-PRS-002", vo.UseCaseAnswerReview, VersionReviewV1, reviewOK, ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if len(o.Preview.Issues) != 2 || o.Preview.Issues[0] != "具体性：数字がない" || len(o.Preview.Improvements) != 1 || !strings.HasPrefix(o.Preview.ImprovedAnswerText, "結論から") {
				t.Fatalf("preview = %+v", o.Preview)
			}
		}},
		{"UT-PRS-003", vo.UseCaseResearchSummarize, VersionResearchV1, researchOK, ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if len(o.Preview.Highlights) != 3 || o.Preview.IdealCandidate != "自ら課題を見つけて動ける人" || len(o.Preview.RecentTopics) != 1 {
				t.Fatalf("preview = %+v", o.Preview)
			}
			if !strings.Contains(o.Preview.Text, "■ 求める人物像") {
				t.Fatalf("formatted summary missing headings: %q", o.Preview.Text)
			}
		}},
		{"UT-PRS-004 no format line uses session version", vo.UseCaseQAGenerate, VersionQAV1, strings.TrimPrefix(qaOK, "【形式:qa-v1】\n"), ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if o.DetectedVersion != VersionQAV1 || len(o.Preview.Items) != 2 {
				t.Fatalf("out = %+v", o)
			}
		}},
		{"UT-PRS-005 format line wins over session version", vo.UseCaseQAGenerate, "qa-v0", qaOK, ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if o.DetectedVersion != VersionQAV1 {
				t.Fatalf("detected = %s", o.DetectedVersion)
			}
		}},
		{"UT-PRS-006 qa missing A3", vo.UseCaseQAGenerate, VersionQAV1, qaOK + "\n【Q3】最後に質問は？\n【意図3】意欲", ParseHints{}, vo.ParsePartial, func(t *testing.T, o ParseOutput) {
			if len(o.Preview.Items) != 3 || !contains(o.Warnings, "【A3】が見つかりません") {
				t.Fatalf("out = %+v", o)
			}
		}},
		{"UT-PRS-006 review missing improved answer", vo.UseCaseAnswerReview, VersionReviewV1, strings.Split(reviewOK, "【改善後の回答】")[0], ParseHints{}, vo.ParsePartial, func(t *testing.T, o ParseOutput) {
			if !contains(o.Warnings, "【改善後の回答】が見つかりません") || len(o.Preview.Issues) != 2 {
				t.Fatalf("out = %+v", o)
			}
		}},
		{"UT-PRS-007 free text", vo.UseCaseQAGenerate, VersionQAV1, "面接ではよく志望動機を聞かれます。準備しましょう。", ParseHints{}, vo.ParseFailed, func(t *testing.T, o ParseOutput) {
			if o.Preview.RawText == "" || len(o.Warnings) == 0 {
				t.Fatalf("failed parse must keep raw text and warn: %+v", o)
			}
		}},
		{"UT-PRS-008 drift", vo.UseCaseQAGenerate, VersionQAV1,
			"はい、以下が想定質問です！\n\n**【形式:qa-v1】**\n\n## [Q１] 学生時代に力を入れたことは？\n**【A１】** 学園祭の決済です。\n\n【意図１】主体性\n\n- 【Q3】弱みは？\n【A3】慎重さです。\n【意図3】自己理解\n", ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
				if len(o.Preview.Items) != 2 || o.Preview.Items[0].AnswerText != "学園祭の決済です。" || o.Preview.Items[1].QuestionText != "弱みは？" {
					t.Fatalf("items = %+v", o.Preview.Items)
				}
			}},
		{"UT-PRS-009 es draft", vo.UseCaseESGenerate, VersionESDraftV1, "【形式:esdraft-v1】\n私は貴社を志望します。", ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if o.Preview.Text != "私は貴社を志望します。" {
				t.Fatalf("text = %q", o.Preview.Text)
			}
		}},
		{"UT-PRS-009 summary without format line", vo.UseCaseSummaryGenerate, VersionSummaryV1, "- 結論から話す\n- 数字を入れる", ParseHints{}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if !strings.HasPrefix(o.Preview.Text, "- 結論から話す") {
				t.Fatalf("text = %q", o.Preview.Text)
			}
		}},
		{"UT-PRS-011 unknown format line", vo.UseCaseQAGenerate, VersionQAV1, "【形式:qa-v9】\n【Q1】x\n【A1】y", ParseHints{}, vo.ParseFailed, func(t *testing.T, o ParseOutput) {
			if o.DetectedVersion != "qa-v9" || o.Preview.RawText == "" || len(o.Warnings) == 0 {
				t.Fatalf("out = %+v", o)
			}
		}},
		{"format line of another use case", vo.UseCaseQAGenerate, VersionQAV1, reviewOK, ParseHints{}, vo.ParseFailed, nil},
		{"question count warning", vo.UseCaseQAGenerate, VersionQAV1, qaOK, ParseHints{QuestionCount: 10}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if !contains(o.Warnings, "質問数が指定（10問）と違います（2問）") {
				t.Fatalf("warnings = %v", o.Warnings)
			}
		}},
		{"guess annotation warning", vo.UseCaseESRefine, VersionESRefineV1, "【形式:esrefine-v1】\n売上を2倍にしました（推測）。", ParseHints{CharacterLimit: ip(400)}, vo.ParseSuccess, func(t *testing.T, o ParseOutput) {
			if len(o.Warnings) != 2 {
				t.Fatalf("want length + guess warnings, got %v", o.Warnings)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			o, err := Parse(tc.uc, tc.version, tc.text, tc.hints)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if o.Result != tc.result {
				t.Fatalf("result = %s, want %s (warnings %v)", o.Result, tc.result, o.Warnings)
			}
			if tc.check != nil {
				tc.check(t, o)
			}
		})
	}
}

// UT-PRS-010
func TestParseLimits(t *testing.T) {
	for _, text := range []string{"", "   \n ", strings.Repeat("あ", MaxResponseChars+1)} {
		if _, err := Parse(vo.UseCaseQAGenerate, VersionQAV1, text, ParseHints{}); !domainerror.Is(err, vo.ErrorCodeUnprocessable) {
			t.Fatalf("want 422 for len %d, got %v", len(text), err)
		}
	}
	if _, err := Parse(vo.UseCaseESGenerate, VersionESDraftV1, strings.Repeat("あ", MaxResponseChars), ParseHints{}); err != nil {
		t.Fatalf("exactly the max length must be accepted: %v", err)
	}
}

// 11.7: real-world response samples per use_case/template_version must keep parsing.
func TestParseFixtures(t *testing.T) {
	dirs, _ := filepath.Glob(filepath.Join("testdata", "responses", "*", "*"))
	if len(dirs) == 0 {
		t.Fatal("no response fixtures under testdata/responses")
	}
	for _, dir := range dirs {
		uc, version := vo.UseCase(filepath.Base(filepath.Dir(dir))), filepath.Base(dir)
		files, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
		for _, f := range files {
			t.Run(filepath.Join(string(uc), version, filepath.Base(f)), func(t *testing.T) {
				b, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				want := vo.ParseSuccess
				switch {
				case strings.Contains(f, "partial"):
					want = vo.ParsePartial
				case strings.Contains(f, "failed"):
					want = vo.ParseFailed
				}
				o, err := Parse(uc, version, string(b), ParseHints{})
				if err != nil {
					t.Fatal(err)
				}
				if o.Result != want {
					t.Fatalf("result = %s, want %s (warnings %v)", o.Result, want, o.Warnings)
				}
			})
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
