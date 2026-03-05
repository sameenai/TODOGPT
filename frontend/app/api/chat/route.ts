import { streamText, convertToCoreMessages, tool } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';
import { z } from 'zod';
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
    "You can also take actions: create todos, complete them, update their priority, or delete them.",
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
      lines.push(`- [${t.id}] [${prio}] ${t.title} (${t.source})`);
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
    maxSteps: 5,
    tools: {
      create_todo: tool({
        description: 'Create a new todo item for the user',
        parameters: z.object({
          title: z.string().describe('The todo title'),
          priority: z.enum(['low', 'medium', 'high', 'urgent']).optional().describe('Priority level'),
        }),
        execute: async ({ title, priority }) => {
          const prioMap = { low: 0, medium: 1, high: 2, urgent: 3 };
          const prio = prioMap[priority ?? 'medium'];
          try {
            const res = await fetch(`${GO_BACKEND}/api/todos`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ title, priority: prio, status: 0 }),
              signal: AbortSignal.timeout(5_000),
            });
            if (!res.ok) return `Failed to create todo (${res.status})`;
            const todo = await res.json() as { title: string };
            return `Created todo: "${todo.title}"`;
          } catch {
            return 'Failed to create todo — backend unavailable';
          }
        },
      }),

      complete_todo: tool({
        description: 'Mark a todo as complete by its ID or partial title match',
        parameters: z.object({
          id: z.string().optional().describe('The exact todo ID (preferred)'),
          title_match: z.string().optional().describe('Partial title to find the todo'),
        }),
        execute: async ({ id, title_match }) => {
          try {
            let todoId = id;
            if (!todoId && title_match) {
              const res = await fetch(`${GO_BACKEND}/api/todos`, { signal: AbortSignal.timeout(5_000) });
              if (!res.ok) return 'Failed to fetch todos';
              const todos = await res.json() as Array<{ id: string; title: string; status: number }>;
              const match = todos.find(t =>
                t.status !== 2 && t.title.toLowerCase().includes(title_match.toLowerCase())
              );
              if (!match) return `No active todo matching "${title_match}"`;
              todoId = match.id;
            }
            if (!todoId) return 'Provide either id or title_match';
            const res = await fetch(`${GO_BACKEND}/api/todos/${todoId}`, {
              method: 'PATCH',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ status: 2 }),
              signal: AbortSignal.timeout(5_000),
            });
            return res.ok ? 'Todo marked as complete' : `Failed to complete todo (${res.status})`;
          } catch {
            return 'Failed to complete todo — backend unavailable';
          }
        },
      }),

      update_todo: tool({
        description: 'Update an existing todo — change its title, priority, or notes',
        parameters: z.object({
          id: z.string().describe('The todo ID to update'),
          title: z.string().optional(),
          priority: z.enum(['low', 'medium', 'high', 'urgent']).optional(),
          notes: z.string().optional(),
        }),
        execute: async ({ id, title, priority, notes }) => {
          const prioMap = { low: 0, medium: 1, high: 2, urgent: 3 };
          const body: Record<string, unknown> = {};
          if (title) body.title = title;
          if (priority) body.priority = prioMap[priority];
          if (notes !== undefined) body.notes = notes;
          try {
            const res = await fetch(`${GO_BACKEND}/api/todos/${id}`, {
              method: 'PATCH',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(body),
              signal: AbortSignal.timeout(5_000),
            });
            return res.ok ? 'Todo updated' : `Failed to update todo (${res.status})`;
          } catch {
            return 'Failed to update todo — backend unavailable';
          }
        },
      }),

      delete_todo: tool({
        description: 'Delete a todo item permanently',
        parameters: z.object({
          id: z.string().describe('The todo ID to delete'),
        }),
        execute: async ({ id }) => {
          try {
            const res = await fetch(`${GO_BACKEND}/api/todos/${id}`, {
              method: 'DELETE',
              signal: AbortSignal.timeout(5_000),
            });
            return res.ok ? 'Todo deleted' : `Failed to delete todo (${res.status})`;
          } catch {
            return 'Failed to delete todo — backend unavailable';
          }
        },
      }),
    },
  });

  return result.toDataStreamResponse();
}
