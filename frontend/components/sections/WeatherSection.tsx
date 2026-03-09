import type { Weather } from '@/lib/types';
import { StatusBadge } from '@/components/StatusBadge';
import { FetchErrorBanner } from '@/components/FetchErrorBanner';

export function WeatherSection({ weather, isLive, fetchError }: { weather: Weather | undefined; isLive?: boolean; fetchError?: string }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
      <div className="panel-header -mx-4 -mt-4 mb-4 px-4">
        <h3 className="section-title">Weather</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && fetchError && <FetchErrorBanner error={fetchError} />}
        </div>
      </div>
      {weather ? (
        <div className="flex items-end gap-4">
          <div>
            <div className="text-6xl font-bold text-white leading-none tabular-nums">
              {Math.round(weather.temperature)}&deg;
            </div>
            <div className="text-xs text-gray-500 mt-1 font-medium uppercase tracking-wide">
              {weather.units === 'metric' ? 'Celsius' : 'Fahrenheit'}
            </div>
          </div>
          <div className="pb-0.5">
            <div className="font-medium text-gray-100 capitalize text-base">{weather.description}</div>
            <div className="text-sm text-gray-400">{weather.city}</div>
            <div className="text-xs text-gray-600 mt-1.5 space-x-2">
              <span>Feels {Math.round(weather.feels_like)}&deg;</span>
              <span>&middot;</span>
              <span>{weather.humidity}% hum.</span>
              <span>&middot;</span>
              <span>{Math.round(weather.wind_speed)} mph</span>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-gray-600 text-sm py-2">No weather data available.</div>
      )}
    </div>
  );
}
