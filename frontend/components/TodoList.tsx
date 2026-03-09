'use client';

import { useState } from 'react';
import type { TodoItem, RecurringRule } from '@/lib/types';
import { createTodo, updateTodo, deleteTodo, setRecurring } from '@/lib/api';
import { PRIORITY_LABEL, PRIORITY_COLOR } from '@/lib/utils';

const RECUR_LABELS: Record<string, string> = {
  daily: 'Daily',
  weekdays: 'Weekdays',
  weekly: 'Weekly',
};

type Filter = 'all' | 'pending' | 'active' | 'done' | 'urgent';

interface Props {
  todos: TodoItem[];
  onTodosChange: (todos: TodoItem[]) => void;
}

export function TodoList({ todos, onTodosChange }: Props) {
  const [input, setInput] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  const [adding, setAdding] = useState(false);
  const [recurringOpen, setRecurringOpen] = useState<string | null>(null);

  const filtered = todos.filter(t => {
    if (filter === 'all')     return t.status === 0 || t.status === 1;
    if (filter === 'pending') return t.status === 0;
    if (filter === 'active')  return t.status === 1;
    if (filter === 'done')    return t.status === 2;
    if (filter === 'urgent')  return t.priority === 3 && (t.status === 0 || t.status === 1);
    return true;
  });

  const pendingCount = todos.filter(t => t.status === 0 || t.status === 1).length;

  async function handleAdd() {
    const title = input.trim();
    if (!title) return;
    setAdding(true);
    try {
      const created = await createTodo(title);
      onTodosChange([...todos, created]);
      setInput('');
    } finally {
      setAdding(false);
    }
  }

  async function handleComplete(todo: TodoItem) {
    const next = todo.status === 2 ? 0 : 2;
    onTodosChange(todos.map(t => t.id === todo.id ? { ...t, status: next as TodoItem['status'] } : t));
    try {
      await updateTodo(todo.id, { status: next });
    } catch {
      onTodosChange(todos); // revert
    }
  }

  async function handleToggleActive(todo: TodoItem) {
    const next = todo.status === 1 ? 0 : 1;
    onTodosChange(todos.map(t => t.id === todo.id ? { ...t, status: next as TodoItem['status'] } : t));
    try {
      await updateTodo(todo.id, { status: next });
    } catch {
      onTodosChange(todos);
    }
  }

  async function handleDelete(id: string) {
    onTodosChange(todos.filter(t => t.id !== id));
    try {
      await deleteTodo(id);
    } catch {
      onTodosChange(todos);
    }
  }

  async function handleSetRecurring(todo: TodoItem, freq: RecurringRule['frequency'] | null) {
    setRecurringOpen(null);
    const rule: RecurringRule | null = freq ? { frequency: freq, enabled: true } : null;
    onTodosChange(todos.map(t => t.id === todo.id ? { ...t, recurring: rule ?? undefined } : t));
    try {
      await setRecurring(todo.id, rule);
    } catch {
      onTodosChange(todos);
    }
  }

  const FILTERS: { key: Filter; label: string }[] = [
    { key: 'all', label: 'All' },
    { key: 'pending', label: 'Pending' },
    { key: 'active', label: 'Active' },
    { key: 'done', label: 'Done' },
    { key: 'urgent', label: 'Urgent' },
  ];

  return (
    <div className="panel">
      <div className="panel-header">
        <h3 className="section-title">Action Items</h3>
        <span className={`text-xs font-bold px-2 py-0.5 rounded-full tabular-nums ${
          pendingCount > 0 ? 'bg-rose-950/60 text-rose-300 border border-rose-800/40' : 'bg-emerald-950/60 text-emerald-300 border border-emerald-800/40'
        }`}>
          {pendingCount}
        </span>
      </div>

      {/* Add input */}
      <div className="flex gap-2 p-3 border-b border-gray-800">
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleAdd()}
          placeholder="Add a task…"
          className="flex-1 bg-gray-800/80 text-gray-100 placeholder-gray-600 text-sm px-3 py-2 rounded-lg border border-gray-700/60 focus:outline-none focus:border-cyan-500/60 transition-colors"
        />
        <button
          onClick={handleAdd}
          disabled={adding || !input.trim()}
          className="px-3 py-2 bg-cyan-600 hover:bg-cyan-700 text-white text-sm font-semibold rounded-lg disabled:opacity-40 transition-colors"
        >
          Add
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-1 px-3 py-2 border-b border-gray-800">
        {FILTERS.map(f => (
          <button
            key={f.key}
            onClick={() => setFilter(f.key)}
            className={`text-xs px-2.5 py-1 rounded-full transition-colors font-medium ${
              filter === f.key
                ? 'bg-cyan-600 text-white'
                : 'text-gray-500 hover:text-gray-300 hover:bg-gray-800/60'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* List */}
      <div className="overflow-y-auto max-h-96 divide-y divide-gray-800/60">
        {filtered.length === 0 ? (
          <div className="py-8 text-center text-gray-600 text-sm">
            {filter === 'done' ? 'No completed tasks.' : 'All clear!'}
          </div>
        ) : (
          filtered.map(todo => (
            <div
              key={todo.id}
              className="flex items-start gap-3 px-4 py-3 hover:bg-gray-800/40 group transition-colors"
            >
              {/* Checkbox */}
              <button
                onClick={() => handleComplete(todo)}
                className={`mt-0.5 w-4 h-4 rounded-md border flex-shrink-0 flex items-center justify-center text-xs transition-all ${
                  todo.status === 2
                    ? 'bg-emerald-500 border-emerald-500 text-white'
                    : 'border-gray-600 hover:border-cyan-500'
                }`}
              >
                {todo.status === 2 && '\u2713'}
              </button>

              <div className="flex-1 min-w-0">
                <div className={`text-sm leading-snug ${todo.status === 2 ? 'line-through text-gray-600' : 'text-gray-100'}`}>
                  {todo.title}
                </div>
                <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                  <span className={`text-xs font-medium ${PRIORITY_COLOR[todo.priority]}`}>
                    {PRIORITY_LABEL[todo.priority]}
                  </span>
                  <span className="text-xs text-gray-700">{todo.source}</span>
                  {todo.status === 1 && (
                    <span className="text-xs text-amber-400 font-medium">active</span>
                  )}
                  {todo.recurring?.enabled && (
                    <span className="text-xs text-cyan-500/80 font-medium" title={`Recurs ${todo.recurring.frequency}`}>
                      ↻ {RECUR_LABELS[todo.recurring.frequency] ?? todo.recurring.frequency}
                    </span>
                  )}
                </div>
              </div>

              {/* Hover actions */}
              <div className="relative flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => handleToggleActive(todo)}
                  className="p-1.5 text-xs text-gray-600 hover:text-amber-400 transition-colors rounded"
                  title="Toggle active"
                  aria-label="Toggle active"
                >
                  &#9679;
                </button>
                <button
                  onClick={() => setRecurringOpen(recurringOpen === todo.id ? null : todo.id)}
                  className={`p-1.5 text-xs transition-colors rounded ${
                    todo.recurring?.enabled ? 'text-cyan-400 hover:text-cyan-300' : 'text-gray-600 hover:text-cyan-400'
                  }`}
                  title="Set recurrence"
                  aria-label="Set recurrence"
                >
                  ↻
                </button>
                <button
                  onClick={() => handleDelete(todo.id)}
                  className="p-1.5 text-xs text-gray-600 hover:text-rose-400 transition-colors rounded"
                  title="Delete"
                  aria-label="Delete todo"
                >
                  &#10005;
                </button>

                {/* Recurrence dropdown */}
                {recurringOpen === todo.id && (
                  <div className="absolute right-0 top-7 z-10 w-32 bg-gray-800 border border-gray-700/60 rounded-lg shadow-2xl overflow-hidden">
                    {(['daily', 'weekdays', 'weekly'] as const).map(freq => (
                      <button
                        key={freq}
                        onClick={() => handleSetRecurring(todo, freq)}
                        className={`w-full text-left px-3 py-2 text-xs hover:bg-gray-700/60 transition-colors ${
                          todo.recurring?.frequency === freq && todo.recurring.enabled ? 'text-cyan-400' : 'text-gray-400'
                        }`}
                      >
                        {RECUR_LABELS[freq]}
                      </button>
                    ))}
                    <button
                      onClick={() => handleSetRecurring(todo, null)}
                      className="w-full text-left px-3 py-2 text-xs text-gray-600 hover:bg-gray-700/60 hover:text-gray-300 transition-colors border-t border-gray-700/60"
                    >
                      None
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
