// DesignDocument/3_UIUXDesign.md 3.4.1, 3.5.2 - 面接対策ルーム
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { BottomAction } from "../components/ui/BottomAction";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import { useAssets } from "../hooks/useAssets";
import { useCompanies } from "../hooks/useCompanies";
import { api } from "../lib/api/client";

export function PrepPage() {
  const navigate = useNavigate();
  const { data: companies, isLoading: loadingCompanies } = useCompanies();
  const { data: assets } = useAssets();

  const [selectedCompany, setSelectedCompany] = useState("");
  const [jobTitle, setJobTitle] = useState("");
  const [selectedAssets, setSelectedAssets] = useState<string[]>([]);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleAsset = (id: string) => {
    setSelectedAssets((prev) =>
      prev.includes(id) ? prev.filter((a) => a !== id) : [...prev, id],
    );
  };

  const handleGenerate = async () => {
    if (!selectedCompany) return;
    setGenerating(true);
    setError(null);
    try {
      const result = await api.post<{ generation_id: string }>(
        "/api/ai/qa-generate",
        {
          company_id: selectedCompany,
          job_title: jobTitle,
          asset_ids: selectedAssets,
        },
      );
      navigate(`/prep/result/${result.generation_id}`);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "生成に失敗しました。再試行してください。",
      );
    } finally {
      setGenerating(false);
    }
  };

  if (loadingCompanies) return <LoadingSpinner />;

  return (
    <div className="animate-fade-in pb-28">
      <PageHeader title="面接対策" showBack />

      <div className="p-4 space-y-6">
        {/* 企業選択 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            企業を選択 <span className="text-red-500">*</span>
          </label>
          {companies && companies.length > 0 ? (
            <select
              value={selectedCompany}
              onChange={(e) => {
                setSelectedCompany(e.target.value);
                const c = companies.find((c) => c.id === e.target.value);
                if (c?.job_title) setJobTitle(c.job_title);
              }}
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all appearance-none"
            >
              <option value="">企業を選択</option>
              {companies.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          ) : (
            <div className="text-center py-4 bg-gray-50 rounded-xl">
              <p className="text-sm text-gray-500 mb-2">
                まず企業を登録してください
              </p>
              <button
                onClick={() => navigate("/companies/new")}
                className="text-sm text-orange-500 font-medium hover:underline"
              >
                企業を追加
              </button>
            </div>
          )}
        </div>

        {/* 職種 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            職種
          </label>
          <input
            value={jobTitle}
            onChange={(e) => setJobTitle(e.target.value)}
            placeholder="例: ソフトウェアエンジニア"
            className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none transition-all"
          />
        </div>

        {/* 自己資産の選択 */}
        <div>
          <label className="block text-sm font-semibold text-gray-900 mb-2">
            参照する自己資産（任意）
          </label>
          {assets && assets.length > 0 ? (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {assets.map((asset) => (
                <label
                  key={asset.id}
                  className={`flex items-center gap-3 p-3 rounded-xl cursor-pointer transition-colors ${
                    selectedAssets.includes(asset.id)
                      ? "bg-orange-50 border border-orange-200"
                      : "bg-gray-50 border border-transparent"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selectedAssets.includes(asset.id)}
                    onChange={() => toggleAsset(asset.id)}
                    className="w-4 h-4 rounded border-gray-300 text-orange-500 focus:ring-orange-500"
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 truncate">
                      {asset.title}
                    </p>
                  </div>
                </label>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500 py-2">
              自己資産を登録すると、より精度の高い質問が生成されます
            </p>
          )}
        </div>

        {error && (
          <div className="bg-red-50 text-red-700 text-sm rounded-xl p-3">
            {error}
          </div>
        )}
      </div>

      <BottomAction
        label="想定Q&Aを生成"
        onClick={handleGenerate}
        loading={generating}
        disabled={!selectedCompany || generating}
      />
    </div>
  );
}
