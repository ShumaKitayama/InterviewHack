// DesignDocument/3_UIUXDesign.md - チップ選択コンポーネント

interface ChipSelectorProps<T extends string> {
  options: readonly { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
}

export function ChipSelector<T extends string>({
  options,
  value,
  onChange,
}: ChipSelectorProps<T>) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => {
        const isSelected = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`px-4 py-2 rounded-full text-sm font-medium transition-all ${
              isSelected
                ? "bg-gray-900 text-white"
                : "bg-white text-gray-700 border border-gray-200 hover:border-gray-400"
            }`}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
