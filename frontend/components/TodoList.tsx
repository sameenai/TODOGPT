'use client';

import { useState } from 'react';
import type { TodoItem } from '@/lib/types';
import { createTodo, updateTodo, deleteTodo } from '@/lib/api';
import { PRIORITY_LABEL, PRIORITY_COLOR } from '@/lib/utils';

type Filter = 'all' | 'pending' | 'active' | 'done' | 'urgent';

interface Props {
  todos: TodoItem[];
  onTodosChange: (todos: TodoItem[]) => void;
}

export function TodoList({ todos, onTodosChange }: Props) {
  const [input, setInput] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  const [adding, setAdding] = useState(false);

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

  const FILTERS: { key: Filter; label: string }[] = [
    { key: 'all', label: 'All' },
    { key: 'pending', label: 'Pending' },
    { key: 'active', label: 'Active' },
    { key: 'done', label: 'Done' },
    { key: 'urgent', label: 'Urgent' },
  ];

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Action Items</h3>
        <span className={`text-xs font-bold px-2 py-0.5 rounded ${
          pendingCount > 0 ? 'bg-red-900 text-red-300' : 'bg-green-900 text-green-300'
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
          placeholder="Add a task..."
          className="flex-1 bg-gray-800 text-gray-100 placeholder-gray-500 text-sm px-3 py-2 rounded border border-gray-700 focus:outline-none focus:border-cyan-500"
        />
        <button
          onClick={handleAdd}
          disabled={adding || !input.trim()}
          className="px-3 py-2 bg-cyan-600 hover:bg-cyan-700 text-white text-sm font-medium rounded disabled:opacity-50 transition-colors"
        >
          Add
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-1 p-2 border-b border-gray-800">
        {FILTERS.map(f => (
          <button
            key={f.key}
            onClick={() => setFilter(f.key)}
            className={`text-xs px-2 py-1 rounded transition-colors ${
              filter === f.key ? 'bg-cyan-600 text-white' : 'text-gray-400 hover:text-gray-200'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* List */}
      <div className="overflow-y-auto max-h-96 divide-y divide-gray-800">
        {filtered.length === 0 ? (
          <div className="py-8 text-center text-gray-500 text-sm">
            {filter === 'done' ? 'No completed tasks.' : 'All clear!'}
          </div>
        ) : (
          filtered.map(todo => (
            <div
              key={todo.id}
              className="flex items-start gap-3 px-4 py-3 hover:bg-gray-800/50 group transition-colors"
            >
              {/* Checkbox */}
              <button
                onClick={() => handleComplete(todo)}
                className={`mt-0.5 w-4 h-4 rounded border flex-shrink-0 flex items-center justify-center text-xs transition-colors ${
                  todo.status === 2
                    ? 'bg-green-500 border-green-500 text-white'
                    : 'border-gray-600 hover:border-cyan-400'
                }`}
              >
                {todo.status === 2 && '\u2713'}
              </button>

              <div className="flex-1 min-w-0">
                <div className={`text-sm ${todo.status === 2 ? 'line-through text-gray-500' : 'text-gray-100'}`}>
                  {todo.title}
                </div>
                <div className="flex items-center gap-2 mt-0.5">
                  <span className={`text-xs ${PRIORITY_COLOR[todo.priority]}`}>
                    {PRIORITY_LABEL[todo.priority]}
                  </span>
                  <span className="text-xs text-gray-600">{todo.source}</span>
                  {todo.status === 1 && (
                    <span className="text-xs text-yellow-400 font-medium">active</span>
                  )}
                </div>
              </div>

              {/* Hover actions */}
              <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => handleToggleActive(todo)}
                  className="p-1 text-xs text-gray-400 hover:text-yellow-400 transition-colors"
                  title="Toggle active"
                >
                  &#9679;
                </button>
                <button
                  onClick={() => handleDelete(todo.id)}
                  className="p-1 text-xs text-gray-400 hover:text-red-400 transition-colors"
                  title="Delete"
                >
                  &#10005;
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
