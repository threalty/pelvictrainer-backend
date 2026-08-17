import { useEffect, useState } from 'react';
import { getCohortAnalysis, type CohortAnalysis } from '../lib/api';

interface Props {
  token: string;
}

// Цвет ячейки в зависимости от процента удержания
function cellColor(percent: number): string {
  if (percent <= 0) return 'bg-gray-800/50 text-gray-600';
  if (percent < 25) return 'bg-red-900/60 text-red-200';
  if (percent < 50) return 'bg-orange-900/60 text-orange-200';
  if (percent < 75) return 'bg-yellow-900/60 text-yellow-200';
  return 'bg-green-900/70 text-green-200';
}

export default function CohortHeatmap({ token }: Props) {
  const [data, setData] = useState<CohortAnalysis | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [token]);

  const loadData = async () => {
    setLoading(true);
    try {
      setData(await getCohortAnalysis(token));
    } catch (err) {
      console.error('Ошибка загрузки когорт:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6 flex items-center justify-center text-gray-400 h-64">
        Загрузка когорт...
      </div>
    );
  }

  if (!data || data.cohorts.length === 0) {
    return (
      <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
        <h3 className="text-lg font-semibold text-white mb-1">🧬 Когортный анализ</h3>
        <p className="text-sm text-gray-400 text-center py-12">
          Пока нет данных для когортного анализа
        </p>
      </div>
    );
  }

  const { cohorts, retention } = data;

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-2xl p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-white">🧬 Когортный анализ</h3>
          <p className="text-sm text-gray-400">
            % пользователей, вернувшихся через N недель после регистрации
          </p>
        </div>
      </div>

      {/* Карточки retention */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 text-center">
          <div className="text-gray-400 text-xs mb-1">D1 Retention</div>
          <div className={`text-2xl font-bold ${retention.d1 >= 40 ? 'text-green-400' : retention.d1 >= 20 ? 'text-yellow-400' : 'text-red-400'}`}>
            {retention.d1.toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500 mt-1">тренировались в 1-й день</div>
        </div>
        <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 text-center">
          <div className="text-gray-400 text-xs mb-1">D7 Retention</div>
          <div className={`text-2xl font-bold ${retention.d7 >= 40 ? 'text-green-400' : retention.d7 >= 20 ? 'text-yellow-400' : 'text-red-400'}`}>
            {retention.d7.toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500 mt-1">активны первую неделю</div>
        </div>
        <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 text-center">
          <div className="text-gray-400 text-xs mb-1">D30 Retention</div>
          <div className={`text-2xl font-bold ${retention.d30 >= 40 ? 'text-green-400' : retention.d30 >= 20 ? 'text-yellow-400' : 'text-red-400'}`}>
            {retention.d30.toFixed(0)}%
          </div>
          <div className="text-xs text-gray-500 mt-1">активны первый месяц</div>
        </div>
      </div>

      {/* Heatmap таблица */}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="text-left text-xs font-medium text-gray-400 uppercase px-2 py-2">Когорта</th>
              <th className="text-center text-xs font-medium text-gray-400 uppercase px-2 py-2">Размер</th>
              {Array.from({ length: 8 }, (_, i) => (
                <th key={i} className="text-center text-xs font-medium text-gray-400 uppercase px-1 py-2">
                  W{i}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {cohorts.map((cohort) => (
              <tr key={cohort.cohort_week}>
                <td className="px-2 py-1 text-gray-300 whitespace-nowrap">
                  {new Date(cohort.cohort_week).toLocaleDateString('ru-RU', {
                    day: '2-digit',
                    month: '2-digit',
                  })}
                </td>
                <td className="px-2 py-1 text-center text-gray-400">{cohort.cohort_size}</td>
                {cohort.weeks.map((week) => (
                  <td key={week.week} className="px-1 py-1">
                    <div
                      className={`${cellColor(week.percent)} rounded-md text-center py-1.5 text-xs font-medium`}
                      title={`${week.active} из ${cohort.cohort_size} (${week.percent.toFixed(0)}%)`}
                    >
                      {week.percent > 0 ? `${week.percent.toFixed(0)}%` : '·'}
                    </div>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-gray-500 mt-4">
        💡 W0 — неделя регистрации. Зелёные ячейки = пользователи возвращаются.
        Цель: D7 &gt; 40% для product-market fit.
      </p>
    </div>
  );
}
