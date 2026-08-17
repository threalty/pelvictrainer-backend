import { useEffect, useState } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { getRegistrationsByDay, type DayCount } from '../lib/api';

interface Props {
  token: string;
  days?: number;
}

export default function RegistrationsChart({ token, days = 30 }: Props) {
  const [data, setData] = useState<DayCount[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [token, days]);

  const loadData = async () => {
    setLoading(true);
    try {
      const result = await getRegistrationsByDay(token, days);
      setData(result);
    } catch (err) {
      console.error('Ошибка загрузки графика:', err);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' });
  };

  const total = data.reduce((sum, d) => sum + d.count, 0);

  if (loading) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center text-gray-400">
        Загрузка графика...
      </div>
    );
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-white">📈 Регистрации</h3>
          <p className="text-sm text-gray-400">За последние {days} дней · всего {total}</p>
        </div>
      </div>

      <ResponsiveContainer width="100%" height={240}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="colorCount" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#8b1e3f" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#8b1e3f" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            stroke="#6b7280"
            fontSize={11}
            interval="preserveStartEnd"
          />
          <YAxis
            stroke="#6b7280"
            fontSize={11}
            allowDecimals={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: '#111827',
              border: '1px solid #374151',
              borderRadius: '8px',
            }}
            labelFormatter={(label) => {
              try {
                return new Date(String(label)).toLocaleDateString('ru-RU');
              } catch {
                return String(label);
              }
            }}
            labelStyle={{ color: '#e5e7eb' }}
            itemStyle={{ color: '#f9a8d4' }}
          />
          <Area
            type="monotone"
            dataKey="count"
            stroke="#8b1e3f"
            strokeWidth={2}
            fill="url(#colorCount)"
            name="Регистраций"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
