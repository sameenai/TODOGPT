import type { EmailMessage } from '@/lib/types';
import { formatDate } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';
import { NotAvailable } from '@/components/ConnectPrompt';
import { FetchErrorBanner } from '@/components/FetchErrorBanner';

export function EmailSection({ emails, isLive, isAvailable, fetchError }: {
  emails: EmailMessage[];
  isLive?: boolean;
  isAvailable?: boolean;
  fetchError?: string;
}) {
  const unread = emails?.filter(e => e.is_unread) ?? [];

  return (
    <div className="panel">
      <div className="panel-header">
        <h3 className="section-title">Email</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && fetchError && <FetchErrorBanner error={fetchError} />}
          {isLive && (
            <span className={`text-xs font-bold px-2 py-0.5 rounded-full tabular-nums ${
              unread.length > 0 ? 'bg-rose-950/60 text-rose-300 border border-rose-800/40' : 'bg-emerald-950/60 text-emerald-300 border border-emerald-800/40'
            }`}>
              {unread.length}
            </span>
          )}
        </div>
      </div>
      {isLive ? (
        <div className="divide-y divide-gray-800/80">
          {unread.length === 0 ? (
            <div className="px-4 py-6 text-center text-emerald-400/80 text-sm">Inbox zero!</div>
          ) : unread.slice(0, 5).map(e => (
            <div key={e.id} className="px-4 py-3 hover:bg-gray-800/30 transition-colors">
              <div className="flex items-start justify-between gap-2">
                <div className="text-sm font-medium text-gray-100 truncate min-w-0">
                  {e.is_starred && <span className="text-amber-400 mr-1">&#9733;</span>}
                  {e.subject}
                </div>
                <div className="text-xs text-gray-600 flex-shrink-0 tabular-nums">{formatDate(e.date)}</div>
              </div>
              <div className="text-xs text-gray-500 mt-0.5 truncate">{e.from}</div>
              {e.snippet && (
                <div className="text-xs text-gray-700 mt-0.5 truncate">{e.snippet}</div>
              )}
            </div>
          ))}
        </div>
      ) : isAvailable === false ? (
        <NotAvailable name="Email" />
      ) : null}
    </div>
  );
}
