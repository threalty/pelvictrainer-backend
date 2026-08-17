import { useEffect, useState } from 'react';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import { getSubscriptionBreakdown, type SubscriptionBreakdown } from '../lib/api';

interface Props {
  token: string;
}

const COLORS: Record<string, string> = {
  free: '#4b5563',
  monthly: '#3b82f6',
  yearly: '#a855f7',
  lifetime: '#eab308',
};

const LABELS: Record<string, string> = {
  free: 'Free',
  monthly: 'Месяц',
  yearly: 'Год',
  lifetime: 'Lifetime',
};

export default function SubscriptionsChart({ token }: Props) {
  const [data, setData] = useState<SubscriptionBreakdown | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [token]);

  const loadData = async () => {
    setLoading(true);
    try {
      setData(await getSubscriptionBreakdown(token));
    } catch (err) {
      console.error('Ошибка:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading || !data) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center text-gray-400">
        Загрузка...
      </div>
    );
  }

  const chartData = [
    { name: 'free', value: data.free.count },
    ...data.active.map((p) => ({ name: p.plan, value: p.count })),
  ].filter((d) => d.value > 0);

  const total = chartData.reduce((sum, d) => sum + d.value, 0);

  if (total === 0) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center text-gray-400">
        Пока нет пользователей
      </div>
    );
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
      <h3 className="text-lg font-semibold text-white mb-1">💎 Подписки</h3>
      <p className="text-sm text-gray-400 mb-4">Распределение · всего {total}</p>

      <ResponsiveContainer width="100%" height={240}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={85}
            paddingAngle={2}
            dataKey="value"
          >
            {chartData.map((entry, i) => (
              <Cell key={i} fill={COLORS[entry.name] || '#6b7280'} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              backgroundColor: '#111827',
              border: '1px solid #374151',
              borderRadius: '8px',
            }}
            formatter={(value: any, name: any) => {
              const val = Number(value);
              const nameStr = String(name);
              const percent = ((val / total) * 100).toFixed(1);
              return [
                `${val} (${percent}%)`,
                LABELS[nameStr] || nameStr,
              ];
            }}
          />
          <Legend
            formatter={(value: any) => (
              <span className="text-gray-300 text-sm">{LABELS[String(value)] || String(value)}</span>
            )}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
