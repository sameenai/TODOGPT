import type { NewsItem } from '@/lib/types';
import { timeAgo } from '@/lib/utils';

export function NewsSection({ news }: { news: NewsItem[] }) {
  if (!news?.length) return null;

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden mb-4">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Top News</h3>
        <span className="text-xs text-cyan-400 font-semibold">LIVE</span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 divide-x divide-y divide-gray-800">
        {news.slice(0, 8).map((item, i) => (
          <a
            key={i}
            href={item.url}
            target="_blank"
            rel="noopener noreferrer"
            className="p-4 hover:bg-gray-800 transition-colors block"
          >
            <div className="text-sm font-medium text-gray-100 line-clamp-2 mb-2 leading-snug">
              {item.title}
            </div>
            <div className="text-xs text-gray-500">
              {item.source} &middot; {timeAgo(item.published_at)}
            </div>
          </a>
        ))}
      </div>
    </div>
  );
}
