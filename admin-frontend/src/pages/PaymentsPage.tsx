import { useEffect, useState } from 'react';
import { getAdminPayments, getRevenueStats, type AdminPayment, type RevenueStats } from '../lib/api';

interface Props {
  token: string;
  onBack: () => void;
}

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-900/40 text-yellow-300 border-yellow-700',
  processing: 'bg-blue-900/40 text-blue-300 border-blue-700',
  succeeded: 'bg-green-900/40 text-green-300 border-green-700',
  failed: 'bg-red-900/40 text-red-300 border-red-700',
  refunded: 'bg-gray-800 text-gray-400 border-gray-700',
};

const planLabels: Record<string, string> = {
  monthly: '📅 Месяц',
  yearly: '📆 Год',
  lifetime: '♾️ Lifetime',
};

export default function PaymentsPage({ token, onBack }: Props) {
  const [payments, setPayments] = useState<AdminPayment[]>([]);
  const [revenue, setRevenue] = useState<RevenueStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [token]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [paymentsData, revenueData] = await Promise.all([
        getAdminPayments(token),
        getRevenueStats(token),
      ]);
      setPayments(paymentsData);
      setRevenue(revenueData);
    } catch (err) {
      console.error('Failed to load:', err);
    } finally {
      setLoading(false);
    }
  };

  const formatAmount = (cents: number) => (cents / 100).toLocaleString('ru-RU') + ' ₽';
  const formatDate = (dateStr: string) => new Date(dateStr).toLocaleString('ru-RU');

  return (
    <div className="min-h-screen">
      <header className="bg-gray-900 border-b border-gray-800 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button onClick={onBack} className="text-gray-400 hover:text-white">← Назад</button>
            <h1 className="text-lg font-bold text-white">💰 Платежи</h1>
          </div>
          <button onClick={loadData} className="text-sm text-gray-400 hover:text-white">⟳ Обновить</button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        {revenue && (
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-8">
            <MetricCard label="Общая выручка" value={formatAmount(revenue.total_revenue_cents)} accent="text-green-400" />
            <MetricCard label="За месяц" value={formatAmount(revenue.month_revenue_cents)} accent="text-blue-400" />
            <MetricCard label="За неделю" value={formatAmount(revenue.week_revenue_cents)} accent="text-purple-400" />
            <MetricCard label="Платежей" value={revenue.total_payments} />
            <MetricCard label="Активных подписок" value={revenue.active_subscriptions} accent="text-yellow-400" />
          </div>
        )}

        <div className="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-800">
            <h2 className="text-lg font-semibold text-white">Все платежи ({payments.length})</h2>
          </div>

          {loading ? (
            <div className="p-12 text-center text-gray-400">Загрузка...</div>
          ) : payments.length === 0 ? (
            <div className="p-12 text-center text-gray-400">Платежей пока нет</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-800/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">ID</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Пользователь</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Сумма</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">План</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Статус</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Дата</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {payments.map((p) => (
                    <tr key={p.id} className="hover:bg-gray-800/30">
                      <td className="px-6 py-4 text-sm text-gray-500">#{p.id}</td>
                      <td className="px-6 py-4 text-sm">
                        <div className="text-white">{p.user_name}</div>
                        <div className="text-gray-500 text-xs">{p.user_email}</div>
                      </td>
                      <td className="px-6 py-4 text-sm font-medium text-white">{formatAmount(p.amount_cents)}</td>
                      <td className="px-6 py-4 text-sm">{planLabels[p.plan] || p.plan}</td>
                      <td className="px-6 py-4">
                        <span className={`text-xs px-2 py-1 rounded border ${statusColors[p.status] || ''}`}>
                          {p.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">{formatDate(p.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}

function MetricCard({ label, value, accent = 'text-white' }: { label: string; value: string | number; accent?: string }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-5">
      <div className="text-gray-400 text-xs mb-2">{label}</div>
      <div className={`text-2xl font-bold ${accent}`}>{value}</div>
    </div>
  );
}