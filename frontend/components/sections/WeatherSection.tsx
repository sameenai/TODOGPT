import type { Weather } from '@/lib/types';

export function WeatherSection({ weather }: { weather: Weather | undefined }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
      <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">Weather</h3>
      {weather ? (
        <div className="flex items-center gap-4">
          <div className="text-5xl font-bold text-white">{Math.round(weather.temperature)}&deg;</div>
          <div>
            <div className="font-medium text-gray-100 capitalize">{weather.description}</div>
            <div className="text-sm text-gray-400">{weather.city}</div>
            <div className="text-xs text-gray-500 mt-1">
              Feels {Math.round(weather.feels_like)}&deg;
              &nbsp;&middot;&nbsp;{weather.humidity}% humidity
              &nbsp;&middot;&nbsp;{Math.round(weather.wind_speed)} mph
            </div>
          </div>
        </div>
      ) : (
        <div className="text-gray-500 text-sm">No weather data available.</div>
      )}
    </div>
  );
}
