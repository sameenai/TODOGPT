import type { SlackMessage } from '@/lib/types';
import { timeAgo } from '@/lib/utils';
import { StatusBadge } from '@/components/StatusBadge';
import { NotAvailable } from '@/components/ConnectPrompt';
import { FetchErrorBanner } from '@/components/FetchErrorBanner';

export function SlackSection({ messages, isLive, isAvailable, fetchError }: {
  messages: SlackMessage[];
  isLive?: boolean;
  isAvailable?: boolean;
  fetchError?: string;
}) {
  return (
    <div className="panel">
      <div className="panel-header">
        <h3 className="section-title">Slack</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && fetchError && <FetchErrorBanner error={fetchError} />}
          {isLive && (
            <span className="text-xs font-bold bg-gray-800 text-gray-400 px-2 py-0.5 rounded-full tabular-nums">
              {messages?.length ?? 0}
            </span>
          )}
        </div>
      </div>
      {isLive ? (
        <div className="divide-y divide-gray-800/80">
          {!messages?.length ? (
            <div className="px-4 py-6 text-center text-gray-600 text-sm">No messages</div>
          ) : messages.slice(0, 5).map((m, i) => (
            <div key={i} className="px-4 py-3 hover:bg-gray-800/30 transition-colors">
              <div className="flex items-center gap-2 mb-1">
                <span className={`text-xs font-semibold ${
                  m.is_dm ? 'text-cyan-400' : m.is_urgent ? 'text-rose-400' : 'text-gray-400'
                }`}>
                  {m.is_dm ? `@${m.user}` : `#${m.channel}`}
                </span>
                {!m.is_dm && <span className="text-xs text-gray-600">{m.user}</span>}
                <span className="text-xs text-gray-700 ml-auto">{timeAgo(m.timestamp)}</span>
              </div>
              <div className="text-sm text-gray-300 truncate">{m.text}</div>
            </div>
          ))}
        </div>
      ) : isAvailable === false ? (
        <NotAvailable name="Slack" />
      ) : null}
    </div>
  );
}
