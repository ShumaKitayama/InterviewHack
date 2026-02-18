// DesignDocument/7_BackendDesign.md 7.9 / DesignDocument/8_DBDesign.dbml generation_jobs.job_type
package vo

type JobType string

const (
	JobTypeQAGenerate        JobType = "qa_generate"
	JobTypeAnswerReview      JobType = "answer_review"
	JobTypeSummaryGenerate   JobType = "summary_generate"
	JobTypeResearchSummarize JobType = "research_summarize"
	JobTypeESGenerate        JobType = "es_generate"
	JobTypeESRefine          JobType = "es_refine"
)

func (j JobType) String() string {
	return string(j)
}

func (j JobType) IsValid() bool {
	switch j {
	case JobTypeQAGenerate, JobTypeAnswerReview, JobTypeSummaryGenerate,
		JobTypeResearchSummarize, JobTypeESGenerate, JobTypeESRefine:
		return true
	}
	return false
}
