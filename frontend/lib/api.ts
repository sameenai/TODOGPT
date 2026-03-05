// Client-side API calls.
// All paths go through Next.js rewrites: /api/go/* → Go backend /api/*
import type { Briefing, TodoItem } from './types';

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
