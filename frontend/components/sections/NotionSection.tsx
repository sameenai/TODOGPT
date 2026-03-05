import type { NotionPage } from '@/lib/types';
import { timeAgo } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';

export function NotionSection({ pages, isLive }: { pages: NotionPage[]; isLive?: boolean }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Notion</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
            {pages?.length ?? 0}
          </span>
        </div>
      </div>
      <div className="divide-y divide-gray-800">
        {!pages?.length ? (
          <div className="px-4 py-6 text-center text-gray-500 text-sm">No pages</div>
        ) : pages.slice(0, 5).map(p => (
          <a
            key={p.id}
            href={p.url || '#'}
            target="_blank"
            rel="noopener noreferrer"
            className="block px-4 py-3 hover:bg-gray-800 transition-colors"
          >
            <div className="flex items-center justify-between gap-2">
              <div className="text-sm text-gray-100 truncate">{p.title}</div>
              <div className="text-xs text-gray-500 flex-shrink-0">{timeAgo(p.updated_at)}</div>
            </div>
            <div className="flex items-center gap-2 mt-0.5">
              <span className="text-xs text-gray-600">{p.database}</span>
              {p.status && <span className="text-xs text-gray-500">{p.status}</span>}
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
