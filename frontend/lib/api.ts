// Client-side API calls.
// All paths go through Next.js rewrites: /api/go/* → Go backend /api/*
import type { Briefing, TodoItem, ConfigResponse, TimeBlock, RecurringRule } from './types';

const BASE = '/api/go';

export async function fetchBriefing(): Promise<Briefing> {
  const res = await fetch(`${BASE}/briefing`);
  if (!res.ok) throw new Error('Failed to fetch briefing');
  return res.json();
}

export async function fetchTodos(): Promise<TodoItem[]> {
  const res = await fetch(`${BASE}/todos`);
  if (!res.ok) throw new Error('Failed to fetch todos');
  return res.json();
}

export async function createTodo(title: string): Promise<TodoItem> {
  const res = await fetch(`${BASE}/todos`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, priority: 1, status: 0 }),
  });
  if (!res.ok) throw new Error('Failed to create todo');
  return res.json();
}

export async function updateTodo(
  id: string,
  updates: { title?: string; status?: number; priority?: number; notes?: string },
): Promise<void> {
  const res = await fetch(`${BASE}/todos/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  });
  if (!res.ok) throw new Error('Failed to update todo');
}

export async function deleteTodo(id: string): Promise<void> {
  const res = await fetch(`${BASE}/todos/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete todo');
}

export async function setRecurring(id: string, rule: RecurringRule | null): Promise<void> {
  const res = await fetch(`${BASE}/todos/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ recurring: rule }),
  });
  if (!res.ok) throw new Error('Failed to set recurring rule');
}

export async function fetchConfig(): Promise<ConfigResponse> {
  const res = await fetch(`${BASE}/config`);
  if (!res.ok) throw new Error('Failed to fetch config');
  return res.json();
}

export async function saveConfig(cfg: Partial<ConfigResponse>): Promise<{ ok: boolean; message: string }> {
  const res = await fetch(`${BASE}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg),
  });
  if (!res.ok) throw new Error('Failed to save config');
  return res.json();
}

export async function fetchDailyReview(): Promise<string> {
  const res = await fetch(`${BASE}/review`);
  if (!res.ok) throw new Error('Failed to fetch review');
  const data: { review: string } = await res.json();
  return data.review;
}

export async function fetchTimeBlocks(): Promise<TimeBlock[]> {
  const res = await fetch(`${BASE}/timeblocks`);
  if (!res.ok) throw new Error('Failed to fetch time blocks');
  const data: { blocks: TimeBlock[] } = await res.json();
  return data.blocks;
}

export interface AuthStatus {
  google: { configured: boolean; connected: boolean };
}

export async function fetchAuthStatus(): Promise<AuthStatus> {
  const res = await fetch(`${BASE}/auth/status`);
  if (!res.ok) throw new Error('Failed to fetch auth status');
  return res.json();
}

export function googleAuthURL(): string {
  const returnTo = typeof window !== 'undefined' ? window.location.origin : '';
  return `${BASE}/auth/google?return_to=${encodeURIComponent(returnTo)}`;
}

export async function disconnectGoogle(): Promise<void> {
  const res = await fetch(`${BASE}/auth/google/disconnect`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to disconnect Google');
}
