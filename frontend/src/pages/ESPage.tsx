// DesignDocument/3_UIUXDesign.md 3.5.4, 5_FrontendDesign.md 5.4.1 - ESプロジェクト一覧
import { useNavigate } from "react-router-dom";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useESProjects } from "../hooks/useES";
import { ES_STATUS_LABEL } from "../lib/constants";

const statusColor: Record<string, string> = {
  draft: "bg-gray-100 text-gray-700",
  in_review: "bg-yellow-100 text-yellow-700",
  completed: "bg-green-100 text-green-700",
  submitted: "bg-blue-100 text-blue-700",
};

export function ESPage() {
  const navigate = useNavigate();
  const { data: projects, isLoading, error, refetch } = useESProjects();

  return (
    <div className="animate-fade-in">
      <PageHeader
        title="ES作成"
        showBack
        rightAction={{
          label: "+新規",
          onClick: () => navigate("/es/new"),
        }}
      />

      {isLoading && <LoadingSpinner />}
      {error && <ErrorDisplay onRetry={refetch} />}

      {!isLoading && !error && projects?.length === 0 && (
        <EmptyState
          icon={
            <svg
              className="w-8 h-8"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
            >
              <path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          }
          title="ESプロジェクトがありません"
          description="企業ごとにESを管理し、AIで作成・推敲できます"
          actionLabel="ESを作成"
          actionTo="/es/new"
        />
      )}

      {projects && projects.length > 0 && (
        <div className="p-4 space-y-3">
          {projects.map((project) => (
            <button
              key={project.id}
              onClick={() => navigate(`/es/${project.id}`)}
              className="w-full bg-white rounded-xl p-4 text-left active:bg-gray-50 transition-colors shadow-sm"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900">
                    {project.company?.name ?? "企業未設定"}
                  </p>
                  <div className="flex items-center gap-2 mt-1.5">
                    <span
                      className={`px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor[project.status] ?? statusColor.draft}`}
                    >
                      {ES_STATUS_LABEL[project.status] ?? project.status}
                    </span>
                    {project.deadline && (
                      <span className="text-xs text-gray-400">
                        期限: {project.deadline}
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
