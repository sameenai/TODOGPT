import type { SlackMessage } from '@/lib/types';
import { timeAgo } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';

export function SlackSection({ messages, isLive }: { messages: SlackMessage[]; isLive?: boolean }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Slack</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
            {messages?.length ?? 0}
          </span>
        </div>
      </div>
      <div className="divide-y divide-gray-800">
        {!messages?.length ? (
          <div className="px-4 py-6 text-center text-gray-500 text-sm">No messages</div>
        ) : messages.slice(0, 5).map((m, i) => (
          <div key={i} className="px-4 py-3">
            <div className="flex items-center gap-2 mb-1">
              <span className={`text-xs font-medium ${
                m.is_dm ? 'text-cyan-400' : m.is_urgent ? 'text-red-400' : 'text-gray-400'
              }`}>
                {m.is_dm ? `@${m.user}` : `#${m.channel}`}
              </span>
              {!m.is_dm && <span className="text-xs text-gray-600">{m.user}</span>}
              <span className="text-xs text-gray-600 ml-auto">{timeAgo(m.timestamp)}</span>
            </div>
            <div className="text-sm text-gray-300 truncate">{m.text}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
