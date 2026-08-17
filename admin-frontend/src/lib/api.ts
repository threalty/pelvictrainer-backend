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

export interface Subscription {
  id: number;
  user_id: number;
  plan: 'monthly' | 'yearly' | 'lifetime';
  status: 'active' | 'cancelled' | 'expired';
  starts_at: string;
  expires_at: string | null;
  created_at: string;
  user_email?: string;
  user_name?: string;
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
  if (res.status === 401) throw new Error('UNAUTHORIZED');
  if (!res.ok) throw new Error('Ошибка загрузки пользователей');
  const data = await res.json();
  return data.users || [];
}

export async function getSubscriptions(token: string): Promise<Subscription[]> {
  const res = await fetch(`${API_URL}/api/v1/subscriptions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) throw new Error('UNAUTHORIZED');
  if (!res.ok) throw new Error('Ошибка загрузки подписок');
  const data = await res.json();
  return data.subscriptions || [];
}

export async function activateSubscription(
  token: string,
  userId: number,
  plan: 'monthly' | 'yearly' | 'lifetime'
): Promise<{ subscription: Subscription }> {
  const res = await fetch(`${API_URL}/api/v1/users/${userId}/subscription`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ plan }),
  });
  if (res.status === 401) throw new Error('UNAUTHORIZED');
  if (!res.ok) throw new Error('Ошибка активации подписки');
  return res.json();
}

export async function cancelSubscription(token: string, subscriptionId: number): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/subscriptions/${subscriptionId}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 401) throw new Error('UNAUTHORIZED');
  if (!res.ok) throw new Error('Ошибка отмены подписки');
}
