import type { Briefing } from '@/lib/types';

function Bar({ label, pct, color }: { label: string; pct: number; color: string }) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-xs text-gray-400 w-14 flex-shrink-0">{label}</span>
      <div className="flex-1 bg-gray-800 rounded-full h-1.5">
        <div
          className={`h-1.5 rounded-full transition-all duration-500 ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs text-gray-400 w-8 text-right flex-shrink-0">{pct}%</span>
    </div>
  );
}

export function InboxZeroProgress({ briefing }: { briefing: Briefing }) {
  const unreadEmails = briefing.unread_emails?.filter(e => e.is_unread).length ?? 0;
  const slackMsgs = briefing.slack_messages?.length ?? 0;
  const unreadGH = briefing.github_notifications?.filter(n => n.unread).length ?? 0;
  const todos = briefing.todos ?? [];
  const doneTodos = todos.filter(t => t.status === 2).length;
  const totalTodos = todos.length;

  const emailPct = Math.max(0, Math.min(100, Math.round(100 - unreadEmails * 5)));
  const slackPct = Math.max(0, Math.min(100, Math.round(100 - slackMsgs * 10)));
  const ghPct = Math.max(0, Math.min(100, Math.round(100 - unreadGH * 10)));
  const todoPct = totalTodos > 0 ? Math.round((doneTodos / totalTodos) * 100) : 100;

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
      <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">
        Inbox Zero Progress
      </h3>
      <div className="space-y-3">
        <Bar label="Email"  pct={emailPct} color={emailPct === 100 ? 'bg-green-500' : 'bg-red-500'} />
        <Bar label="Slack"  pct={slackPct} color={slackPct >= 80 ? 'bg-green-500' : 'bg-yellow-500'} />
        <Bar label="GitHub" pct={ghPct}    color={ghPct === 100 ? 'bg-green-500' : 'bg-blue-500'} />
        <Bar label="Tasks"  pct={todoPct}  color={todoPct >= 80 ? 'bg-green-500' : 'bg-cyan-500'} />
      </div>
    </div>
  );
}
