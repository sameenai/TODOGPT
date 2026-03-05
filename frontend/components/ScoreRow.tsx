import type { Briefing } from '@/lib/types';

function inboxZeroScore(b: Briefing): number {
  const unreadEmails = b.unread_emails?.filter(e => e.is_unread).length ?? 0;
  const slackMsgs = b.slack_messages?.length ?? 0;
  const unreadGH = b.github_notifications?.filter(n => n.unread).length ?? 0;
  const todos = b.todos ?? [];
  const doneTodos = todos.filter(t => t.status === 2).length;
  const totalTodos = todos.length;

  const emailPct = Math.max(0, 100 - unreadEmails * 5);
  const slackPct = Math.max(0, 100 - slackMsgs * 10);
  const ghPct = Math.max(0, 100 - unreadGH * 10);
  const todoPct = totalTodos > 0 ? Math.round((doneTodos / totalTodos) * 100) : 100;

  return Math.round((emailPct + slackPct + ghPct + todoPct) / 4);
}

interface Card {
  label: string;
  value: string | number;
  icon: string;
  color: string;
}

export function ScoreRow({ briefing }: { briefing: Briefing }) {
  const unreadEmails = briefing.unread_emails?.filter(e => e.is_unread).length ?? 0;
  const slackCount = briefing.slack_messages?.length ?? 0;
  const ghCount = briefing.github_notifications?.filter(n => n.unread).length ?? 0;
  const eventCount = briefing.events?.length ?? 0;
  const pendingTodos = briefing.todos?.filter(t => t.status === 0 || t.status === 1).length ?? 0;
  const izScore = inboxZeroScore(briefing);

  const cards: Card[] = [
    { label: 'Unread Emails', value: unreadEmails, icon: '\u2709', color: unreadEmails > 0 ? 'text-red-400' : 'text-green-400' },
    { label: 'Slack', value: slackCount, icon: '\ud83d\udcac', color: slackCount > 0 ? 'text-yellow-400' : 'text-green-400' },
    { label: 'GitHub', value: ghCount, icon: '\u2699', color: ghCount > 0 ? 'text-blue-400' : 'text-green-400' },
    { label: 'Events Today', value: eventCount, icon: '\ud83d\udcc5', color: 'text-cyan-400' },
    { label: 'Action Items', value: pendingTodos, icon: '\u2611', color: pendingTodos > 0 ? 'text-yellow-400' : 'text-green-400' },
    { label: 'Inbox Zero', value: `${izScore}%`, icon: '\ud83c\udfaf', color: izScore >= 80 ? 'text-green-400' : izScore >= 50 ? 'text-yellow-400' : 'text-red-400' },
  ];

  return (
    <div className="grid grid-cols-3 md:grid-cols-6 gap-3 mb-4">
      {cards.map(c => (
        <div key={c.label} className="bg-gray-900 border border-gray-800 rounded-lg p-3 flex items-center gap-3">
          <span className="text-xl">{c.icon}</span>
          <div>
            <div className={`text-xl font-bold ${c.color}`}>{c.value}</div>
            <div className="text-xs text-gray-500">{c.label}</div>
          </div>
        </div>
      ))}
    </div>
  );
}
