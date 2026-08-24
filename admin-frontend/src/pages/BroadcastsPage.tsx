import { useEffect, useState } from 'react';
import {
  getBroadcasts,
  createBroadcast,
  sendBroadcast,
  type Broadcast,
  type CreateBroadcastRequest,
} from '../lib/api';

interface Props {
  token: string;
  onBack: () => void;
}

export default function BroadcastsPage({ token, onBack }: Props) {
  const [broadcasts, setBroadcasts] = useState<Broadcast[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [audience, setAudience] = useState<'all' | 'premium' | 'free' | 'inactive'>('all');
  const [submitting, setSubmitting] = useState(false);
  const [sendingId, setSendingId] = useState<number | null>(null);

  const loadData = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await getBroadcasts(token);
      setBroadcasts(data);
    } catch (err) {
      setError('Не удалось загрузить рассылки');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [token]);

  const handleCreate = async () => {
    if (!subject.trim() || !body.trim()) {
      alert('Заполните тему и содержимое');
      return;
    }

    setSubmitting(true);
    try {
      const request: CreateBroadcastRequest = {
        subject: subject.trim(),
        body: body.trim(),
        audience,
      };
      await createBroadcast(token, request);
      setSubject('');
      setBody('');
      setShowCreateForm(false);
      await loadData();
    } catch (err) {
      alert('Ошибка: ' + (err instanceof Error ? err.message : 'неизвестно'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleSend = async (broadcastId: number) => {
    if (!confirm('Отправить рассылку? Это действие нельзя отменить.')) return;

    setSendingId(broadcastId);
    try {
      const result = await sendBroadcast(token, broadcastId);
      alert(`✅ ${result.message}\nПолучателей: ${result.total_recipients}`);
      await loadData();
    } catch (err) {
      alert('Ошибка: ' + (err instanceof Error ? err.message : 'неизвестно'));
    } finally {
      setSendingId(null);
    }
  };

  const statusBadge = (status: string) => {
    const styles: Record<string, string> = {
      pending: 'bg-yellow-900/40 text-yellow-300 border-yellow-700',
      sending: 'bg-blue-900/40 text-blue-300 border-blue-700',
      sent: 'bg-green-900/40 text-green-300 border-green-700',
      failed: 'bg-red-900/40 text-red-300 border-red-700',
    };
    const labels: Record<string, string> = {
      pending: '⏸️ Черновик',
      sending: '📤 Отправляется',
      sent: '✅ Отправлено',
      failed: '❌ Ошибка',
    };
    return (
      <span className={`text-xs px-2 py-1 rounded border ${styles[status] || 'bg-gray-800 text-gray-400 border-gray-700'}`}>
        {labels[status] || status}
      </span>
    );
  };

  const audienceLabel = (aud: string) => {
    const labels: Record<string, string> = {
      all: '👥 Все пользователи',
      premium: '💎 Premium',
      free: '🆓 Free',
      inactive: '😴 Неактивные 7+ дней',
    };
    return labels[aud] || aud;
  };

  return (
    <div className="min-h-screen">
      <header className="bg-gray-900 border-b border-gray-800 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <button
              onClick={onBack}
              className="text-gray-400 hover:text-white transition-colors"
            >
              ← Назад
            </button>
            <span className="text-2xl">📧</span>
            <h1 className="text-lg font-bold text-white">Email рассылки</h1>
          </div>
          <div className="flex items-center gap-4">
            <button
              onClick={loadData}
              disabled={loading}
              className="text-sm text-gray-400 hover:text-white disabled:opacity-50 transition-colors"
            >
              ⟳ Обновить
            </button>
            <button
              onClick={() => setShowCreateForm(true)}
              className="text-sm bg-red-600 hover:bg-red-700 text-white font-semibold rounded-lg px-4 py-2 transition-colors"
            >
              + Новая рассылка
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        {showCreateForm && (
          <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 mb-6">
            <h2 className="text-lg font-semibold text-white mb-4">Создать рассылку</h2>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-400 mb-2">
                  Тема письма
                </label>
                <input
                  type="text"
                  value={subject}
                  onChange={(e) => setSubject(e.target.value)}
                  placeholder="Например: Новые функции в PelvicTrainer"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-red-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-400 mb-2">
                  Аудитория
                </label>
                <select
                  value={audience}
                  onChange={(e) => setAudience(e.target.value as any)}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white focus:outline-none focus:ring-2 focus:ring-red-500"
                >
                  <option value="all">👥 Все пользователи</option>
                  <option value="premium">💎 Premium</option>
                  <option value="free">🆓 Free</option>
                  <option value="inactive">😴 Неактивные 7+ дней</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-400 mb-2">
                  Содержимое (HTML поддерживается)
                </label>
                <textarea
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  rows={8}
                  placeholder="<h1>Привет!</h1><p>Мы добавили новые функции...</p>"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-red-500 font-mono text-sm"
                />
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  onClick={() => {
                    setShowCreateForm(false);
                    setSubject('');
                    setBody('');
                  }}
                  disabled={submitting}
                  className="flex-1 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 text-white rounded-lg px-4 py-3 transition-colors"
                >
                  Отмена
                </button>
                <button
                  onClick={handleCreate}
                  disabled={submitting}
                  className="flex-1 bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white font-semibold rounded-lg px-4 py-3 transition-colors"
                >
                  {submitting ? 'Создание...' : 'Создать'}
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-800">
            <h2 className="text-lg font-semibold text-white">История рассылок</h2>
          </div>

          {loading ? (
            <div className="p-12 text-center text-gray-400">Загрузка...</div>
          ) : error ? (
            <div className="p-12 text-center text-red-400">❌ {error}</div>
          ) : broadcasts.length === 0 ? (
            <div className="p-12 text-center text-gray-400">
              <div className="text-4xl mb-2">📭</div>
              <div>Пока нет рассылок</div>
              <div className="text-sm mt-2">Нажмите "Новая рассылка" чтобы создать первую</div>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-800/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">ID</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Тема</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Аудитория</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Статус</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Отправлено</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-400 uppercase">Создано</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-400 uppercase">Действия</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {broadcasts.map((broadcast) => (
                    <tr key={broadcast.id} className="hover:bg-gray-800/30 transition-colors">
                      <td className="px-6 py-4 text-sm text-gray-500">#{broadcast.id}</td>
                      <td className="px-6 py-4 text-sm font-medium text-white max-w-xs truncate">
                        {broadcast.subject}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {audienceLabel(broadcast.audience)}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        {statusBadge(broadcast.status)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {broadcast.sent_count > 0 ? (
                          <div>
                            <div className="text-green-400">✅ {broadcast.sent_count}</div>
                            {broadcast.failed_count > 0 && (
                              <div className="text-red-400 text-xs">❌ {broadcast.failed_count}</div>
                            )}
                          </div>
                        ) : (
                          <span className="text-gray-600">—</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {new Date(broadcast.created_at).toLocaleString('ru-RU', {
                          day: '2-digit',
                          month: '2-digit',
                          year: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </td>
                      <td className="px-6 py-4 text-sm text-right">
                        {broadcast.status === 'pending' && (
                          <button
                            onClick={() => handleSend(broadcast.id)}
                            disabled={sendingId === broadcast.id}
                            className="text-xs bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white font-semibold rounded-lg px-3 py-1.5 transition-colors"
                          >
                            {sendingId === broadcast.id ? 'Отправка...' : '📤 Отправить'}
                          </button>
                        )}
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