'use client';

import { useRef, useEffect } from 'react';
import { useChat } from '@ai-sdk/react';

const SUGGESTIONS = [
  'What should I focus on today?',
  'Summarize my unread emails',
  'What meetings do I have?',
  'What are my most urgent tasks?',
];

interface Props {
  open: boolean;
  onToggle: () => void;
}

export function BriefingChat({ open, onToggle }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const { messages, input, handleInputChange, handleSubmit, status, append } = useChat({
    api: '/api/chat',
  });

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const isStreaming = status === 'streaming' || status === 'submitted';

  return (
    <>
      {/* Floating toggle button */}
      <button
        onClick={onToggle}
        className="fixed bottom-6 right-6 z-50 w-14 h-14 rounded-full bg-cyan-600 hover:bg-cyan-500 text-white text-xl shadow-lg flex items-center justify-center transition-colors"
        aria-label="Toggle AI assistant"
      >
        {open ? '\u2715' : '\ud83e\udd16'}
      </button>

      {/* Chat panel */}
      {open && (
        <div className="fixed bottom-24 right-6 z-50 w-96 h-[520px] bg-gray-900 border border-gray-800 rounded-xl shadow-2xl flex flex-col overflow-hidden">
          {/* Header */}
          <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
            <span className="text-lg">\ud83e\udd16</span>
            <div>
              <div className="text-sm font-semibold text-white">Briefing Assistant</div>
              <div className="text-xs text-gray-500">Powered by Claude</div>
            </div>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4 space-y-3">
            {messages.length === 0 ? (
              <div className="space-y-2">
                <p className="text-xs text-gray-500 text-center mb-3">Ask me about your day</p>
                {SUGGESTIONS.map(q => (
                  <button
                    key={q}
                    onClick={() => append({ role: 'user', content: q })}
                    className="w-full text-left text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 px-3 py-2 rounded-lg transition-colors"
                  >
                    {q}
                  </button>
                ))}
              </div>
            ) : (
              messages.map(m => (
                <div key={m.id} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[85%] rounded-xl px-3 py-2 text-sm ${
                    m.role === 'user'
                      ? 'bg-cyan-700 text-white'
                      : 'bg-gray-800 text-gray-100'
                  }`}>
                    {m.parts.map((part, i) =>
                      part.type === 'text' ? (
                        <span key={i} className="whitespace-pre-wrap">{part.text}</span>
                      ) : null
                    )}
                  </div>
                </div>
              ))
            )}

            {isStreaming && messages[messages.length - 1]?.role === 'user' && (
              <div className="flex justify-start">
                <div className="bg-gray-800 rounded-xl px-3 py-2">
                  <span className="text-gray-400 text-sm animate-pulse">...</span>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>

          {/* Input */}
          <form onSubmit={handleSubmit} className="flex gap-2 p-3 border-t border-gray-800">
            <input
              value={input}
              onChange={handleInputChange}
              placeholder="Ask about your day..."
              disabled={isStreaming}
              className="flex-1 bg-gray-800 text-gray-100 placeholder-gray-500 text-sm px-3 py-2 rounded-lg border border-gray-700 focus:outline-none focus:border-cyan-500 disabled:opacity-50"
            />
            <button
              type="submit"
              disabled={isStreaming || !input.trim()}
              className="px-3 py-2 bg-cyan-600 hover:bg-cyan-500 text-white text-sm rounded-lg disabled:opacity-50 transition-colors"
            >
              Send
            </button>
          </form>
        </div>
      )}
    </>
  );
}
