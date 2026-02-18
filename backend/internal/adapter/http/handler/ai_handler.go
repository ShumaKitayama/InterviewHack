// DesignDocument/7_BackendDesign.md 7.5.4 AI（OpenAI + クレジット消費）
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/usecase/ai"
)

type AIHandler struct {
	qaGenerateUC        *ai.QAGenerateUsecase
	answerReviewUC      *ai.AnswerReviewUsecase
	summaryGenerateUC   *ai.SummaryGenerateUsecase
	researchSummarizeUC *ai.ResearchSummarizeUsecase
	esGenerateUC        *ai.ESGenerateUsecase
	esRefineUC          *ai.ESRefineUsecase
}

func NewAIHandler(
	qaGenerateUC *ai.QAGenerateUsecase,
	answerReviewUC *ai.AnswerReviewUsecase,
	summaryGenerateUC *ai.SummaryGenerateUsecase,
	researchSummarizeUC *ai.ResearchSummarizeUsecase,
	esGenerateUC *ai.ESGenerateUsecase,
	esRefineUC *ai.ESRefineUsecase,
) *AIHandler {
	return &AIHandler{
		qaGenerateUC:        qaGenerateUC,
		answerReviewUC:      answerReviewUC,
		summaryGenerateUC:   summaryGenerateUC,
		researchSummarizeUC: researchSummarizeUC,
		esGenerateUC:        esGenerateUC,
		esRefineUC:          esRefineUC,
	}
}

// QAGenerate handles POST /ai/qa-generate
func (h *AIHandler) QAGenerate(c *gin.Context) {
	var req dto.QAGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.qaGenerateUC.Execute(c.Request.Context(), &ai.QAGenerateInput{
		UserID:        userID,
		CompanyID:     req.CompanyID,
		JobTitle:      req.JobTitle,
		AssetIDs:      req.AssetIDs,
		QuestionCount: req.QuestionCount,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.QAGenerateResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}

// AnswerReview handles POST /ai/answer-review
func (h *AIHandler) AnswerReview(c *gin.Context) {
	var req dto.AnswerReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.answerReviewUC.Execute(c.Request.Context(), &ai.AnswerReviewInput{
		UserID:       userID,
		QuestionText: req.QuestionText,
		AnswerText:   req.AnswerText,
		AssetIDs:     req.AssetIDs,
		Strictness:   req.Strictness,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.AnswerReviewResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}

// SummaryGenerate handles POST /ai/summary-generate
func (h *AIHandler) SummaryGenerate(c *gin.Context) {
	var req dto.SummaryGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.summaryGenerateUC.Execute(c.Request.Context(), &ai.SummaryGenerateInput{
		UserID:          userID,
		InterviewID:     req.InterviewID,
		InterviewLogIDs: req.InterviewLogIDs,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.SummaryGenerateResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}

// ResearchSummarize handles POST /ai/research-summarize
func (h *AIHandler) ResearchSummarize(c *gin.Context) {
	var req dto.ResearchSummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.researchSummarizeUC.Execute(c.Request.Context(), &ai.ResearchSummarizeInput{
		UserID:  userID,
		Type:    req.Type,
		Content: req.Content,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.ResearchSummarizeResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}

// ESGenerate handles POST /ai/es-generate
func (h *AIHandler) ESGenerate(c *gin.Context) {
	var req dto.ESGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.esGenerateUC.Execute(c.Request.Context(), &ai.ESGenerateInput{
		UserID:              userID,
		ESProjectID:         req.ESProjectID,
		SectionTitle:        req.SectionTitle,
		AssetIDs:            req.AssetIDs,
		ResearchMaterialIDs: req.ResearchMaterialIDs,
		CharacterLimit:      req.CharacterLimit,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.ESGenerateResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}

// ESRefine handles POST /ai/es-refine
func (h *AIHandler) ESRefine(c *gin.Context) {
	var req dto.ESRefineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		presenter.RespondError(c, err)
		return
	}

	userID := c.GetString("user_id")
	output, err := h.esRefineUC.Execute(c.Request.Context(), &ai.ESRefineInput{
		UserID:       userID,
		CurrentText:  req.CurrentText,
		Instructions: req.Instructions,
	})
	if err != nil {
		presenter.RespondError(c, err)
		return
	}

	presenter.RespondOK(c, dto.ESRefineResponse{
		GenerationID: output.GenerationID,
		RawContent:   output.RawContent,
	})
}
