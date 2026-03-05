import type { GitHubNotification } from '@/lib/types';
import { timeAgo } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';

export function GitHubSection({ notifications, isLive }: { notifications: GitHubNotification[]; isLive?: boolean }) {
  const unread = notifications?.filter(n => n.unread) ?? [];

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">GitHub</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
            {unread.length}
          </span>
        </div>
      </div>
      <div className="divide-y divide-gray-800">
        {!unread.length ? (
          <div className="px-4 py-6 text-center text-green-400 text-sm">No unread notifications</div>
        ) : unread.slice(0, 5).map(n => (
          <a
            key={n.id}
            href={n.url || '#'}
            target="_blank"
            rel="noopener noreferrer"
            className="block px-4 py-3 hover:bg-gray-800 transition-colors"
          >
            <div className="flex items-start gap-2">
              <span className="text-xs bg-gray-800 text-gray-400 px-1.5 py-0.5 rounded mt-0.5 flex-shrink-0 font-mono">
                {n.type === 'PullRequest' ? 'PR' : n.type === 'Issue' ? 'IS' : n.type.slice(0, 2).toUpperCase()}
              </span>
              <div className="min-w-0">
                <div className="text-sm text-gray-100 truncate">{n.title}</div>
                <div className="text-xs text-gray-500 mt-0.5">
                  {n.repo} &middot; {n.reason} &middot; {timeAgo(n.updated_at)}
                </div>
              </div>
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
