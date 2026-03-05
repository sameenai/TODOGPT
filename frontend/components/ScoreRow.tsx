import type { Briefing } from '@/lib/types';

interface Card {
  label: string;
  value: string | number;
  icon: string;
  color: string;
  live: boolean;
}

export function ScoreRow({ briefing }: { briefing: Briefing }) {
  const s = briefing.integration_statuses ?? {};

  const ghCount = briefing.github_notifications?.filter(n => n.unread).length ?? 0;
  const pendingTodos = briefing.todos?.filter(t => t.status === 0 || t.status === 1).length ?? 0;

  // Only count live integrations toward the score
  const liveSources: number[] = [];
  if (s.github) {
    const ghPct = Math.max(0, 100 - ghCount * 10);
    liveSources.push(ghPct);
  }
  const todos = briefing.todos ?? [];
  const doneTodos = todos.filter(t => t.status === 2).length;
  const totalTodos = todos.length;
  const todoPct = totalTodos > 0 ? Math.round((doneTodos / totalTodos) * 100) : 100;
  liveSources.push(todoPct);

  const izScore = liveSources.length > 0
    ? Math.round(liveSources.reduce((a, b) => a + b, 0) / liveSources.length)
    : 100;

  const cards: Card[] = [
    {
      label: 'Unread Emails',
      value: s.email ? (briefing.unread_emails?.filter(e => e.is_unread).length ?? 0) : '—',
      icon: '✉',
      color: s.email ? ((briefing.unread_emails?.filter(e => e.is_unread).length ?? 0) > 0 ? 'text-red-400' : 'text-green-400') : 'text-gray-600',
      live: !!s.email,
    },
    {
      label: 'Slack',
      value: s.slack ? (briefing.slack_messages?.length ?? 0) : '—',
      icon: '💬',
      color: s.slack ? ((briefing.slack_messages?.length ?? 0) > 0 ? 'text-yellow-400' : 'text-green-400') : 'text-gray-600',
      live: !!s.slack,
    },
    {
      label: 'GitHub',
      value: s.github ? ghCount : '—',
      icon: '⚙',
      color: s.github ? (ghCount > 0 ? 'text-blue-400' : 'text-green-400') : 'text-gray-600',
      live: !!s.github,
    },
    {
      label: 'Events Today',
      value: s.calendar ? (briefing.events?.length ?? 0) : '—',
      icon: '📅',
      color: s.calendar ? 'text-cyan-400' : 'text-gray-600',
      live: !!s.calendar,
    },
    {
      label: 'Action Items',
      value: pendingTodos,
      icon: '☑',
      color: pendingTodos > 0 ? 'text-yellow-400' : 'text-green-400',
      live: true,
    },
    {
      label: 'Focus Score',
      value: `${izScore}%`,
      icon: '🎯',
      color: izScore >= 80 ? 'text-green-400' : izScore >= 50 ? 'text-yellow-400' : 'text-red-400',
      live: true,
    },
  ];

  return (
    <div className="grid grid-cols-3 md:grid-cols-6 gap-3 mb-4">
      {cards.map(c => (
        <div key={c.label} className={`bg-gray-900 border rounded-lg p-3 flex items-center gap-3 ${c.live ? 'border-gray-800' : 'border-gray-800/50'}`}>
          <span className={`text-xl ${c.live ? '' : 'opacity-30'}`}>{c.icon}</span>
          <div>
            <div className={`text-xl font-bold ${c.color}`}>{c.value}</div>
            <div className="text-xs text-gray-500">{c.label}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
