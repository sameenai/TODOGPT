import type { NewsItem } from '@/lib/types';
import { timeAgo } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';
import { FetchErrorBanner } from '@/components/FetchErrorBanner';

export function NewsSection({ news, isLive, fetchError }: { news: NewsItem[]; isLive?: boolean; fetchError?: string }) {
  if (!news?.length) return null;

  return (
    <div className="panel mb-4">
      <div className="panel-header">
        <h3 className="section-title">Top News</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && fetchError && <FetchErrorBanner error={fetchError} />}
        </div>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 divide-x divide-y divide-gray-800/80">
        {news.slice(0, 8).map((item, i) => (
          <a
            key={i}
            href={item.url}
            target="_blank"
            rel="noopener noreferrer"
            className="p-4 hover:bg-gray-800/50 transition-colors group block"
          >
            <div className="text-sm font-medium text-gray-200 group-hover:text-white line-clamp-2 mb-2 leading-snug transition-colors">
              {item.title}
            </div>
            <div className="text-xs text-gray-600">
              {item.source}&nbsp;&middot;&nbsp;{timeAgo(item.published_at)}
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
