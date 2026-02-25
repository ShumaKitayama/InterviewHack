// DesignDocument/3_UIUXDesign.md 3.5.5 - ESエディタ
import { useState } from "react";
import { useParams } from "react-router-dom";
import { ErrorDisplay } from "../components/ui/ErrorDisplay";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { Modal } from "../components/ui/Modal";
import { PageHeader } from "../components/ui/PageHeader";
import {
  useCreateESSection,
  useESProject,
  useESSections,
  useUpdateESSection,
} from "../hooks/useES";

export function ESEditorPage() {
  const { id } = useParams<{ id: string }>();
  const { data: project, isLoading: loadingProject } = useESProject(id);
  const { data: sections, isLoading: loadingSections } = useESSections(id);
  const createSection = useCreateESSection();
  const updateSection = useUpdateESSection();

  const [showAddModal, setShowAddModal] = useState(false);
  const [newSectionTitle, setNewSectionTitle] = useState("");
  const [newSectionLimit, setNewSectionLimit] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");

  if (loadingProject || loadingSections) return <LoadingSpinner />;
  if (!project) return <ErrorDisplay message="プロジェクトが見つかりません" />;

  const handleAddSection = async () => {
    if (!id || !newSectionTitle.trim()) return;
    await createSection.mutateAsync({
      projectId: id,
      input: {
        title: newSectionTitle.trim(),
        content_draft: "",
        character_limit: newSectionLimit
          ? parseInt(newSectionLimit)
          : undefined,
      },
    });
    setNewSectionTitle("");
    setNewSectionLimit("");
    setShowAddModal(false);
  };

  const handleSaveSection = async (sectionId: string) => {
    if (!id) return;
    await updateSection.mutateAsync({
      id: sectionId,
      projectId: id,
      content: editContent,
    });
    setEditingId(null);
  };

  return (
    <div className="animate-fade-in pb-8">
      <PageHeader
        title={project.company?.name ?? "ESエディタ"}
        showBack
        rightAction={{
          label: "+セクション",
          onClick: () => setShowAddModal(true),
        }}
      />

      <div className="p-4 space-y-4">
        {/* プロジェクト情報 */}
        <div className="bg-white rounded-xl p-4 shadow-sm">
          <div className="flex items-center gap-3 text-sm text-gray-600">
            {project.deadline && <span>期限: {project.deadline}</span>}
            <span className="px-2 py-0.5 bg-gray-100 rounded-full text-xs">
              {project.status}
            </span>
          </div>
        </div>

        {/* セクション一覧 */}
        {sections?.length === 0 && (
          <div className="text-center py-8">
            <p className="text-sm text-gray-500 mb-3">
              セクションを追加して執筆を始めましょう
            </p>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-5 py-2.5 bg-orange-500 text-white text-sm font-medium rounded-full hover:bg-orange-600 transition-colors"
            >
              セクションを追加
            </button>
          </div>
        )}

        {sections?.map((section) => (
          <div
            key={section.id}
            className="bg-white rounded-xl shadow-sm overflow-hidden"
          >
            <div className="p-4 border-b border-gray-50">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-900">
                  {section.title}
                </h3>
                {section.character_limit && (
                  <span className="text-xs text-gray-400">
                    {(section.content_draft ?? "").length}/
                    {section.character_limit}字
                  </span>
                )}
              </div>
            </div>

            {editingId === section.id ? (
              <div className="p-4 space-y-3">
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  rows={8}
                  className="w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg text-sm resize-none focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none"
                  maxLength={section.character_limit ?? undefined}
                />
                <div className="flex gap-2 justify-end">
                  <button
                    onClick={() => setEditingId(null)}
                    className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg"
                  >
                    キャンセル
                  </button>
                  <button
                    onClick={() => handleSaveSection(section.id)}
                    className="px-4 py-2 text-sm bg-orange-500 text-white rounded-lg hover:bg-orange-600"
                  >
                    保存
                  </button>
                </div>
              </div>
            ) : (
              <button
                onClick={() => {
                  setEditingId(section.id);
                  setEditContent(section.content_draft ?? "");
                }}
                className="w-full p-4 text-left text-sm text-gray-700 hover:bg-gray-50 transition-colors min-h-[80px]"
              >
                {section.content_draft || (
                  <span className="text-gray-400">タップして執筆開始...</span>
                )}
              </button>
            )}
          </div>
        ))}
      </div>

      <Modal
        open={showAddModal}
        onClose={() => setShowAddModal(false)}
        title="セクション追加"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              セクション名
            </label>
            <input
              value={newSectionTitle}
              onChange={(e) => setNewSectionTitle(e.target.value)}
              placeholder="例: 志望動機、自己PR"
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              文字数制限（任意）
            </label>
            <input
              value={newSectionLimit}
              onChange={(e) => setNewSectionLimit(e.target.value)}
              type="number"
              placeholder="例: 400"
              className="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-orange-500 focus:border-transparent outline-none"
            />
          </div>
          <button
            onClick={handleAddSection}
            disabled={!newSectionTitle.trim()}
            className="w-full py-3 bg-orange-500 text-white rounded-xl text-sm font-semibold hover:bg-orange-600 disabled:opacity-50 transition-colors"
          >
            追加
          </button>
        </div>
      </Modal>
    </div>
  );
}
