import { useEffect, useState } from 'react';
import { getUsers, type User } from '../lib/api';

interface Props {
  token: string;
  onLogout: () => void;
}

export default function Dashboard({ token, onLogout }: Props) {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadUsers();
  }, [token]);

  const loadUsers = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await getUsers(token);
      setUsers(data);
    } catch (err) {
      if (err instanceof Error && err.message === 'UNAUTHORIZED') {
        onLogout();
      } else {
        setError('Не удалось загрузить пользователей');
      }
    } finally {
      setLoading(false);
    }
  };

  const newThisWeek = users.filter(
    (u) => new Date(u.created_at).getTime() > Date.now() - 7 * 24 * 3600 * 1000
  ).length;

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
            <div className="text-gray-400 text-sm mb-1">💰 Активных подписок</div>
            <div className="text-3xl font-bold text-green-400">0</div>
            <div className="text-xs text-gray-500 mt-1">скоро: управление подписками</div>
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-800 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white">Пользователи</h2>
            <button
              onClick={loadUsers}
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
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">ID</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Имя</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Email</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">Регистрация</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {users.map((user) => (
                    <tr key={user.id} className="hover:bg-gray-800/30 transition-colors">
                      <td className="px-6 py-4 text-sm text-gray-500">#{user.id}</td>
                      <td className="px-6 py-4 text-sm font-medium text-white">{user.name}</td>
                      <td className="px-6 py-4 text-sm text-gray-400">{user.email}</td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {new Date(user.created_at).toLocaleDateString('ru-RU', {
                          day: '2-digit',
                          month: '2-digit',
                          year: 'numeric',
                        })}
                      </td>
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
