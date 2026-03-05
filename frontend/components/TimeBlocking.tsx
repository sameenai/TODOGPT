'use client';

import { useState } from 'react';
import type { TimeBlock } from '@/lib/types';
import { fetchTimeBlocks } from '@/lib/api';

const COLOR_CLASS: Record<string, string> = {
  red: 'border-red-500 bg-red-950/40',
  orange: 'border-orange-400 bg-orange-950/40',
  blue: 'border-blue-500 bg-blue-950/40',
  gray: 'border-gray-600 bg-gray-800/40',
};

const DOT_CLASS: Record<string, string> = {
  red: 'bg-red-500',
  orange: 'bg-orange-400',
  blue: 'bg-blue-500',
  gray: 'bg-gray-500',
};

function BlockCard({ block }: { block: TimeBlock }) {
  const color = block.color ?? 'blue';
  return (
    <div className={`border-l-2 rounded-r px-3 py-2 ${COLOR_CLASS[color] ?? COLOR_CLASS.blue}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="text-sm font-medium text-gray-100 truncate">{block.title}</div>
          {block.notes && (
            <div className="text-xs text-gray-500 mt-0.5 truncate">{block.notes}</div>
          )}
        </div>
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <span className={`w-2 h-2 rounded-full flex-shrink-0 ${DOT_CLASS[color] ?? DOT_CLASS.blue}`} aria-hidden="true" />
          <span className="text-xs text-gray-400 font-mono whitespace-nowrap">
            {block.start}–{block.end}
          </span>
        </div>
      </div>
    </div>
  );
}

export function TimeBlocking() {
  const [blocks, setBlocks] = useState<TimeBlock[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [generated, setGenerated] = useState(false);

  async function suggest() {
    setLoading(true);
    setError('');
    try {
      const result = await fetchTimeBlocks();
      if (result.length === 0 && !generated) {
        setError('AI is not configured. Enable it in Settings and add your Anthropic API key.');
      } else {
        setBlocks(result);
        setGenerated(true);
      }
    } catch {
      setError('Failed to generate schedule — check that the server is running.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800">
        <div>
          <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Time Blocking</h3>
          <p className="text-xs text-gray-600 mt-0.5">AI-suggested schedule</p>
        </div>
        <button
          onClick={suggest}
          disabled={loading}
          className="text-xs px-2.5 py-1 bg-cyan-700 hover:bg-cyan-600 text-white rounded transition-colors disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500"
        >
          {loading ? '…' : blocks.length ? '↻ Refresh' : 'Suggest'}
        </button>
      </div>

      <div className="p-3">
        {error && (
          <p className="text-xs text-gray-500 text-center py-2">{error}</p>
        )}

        {!error && !loading && blocks.length === 0 && (
          <p className="text-xs text-gray-600 text-center py-4">
            Click Suggest to generate a focused work schedule based on your todos and calendar.
          </p>
        )}

        {loading && (
          <div className="flex items-center gap-2 py-3 text-xs text-gray-500">
            <span className="animate-spin">⟳</span>
            <span>Analysing your day…</span>
          </div>
        )}

        {!loading && blocks.length > 0 && (
          <div className="space-y-2">
            {blocks.map((b, i) => (
              <BlockCard key={`${b.start}-${i}`} block={b} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
