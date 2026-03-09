'use client';

import { useState } from 'react';

interface Step {
  text: string;
  url?: string;
}

interface ConnectPromptProps {
  title: string;
  steps: Step[];
  configSnippet: string;
}

export function ConnectPrompt({ title, steps, configSnippet }: ConnectPromptProps) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(configSnippet).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="p-4 space-y-4">
      {/* Header */}
      <div className="flex items-center gap-2.5">
        <div className="w-1 h-5 bg-gradient-to-b from-amber-400 to-amber-600 rounded-full flex-shrink-0" />
        <span className="text-sm font-semibold text-gray-200">Connect {title}</span>
      </div>

      {/* Steps */}
      <ol className="space-y-2.5">
        {steps.map((s, i) => (
          <li key={i} className="flex items-start gap-3">
            <span className="flex-shrink-0 w-5 h-5 rounded-full bg-gray-800 border border-gray-700 text-xs text-gray-500 flex items-center justify-center font-mono mt-0.5">
              {i + 1}
            </span>
            <span className="text-xs text-gray-400 leading-relaxed">
              {s.url ? (
                <a
                  href={s.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-cyan-400 hover:text-cyan-300 underline underline-offset-2 transition-colors"
                >
                  {s.text} ↗
                </a>
              ) : s.text}
            </span>
          </li>
        ))}
      </ol>

      {/* Config snippet */}
      <div className="rounded-lg border border-gray-700/60 overflow-hidden">
        <div className="flex items-center justify-between px-3 py-1.5 bg-gray-800/80 border-b border-gray-700/60">
          <code className="text-xs text-gray-500">~/.daily-briefing/config.json</code>
          <button
            onClick={copy}
            className="text-xs text-gray-500 hover:text-gray-200 transition-colors px-2 py-0.5 rounded hover:bg-gray-700/60"
          >
            {copied ? '✓ copied' : 'copy'}
          </button>
        </div>
        <pre className="text-xs text-emerald-300/80 bg-gray-950 p-3 overflow-x-auto leading-relaxed">{configSnippet}</pre>
      </div>

      {/* Footer */}
      <p className="text-xs text-gray-600">
        Restart the server after saving the config.
      </p>
    </div>
  );
}

export function NotAvailable({ name }: { name: string }) {
  return (
    <div className="p-4 space-y-2">
      <div className="flex items-center gap-2">
        <div className="w-1.5 h-4 bg-gray-600 rounded-full flex-shrink-0" />
        <span className="text-sm font-semibold text-gray-400">{name} — Not yet implemented</span>
      </div>
      <p className="text-xs text-gray-600 pl-4">
        This integration does not have a real API connection yet. The section will appear once it is built.
      </p>
    </div>
  );
}
