'use client';

import { useState } from 'react';
import { fetchDailyReview } from '@/lib/api';

interface Props {
  open: boolean;
  onClose: () => void;
}

export function DailyReview({ open, onClose }: Props) {
  const [review, setReview] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function generate() {
    setLoading(true);
    setError('');
    setReview('');
    try {
      const text = await fetchDailyReview();
      if (!text) {
        setError('AI is not configured. Enable it in Settings and add your Anthropic API key.');
      } else {
        setReview(text);
      }
    } catch {
      setError('Failed to generate review — check that the server is running.');
    } finally {
      setLoading(false);
    }
  }

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div className="relative w-full max-w-lg bg-gray-950 border border-gray-800 rounded-xl shadow-2xl flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
          <div>
            <h2 className="text-sm font-semibold text-white">End-of-Day Review</h2>
            <p className="text-xs text-gray-500 mt-0.5">AI-generated summary of your day</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white text-lg leading-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 rounded"
            aria-label="Close review"
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-5 min-h-[200px]">
          {!review && !loading && !error && (
            <div className="flex flex-col items-center justify-center h-full py-8 text-center">
              <div className="text-4xl mb-3">🌅</div>
              <p className="text-sm text-gray-400 mb-1">Ready to wrap up your day?</p>
              <p className="text-xs text-gray-600">Claude will review your completed tasks, carry-forwards, and suggest improvements.</p>
            </div>
          )}

          {loading && (
            <div className="flex items-center gap-2 text-sm text-gray-400">
              <span className="animate-spin">⟳</span>
              <span>Generating review…</span>
            </div>
          )}

          {error && (
            <div className="bg-red-950/50 border border-red-800 rounded-lg p-3 text-sm text-red-400">
              {error}
            </div>
          )}

          {review && (
            <div className="text-sm text-gray-200 leading-relaxed whitespace-pre-wrap">
              {review}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-5 py-4 border-t border-gray-800">
          <p className="text-xs text-gray-600">Powered by Claude</p>
          <button
            onClick={generate}
            disabled={loading}
            className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-medium rounded disabled:opacity-50 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500"
          >
            {loading ? 'Generating…' : review ? 'Regenerate' : 'Generate Review'}
          </button>
        </div>
      </div>
    </div>
  );
}
