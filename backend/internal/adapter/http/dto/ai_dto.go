// DesignDocument/7_BackendDesign.md 7.5.4 AI（OpenAI + クレジット消費）
package dto

// --- QA Generate ---

type QAGenerateRequest struct {
	CompanyID     string   `json:"companyId" binding:"required"`
	CompanyName   string   `json:"companyName"`
	JobTitle      string   `json:"jobTitle"`
	AssetIDs      []string `json:"assetIds"`
	AssetTexts    []string `json:"assetTexts"`
	QuestionCount int      `json:"questionCount"`
}

type QAItem struct {
	QuestionText       string   `json:"questionText"`
	AnswerText         string   `json:"answerText"`
	ReferencedAssetIDs []string `json:"referencedAssetIds"`
}

type QAGenerateResponse struct {
	GenerationID string   `json:"generationId"`
	Items        []QAItem `json:"items"`
}

// --- Answer Review ---

type AnswerReviewRequest struct {
	QuestionText string   `json:"questionText" binding:"required"`
	AnswerText   string   `json:"answerText" binding:"required"`
	AssetIDs     []string `json:"assetIds"`
	AssetTexts   []string `json:"assetTexts"`
	Strictness   string   `json:"strictness"`
}

type AnswerReviewResponse struct {
	GenerationID       string   `json:"generationId"`
	Issues             []string `json:"issues"`
	Improvements       []string `json:"improvements"`
	ImprovedAnswerText string   `json:"improvedAnswerText"`
	ReferencedAssetIDs []string `json:"referencedAssetIds"`
}

// --- Summary Generate ---

type SummaryGenerateRequest struct {
	InterviewID     string   `json:"interviewId"`
	InterviewLogIDs []string `json:"interviewLogIds"`
	LogTexts        []string `json:"logTexts"`
}

type SummaryGenerateResponse struct {
	GenerationID   string   `json:"generationId"`
	SummaryBullets []string `json:"summaryBullets"`
}

// --- Research Summarize ---

type ResearchSummarizeRequest struct {
	Type    string `json:"type" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type ResearchSummarizeResponse struct {
	GenerationID    string   `json:"generationId"`
	Summary         string   `json:"summary"`
	KeyPoints       []string `json:"keyPoints"`
	UsageSuggestion string   `json:"usageSuggestion"`
}

// --- ES Generate ---

type ESGenerateRequest struct {
	ESProjectID         string   `json:"esProjectId" binding:"required"`
	SectionTitle        string   `json:"sectionTitle" binding:"required"`
	AssetIDs            []string `json:"assetIds"`
	AssetTexts          []string `json:"assetTexts"`
	ResearchMaterialIDs []string `json:"researchMaterialIds"`
	ResearchTexts       []string `json:"researchTexts"`
	CompanyName         string   `json:"companyName"`
	JobTitle            string   `json:"jobTitle"`
	CharacterLimit      int      `json:"characterLimit" binding:"required"`
}

type ESGenerateResponse struct {
	GenerationID string `json:"generationId"`
	DraftText    string `json:"draftText"`
	Rationale    string `json:"rationale"`
}

// --- ES Refine ---

type ESRefineRequest struct {
	CurrentText  string `json:"currentText" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
}

type ESRefineResponse struct {
	GenerationID string   `json:"generationId"`
	RefinedText  string   `json:"refinedText"`
	Changes      []string `json:"changes"`
}
