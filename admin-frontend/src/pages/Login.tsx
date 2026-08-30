import { useState } from 'react';
import { login, verify2FA, verify2FABackup } from '../lib/api';

interface Props {
  onLogin: (accessToken: string, refreshToken: string) => void;
}

export default function Login({ onLogin }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // === 2FA состояние ===
  const [requires2FA, setRequires2FA] = useState(false);
  const [userId2FA, setUserId2FA] = useState<number | null>(null);
  const [email2FA, setEmail2FA] = useState('');
  const [code, setCode] = useState('');
  const [useBackup, setUseBackup] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const data = await login(email, password);

      if (data.requires_2fa && data.user_id) {
        setRequires2FA(true);
        setUserId2FA(data.user_id);
        setEmail2FA(data.email || email);
      } else if (data.access_token && data.refresh_token) {
        onLogin(data.access_token, data.refresh_token);
      } else {
        throw new Error('Не удалось получить токены');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  };

  const handle2FASubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const data = useBackup
        ? await verify2FABackup(userId2FA!, code)
        : await verify2FA(userId2FA!, code);

      onLogin(data.access_token, data.refresh_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Неверный код');
    } finally {
      setLoading(false);
    }
  };

  // === Экран ввода кода 2FA ===
  if (requires2FA && userId2FA !== null) {
    return (
      <div className="min-h-screen flex items-center justify-center px-4">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <div className="text-6xl mb-4">🔐</div>
            <h1 className="text-3xl font-bold text-white">Двухфакторная аутентификация</h1>
            <p className="text-gray-400 mt-2">{email2FA}</p>
          </div>

          <form
            onSubmit={handle2FASubmit}
            className="bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-5 shadow-xl"
          >
            {error && (
              <div className="bg-red-900/40 border border-red-700 text-red-200 text-sm rounded-lg px-4 py-3">
                ❌ {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                {useBackup ? 'Backup-код' : '6-значный код'}
              </label>
              <p className="text-xs text-gray-500 mb-3">
                {useBackup
                  ? 'Введите один из сохранённых backup-кодов (например ABCD-EFGH)'
                  : 'Введите код из Google Authenticator или Яндекс.Ключ'}
              </p>
              <input
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
                placeholder={useBackup ? 'ABCD-EFGH' : '123456'}
                maxLength={useBackup ? 9 : 6}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-bordeaux-600 focus:border-transparent transition text-center text-lg tracking-widest font-mono"
              />
            </div>

            <button
              type="submit"
              disabled={loading || code.length < 6}
              className="w-full bg-bordeaux-600 hover:bg-bordeaux-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold rounded-lg px-4 py-3 transition-colors"
            >
              {loading ? 'Проверяем...' : 'Подтвердить'}
            </button>

            <button
              type="button"
              onClick={() => {
                setUseBackup(!useBackup);
                setCode('');
                setError('');
              }}
              className="w-full text-sm text-bordeaux-400 hover:text-bordeaux-300 transition-colors"
            >
              {useBackup ? '← Использовать код из приложения' : 'Использовать backup-код →'}
            </button>
          </form>

          <button
            onClick={() => {
              setRequires2FA(false);
              setUserId2FA(null);
              setEmail2FA('');
              setCode('');
              setError('');
            }}
            className="w-full mt-4 text-gray-400 hover:text-gray-300 text-sm transition-colors"
          >
            ← Назад к входу
          </button>
        </div>
      </div>
    );
  }

  // === Обычная форма логина ===
  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="text-6xl mb-4">💪</div>
          <h1 className="text-3xl font-bold text-white">PelvicTrainer Admin</h1>
          <p className="text-gray-400 mt-2">Панель управления сервисом</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-gray-900 border border-gray-800 rounded-2xl p-8 space-y-5 shadow-xl"
        >
          {error && (
            <div className="bg-red-900/40 border border-red-700 text-red-200 text-sm rounded-lg px-4 py-3">
              ❌ {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="admin@pelvictrainer.ru"
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-bordeaux-600 focus:border-transparent transition"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Пароль</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="••••••••"
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-bordeaux-600 focus:border-transparent transition"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-bordeaux-600 hover:bg-bordeaux-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold rounded-lg px-4 py-3 transition-colors"
          >
            {loading ? 'Входим...' : 'Войти'}
          </button>
        </form>

        <p className="text-center text-gray-500 text-xs mt-6">
          🔒 Защищённое соединение · JWT аутентификация
        </p>
      </div>
    </div>
  );
}