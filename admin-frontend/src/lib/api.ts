const API_URL = '';

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

export interface Overview {
  total_users: number;
  new_users_7d: number;
  new_users_30d: number;
  active_subs: number;
  mrr_rub: number;
  conversion_rate: number;
  dau: number;
  wau: number;
}

export interface DayCount {
  date: string;
  count: number;
}

export interface PlanBreakdown {
  plan: string;
  count: number;
  mrr: number;
}

export interface SubscriptionBreakdown {
  active: PlanBreakdown[];
  free: { count: number; label: string };
  total_users: number;
}

const authFetch = async (url: string, token: string, options: RequestInit = {}) => {
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...(options.headers || {}),
    },
  });
  if (res.status === 401) throw new Error('UNAUTHORIZED');
  return res;
};

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
  const res = await authFetch(`${API_URL}/api/v1/users`, token);
  if (!res.ok) throw new Error('Ошибка загрузки');
  const data = await res.json();
  return data.users || [];
}

export async function getSubscriptions(token: string): Promise<Subscription[]> {
  const res = await authFetch(`${API_URL}/api/v1/subscriptions`, token);
  if (!res.ok) throw new Error('Ошибка загрузки');
  const data = await res.json();
  return data.subscriptions || [];
}

export async function activateSubscription(
  token: string,
  userId: number,
  plan: 'monthly' | 'yearly' | 'lifetime'
) {
  const res = await authFetch(`${API_URL}/api/v1/users/${userId}/subscription`, token, {
    method: 'POST',
    body: JSON.stringify({ plan }),
  });
  if (!res.ok) throw new Error('Ошибка активации');
  return res.json();
}

export async function cancelSubscription(token: string, subscriptionId: number) {
  const res = await authFetch(`${API_URL}/api/v1/subscriptions/${subscriptionId}`, token, {
    method: 'DELETE',
  });
  if (!res.ok) throw new Error('Ошибка отмены');
}

export async function getOverview(token: string): Promise<Overview> {
  const res = await authFetch(`${API_URL}/api/v1/analytics/overview`, token);
  if (!res.ok) throw new Error('Ошибка загрузки');
  return res.json();
}

export async function getRegistrationsByDay(token: string, days: number = 30): Promise<DayCount[]> {
  const res = await authFetch(`${API_URL}/api/v1/analytics/registrations?days=${days}`, token);
  if (!res.ok) throw new Error('Ошибка загрузки');
  const data = await res.json();
  return data.data || [];
}

export async function getSubscriptionBreakdown(token: string): Promise<SubscriptionBreakdown> {
  const res = await authFetch(`${API_URL}/api/v1/analytics/subscriptions`, token);
  if (!res.ok) throw new Error('Ошибка загрузки');
  return res.json();
}

// ===== COHORT ANALYSIS =====

export interface CohortWeek {
  week: number;
  active: number;
  percent: number;
}

export interface CohortData {
  cohort_week: string;
  cohort_size: number;
  weeks: CohortWeek[];
}

export interface Retention {
  d1: number;
  d7: number;
  d30: number;
}

export interface CohortAnalysis {
  cohorts: CohortData[];
  retention: Retention;
}

export async function getCohortAnalysis(token: string): Promise<CohortAnalysis> {
  const res = await authFetch(`${API_URL}/api/v1/analytics/cohorts`, token);
  if (!res.ok) throw new Error('Ошибка загрузки когорт');
  return res.json();
}

export interface AdminPayment {
  id: number;
  user_id: number;
  user_email: string;
  user_name: string;
  amount_cents: number;
  currency: string;
  plan: string;
  status: string;
  created_at: string;
}

export interface RevenueStats {
  total_revenue_cents: number;
  month_revenue_cents: number;
  week_revenue_cents: number;
  total_payments: number;
  active_subscriptions: number;
}

export async function getAdminPayments(token: string): Promise<AdminPayment[]> {
  const res = await authFetch(`${API_URL}/api/v1/payments`, token);
  if (!res.ok) throw new Error('Ошибка загрузки платежей');
  const data = await res.json();
  return data.payments || [];
}

export async function getRevenueStats(token: string): Promise<RevenueStats> {
  const res = await authFetch(`${API_URL}/api/v1/revenue`, token);
  if (!res.ok) throw new Error('Ошибка загрузки статистики');
  return res.json();
}

// ============ BROADCASTS (РАССЫЛКИ) ============

export interface Broadcast {
  id: number;
  subject: string;
  body: string;
  audience: 'all' | 'premium' | 'free' | 'inactive';
  status: 'pending' | 'sending' | 'sent' | 'failed';
  sent_count: number;
  failed_count: number;
  created_at: string;
  sent_at: string | null;
}

export interface CreateBroadcastRequest {
  subject: string;
  body: string;
  audience: 'all' | 'premium' | 'free' | 'inactive';
}

export async function getBroadcasts(token: string): Promise<Broadcast[]> {
  const res = await authFetch(`${API_URL}/api/v1/broadcasts`, token);
  if (!res.ok) throw new Error('Не удалось загрузить рассылки');
  const data = await res.json();
  return data.broadcasts || [];
}

export async function createBroadcast(
  token: string,
  request: CreateBroadcastRequest
): Promise<{ broadcast_id: number }> {
  const res = await authFetch(`${API_URL}/api/v1/broadcasts`, token, {
    method: 'POST',
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Ошибка создания' }));
    throw new Error(error.error || 'Не удалось создать рассылку');
  }
  return res.json();
}

export async function sendBroadcast(
  token: string,
  broadcastId: number
): Promise<{ message: string; total_recipients: number }> {
  const res = await authFetch(`${API_URL}/api/v1/broadcasts/${broadcastId}/send`, token, {
    method: 'POST',
  });
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Ошибка отправки' }));
    throw new Error(error.error || 'Не удалось отправить рассылку');
  }
  return res.json();
}