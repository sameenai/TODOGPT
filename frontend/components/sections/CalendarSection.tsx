import type { CalendarEvent } from '@/lib/types';
import { formatTime } from '@/lib/utils';

export function CalendarSection({ events }: { events: CalendarEvent[] }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Calendar</h3>
        <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
          {events?.length ?? 0}
        </span>
      </div>
      <div className="divide-y divide-gray-800">
        {!events?.length ? (
          <div className="px-4 py-6 text-center text-gray-500 text-sm">No events today</div>
        ) : events.map(e => (
          <div key={e.id} className="px-4 py-3">
            <div className="flex items-start justify-between gap-2">
              <div className="text-sm font-medium text-gray-100">{e.title}</div>
              <div className="text-xs text-gray-500 flex-shrink-0 font-mono">
                {e.all_day ? 'all day' : formatTime(e.start_time)}
              </div>
            </div>
            {(e.location || e.meeting_url) && (
              <div className="text-xs text-gray-500 mt-0.5">
                {e.location || '(virtual)'}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
