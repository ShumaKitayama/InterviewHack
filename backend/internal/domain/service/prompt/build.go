// Package prompt builds AI連携 prompts and parses pasted responses as pure functions.
// DesignDocument/4_LLMDesign.md 4.2 / 4.3, DirectionChange_20261022.md 2.10.1
package prompt

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

const (
	// RecommendedMaxPromptChars is the recommended upper bound of a whole prompt (4.3.5).
	RecommendedMaxPromptChars = 8000
	// MaxMaterialChars is the per-material limit when content_raw is used (4.3.5).
	MaxMaterialChars = 3000
	// MaxResearchChars is the content_raw limit for research_summarize (4.3.5).
	MaxResearchChars = 8000
	// DefaultQuestionCount is the default number of questions for qa_generate.
	DefaultQuestionCount = 10
	// MaxQuestionCount bounds questionCount.
	MaxQuestionCount = 30
	// DefaultAnswerLength is the default answer length guide for qa_generate.
	DefaultAnswerLength = 300
	// DefaultESLength is used when an ES section has no character limit.
	DefaultESLength = 400
	// RecentLogsForSummary is how many recent reviewed logs summary_generate uses.
	RecentLogsForSummary = 10
)

// WarningPromptTooLong is returned when a prompt exceeds RecommendedMaxPromptChars.
const WarningPromptTooLong = "プロンプトが長すぎます（推奨 8,000 字以内）。資料を減らすか、先に資料を要約してください"

// Options are the per-use-case options stored in input_refs.options.
type Options struct {
	QuestionCount int
	Strictness    string // normal | hard
	Instructions  string // es_refine
}

// BuildInput is everything a template needs. Callers load only the user's own data.
type BuildInput struct {
	UseCase      vo.UseCase
	Company      *entity.Company
	Assets       []entity.Asset
	Materials    []entity.ResearchMaterial
	Interview    *entity.Interview
	InterviewLog *entity.InterviewLog
	FeedbackLogs []entity.InterviewLog
	Section      *entity.ESSection
	Options      Options
}

// Prompt is a generated prompt.
type Prompt struct {
	Text            string
	TemplateVersion string
	Warnings        []string
}

// Build renders the latest template version for in.UseCase.
func Build(in BuildInput) (Prompt, error) {
	version, ok := LatestVersion(in.UseCase)
	if !ok {
		return Prompt{}, domainerror.ErrValidation(fmt.Sprintf("未知の useCase です: %s", in.UseCase))
	}
	return BuildVersion(in, version)
}

// BuildVersion renders a specific template version. It is used when a session's
// prompt is re-generated from input_refs (GET /prompt-sessions/{id}/prompt).
func BuildVersion(in BuildInput, version string) (Prompt, error) {
	uc, ok := UseCaseOf(version)
	if !ok || uc != in.UseCase {
		return Prompt{}, domainerror.ErrValidation(fmt.Sprintf("未知のテンプレートです: %s", version))
	}
	var text string
	var err error
	switch version {
	case VersionQAV1:
		text, err = buildQA(in)
	case VersionReviewV1:
		text, err = buildReview(in)
	case VersionSummaryV1:
		text, err = buildSummary(in)
	case VersionResearchV1:
		text, err = buildResearch(in)
	case VersionESDraftV1:
		text, err = buildESDraft(in)
	case VersionESRefineV1:
		text, err = buildESRefine(in)
	default:
		return Prompt{}, domainerror.ErrValidation(fmt.Sprintf("未知のテンプレートです: %s", version))
	}
	if err != nil {
		return Prompt{}, err
	}
	p := Prompt{Text: text, TemplateVersion: version}
	if utf8.RuneCountInString(text) > RecommendedMaxPromptChars {
		p.Warnings = append(p.Warnings, WarningPromptTooLong)
	}
	return p, nil
}

// sanitize replaces the marker brackets in user data so they cannot collide
// with output markers (4.3.2).
func sanitize(s string) string {
	return strings.NewReplacer("【", "〔", "】", "〕").Replace(strings.TrimSpace(s))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（未入力）"
	}
	return s
}

func truncateRunes(s string, max int) (string, bool) {
	if utf8.RuneCountInString(s) <= max {
		return s, false
	}
	r := []rune(s)
	return string(r[:max]), true
}

var assetTypeLabels = map[string]string{
	"academic": "学歴", "work": "職務経歴", "project": "プロジェクト", "skill": "スキル",
	"strength": "強み", "weakness": "弱み", "episode": "エピソード", "other": "その他",
}

func assetTypeLabel(t string) string {
	if l, ok := assetTypeLabels[t]; ok {
		return l
	}
	return sanitize(t)
}

// formatAssets renders assets as "[種別] タイトル（タグ）：本文" (4.3.2).
func formatAssets(assets []entity.Asset) string {
	if len(assets) == 0 {
		return "（選択された自己資産はありません）"
	}
	var b strings.Builder
	for i, a := range assets {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[" + assetTypeLabel(a.Type) + "] " + sanitize(a.Title))
		if len(a.TagNames) > 0 {
			tags := make([]string, len(a.TagNames))
			for j, t := range a.TagNames {
				tags[j] = sanitize(t)
			}
			b.WriteString("（" + strings.Join(tags, "、") + "）")
		}
		b.WriteString("：" + sanitize(a.Content))
	}
	return b.String()
}

// formatMaterials renders research materials, preferring summary over content_raw
// and truncating content_raw to MaxMaterialChars (4.3.5).
func formatMaterials(materials []entity.ResearchMaterial) string {
	if len(materials) == 0 {
		return "（選択された企業研究資料はありません）"
	}
	var b strings.Builder
	for i, m := range materials {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[" + sanitize(m.Title) + "]\n")
		if s := strings.TrimSpace(deref(m.Summary)); s != "" {
			b.WriteString(sanitize(s))
			continue
		}
		body, truncated := truncateRunes(sanitize(deref(m.ContentRaw)), MaxMaterialChars)
		b.WriteString(orDash(body))
		if truncated {
			b.WriteString("\n（資料が長いため、先頭 3,000 字までを掲載しています）")
		}
	}
	return b.String()
}

func companyLine(c *entity.Company) string {
	if c == nil {
		return "（企業情報なし）"
	}
	return sanitize(c.Name) + " / " + orDash(sanitize(deref(c.JobTitle))) + " / " + orDash(sanitize(deref(c.SelectionPhase)))
}

func jobTitle(c *entity.Company) string {
	if c == nil || strings.TrimSpace(deref(c.JobTitle)) == "" {
		return "志望職種"
	}
	return sanitize(deref(c.JobTitle))
}

func buildQA(in BuildInput) (string, error) {
	if in.Company == nil {
		return "", domainerror.ErrValidation("companyId は必須です")
	}
	count := in.Options.QuestionCount
	if count == 0 {
		count = DefaultQuestionCount
	}
	if count < 1 || count > MaxQuestionCount {
		return "", domainerror.ErrUnprocessable(fmt.Sprintf("questionCount は 1〜%d の範囲で指定してください", MaxQuestionCount))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "あなたは%sの採用面接を担当する経験豊富な面接官であり、キャリアコーチです。\n", jobTitle(in.Company))
	fmt.Fprintf(&b, "以下の候補者情報と企業情報をもとに、この面接で聞かれやすく、深掘りされやすい質問を%d問作り、\n", count)
	b.WriteString("それぞれに候補者本人の経験だけを使った模範解答を書いてください。\n\n")
	b.WriteString("# 制約\n")
	b.WriteString("- 候補者情報に書かれていない経歴・実績・数値を作らないこと\n")
	b.WriteString("- 推測を含む場合は、その文の末尾に（推測）と書くこと\n")
	fmt.Fprintf(&b, "- 模範解答は1問あたり%d字程度。結論→理由→具体例→学びの順で書くこと\n", DefaultAnswerLength)
	b.WriteString("- 一般論だけで終わらせないこと\n\n")
	b.WriteString("# 企業情報\n")
	b.WriteString(companyLine(in.Company) + "\n")
	b.WriteString(orDash(sanitize(deref(in.Company.MotivationMemo))) + "\n")
	b.WriteString(formatMaterials(in.Materials) + "\n\n")
	b.WriteString("# 候補者情報（自己資産）\n")
	b.WriteString(formatAssets(in.Assets) + "\n\n")
	b.WriteString("# 出力形式（この形式以外の文章は書かないこと）\n")
	b.WriteString("1行目に必ず「【形式:qa-v1】」と書き、続けて以下を質問数だけ繰り返してください。\n")
	b.WriteString("【Q1】質問文\n【A1】模範解答\n【意図1】この質問で面接官が確認したいこと\n")
	return b.String(), nil
}

func buildReview(in BuildInput) (string, error) {
	if in.InterviewLog == nil {
		return "", domainerror.ErrValidation("interviewLogId は必須です")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "あなたは%sの採用面接官であり、厳しめだが建設的なキャリアコーチです。\n", jobTitle(in.Company))
	b.WriteString("以下は候補者が実際の面接で受けた質問と、その場での回答です。\n")
	b.WriteString("論理性・具体性・説得力・構成の観点で添削し、次回使える改善後の回答を書いてください。\n\n")
	b.WriteString("# 制約\n")
	b.WriteString("- 否定だけで終わらせず、「どう直せばよいか」を具体的に書くこと\n")
	b.WriteString("- 改善後の回答には、候補者情報にあるエピソードだけを使うこと。新しい経歴を作らないこと\n")
	b.WriteString("- 使ったエピソードのタイトルを改善案の中で明記すること\n")
	b.WriteString("- 推測を含む場合は、その文の末尾に（推測）と書くこと\n")
	if in.Options.Strictness == "hard" {
		b.WriteString("- 評価は特に厳しくし、曖昧な表現や根拠のない主張はすべて指摘すること\n")
	}
	b.WriteString("\n# 面接情報\n")
	b.WriteString(companyLine(in.Company) + "\n")
	b.WriteString("# 質問\n" + sanitize(in.InterviewLog.QuestionText) + "\n")
	b.WriteString("# 候補者の回答\n" + sanitize(in.InterviewLog.AnswerText) + "\n")
	b.WriteString("# 自己評価\n" + orDash(sanitize(deref(in.InterviewLog.SelfEvaluation))) + "\n")
	b.WriteString("# 候補者情報（自己資産）\n" + formatAssets(in.Assets) + "\n\n")
	b.WriteString("# 出力形式（この形式以外の文章は書かないこと）\n")
	b.WriteString("【形式:review-v1】\n【指摘】\n- 指摘（観点：論理性 / 具体性 / 説得力 / 構成 のいずれかを先頭に書く）\n")
	b.WriteString("【改善案】\n- 改善の要点\n【改善後の回答】\n改善後の回答文\n")
	return b.String(), nil
}

func buildSummary(in BuildInput) (string, error) {
	var items []string
	for _, l := range in.FeedbackLogs {
		if fb := strings.TrimSpace(deref(l.AIFeedbackText)); fb != "" {
			items = append(items, "- 質問："+sanitize(l.QuestionText)+"\n  添削："+sanitize(fb))
		}
	}
	if len(items) == 0 {
		return "", domainerror.ErrUnprocessable("要約できる添削結果がありません。先に面接ログの添削を取り込んでください")
	}
	var b strings.Builder
	b.WriteString("以下は、ある就活生の直近の面接に対する添削結果です。\n")
	b.WriteString("次の面接で特に意識すべきポイントを3〜5項目、行動に移せる短文で箇条書きにしてください。\n\n")
	b.WriteString("# 制約\n- 添削結果に書かれていないことは書かないこと\n\n")
	b.WriteString("# 添削結果\n" + strings.Join(items, "\n") + "\n\n")
	b.WriteString("# 出力形式\n1行目に「【形式:summary-v1】」と書き、続けて箇条書きだけを書いてください。\n")
	return b.String(), nil
}

func buildResearch(in BuildInput) (string, error) {
	if len(in.Materials) != 1 {
		return "", domainerror.ErrValidation("research_summarize は企業研究資料を 1 件だけ指定してください")
	}
	m := in.Materials[0]
	body, truncated := truncateRunes(sanitize(deref(m.ContentRaw)), MaxResearchChars)
	if strings.TrimSpace(body) == "" {
		return "", domainerror.ErrUnprocessable("資料の本文がありません")
	}
	companyName := "志望企業"
	if in.Company != nil {
		companyName = sanitize(in.Company.Name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "以下は%sに関する資料の本文です（Webページや PDF から抽出したもので、ノイズを含むことがあります）。\n", companyName)
	fmt.Fprintf(&b, "%s志望の就活生が「なぜこの会社か」を語るために使える情報を整理してください。\n\n", jobTitle(in.Company))
	b.WriteString("# 制約\n- 広告、メニュー、ヘッダー、フッターなどのノイズは無視すること\n- 資料に書かれていないことは書かないこと\n\n")
	if in.Company != nil {
		if memo := strings.TrimSpace(deref(in.Company.MotivationMemo)); memo != "" {
			b.WriteString("# ユーザーのメモ\n" + sanitize(memo) + "\n\n")
		}
	}
	b.WriteString("# 資料本文\n[" + sanitize(m.Title) + "]\n" + body + "\n")
	if truncated {
		b.WriteString("（資料が長いため、先頭 8,000 字までを掲載しています）\n")
	}
	b.WriteString("\n# 出力形式（この形式以外の文章は書かないこと）\n")
	b.WriteString("【形式:research-v1】\n【魅力・特徴】\n- 3点\n【求める人物像】\n本文\n【最近のトピック】\n- 箇条書き\n")
	return b.String(), nil
}

func lengthConstraint(limit *int) string {
	if limit != nil && *limit > 0 {
		return fmt.Sprintf("%d字の±10%%以内に収めること", *limit)
	}
	return fmt.Sprintf("%d字程度で書くこと", DefaultESLength)
}

func buildESDraft(in BuildInput) (string, error) {
	if in.Section == nil {
		return "", domainerror.ErrValidation("esSectionId は必須です")
	}
	var b strings.Builder
	b.WriteString("あなたは新卒・転職のES添削を専門とするキャリアコーチです。\n")
	fmt.Fprintf(&b, "以下の企業情報と候補者情報を使い、「%s」のES原稿を書いてください。\n\n", sanitize(in.Section.Title))
	b.WriteString("# 制約\n")
	b.WriteString("- 企業の特徴と、候補者の強み・経験の接点を明確にすること\n")
	b.WriteString("- 結論ファースト（PREP法）で書くこと\n")
	b.WriteString("- " + lengthConstraint(in.Section.CharacterLimit) + "\n")
	b.WriteString("- 候補者情報にない経歴・実績・数値を作らないこと\n\n")
	b.WriteString("# 企業情報\n")
	if in.Company != nil {
		b.WriteString(sanitize(in.Company.Name) + " / " + orDash(sanitize(deref(in.Company.JobTitle))) + "\n")
	}
	b.WriteString(formatMaterials(in.Materials) + "\n")
	b.WriteString("# 候補者情報（自己資産）\n" + formatAssets(in.Assets) + "\n\n")
	b.WriteString("# 出力形式\n1行目に「【形式:esdraft-v1】」と書き、続けてES本文だけを書いてください。解説や前置きは書かないこと。\n")
	return b.String(), nil
}

func buildESRefine(in BuildInput) (string, error) {
	if in.Section == nil {
		return "", domainerror.ErrValidation("esSectionId は必須です")
	}
	draft := strings.TrimSpace(deref(in.Section.ContentDraft))
	if draft == "" {
		return "", domainerror.ErrUnprocessable("推敲する原稿がありません。先にドラフトを作成してください")
	}
	instructions := strings.TrimSpace(in.Options.Instructions)
	if instructions == "" {
		instructions = "全体の読みやすさと説得力を高めてください"
	}
	var b strings.Builder
	b.WriteString("以下のES原稿を、修正指示に従って書き直してください。\n\n")
	b.WriteString("# 修正指示\n" + sanitize(instructions) + "\n")
	b.WriteString("# 制約\n")
	b.WriteString("- " + lengthConstraint(in.Section.CharacterLimit) + "\n")
	b.WriteString("- 原稿にない経歴・実績・数値を加えないこと\n")
	b.WriteString("- 文章の自然さを保つこと\n")
	b.WriteString("# 現在の原稿\n" + sanitize(draft) + "\n")
	var ideals []string
	for _, m := range in.Materials {
		if s := strings.TrimSpace(deref(m.Summary)); s != "" {
			ideals = append(ideals, sanitize(s))
		}
	}
	if len(ideals) > 0 {
		b.WriteString("# （任意）企業の求める人物像\n" + strings.Join(ideals, "\n") + "\n")
	}
	b.WriteString("\n# 出力形式\n1行目に「【形式:esrefine-v1】」と書き、続けて書き直したES本文だけを書いてください。\n")
	return b.String(), nil
}
