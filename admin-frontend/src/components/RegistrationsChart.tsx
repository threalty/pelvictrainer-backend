import { useEffect, useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { getRegistrationsByDay, type DayCount } from '../lib/api';

interface Props {
  token: string;
  days?: number;
}

export default function RegistrationsChart({ token, days = 30 }: Props) {
  const [data, setData] = useState<DayCount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const loadData = async () => {
      try {
        setLoading(true);
        const result = await getRegistrationsByDay(token, days);
        setData(Array.isArray(result) ? result : []);
      } catch (err) {
        setError('Не удалось загрузить данные');
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, [token, days]);

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

  if (data.length === 0) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 h-80 flex items-center justify-center">
        <div className="text-gray-400">Нет данных о регистрациях</div>
      </div>
    );
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
      <h3 className="text-lg font-semibold text-white mb-4">
        📈 Регистрации за {days} дней
      </h3>
      <ResponsiveContainer width="100%" height={250}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
          <XAxis 
            dataKey="date" 
            stroke="#9CA3AF"
            tick={{ fontSize: 11 }}
            tickFormatter={(value) => {
              const date = new Date(value);
              return `${date.getDate()}.${date.getMonth() + 1}`;
            }}
          />
          <YAxis stroke="#9CA3AF" tick={{ fontSize: 11 }} />
          <Tooltip 
            contentStyle={{ 
              backgroundColor: '#1F2937', 
              border: '1px solid #374151',
              borderRadius: '8px',
              color: '#fff'
            }}
            labelFormatter={(value) => {
              // ИСПРАВЛЕНО: приводим к string
              const dateStr = String(value);
              return new Date(dateStr).toLocaleDateString('ru-RU');
            }}
          />
          <Bar dataKey="count" fill="#8B1538" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}