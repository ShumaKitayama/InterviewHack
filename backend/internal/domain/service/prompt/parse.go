// DesignDocument/4_LLMDesign.md 4.3.7 解析ルール / 4.5 外部AIの揺れへの対処
package prompt

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/domain/vo"
)

// MaxResponseChars is the upper bound of a pasted response (7.5.5).
const MaxResponseChars = 100000

// QAItem is one parsed question/answer. IntentText is preview-only and never saved.
type QAItem struct {
	QuestionText string
	AnswerText   string
	IntentText   string
}

// Preview is the parsed structure. Only the fields of the session's use case are set.
type Preview struct {
	Items              []QAItem
	Issues             []string
	Improvements       []string
	ImprovedAnswerText string
	Highlights         []string
	IdealCandidate     string
	RecentTopics       []string
	// Text is the full text for non-parsed use cases, and the formatted summary
	// (with the three headings) for research_summarize.
	Text string
	// RawText keeps the original response when parsing failed, for manual editing.
	RawText string
}

// ParseOutput is the result of Parse.
type ParseOutput struct {
	Result          vo.ParseResult
	DetectedVersion string
	Preview         Preview
	Warnings        []string
}

// ParseHints carries session information used for warnings.
type ParseHints struct {
	QuestionCount  int
	CharacterLimit *int
}

var (
	formatLineRe = regexp.MustCompile(`【\s*形式\s*[:：]\s*([A-Za-z0-9._\-]+)\s*】`)
	// markerRe matches a marker at the start of a (decoration-stripped) line.
	markerRe = regexp.MustCompile(`^【\s*([^【】]+?)\s*】\s*(.*)$`)
	bulletRe = regexp.MustCompile(`^\s*(?:[-*・•●]|\d+[.)．、])\s*`)
	// noteGuessRe detects the （推測） annotation.
	noteGuessRe = regexp.MustCompile(`[（(]推測[）)]`)
)

// Parse parses a pasted response for a session. sessionVersion is the template
// version recorded on the session; the format line in the response wins over it.
func Parse(uc vo.UseCase, sessionVersion, response string, hints ParseHints) (ParseOutput, error) {
	if strings.TrimSpace(response) == "" {
		return ParseOutput{}, domainerror.ErrUnprocessable("回答が空です")
	}
	if utf8.RuneCountInString(response) > MaxResponseChars {
		return ParseOutput{}, domainerror.ErrUnprocessable("回答が長すぎます（最大 100,000 文字）")
	}

	text := normalize(response)
	version := sessionVersion
	body := text
	if loc := formatLineRe.FindStringSubmatchIndex(text); loc != nil {
		version = text[loc[2]:loc[3]]
		// Anything before the format line (前置き) is ignored.
		body = text[loc[1]:]
	}

	out := ParseOutput{DetectedVersion: version}
	owner, known := UseCaseOf(version)
	if !known || owner != uc {
		out.Result = vo.ParseFailed
		out.Preview.RawText = strings.TrimSpace(response)
		out.Warnings = append(out.Warnings, fmt.Sprintf("このセッションでは解析できない形式です（%s）。手動で修正してください", version))
		return out, nil
	}

	switch version {
	case VersionQAV1:
		parseQA(body, hints, &out)
	case VersionReviewV1:
		parseReview(body, &out)
	case VersionResearchV1:
		parseResearch(body, &out)
	case VersionSummaryV1, VersionESDraftV1, VersionESRefineV1:
		out.Result = vo.ParseSuccess
		out.Preview.Text = strings.TrimSpace(body)
		if uc == vo.UseCaseESGenerate || uc == vo.UseCaseESRefine {
			out.Warnings = append(out.Warnings, lengthWarnings(out.Preview.Text, hints.CharacterLimit)...)
		}
	}
	if out.Result == vo.ParseFailed {
		out.Preview.RawText = strings.TrimSpace(response)
	}
	if noteGuessRe.MatchString(text) {
		out.Warnings = append(out.Warnings, "（推測）の注記がある箇所があります。内容を確認してください")
	}
	return out, nil
}

// normalize absorbs common formatting drift (4.3.7 表記揺れの吸収).
func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = toHalfWidth(s)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = normalizeMarkerLine(line)
	}
	return strings.Join(lines, "\n")
}

var halfBracketMarkerRe = regexp.MustCompile(`^\[\s*((?:形式|Q|A|意図|指摘|改善案|改善後の回答|魅力・特徴|求める人物像|最近のトピック)[^\[\]]*)\]`)

// normalizeMarkerLine strips Markdown decorations around a marker at line start
// and converts half-width [..] markers to 【..】.
func normalizeMarkerLine(line string) string {
	trimmed := strings.TrimSpace(line)
	stripped := strings.TrimLeft(trimmed, "#>*-_ \t")
	stripped = strings.TrimSpace(strings.ReplaceAll(stripped, "**", ""))
	stripped = strings.ReplaceAll(stripped, "__", "")
	if m := halfBracketMarkerRe.FindStringSubmatchIndex(stripped); m != nil {
		stripped = "【" + stripped[m[2]:m[3]] + "】" + stripped[m[1]:]
	}
	if strings.HasPrefix(stripped, "【") {
		return stripped
	}
	return line
}

func toHalfWidth(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '０' && r <= '９':
			return r - '０' + '0'
		case r == 'Ｑ' || r == 'ｑ':
			return 'Q'
		case r == 'Ａ' || r == 'ａ':
			return 'A'
		case r == '［':
			return '['
		case r == '］':
			return ']'
		}
		return r
	}, s)
}

type block struct {
	name    string
	content string
}

// splitBlocks splits text into marker blocks. Text before the first marker is dropped.
func splitBlocks(text string) []block {
	var blocks []block
	var cur *block
	var buf []string
	flush := func() {
		if cur != nil {
			cur.content = strings.TrimSpace(strings.Join(buf, "\n"))
			blocks = append(blocks, *cur)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if m := markerRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			flush()
			cur = &block{name: strings.TrimSpace(m[1])}
			buf = nil
			if rest := strings.TrimSpace(m[2]); rest != "" {
				buf = append(buf, rest)
			}
			continue
		}
		if cur != nil {
			buf = append(buf, line)
		}
	}
	flush()
	return blocks
}

var qaMarkerRe = regexp.MustCompile(`^(Q|A|意図)\s*(\d*)$`)

func parseQA(body string, hints ParseHints, out *ParseOutput) {
	type entry struct{ q, a, intent string }
	entries := map[int]*entry{}
	seq := map[string]int{}
	found := false
	for _, b := range splitBlocks(body) {
		m := qaMarkerRe.FindStringSubmatch(strings.ToUpper(b.name))
		if m == nil {
			continue
		}
		kind := m[1]
		n := 0
		if m[2] != "" {
			n, _ = strconv.Atoi(m[2])
		} else {
			// Unnumbered markers are numbered by their order of appearance.
			seq[kind]++
			n = seq[kind]
		}
		e, ok := entries[n]
		if !ok {
			e = &entry{}
			entries[n] = e
		}
		found = true
		switch kind {
		case "Q":
			e.q = b.content
		case "A":
			e.a = b.content
		case "意図":
			e.intent = b.content
		}
	}
	if !found {
		out.Result = vo.ParseFailed
		out.Warnings = append(out.Warnings, "【Q】【A】の目印が見つかりません。全文を手動で分割してください")
		return
	}
	keys := make([]int, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out.Result = vo.ParseSuccess
	for _, k := range keys {
		e := entries[k]
		if e.q == "" {
			out.Result = vo.ParsePartial
			out.Warnings = append(out.Warnings, fmt.Sprintf("【Q%d】が見つかりません", k))
		}
		if e.a == "" {
			out.Result = vo.ParsePartial
			out.Warnings = append(out.Warnings, fmt.Sprintf("【A%d】が見つかりません", k))
		}
		if e.intent == "" && e.q != "" {
			out.Warnings = append(out.Warnings, fmt.Sprintf("【意図%d】が見つかりません", k))
		}
		out.Preview.Items = append(out.Preview.Items, QAItem{QuestionText: e.q, AnswerText: e.a, IntentText: e.intent})
	}
	if hints.QuestionCount > 0 && len(keys) != hints.QuestionCount {
		out.Warnings = append(out.Warnings, fmt.Sprintf("質問数が指定（%d問）と違います（%d問）", hints.QuestionCount, len(keys)))
	}
}

func listItems(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimSpace(bulletRe.ReplaceAllString(line, ""))
		line = strings.TrimSpace(strings.ReplaceAll(line, "**", ""))
		if line != "" {
			items = append(items, line)
		}
	}
	return items
}

func parseReview(body string, out *ParseOutput) {
	var issues, improvements []string
	var improved string
	var hasIssues, hasImprovements, hasImproved bool
	for _, b := range splitBlocks(body) {
		switch b.name {
		case "指摘":
			hasIssues = true
			issues = listItems(b.content)
		case "改善案":
			hasImprovements = true
			improvements = listItems(b.content)
		case "改善後の回答":
			hasImproved = true
			improved = b.content
		}
	}
	out.Preview.Issues = issues
	out.Preview.Improvements = improvements
	out.Preview.ImprovedAnswerText = improved
	out.Result = resultOf([]bool{hasIssues, hasImprovements, hasImproved})
	if !hasIssues {
		out.Warnings = append(out.Warnings, "【指摘】が見つかりません")
	}
	if !hasImprovements {
		out.Warnings = append(out.Warnings, "【改善案】が見つかりません")
	}
	if !hasImproved {
		out.Warnings = append(out.Warnings, "【改善後の回答】が見つかりません")
	}
	if out.Result == vo.ParseFailed {
		out.Warnings = []string{"【指摘】【改善案】【改善後の回答】の目印が見つかりません。全文を手動で分割してください"}
	}
}

func parseResearch(body string, out *ParseOutput) {
	var has [3]bool
	for _, b := range splitBlocks(body) {
		switch b.name {
		case "魅力・特徴", "魅力", "特徴":
			has[0] = true
			out.Preview.Highlights = listItems(b.content)
		case "求める人物像":
			has[1] = true
			out.Preview.IdealCandidate = b.content
		case "最近のトピック":
			has[2] = true
			out.Preview.RecentTopics = listItems(b.content)
		}
	}
	out.Result = resultOf(has[:])
	names := []string{"【魅力・特徴】", "【求める人物像】", "【最近のトピック】"}
	for i, ok := range has {
		if !ok {
			out.Warnings = append(out.Warnings, names[i]+"が見つかりません")
		}
	}
	if out.Result == vo.ParseFailed {
		out.Warnings = []string{"【魅力・特徴】【求める人物像】【最近のトピック】の目印が見つかりません。全文を手動で分割してください"}
		return
	}
	out.Preview.Text = FormatResearchSummary(out.Preview.Highlights, out.Preview.IdealCandidate, out.Preview.RecentTopics)
}

// FormatResearchSummary renders the research summary saved to
// research_materials.summary with the three headings (4.2.4).
func FormatResearchSummary(highlights []string, ideal string, topics []string) string {
	var b strings.Builder
	b.WriteString("■ 魅力・特徴\n")
	for _, h := range highlights {
		b.WriteString("- " + h + "\n")
	}
	b.WriteString("■ 求める人物像\n" + strings.TrimSpace(ideal) + "\n")
	b.WriteString("■ 最近のトピック\n")
	for _, t := range topics {
		b.WriteString("- " + t + "\n")
	}
	return strings.TrimSpace(b.String())
}

func resultOf(found []bool) vo.ParseResult {
	n := 0
	for _, f := range found {
		if f {
			n++
		}
	}
	switch n {
	case 0:
		return vo.ParseFailed
	case len(found):
		return vo.ParseSuccess
	default:
		return vo.ParsePartial
	}
}

func lengthWarnings(text string, limit *int) []string {
	if limit == nil || *limit <= 0 {
		return nil
	}
	n := utf8.RuneCountInString(strings.TrimSpace(text))
	lo := float64(*limit) * 0.9
	hi := float64(*limit) * 1.1
	if float64(n) < lo || float64(n) > hi {
		return []string{fmt.Sprintf("文字数が指定（%d字の±10%%）の範囲外です（%d字）", *limit, n)}
	}
	return nil
}
