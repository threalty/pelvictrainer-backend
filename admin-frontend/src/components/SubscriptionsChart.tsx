import { useEffect, useState } from 'react';
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';
import { getSubscriptionBreakdown, type SubscriptionBreakdown } from '../lib/api';

interface Props {
  token: string;
}

const COLORS = ['#8B1538', '#cf2d5d', '#ee7798', '#6B7280'];

export default function SubscriptionsChart({ token }: Props) {
  const [data, setData] = useState<SubscriptionBreakdown | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        const result = await getSubscriptionBreakdown(token);
        setData(result);
      } catch (err) {
        setError('Не удалось загрузить данные');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [token]);

  if (loading) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center">
        <div className="text-gray-400">Загрузка...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center">
        <div className="text-red-400">❌ {error}</div>
      </div>
    );
  }

  const activePlans = Array.isArray(data?.active) ? data.active : [];
  const freeCount = data?.free?.count ?? 0;
  
  const chartData = [
    ...activePlans.map(p => ({ 
      name: `${p.plan} (${p.count})`, 
      value: p.count 
    })),
    ...(freeCount > 0 ? [{ name: `Free (${freeCount})`, value: freeCount }] : [])
  ];

  if (chartData.length === 0) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center">
        <div className="text-gray-400">Нет данных о подписках</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
      <h3 className="text-lg font-semibold text-white mb-4">
        💎 Структура подписок
      </h3>
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            labelLine={false}
            outerRadius={80}
            fill="#8884d8"
            dataKey="value"
          >
            {chartData.map((_entry, index) => (
              // ИСПРАВЛЕНО: используем _entry (подчёркивание) вместо entry
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip 
            contentStyle={{ 
              backgroundColor: '#1F2937', 
              border: '1px solid #374151',
              borderRadius: '8px',
              color: '#fff'
            }}
          />
          <Legend 
            wrapperStyle={{ color: '#fff' }}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}