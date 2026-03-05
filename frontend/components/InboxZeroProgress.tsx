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
  const s = briefing.integration_statuses ?? {};
  const todos = briefing.todos ?? [];
  const doneTodos = todos.filter(t => t.status === 2).length;
  const totalTodos = todos.length;
  const todoPct = totalTodos > 0 ? Math.round((doneTodos / totalTodos) * 100) : 100;

  const bars: { label: string; pct: number; color: string }[] = [];

  if (s.email) {
    const unreadEmails = briefing.unread_emails?.filter(e => e.is_unread).length ?? 0;
    const pct = Math.max(0, Math.min(100, Math.round(100 - unreadEmails * 5)));
    bars.push({ label: 'Email', pct, color: pct === 100 ? 'bg-green-500' : 'bg-red-500' });
  }
  if (s.slack) {
    const slackMsgs = briefing.slack_messages?.length ?? 0;
    const pct = Math.max(0, Math.min(100, Math.round(100 - slackMsgs * 10)));
    bars.push({ label: 'Slack', pct, color: pct >= 80 ? 'bg-green-500' : 'bg-yellow-500' });
  }
  if (s.github) {
    const unreadGH = briefing.github_notifications?.filter(n => n.unread).length ?? 0;
    const pct = Math.max(0, Math.min(100, Math.round(100 - unreadGH * 10)));
    bars.push({ label: 'GitHub', pct, color: pct === 100 ? 'bg-green-500' : 'bg-blue-500' });
  }
  bars.push({ label: 'Tasks', pct: todoPct, color: todoPct >= 80 ? 'bg-green-500' : 'bg-cyan-500' });

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
      <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">
        Focus Progress
      </h3>
      <div className="space-y-3">
        {bars.map(b => <Bar key={b.label} label={b.label} pct={b.pct} color={b.color} />)}
      </div>
      {bars.length === 1 && (
        <p className="text-xs text-gray-600 mt-3">Connect Email, Slack, or GitHub for more signals.</p>
      )}
    </div>
  );
}
