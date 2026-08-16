const API_URL = 'https://api.pelvictrainer.ru';

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: { id: number; email: string; name: string };
}

export interface User {
  id: number;
  email: string;
  name: string;
  created_at: string;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Ошибка входа' }));
    throw new Error(err.error || 'Ошибка входа');
  }

  return res.json();
}

export async function getUsers(token: string): Promise<User[]> {
  const res = await fetch(`${API_URL}/api/v1/users`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (res.status === 401) {
    throw new Error('UNAUTHORIZED');
  }

  if (!res.ok) {
    throw new Error('Ошибка загрузки пользователей');
  }

  const data = await res.json();
  return data.users || [];
}
