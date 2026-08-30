import { useState, useEffect } from 'react';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';

export default function App() {
  const [token, setToken] = useState<string | null>(() => {
    const stored = localStorage.getItem('access_token');
    // Возвращаем null если токен пустой или 'undefined'
    return stored && stored !== 'undefined' && stored !== 'null' ? stored : null;
  });

  useEffect(() => {
    const handleUnauthorized = () => {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      setToken(null);
    };

    window.addEventListener('unauthorized', handleUnauthorized);
    return () => window.removeEventListener('unauthorized', handleUnauthorized);
  }, []);

  const handleLogin = (accessToken: string, refreshToken: string) => {
    // Защита от undefined/null токенов
    if (!accessToken || accessToken === 'undefined' || accessToken === 'null') {
      console.error('Invalid access token received');
      return;
    }
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    setToken(accessToken);
  };

  const handleLogout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    setToken(null);
  };

  return token ? (
    <Dashboard token={token} onLogout={handleLogout} />
  ) : (
    <Login onLogin={handleLogin} />
  );
}