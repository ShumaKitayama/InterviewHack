// DesignDocument/5_FrontendDesign.md 5.4.1, 3_UIUXDesign.md 3.3.2 - 企業一覧
import { useNavigate } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { useCompanies } from "../hooks/useCompanies";
import { SELECTION_PHASE_LABEL } from "../lib/constants";

const phaseColorMap: Record<string, string> = {
  researching: "bg-gray-100 text-gray-700",
  es_submitted: "bg-blue-100 text-blue-700",
  first_interview: "bg-cyan-100 text-cyan-700",
  second_interview: "bg-indigo-100 text-indigo-700",
  final_interview: "bg-purple-100 text-purple-700",
  offer: "bg-green-100 text-green-700",
  rejected: "bg-red-100 text-red-700",
};

export function CompaniesPage() {
  const navigate = useNavigate();
  const { data: companies, isLoading, error, refetch } = useCompanies();

  return (
    <div className="animate-fade-in">
      <div className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
        <div className="flex items-center justify-between px-4 h-14">
          <h1 className="text-lg font-bold text-gray-900">企業DB</h1>
          <button
            onClick={() => navigate("/companies/new")}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-orange-500 text-white hover:bg-orange-600 transition-colors"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={2.5}
            >
              <path d="M12 4v16m8-8H4" />
            </svg>
          </button>
        </div>
      </div>

      {isLoading && <LoadingSpinner />}
      {error && <ErrorDisplay onRetry={refetch} />}

      {!isLoading && !error && companies?.length === 0 && (
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
            </svg>
          }
          title="企業がまだ登録されていません"
          description="志望企業を登録して、面接対策やES作成に活用しましょう"
          actionLabel="企業を追加"
          actionTo="/companies/new"
        />
      )}

      {companies && companies.length > 0 && (
        <div className="p-4 space-y-3">
          {companies.map((company) => (
            <button
              key={company.id}
              onClick={() => navigate(`/companies/${company.id}`)}
              className="w-full bg-white rounded-xl p-4 text-left active:bg-gray-50 transition-colors shadow-sm"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900">
                    {company.name}
                  </p>
                  {company.job_title && (
                    <p className="text-xs text-gray-500 mt-0.5">
                      {company.job_title}
                    </p>
                  )}
                  <div className="flex items-center gap-2 mt-2">
                    {company.selection_phase && (
                      <span
                        className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-medium ${phaseColorMap[company.selection_phase] ?? phaseColorMap.researching}`}
                      >
                        {SELECTION_PHASE_LABEL[company.selection_phase] ??
                          company.selection_phase}
                      </span>
                    )}
                  </div>
                </div>
                <svg
                  className="w-4 h-4 text-gray-300 ml-2 mt-1 flex-shrink-0"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
