// DesignDocument/3_UIUXDesign.md - ページヘッダーコンポーネント
import { useNavigate } from "react-router-dom";

interface PageHeaderProps {
  title: string;
  showBack?: boolean;
  rightAction?: {
    label: string;
    onClick: () => void;
    disabled?: boolean;
  };
}

export function PageHeader({ title, showBack, rightAction }: PageHeaderProps) {
  const navigate = useNavigate();

  return (
    <div className="sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-gray-100">
      <div className="flex items-center justify-between px-4 h-14">
        {showBack ? (
          <button
            onClick={() => navigate(-1)}
            className="flex items-center gap-1 text-gray-600 -ml-1 p-1"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
          </button>
        ) : (
          <div className="w-8" />
        )}
        <h1 className="text-base font-semibold text-gray-900">{title}</h1>
        {rightAction ? (
          <button
            onClick={rightAction.onClick}
            disabled={rightAction.disabled}
            className="text-sm font-semibold text-orange-500 disabled:opacity-40 px-2 py-1 rounded-lg hover:bg-orange-50 transition-colors"
          >
            {rightAction.label}
          </button>
        ) : (
          <div className="w-8" />
        )}
      </div>
    </div>
  );
}
