import type { CalendarEvent } from '@/lib/types';
import { formatTime } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';
import { ConnectPrompt } from '@/components/ConnectPrompt';
import { FetchErrorBanner } from '@/components/FetchErrorBanner';

export function CalendarSection({ events, isLive, isAvailable, fetchError }: {
  events: CalendarEvent[];
  isLive?: boolean;
  isAvailable?: boolean;
  fetchError?: string;
}) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Calendar</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && fetchError && <FetchErrorBanner error={fetchError} />}
          {isLive && (
            <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
              {events?.length ?? 0}
            </span>
          )}
        </div>
      </div>
      {isLive ? (
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
      ) : (
        <ConnectPrompt
          title="Calendar"
          steps={[
            { text: 'Open Google Calendar → Settings (⚙) → the calendar you want → Integrate calendar' },
            { text: 'Copy the Secret address in iCal format URL (keep it private — it grants read access)' },
            { text: 'iCloud: System Settings → Apple ID → iCloud → Passwords & Keychain → share calendar → copy link', },
            { text: 'Add to ~/.daily-briefing/config.json:' },
          ]}
          configSnippet={`"google": {\n  "ical_url": "https://calendar.google.com/calendar/ical/...secret.../basic.ics"\n}`}
        />
      )}
    </div>
  );
}
