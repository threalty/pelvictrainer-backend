import { useEffect, useState } from 'react';
import {
  getUsers,
  getSubscriptions,
  activateSubscription,
  cancelSubscription,
  type User,
  type Subscription,
} from '../lib/api';

interface Props {
  token: string;
  onLogout: () => void;
}

export default function Dashboard({ token, onLogout }: Props) {
  const [users, setUsers] = useState<User[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Модалка
  const [modalUser, setModalUser] = useState<User | null>(null);
  const [selectedPlan, setSelectedPlan] = useState<'monthly' | 'yearly' | 'lifetime'>('monthly');
  const [submitting, setSubmitting] = useState(false);

  const loadData = async () => {
    setLoading(true);
    setError('');
    try {
      const [usersData, subsData] = await Promise.all([
        getUsers(token),
        getSubscriptions(token),
      ]);
      setUsers(usersData);
      setSubscriptions(subsData);
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

  // Мапа user_id -> активная подписка
  const activeSubMap = new Map<number, Subscription>();
  subscriptions.forEach((s) => {
    if (s.status === 'active') {
      activeSubMap.set(s.user_id, s);
    }
  });

  const newThisWeek = users.filter(
    (u) => new Date(u.created_at).getTime() > Date.now() - 7 * 24 * 3600 * 1000
  ).length;

  const activeSubsCount = subscriptions.filter((s) => s.status === 'active').length;

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

  return (
    <div className="min-h-screen">
      <header className="bg-gray-900 border-b border-gray-800 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-2xl">💪</span>
            <h1 className="text-lg font-bold text-white">PelvicTrainer Admin</h1>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-xs text-green-400 bg-green-900/30 border border-green-800 rounded-full px-3 py-1">
              ● API онлайн
            </span>
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
        {/* Статистика */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
            <div className="text-gray-400 text-sm mb-1">👥 Всего пользователей</div>
            <div className="text-3xl font-bold text-white">{users.length}</div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
            <div className="text-gray-400 text-sm mb-1">🆕 Новых за неделю</div>
            <div className="text-3xl font-bold text-bordeaux-400">{newThisWeek}</div>
          </div>
          <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
            <div className="text-gray-400 text-sm mb-1">💎 Активных подписок</div>
            <div className="text-3xl font-bold text-green-400">{activeSubsCount}</div>
          </div>
        </div>

        {/* Таблица пользователей */}
        <div className="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-800 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white">Пользователи</h2>
            <button
              onClick={loadData}
              className="text-sm text-bordeaux-400 hover:text-bordeaux-300 transition-colors"
            >
              ⟳ Обновить
            </button>
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
                      <tr key={user.id} className="hover:bg-gray-800/30 transition-colors">
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
                              onClick={() => setModalUser(user)}
                              className="text-xs text-bordeaux-400 hover:text-bordeaux-300 transition-colors"
                            >
                              {sub ? 'Изменить' : '+ Подписка'}
                            </button>
                            {sub && (
                              <button
                                onClick={() => handleCancel(sub.id)}
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

      {/* Модалка активации подписки */}
      {modalUser && (
        <div
          className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4"
          onClick={() => !submitting && setModalUser(null)}
        >
          <div
            className="bg-gray-900 border border-gray-800 rounded-2xl p-8 max-w-md w-full"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-xl font-bold text-white mb-2">
              Активировать подписку
            </h2>
            <p className="text-gray-400 text-sm mb-6">
              для <span className="text-white font-medium">{modalUser.name}</span> ({modalUser.email})
            </p>

            <div className="space-y-3 mb-6">
              {(['monthly', 'yearly', 'lifetime'] as const).map((plan) => (
                <label
                  key={plan}
                  className={`flex items-center justify-between p-4 rounded-xl border cursor-pointer transition-colors ${
                    selectedPlan === plan
                      ? 'border-bordeaux-600 bg-bordeaux-950/30'
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
                      className="w-4 h-4 accent-bordeaux-600"
                    />
                    <div>
                      <div className="text-white font-medium">{planBadge(plan)}</div>
                      <div className="text-xs text-gray-400 mt-1">
                        {plan === 'monthly' && 'Доступ на 1 месяц'}
                        {plan === 'yearly' && 'Доступ на 1 год (выгодно)'}
                        {plan === 'lifetime' && 'Бессрочный доступ'}
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
                className="flex-1 bg-bordeaux-600 hover:bg-bordeaux-700 disabled:opacity-50 text-white font-semibold rounded-lg px-4 py-3 transition-colors"
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
