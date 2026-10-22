package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/interviewhack/backend/internal/adapter/http/dto"
	"github.com/interviewhack/backend/internal/adapter/http/presenter"
	"github.com/interviewhack/backend/internal/domain/entity"
	domainerror "github.com/interviewhack/backend/internal/domain/error"
	"github.com/interviewhack/backend/internal/port/repository"
	"github.com/interviewhack/backend/internal/usecase/asset"
	"github.com/interviewhack/backend/internal/usecase/company"
	"github.com/interviewhack/backend/internal/usecase/es"
	"github.com/interviewhack/backend/internal/usecase/interview"
	"github.com/interviewhack/backend/internal/usecase/qa"
	"github.com/interviewhack/backend/internal/usecase/research"
)

// ---- Account (7.5.2) ----

// GetMe is GET /me.
func (h *Handler) GetMe(c *gin.Context) {
	u, err := h.Account.Get(c.Request.Context(), userID(c))
	respond(c, u, err, dto.MeFrom, false)
}

// PatchMe is PATCH /me.
func (h *Handler) PatchMe(c *gin.Context) {
	var in struct {
		DisplayName dto.Opt[string] `json:"displayName"`
	}
	if !bind(c, &in) {
		return
	}
	u, err := h.Account.UpdateDisplayName(c.Request.Context(), userID(c), dto.StrPtr(in.DisplayName))
	respond(c, u, err, dto.MeFrom, false)
}

// DeleteMe is DELETE /me (退会).
func (h *Handler) DeleteMe(c *gin.Context) {
	respondNoContent(c, h.Account.Delete(c.Request.Context(), userID(c)))
}

// ---- Assets / tags ----

func assetInput(c *gin.Context, in dto.AssetInput) (asset.Input, bool) {
	out := asset.Input{Type: dto.StrPtr(in.Type), Title: dto.StrPtr(in.Title), Content: dto.StrPtr(in.Content)}
	if in.TagIDs.Set {
		ids := []string{}
		if in.TagIDs.Val != nil {
			ids = *in.TagIDs.Val
		}
		if !uuidList(c, "tagIds", ids) {
			return out, false
		}
		out.TagIDs = &ids
	}
	return out, true
}

// ListAssets is GET /assets.
func (h *Handler) ListAssets(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	tagID, ok := uuidQuery(c, "tagId")
	if !ok {
		return
	}
	page, err := h.Assets.List(c.Request.Context(), userID(c), repository.AssetFilter{
		Type: c.Query("type"), TagID: tagID, Query: c.Query("q"), SortBy: c.Query("sort"),
	}, p)
	respondList(c, page, err, dto.AssetFrom)
}

// CreateAsset is POST /assets.
func (h *Handler) CreateAsset(c *gin.Context) {
	var in dto.AssetInput
	if !bind(c, &in) {
		return
	}
	ai, ok := assetInput(c, in)
	if !ok {
		return
	}
	a, err := h.Assets.Create(c.Request.Context(), userID(c), ai)
	respond(c, a, err, dto.AssetFrom, true)
}

// GetAsset is GET /assets/:id.
func (h *Handler) GetAsset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	a, err := h.Assets.Get(c.Request.Context(), userID(c), id)
	respond(c, a, err, dto.AssetFrom, false)
}

// PatchAsset is PATCH /assets/:id.
func (h *Handler) PatchAsset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.AssetInput
	if !bind(c, &in) {
		return
	}
	ai, ok := assetInput(c, in)
	if !ok {
		return
	}
	a, err := h.Assets.Update(c.Request.Context(), userID(c), id, ai)
	respond(c, a, err, dto.AssetFrom, false)
}

// DeleteAsset is DELETE /assets/:id.
func (h *Handler) DeleteAsset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Assets.Delete(c.Request.Context(), userID(c), id))
}

func tagFrom(t entity.AssetTag) dto.Tag { return dto.Tag{ID: t.ID, Name: t.Name} }

// ListTags is GET /asset-tags.
func (h *Handler) ListTags(c *gin.Context) {
	tags, err := h.Assets.ListTags(c.Request.Context(), userID(c))
	respondList(c, repository.Page[entity.AssetTag]{Items: tags}, err, tagFrom)
}

// CreateTag is POST /asset-tags.
func (h *Handler) CreateTag(c *gin.Context) {
	var in dto.TagInput
	if !bind(c, &in) {
		return
	}
	t, err := h.Assets.CreateTag(c.Request.Context(), userID(c), in.Name)
	respond(c, t, err, tagFrom, true)
}

// PatchTag is PATCH /asset-tags/:id.
func (h *Handler) PatchTag(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.TagInput
	if !bind(c, &in) {
		return
	}
	t, err := h.Assets.UpdateTag(c.Request.Context(), userID(c), id, in.Name)
	respond(c, t, err, tagFrom, false)
}

// GetTag is GET /asset-tags/:id.
func (h *Handler) GetTag(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	tags, err := h.Assets.ListTags(c.Request.Context(), userID(c))
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	for _, t := range tags {
		if t.ID == id {
			presenter.RespondOK(c, tagFrom(t))
			return
		}
	}
	presenter.RespondError(c, domainerror.ErrNotFound("タグ"))
}

// DeleteTag is DELETE /asset-tags/:id.
func (h *Handler) DeleteTag(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Assets.DeleteTag(c.Request.Context(), userID(c), id))
}

// ---- Companies ----

func companyInput(in dto.CompanyInput) company.Input {
	return company.Input{Name: dto.StrPtr(in.Name), JobTitle: dto.StrPtr(in.JobTitle), SelectionPhase: dto.StrPtr(in.SelectionPhase),
		MotivationMemo: dto.StrPtr(in.MotivationMemo), ReferenceURL: dto.StrPtr(in.ReferenceURL)}
}

// ListCompanies is GET /companies.
func (h *Handler) ListCompanies(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	page, err := h.Companies.List(c.Request.Context(), userID(c), repository.CompanyFilter{Query: c.Query("q"), SelectionPhase: c.Query("selectionPhase")}, p)
	respondList(c, page, err, dto.CompanyFrom)
}

// CreateCompany is POST /companies.
func (h *Handler) CreateCompany(c *gin.Context) {
	var in dto.CompanyInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Companies.Create(c.Request.Context(), userID(c), companyInput(in))
	respond(c, v, err, dto.CompanyFrom, true)
}

// GetCompany is GET /companies/:id.
func (h *Handler) GetCompany(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.Companies.Get(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.CompanyFrom, false)
}

// PatchCompany is PATCH /companies/:id.
func (h *Handler) PatchCompany(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.CompanyInput
	if !bind(c, &in) {
		return
	}
	v, err := h.Companies.Update(c.Request.Context(), userID(c), id, companyInput(in))
	respond(c, v, err, dto.CompanyFrom, false)
}

// DeleteCompany is DELETE /companies/:id.
func (h *Handler) DeleteCompany(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Companies.Delete(c.Request.Context(), userID(c), id))
}

// ---- Research materials ----

func researchInput(in dto.ResearchInput) research.Input {
	return research.Input{CompanyID: in.CompanyID.Val, Title: dto.StrPtr(in.Title), Type: dto.StrPtr(in.Type),
		SourceURL: dto.StrPtr(in.SourceURL), ContentRaw: dto.StrPtr(in.ContentRaw), Summary: dto.StrPtr(in.Summary)}
}

// ListResearch is GET /research-materials?companyId=.
func (h *Handler) ListResearch(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	companyID := c.Query("companyId")
	if !dto.IsUUID(companyID) {
		presenter.RespondError(c, domainerror.ErrValidation("companyId は必須です"))
		return
	}
	page, err := h.Research.List(c.Request.Context(), userID(c), companyID, p)
	respondList(c, page, err, dto.ResearchFrom)
}

// CreateResearch is POST /research-materials.
func (h *Handler) CreateResearch(c *gin.Context) {
	var in dto.ResearchInput
	if !bind(c, &in) || !uuidOpt(c, "companyId", in.CompanyID) {
		return
	}
	v, err := h.Research.Create(c.Request.Context(), userID(c), researchInput(in))
	respond(c, v, err, dto.ResearchFrom, true)
}

// GetResearch is GET /research-materials/:id.
func (h *Handler) GetResearch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.Research.Get(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.ResearchFrom, false)
}

// PatchResearch is PATCH /research-materials/:id.
func (h *Handler) PatchResearch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.ResearchInput
	if !bind(c, &in) {
		return
	}
	ri := researchInput(in)
	ri.CompanyID = nil
	v, err := h.Research.Update(c.Request.Context(), userID(c), id, ri)
	respond(c, v, err, dto.ResearchFrom, false)
}

// DeleteResearch is DELETE /research-materials/:id.
func (h *Handler) DeleteResearch(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Research.Delete(c.Request.Context(), userID(c), id))
}

// ---- ES projects / sections / versions ----

func esProjectInput(c *gin.Context, in dto.ESProjectInput) (es.ProjectInput, bool) {
	out := es.ProjectInput{CompanyID: in.CompanyID.Val, Status: in.Status.Val}
	if !uuidOpt(c, "companyId", in.CompanyID) {
		return out, false
	}
	if in.Deadline.Set {
		var d *time.Time
		if in.Deadline.Val != nil && *in.Deadline.Val != "" {
			t, ok := dto.ParseDate(*in.Deadline.Val)
			if !ok {
				presenter.RespondError(c, domainerror.ErrValidation("deadline は YYYY-MM-DD 形式で指定してください"))
				return out, false
			}
			d = &t
		}
		out.Deadline = &d
	}
	for _, f := range []struct {
		name string
		opt  dto.Opt[[]string]
		dst  **[]string
	}{{"assetIds", in.AssetIDs, &out.AssetIDs}, {"researchMaterialIds", in.ResearchMaterialIDs, &out.ResearchMaterialIDs}} {
		if !f.opt.Set {
			continue
		}
		ids := []string{}
		if f.opt.Val != nil {
			ids = *f.opt.Val
		}
		if !uuidList(c, f.name, ids) {
			return out, false
		}
		*f.dst = &ids
	}
	return out, true
}

// ListESProjects is GET /es-projects.
func (h *Handler) ListESProjects(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	companyID, ok := uuidQuery(c, "companyId")
	if !ok {
		return
	}
	page, err := h.ES.ListProjects(c.Request.Context(), userID(c), repository.ESFilter{CompanyID: companyID, Status: c.Query("status")}, p)
	respondList(c, page, err, dto.ESProjectFrom)
}

// CreateESProject is POST /es-projects.
func (h *Handler) CreateESProject(c *gin.Context) {
	var in dto.ESProjectInput
	if !bind(c, &in) {
		return
	}
	pi, ok := esProjectInput(c, in)
	if !ok {
		return
	}
	v, err := h.ES.CreateProject(c.Request.Context(), userID(c), pi)
	respond(c, v, err, dto.ESProjectFrom, true)
}

// GetESProject is GET /es-projects/:id.
func (h *Handler) GetESProject(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.ES.GetProject(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.ESProjectFrom, false)
}

// PatchESProject is PATCH /es-projects/:id.
func (h *Handler) PatchESProject(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.ESProjectInput
	if !bind(c, &in) {
		return
	}
	pi, ok := esProjectInput(c, in)
	if !ok {
		return
	}
	v, err := h.ES.UpdateProject(c.Request.Context(), userID(c), id, pi)
	respond(c, v, err, dto.ESProjectFrom, false)
}

// DeleteESProject is DELETE /es-projects/:id.
func (h *Handler) DeleteESProject(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.ES.DeleteProject(c.Request.Context(), userID(c), id))
}

func sectionInput(in dto.ESSectionInput) es.SectionInput {
	return es.SectionInput{Title: dto.StrPtr(in.Title), ContentDraft: dto.StrPtr(in.ContentDraft), ContentFinal: dto.StrPtr(in.ContentFinal),
		CharacterLimit: dto.Nullable(in.CharacterLimit)}
}

// ListESSections is GET /es-projects/:id/sections.
func (h *Handler) ListESSections(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	items, err := h.ES.ListSections(c.Request.Context(), userID(c), id)
	respondList(c, repository.Page[entity.ESSection]{Items: items}, err, dto.ESSectionFrom)
}

// CreateESSection is POST /es-projects/:id/sections.
func (h *Handler) CreateESSection(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.ESSectionInput
	if !bind(c, &in) {
		return
	}
	v, err := h.ES.CreateSection(c.Request.Context(), userID(c), id, sectionInput(in))
	respond(c, v, err, dto.ESSectionFrom, true)
}

// GetESSection is GET /es-sections/:id.
func (h *Handler) GetESSection(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.ES.GetSection(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.ESSectionFrom, false)
}

// PatchESSection is PATCH /es-sections/:id.
func (h *Handler) PatchESSection(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.ESSectionInput
	if !bind(c, &in) {
		return
	}
	v, err := h.ES.UpdateSection(c.Request.Context(), userID(c), id, sectionInput(in))
	respond(c, v, err, dto.ESSectionFrom, false)
}

// DeleteESSection is DELETE /es-sections/:id.
func (h *Handler) DeleteESSection(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.ES.DeleteSection(c.Request.Context(), userID(c), id))
}

// ListVersions is GET /es-sections/:id/versions.
func (h *Handler) ListVersions(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	items, err := h.ES.ListVersions(c.Request.Context(), userID(c), id)
	if err != nil {
		presenter.RespondError(c, err)
		return
	}
	presenter.RespondOK(c, gin.H{"items": dto.Map(items, dto.ESSectionVersionFrom)})
}

// SaveVersion is POST /es-sections/:id/versions.
func (h *Handler) SaveVersion(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.ES.SaveVersion(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.ESSectionVersionFrom, true)
}

// RestoreVersion is POST /es-sections/:id/versions/:versionId/restore.
func (h *Handler) RestoreVersion(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	vid, ok := pathID(c, "versionId")
	if !ok {
		return
	}
	v, err := h.ES.RestoreVersion(c.Request.Context(), userID(c), id, vid)
	respond(c, v, err, dto.ESSectionFrom, false)
}

// ---- Interviews / logs ----

func interviewInput(c *gin.Context, in dto.InterviewInput) (interview.InterviewInput, bool) {
	out := interview.InterviewInput{Phase: dto.StrPtr(in.Phase), Result: dto.StrPtr(in.Result), Memo: dto.StrPtr(in.Memo)}
	if !uuidOpt(c, "companyId", in.CompanyID) {
		return out, false
	}
	out.CompanyID = dto.Nullable(in.CompanyID)
	if in.InterviewDate.Set {
		if in.InterviewDate.Val == nil {
			presenter.RespondError(c, domainerror.ErrValidation("interviewDate は必須です"))
			return out, false
		}
		t, ok := dto.ParseDate(*in.InterviewDate.Val)
		if !ok {
			presenter.RespondError(c, domainerror.ErrValidation("interviewDate は YYYY-MM-DD 形式で指定してください"))
			return out, false
		}
		out.InterviewDate = &t
	}
	return out, true
}

// ListInterviews is GET /interviews.
func (h *Handler) ListInterviews(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	companyID, ok := uuidQuery(c, "companyId")
	if !ok {
		return
	}
	f := repository.InterviewFilter{CompanyID: companyID}
	for _, q := range []struct {
		name string
		dst  **time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if v := c.Query(q.name); v != "" {
			t, ok := dto.ParseDate(v)
			if !ok {
				presenter.RespondError(c, domainerror.ErrValidation(q.name+" は YYYY-MM-DD 形式で指定してください"))
				return
			}
			*q.dst = &t
		}
	}
	page, err := h.Interviews.ListInterviews(c.Request.Context(), userID(c), f, p)
	respondList(c, page, err, dto.InterviewFrom)
}

// CreateInterview is POST /interviews.
func (h *Handler) CreateInterview(c *gin.Context) {
	var in dto.InterviewInput
	if !bind(c, &in) {
		return
	}
	ii, ok := interviewInput(c, in)
	if !ok {
		return
	}
	v, err := h.Interviews.CreateInterview(c.Request.Context(), userID(c), ii)
	respond(c, v, err, dto.InterviewFrom, true)
}

// GetInterview is GET /interviews/:id.
func (h *Handler) GetInterview(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.Interviews.GetInterview(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.InterviewFrom, false)
}

// PatchInterview is PATCH /interviews/:id.
func (h *Handler) PatchInterview(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.InterviewInput
	if !bind(c, &in) {
		return
	}
	ii, ok := interviewInput(c, in)
	if !ok {
		return
	}
	v, err := h.Interviews.UpdateInterview(c.Request.Context(), userID(c), id, ii)
	respond(c, v, err, dto.InterviewFrom, false)
}

// DeleteInterview is DELETE /interviews/:id.
func (h *Handler) DeleteInterview(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Interviews.DeleteInterview(c.Request.Context(), userID(c), id))
}

func logInput(in dto.InterviewLogInput) interview.LogInput {
	return interview.LogInput{InterviewID: in.InterviewID.Val, QuestionText: dto.StrPtr(in.QuestionText), AnswerText: dto.StrPtr(in.AnswerText),
		SelfEvaluation: dto.StrPtr(in.SelfEvaluation), AIFeedbackText: dto.StrPtr(in.AIFeedbackText),
		AIImprovedAnswerText: dto.StrPtr(in.AIImprovedAnswerText), Score: dto.Nullable(in.Score)}
}

// ListInterviewLogs is GET /interview-logs?interviewId=.
func (h *Handler) ListInterviewLogs(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	interviewID := c.Query("interviewId")
	if !dto.IsUUID(interviewID) {
		presenter.RespondError(c, domainerror.ErrValidation("interviewId は必須です"))
		return
	}
	page, err := h.Interviews.ListLogs(c.Request.Context(), userID(c), interviewID, p)
	respondList(c, page, err, dto.InterviewLogFrom)
}

// CreateInterviewLog is POST /interview-logs.
func (h *Handler) CreateInterviewLog(c *gin.Context) {
	var in dto.InterviewLogInput
	if !bind(c, &in) || !uuidOpt(c, "interviewId", in.InterviewID) {
		return
	}
	v, err := h.Interviews.CreateLog(c.Request.Context(), userID(c), logInput(in))
	respond(c, v, err, dto.InterviewLogFrom, true)
}

// GetInterviewLog is GET /interview-logs/:id.
func (h *Handler) GetInterviewLog(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.Interviews.GetLog(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.InterviewLogFrom, false)
}

// PatchInterviewLog is PATCH /interview-logs/:id.
func (h *Handler) PatchInterviewLog(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.InterviewLogInput
	if !bind(c, &in) {
		return
	}
	li := logInput(in)
	li.InterviewID = nil
	v, err := h.Interviews.UpdateLog(c.Request.Context(), userID(c), id, li)
	respond(c, v, err, dto.InterviewLogFrom, false)
}

// DeleteInterviewLog is DELETE /interview-logs/:id.
func (h *Handler) DeleteInterviewLog(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.Interviews.DeleteLog(c.Request.Context(), userID(c), id))
}

// ---- Q&A ----

func qaInput(c *gin.Context, in dto.QAInput) (qa.Input, bool) {
	out := qa.Input{QuestionText: dto.StrPtr(in.QuestionText), AnswerText: dto.StrPtr(in.AnswerText), SourceType: in.SourceType.Val,
		IsFavorite: in.IsFavorite.Val, ReviewStatus: in.ReviewStatus.Val}
	if !uuidOpt(c, "companyId", in.CompanyID) {
		return out, false
	}
	out.CompanyID = dto.Nullable(in.CompanyID)
	if in.AssetIDs.Set {
		ids := []string{}
		if in.AssetIDs.Val != nil {
			ids = *in.AssetIDs.Val
		}
		if !uuidList(c, "assetIds", ids) {
			return out, false
		}
		out.AssetIDs = &ids
	}
	return out, true
}

// ListQA is GET /qa.
func (h *Handler) ListQA(c *gin.Context) {
	p, ok := pageParams(c)
	if !ok {
		return
	}
	companyID, ok := uuidQuery(c, "companyId")
	if !ok {
		return
	}
	f := repository.QAFilter{CompanyID: companyID, Query: c.Query("q"), ReviewStatus: c.Query("reviewStatus")}
	switch c.Query("isFavorite") {
	case "true":
		t := true
		f.IsFavorite = &t
	case "false":
		v := false
		f.IsFavorite = &v
	case "":
	default:
		presenter.RespondError(c, domainerror.ErrValidation("isFavorite は true / false で指定してください"))
		return
	}
	page, err := h.QA.List(c.Request.Context(), userID(c), f, p)
	respondList(c, page, err, dto.QAFrom)
}

// CreateQA is POST /qa.
func (h *Handler) CreateQA(c *gin.Context) {
	var in dto.QAInput
	if !bind(c, &in) {
		return
	}
	qi, ok := qaInput(c, in)
	if !ok {
		return
	}
	v, err := h.QA.Create(c.Request.Context(), userID(c), qi)
	respond(c, v, err, dto.QAFrom, true)
}

// GetQA is GET /qa/:id.
func (h *Handler) GetQA(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	v, err := h.QA.Get(c.Request.Context(), userID(c), id)
	respond(c, v, err, dto.QAFrom, false)
}

// PatchQA is PATCH /qa/:id.
func (h *Handler) PatchQA(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var in dto.QAInput
	if !bind(c, &in) {
		return
	}
	qi, ok := qaInput(c, in)
	if !ok {
		return
	}
	v, err := h.QA.Update(c.Request.Context(), userID(c), id, qi)
	respond(c, v, err, dto.QAFrom, false)
}

// DeleteQA is DELETE /qa/:id.
func (h *Handler) DeleteQA(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	respondNoContent(c, h.QA.Delete(c.Request.Context(), userID(c), id))
}
