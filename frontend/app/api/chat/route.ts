import { streamText, convertToCoreMessages } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';
import type { Message } from 'ai';
import type { NextRequest } from 'next/server';

const GO_BACKEND = process.env.GO_BACKEND_URL || 'http://localhost:8080';

function fmtTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

type BriefingLike = Record<string, unknown>;

function buildSystemPrompt(briefing: BriefingLike): string {
  const lines: string[] = [
    `You are a helpful personal assistant. Today is ${new Date().toLocaleDateString('en-US', {
      weekday: 'long', month: 'long', day: 'numeric', year: 'numeric',
    })}.`,
    "You have access to the user's live dashboard data. Be concise and actionable.",
    '',
  ];

  const weather = briefing.weather as BriefingLike | undefined;
  if (weather) {
    lines.push(`WEATHER: ${weather.temperature}°F, ${weather.description} in ${weather.city}`);
  }

  const events = (briefing.events as BriefingLike[]) ?? [];
  if (events.length) {
    lines.push(`\nCALENDAR (${events.length} events today):`);
    events.slice(0, 8).forEach(e => {
      lines.push(`- ${fmtTime(e.start_time as string)} ${e.title}${e.location ? ` @ ${e.location}` : ''}`);
    });
  }

  const emails = ((briefing.unread_emails as BriefingLike[]) ?? []).filter(e => e.is_unread);
  if (emails.length) {
    lines.push(`\nEMAIL (${emails.length} unread):`);
    emails.slice(0, 5).forEach(e => {
      lines.push(`- From ${e.from}: "${e.subject}"${e.snippet ? ` — ${String(e.snippet).slice(0, 80)}` : ''}`);
    });
  }

  const slack = (briefing.slack_messages as BriefingLike[]) ?? [];
  if (slack.length) {
    lines.push(`\nSLACK (${slack.length} messages):`);
    slack.slice(0, 5).forEach(m => {
      lines.push(`- ${m.user} in ${m.channel}: "${String(m.text).slice(0, 80)}"`);
    });
  }

  const todos = (briefing.todos as BriefingLike[]) ?? [];
  const pending = todos.filter(t => t.status === 0 || t.status === 1);
  if (pending.length) {
    lines.push(`\nTODOS (${pending.length} open):`);
    pending.slice(0, 10).forEach(t => {
      const prio = (['low', 'medium', 'high', 'urgent'] as const)[(t.priority as number) ?? 1];
      lines.push(`- [${prio}] ${t.title} (${t.source})`);
    });
  }

  const ghNotifs = (briefing.github_notifications as BriefingLike[]) ?? [];
  const unreadGH = ghNotifs.filter(n => n.unread);
  if (unreadGH.length) {
    lines.push(`\nGITHUB (${unreadGH.length} unread notifications):`);
    unreadGH.slice(0, 5).forEach(n => {
      lines.push(`- [${n.type}] ${n.title} (${n.repo})`);
    });
  }

  return lines.join('\n');
}

export async function POST(req: NextRequest) {
  if (!process.env.ANTHROPIC_API_KEY) {
    return new Response(
      JSON.stringify({ error: 'ANTHROPIC_API_KEY not configured' }),
      { status: 503, headers: { 'Content-Type': 'application/json' } },
    );
  }

  const { messages }: { messages: Message[] } = await req.json();

  let systemPrompt = 'You are a helpful personal assistant for a daily briefing dashboard.';
  try {
    const res = await fetch(`${GO_BACKEND}/api/briefing`, {
      signal: AbortSignal.timeout(8_000),
    });
    if (res.ok) {
      const briefing = await res.json();
      systemPrompt = buildSystemPrompt(briefing as BriefingLike);
    }
  } catch { /* use generic prompt if backend unreachable */ }

  const result = streamText({
    model: anthropic('claude-sonnet-4-6'),
    system: systemPrompt,
    messages: convertToCoreMessages(messages),
  });

  return result.toDataStreamResponse();
}
