// DesignDocument/5_FrontendDesign.md 5.6.2, 3_UIUXDesign.md 3.3.3 - クレジット購入
import { useState } from "react";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import { PageHeader } from "../components/ui/PageHeader";
import {
  useCreditBalance,
  useCreditPackages,
  useCreditTransactions,
} from "../hooks/useCredits";
import { api } from "../lib/api/client";

export function BillingPage() {
  const { data: balance, isLoading: loadingBalance } = useCreditBalance();
  const { data: packages, isLoading: loadingPackages } = useCreditPackages();
  const { data: transactions, isLoading: loadingTx } = useCreditTransactions();

  const [purchasing, setPurchasing] = useState<string | null>(null);

  const handlePurchase = async (packageId: string) => {
    setPurchasing(packageId);
    try {
      const { url } = await api.post<{ url: string }>(
        "/api/billing/checkout-session",
        { package_id: packageId },
      );
      window.location.href = url;
    } catch {
      // handled by api client
    } finally {
      setPurchasing(null);
    }
  };

  const txTypeLabel: Record<string, string> = {
    purchase: "購入",
    consume: "消費",
    bonus: "ボーナス",
    adjustment: "調整",
  };

  if (loadingBalance || loadingPackages) return <LoadingSpinner />;

  return (
    <div className="animate-fade-in pb-8">
      <PageHeader title="クレジット" showBack />

      <div className="p-4 space-y-6">
        {/* 残高表示 */}
        <div className="bg-gradient-to-br from-orange-500 to-orange-600 rounded-2xl p-5 text-white shadow-lg">
          <p className="text-sm text-orange-100">現在の残高</p>
          <p className="text-4xl font-bold mt-1">
            {balance?.balance ?? 0}
            <span className="text-base font-normal ml-1">クレジット</span>
          </p>
        </div>

        {/* パッケージ一覧 */}
        <div>
          <h2 className="text-base font-semibold text-gray-900 mb-3">
            クレジットを購入
          </h2>
          {packages && packages.length > 0 ? (
            <div className="space-y-3">
              {packages.map((pkg) => (
                <button
                  key={pkg.id}
                  onClick={() => handlePurchase(pkg.id)}
                  disabled={purchasing === pkg.id}
                  className="w-full bg-white rounded-xl p-4 flex items-center justify-between active:bg-gray-50 transition-colors shadow-sm disabled:opacity-50"
                >
                  <div>
                    <p className="text-sm font-semibold text-gray-900">
                      {pkg.name}
                    </p>
                    <p className="text-xs text-gray-500 mt-0.5">
                      {pkg.credits} クレジット
                    </p>
                  </div>
                  <span className="px-4 py-2 bg-orange-500 text-white text-sm font-medium rounded-lg">
                    {purchasing === pkg.id ? "処理中..." : "購入"}
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500 text-center py-4">
              購入可能なパッケージがありません
            </p>
          )}
        </div>

        {/* 取引履歴 */}
        <div>
          <h2 className="text-base font-semibold text-gray-900 mb-3">
            利用履歴
          </h2>
          {loadingTx && <LoadingSpinner />}
          {transactions && transactions.length === 0 && (
            <p className="text-sm text-gray-500 text-center py-4">
              まだ利用履歴がありません
            </p>
          )}
          {transactions && transactions.length > 0 && (
            <div className="bg-white rounded-xl shadow-sm divide-y divide-gray-50">
              {transactions.map((tx) => (
                <div
                  key={tx.id}
                  className="px-4 py-3 flex items-center justify-between"
                >
                  <div>
                    <p className="text-sm text-gray-900">
                      {txTypeLabel[tx.type] ?? tx.type}
                    </p>
                    <p className="text-xs text-gray-400">{tx.reason}</p>
                  </div>
                  <span
                    className={`text-sm font-semibold ${tx.amount >= 0 ? "text-green-600" : "text-red-600"}`}
                  >
                    {tx.amount >= 0 ? "+" : ""}
                    {tx.amount}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
