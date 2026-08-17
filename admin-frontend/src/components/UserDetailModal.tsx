import { useEffect, useState } from 'react';

interface User {
  id: number;
  email: string;
  name: string;
  created_at: string;
}

interface Stats {
  total_sessions: number;
  current_streak: number;
  days_since_last: number;
  last_session_date: string | null;
}

interface Session {
  id: number;
  preset_id: number;
  completed_at: string;
  duration_seconds: number;
  repeats_completed: number;
}

interface Props {
  token: string;
  user: User;
  onClose: () => void;
}

const API_URL = 'https://api.pelvictrainer.ru';

export default function UserDetailModal({ token, user, onClose }: Props) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [user.id]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [detailRes, sessionsRes] = await Promise.all([
        fetch(`${API_URL}/api/v1/users/${user.id}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${API_URL}/api/v1/users/${user.id}/sessions`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ]);

      if (detailRes.ok) {
        const detail = await detailRes.json();
        setStats(detail.stats);
      }

      if (sessionsRes.ok) {
        const sessionsData = await sessionsRes.json();
        setSessions(sessionsData.sessions || []);
      }
    } catch (err) {
      console.error('Ошибка загрузки:', err);
    } finally {
      setLoading(false);
    }
  };

  const activityBadge = () => {
    if (!stats) return null;
    const days = stats.days_since_last;
    if (days === 0) {
      return <span className="text-xs bg-green-900/40 text-green-300 border border-green-700 rounded-full px-2 py-0.5">🟢 Активен сегодня</span>;
    } else if (days <= 3) {
      return <span className="text-xs bg-green-900/40 text-green-300 border border-green-700 rounded-full px-2 py-0.5">🟢 Активен</span>;
    } else if (days <= 7) {
      return <span className="text-xs bg-yellow-900/40 text-yellow-300 border border-yellow-700 rounded-full px-2 py-0.5">🟡 {days} дн. назад</span>;
    } else {
      return <span className="text-xs bg-red-900/40 text-red-300 border border-red-700 rounded-full px-2 py-0.5">🔴 {days} дн. назад</span>;
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-gray-900 border border-gray-800 rounded-2xl p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-white mb-1">{user.name}</h2>
            <p className="text-gray-400 text-sm">{user.email}</p>
            <p className="text-gray-500 text-xs mt-1">
              Зарегистрирован: {new Date(user.created_at).toLocaleDateString('ru-RU')}
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white text-2xl leading-none"
          >
            ×
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400">Загрузка...</div>
        ) : (
          <>
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                <div className="text-gray-400 text-xs mb-1">🏋️ Тренировок</div>
                <div className="text-2xl font-bold text-white">
                  {stats?.total_sessions || 0}
                </div>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                <div className="text-gray-400 text-xs mb-1">🔥 Streak</div>
                <div className="text-2xl font-bold text-bordeaux-400">
                  {stats?.current_streak || 0} дн.
                </div>
              </div>
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                <div className="text-gray-400 text-xs mb-1">📅 Активность</div>
                <div className="mt-1">{activityBadge()}</div>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-white mb-3">История тренировок</h3>
              {sessions.length === 0 ? (
                <div className="text-center py-8 text-gray-500 bg-gray-800/30 rounded-xl">
                  Пока нет тренировок
                </div>
              ) : (
                <div className="bg-gray-800/30 rounded-xl overflow-hidden">
                  <table className="w-full">
                    <thead className="bg-gray-800/50">
                      <tr>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Дата</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Длительность</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase">Повторы</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-700">
                      {sessions.map((session) => (
                        <tr key={session.id} className="hover:bg-gray-700/30">
                          <td className="px-4 py-2 text-sm text-gray-300">
                            {new Date(session.completed_at).toLocaleDateString('ru-RU', {
                              day: '2-digit',
                              month: '2-digit',
                              year: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </td>
                          <td className="px-4 py-2 text-sm text-gray-300">
                            {Math.floor(session.duration_seconds / 60)} мин {session.duration_seconds % 60} сек
                          </td>
                          <td className="px-4 py-2 text-sm text-gray-300">
                            {session.repeats_completed}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}