// DesignDocument/7_BackendDesign.md 7.5.4 AI（OpenAI + クレジット消費）
package dto

type QAGenerateRequest struct {
	CompanyID     string   `json:"companyId" binding:"required"`
	JobTitle      string   `json:"jobTitle"`
	AssetIDs      []string `json:"assetIds"`
	QuestionCount int      `json:"questionCount"`
}

type QAGenerateResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}

type AnswerReviewRequest struct {
	QuestionText string   `json:"questionText" binding:"required"`
	AnswerText   string   `json:"answerText" binding:"required"`
	AssetIDs     []string `json:"assetIds"`
	Strictness   string   `json:"strictness"`
}

type AnswerReviewResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}

type SummaryGenerateRequest struct {
	InterviewID     string   `json:"interviewId"`
	InterviewLogIDs []string `json:"interviewLogIds"`
}

type SummaryGenerateResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}

type ResearchSummarizeRequest struct {
	Type    string `json:"type" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type ResearchSummarizeResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}

type ESGenerateRequest struct {
	ESProjectID         string   `json:"esProjectId" binding:"required"`
	SectionTitle        string   `json:"sectionTitle" binding:"required"`
	AssetIDs            []string `json:"assetIds" binding:"required"`
	ResearchMaterialIDs []string `json:"researchMaterialIds" binding:"required"`
	CharacterLimit      int      `json:"characterLimit" binding:"required"`
}

type ESGenerateResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}

type ESRefineRequest struct {
	CurrentText  string `json:"currentText" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
}

type ESRefineResponse struct {
	GenerationID string `json:"generationId"`
	RawContent   string `json:"rawContent"`
}
