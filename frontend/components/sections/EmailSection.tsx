import type { EmailMessage } from '@/lib/types';
import { formatDate } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';

export function EmailSection({ emails, isLive }: { emails: EmailMessage[]; isLive?: boolean }) {
  const unread = emails?.filter(e => e.is_unread) ?? [];

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Email</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          <span className={`text-xs font-bold px-2 py-0.5 rounded ${
            unread.length > 0 ? 'bg-red-900 text-red-300' : 'bg-green-900 text-green-300'
          }`}>
            {unread.length}
          </span>
        </div>
      </div>
      <div className="divide-y divide-gray-800">
        {unread.length === 0 ? (
          <div className="px-4 py-6 text-center text-green-400 text-sm">Inbox zero!</div>
        ) : unread.slice(0, 5).map(e => (
          <div key={e.id} className="px-4 py-3">
            <div className="flex items-start justify-between gap-2">
              <div className="text-sm font-medium text-gray-100 truncate min-w-0">
                {e.is_starred && <span className="text-yellow-400 mr-1">&#9733;</span>}
                {e.subject}
              </div>
              <div className="text-xs text-gray-500 flex-shrink-0">{formatDate(e.date)}</div>
            </div>
            <div className="text-xs text-gray-500 mt-0.5 truncate">{e.from}</div>
            {e.snippet && (
              <div className="text-xs text-gray-600 mt-0.5 truncate">{e.snippet}</div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
