import type { JiraTicket } from '@/lib/types';
import { StatusBadge } from '@/components/StatusBadge';
import { ConnectPrompt } from '@/components/ConnectPrompt';

const PRIORITY_COLOR: Record<string, string> = {
  urgent: 'text-red-400',
  high: 'text-yellow-400',
  medium: 'text-blue-400',
  low: 'text-gray-400',
};

export function JiraSection({ tickets, isLive }: { tickets: JiraTicket[]; isLive?: boolean }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Jira</h3>
        <div className="flex items-center gap-2">
          <StatusBadge live={isLive} />
          {isLive && (
            <span className="text-xs font-bold bg-gray-800 text-gray-300 px-2 py-0.5 rounded">
              {tickets?.length ?? 0}
            </span>
          )}
        </div>
      </div>
      {isLive ? (
        <div className="divide-y divide-gray-800">
          {!tickets?.length ? (
            <div className="px-4 py-6 text-center text-gray-500 text-sm">No tickets</div>
          ) : tickets.slice(0, 5).map(t => (
            <a
              key={t.key}
              href={t.url || '#'}
              target="_blank"
              rel="noopener noreferrer"
              className="block px-4 py-3 hover:bg-gray-800 transition-colors"
            >
              <div className="flex items-start gap-2">
                <span className="text-xs text-gray-500 flex-shrink-0 mt-0.5 font-mono">{t.key}</span>
                <div className="min-w-0">
                  <div className="text-sm text-gray-100 truncate">{t.summary}</div>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className={`text-xs ${PRIORITY_COLOR[t.priority?.toLowerCase()] ?? 'text-gray-400'}`}>
                      {t.priority}
                    </span>
                    <span className="text-xs text-gray-600">{t.status}</span>
                    {t.assignee && <span className="text-xs text-gray-600">{t.assignee}</span>}
                  </div>
                </div>
              </div>
            </a>
          ))}
        </div>
      ) : (
        <ConnectPrompt
          title="Jira"
          steps={[
            { text: 'Go to Atlassian account → Security → Create and manage API tokens', url: 'https://id.atlassian.com/manage-profile/security/api-tokens' },
            { text: 'Create a token and copy it' },
            { text: 'Add to ~/.daily-briefing/config.json:' },
          ]}
          configSnippet={`"jira": {\n  "enabled": true,\n  "base_url": "https://yourorg.atlassian.net",\n  "email": "you@example.com",\n  "api_token": "your_token_here"\n}`}
        />
      )}
    </div>
  );
}
