import { useEffect, useState } from 'react';
import {
  getUsers,
  getSubscriptions,
  getOverview,
  activateSubscription,
  cancelSubscription,
  type User,
  type Subscription,
  type Overview,
} from '../lib/api';
import UserDetailModal from '../components/UserDetailModal';
import RegistrationsChart from '../components/RegistrationsChart';
import SubscriptionsChart from '../components/SubscriptionsChart';
import CohortHeatmap from '../components/CohortHeatmap';
import PaymentsPage from './PaymentsPage';
import BroadcastsPage from './BroadcastsPage';

interface Props {
  token: string;
  onLogout: () => void;
}

export default function Dashboard({ token, onLogout }: Props) {
  const [users, setUsers] = useState<User[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [overview, setOverview] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  const [modalUser, setModalUser] = useState<User | null>(null);
  const [selectedPlan, setSelectedPlan] = useState<'monthly' | 'yearly' | 'lifetime'>('monthly');
  const [submitting, setSubmitting] = useState(false);

  const [currentPage, setCurrentPage] = useState<'dashboard' | 'payments' | 'broadcasts'>('dashboard');

  const loadData = async () => {
    setLoading(true);
    setError('');
    try {
      const [usersData, subsData, overviewData] = await Promise.all([
        getUsers(token),
        getSubscriptions(token),
        getOverview(token),
      ]);
      setUsers(usersData);
      setSubscriptions(subsData);
      setOverview(overviewData);
    } catch (err) {
      if (err instanceof Error && err.message === 'UNAUTHORIZED') {
        onLogout();
      } else {
        setError('Не удалось загрузить данные');
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [token]);

  const activeSubMap = new Map<number, Subscription>();
  subscriptions.forEach((s) => {
    if (s.status === 'active') activeSubMap.set(s.user_id, s);
  });

  const handleActivate = async () => {
    if (!modalUser) return;
    setSubmitting(true);
    try {
      await activateSubscription(token, modalUser.id, selectedPlan);
      setModalUser(null);
      await loadData();
    } catch (err) {
      alert('Ошибка: ' + (err instanceof Error ? err.message : 'неизвестно'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async (subId: number) => {
    if (!confirm('Отменить подписку?')) return;
    try {
      await cancelSubscription(token, subId);
      await loadData();
    } catch (err) {
      alert('Ошибка: ' + (err instanceof Error ? err.message : 'неизвестно'));
    }
  };

  const planBadge = (plan: string) => {
    const styles: Record<string, string> = {
      monthly: 'bg-blue-900/40 text-blue-300 border-blue-700',
      yearly: 'bg-purple-900/40 text-purple-300 border-purple-700',
      lifetime: 'bg-yellow-900/40 text-yellow-300 border-yellow-700',
    };
    const labels: Record<string, string> = {
      monthly: '📅 Месяц',
      yearly: '📆 Год',
      lifetime: '♾️ Lifetime',
    };
    return (
      <span className={`text-xs px-2 py-0.5 rounded border ${styles[plan] || 'bg-gray-800 text-gray-400 border-gray-700'}`}>
        {labels[plan] || plan}
      </span>
    );
  };

  if (currentPage === 'payments') {
    return <PaymentsPage token={token} onBack={() => setCurrentPage('dashboard')} />;
  }

  if (currentPage === 'broadcasts') {
    return <BroadcastsPage token={token} onBack={() => setCurrentPage('dashboard')} />;
  }

  return (
    <div className="min-h-screen">
      <header className="bg-gray-900 border-b border-gray-800 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-2xl">💪</span>
            <h1 className="text-lg font-bold text-white">PelvicTrainer Admin</h1>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs text-green-400 bg-green-900/30 border border-green-800 rounded-full px-3 py-1">
              ● API онлайн
            </span>
            <button
              onClick={() => setCurrentPage('payments')}
              className="text-sm text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg px-4 py-2 transition-colors"
            >
              💰 Платежи
            </button>
            <button
              onClick={() => setCurrentPage('broadcasts')}
              className="text-sm text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg px-4 py-2 transition-colors"
            >
              📧 Рассылки
            </button>
            <button
              onClick={loadData}
              disabled={loading}
              className="text-sm text-gray-400 hover:text-white disabled:opacity-50 transition-colors"
            >
              ⟳ Обновить
            </button>
            <button
              onClick={onLogout}
              className="text-sm text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg px-4 py-2 transition-colors"
            >
              Выйти
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <MetricCard
            icon="💰"
            label="MRR"
            value={overview?.mrr_rub ? `${Math.round(overview.mrr_rub).toLocaleString('ru-RU')} ₽` : '—'}
            sub="мес. регулярный доход"
            accent="text-green-400"
          />
          <MetricCard
            icon="👥"
            label="Всего пользователей"
            value={overview?.total_users ?? '—'}
            sub={`+${overview?.new_users_30d ?? 0} за 30 дн.`}
          />
          <MetricCard
            icon="💎"
            label="Платящих"
            value={overview?.active_subs ?? '—'}
            sub={`конверсия ${(overview?.conversion_rate ?? 0).toFixed(1)}%`}
            accent="text-red-400"
          />
          <MetricCard
            icon="🔥"
            label="DAU / WAU"
            value={overview ? `${overview.dau} / ${overview.wau}` : '—'}
            sub="активные пользователи"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
          <div className="lg:col-span-2">
            <RegistrationsChart token={token} days={30} />
          </div>
          <SubscriptionsChart token={token} />
        </div>

        <div className="mb-8">
          <CohortHeatmap token={token} />
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-800">
            <h2 className="text-lg font-semibold text-white">Пользователи</h2>
          </div>

          {loading ? (
            <div className="p-12 text-center text-gray-400">Загрузка...</div>
          ) : error ? (
            <div className="p-12 text-center text-red-400">❌ {error}</div>
          ) : users.length === 0 ? (
            <div className="p-12 text-center text-gray-400">Пока нет пользователей</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-800/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">ID</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Имя</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Email</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Подписка</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Регистрация</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">Действия</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {users.map((user) => {
                    const sub = activeSubMap.get(user.id);
                    return (
                      <tr
                        key={user.id}
                        className="hover:bg-gray-800/30 transition-colors cursor-pointer"
                        onClick={() => setSelectedUser(user)}
                      >
                        <td className="px-6 py-4 text-sm text-gray-500">#{user.id}</td>
                        <td className="px-6 py-4 text-sm font-medium text-white">{user.name}</td>
                        <td className="px-6 py-4 text-sm text-gray-400">{user.email}</td>
                        <td className="px-6 py-4 text-sm">
                          {sub ? (
                            <div className="flex items-center gap-2">
                              {planBadge(sub.plan)}
                              {sub.expires_at && (
                                <span className="text-xs text-gray-500">
                                  до {new Date(sub.expires_at).toLocaleDateString('ru-RU')}
                                </span>
                              )}
                            </div>
                          ) : (
                            <span className="text-xs text-gray-600">— free —</span>
                          )}
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-400">
                          {new Date(user.created_at).toLocaleDateString('ru-RU')}
                        </td>
                        <td className="px-6 py-4 text-sm text-right">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setModalUser(user);
                              }}
                              className="text-xs text-red-400 hover:text-red-300 transition-colors"
                            >
                              {sub ? 'Изменить' : '+ Подписка'}
                            </button>
                            {sub && (
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleCancel(sub.id);
                                }}
                                className="text-xs text-red-400 hover:text-red-300 transition-colors"
                              >
                                Отменить
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>

      {selectedUser && (
        <UserDetailModal
          token={token}
          user={selectedUser}
          onClose={() => setSelectedUser(null)}
        />
      )}

      {modalUser && (
        <div
          className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4"
          onClick={() => !submitting && setModalUser(null)}
        >
          <div
            className="bg-gray-900 border border-gray-800 rounded-2xl p-8 max-w-md w-full"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-xl font-bold text-white mb-2">Активировать подписку</h2>
            <p className="text-gray-400 text-sm mb-6">
              для <span className="text-white font-medium">{modalUser.name}</span>
            </p>

            <div className="space-y-3 mb-6">
              {(['monthly', 'yearly', 'lifetime'] as const).map((plan) => (
                <label
                  key={plan}
                  className={`flex items-center justify-between p-4 rounded-xl border cursor-pointer transition-colors ${
                    selectedPlan === plan
                      ? 'border-red-600 bg-red-950/30'
                      : 'border-gray-800 bg-gray-800/30 hover:border-gray-700'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <input
                      type="radio"
                      name="plan"
                      value={plan}
                      checked={selectedPlan === plan}
                      onChange={() => setSelectedPlan(plan)}
                      className="w-4 h-4 accent-red-600"
                    />
                    <div>
                      <div className="text-white font-medium">{planBadge(plan)}</div>
                      <div className="text-xs text-gray-400 mt-1">
                        {plan === 'monthly' && '299 ₽/мес'}
                        {plan === 'yearly' && '2 490 ₽/год (выгодно)'}
                        {plan === 'lifetime' && '4 990 ₽ — навсегда'}
                      </div>
                    </div>
                  </div>
                </label>
              ))}
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => setModalUser(null)}
                disabled={submitting}
                className="flex-1 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 text-white rounded-lg px-4 py-3 transition-colors"
              >
                Отмена
              </button>
              <button
                onClick={handleActivate}
                disabled={submitting}
                className="flex-1 bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white font-semibold rounded-lg px-4 py-3 transition-colors"
              >
                {submitting ? 'Активация...' : 'Активировать'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function MetricCard({
  icon,
  label,
  value,
  sub,
  accent = 'text-white',
}: {
  icon: string;
  label: string;
  value: string | number;
  sub?: string;
  accent?: string;
}) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-5">
      <div className="text-gray-400 text-xs mb-2 flex items-center gap-1">
        <span>{icon}</span>
        <span>{label}</span>
      </div>
      <div className={`text-2xl font-bold ${accent}`}>{value}</div>
      {sub && <div className="text-xs text-gray-500 mt-1">{sub}</div>}
    </div>
  );
}