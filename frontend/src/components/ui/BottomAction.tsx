// DesignDocument/3_UIUXDesign.md 3.9 - 画面下固定CTAコンポーネント

interface BottomActionProps {
  label: string;
  onClick?: () => void;
  type?: "button" | "submit";
  disabled?: boolean;
  loading?: boolean;
}

export function BottomAction({
  label,
  onClick,
  type = "button",
  disabled,
  loading,
}: BottomActionProps) {
  return (
    <div className="fixed bottom-16 left-0 right-0 p-4 bg-gradient-to-t from-white via-white to-white/0 pointer-events-none">
      <div className="max-w-lg mx-auto pointer-events-auto">
        <button
          type={type}
          onClick={onClick}
          disabled={disabled || loading}
          className="w-full py-3.5 bg-orange-500 text-white text-base font-semibold rounded-xl hover:bg-orange-600 active:scale-[0.98] disabled:opacity-50 disabled:active:scale-100 transition-all shadow-lg shadow-orange-500/25"
        >
          {loading ? (
            <span className="flex items-center justify-center gap-2">
              <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              処理中...
            </span>
          ) : (
            label
          )}
        </button>
      </div>
    </div>
  );
}
