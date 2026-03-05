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
  return (
    <div className="px-4 py-5 space-y-4">
      <p className="text-sm text-gray-400">
        Connect <span className="text-gray-200 font-medium">{title}</span> by adding credentials to{' '}
        <code className="text-xs bg-gray-800 text-cyan-300 px-1.5 py-0.5 rounded">~/.daily-briefing/config.json</code>
        , then restart the server.
      </p>

      <ol className="space-y-1.5 text-sm text-gray-400 list-decimal list-inside">
        {steps.map((s, i) => (
          <li key={i}>
            {s.url ? (
              <a href={s.url} target="_blank" rel="noopener noreferrer" className="text-cyan-400 underline underline-offset-2 hover:text-cyan-300">
                {s.text}
              </a>
            ) : s.text}
          </li>
        ))}
      </ol>

      <pre className="text-xs bg-gray-950 border border-gray-800 rounded p-3 text-green-300 overflow-x-auto whitespace-pre">{configSnippet}</pre>
    </div>
  );
}

export function NotAvailable({ name }: { name: string }) {
  return (
    <div className="px-4 py-6 text-center">
      <p className="text-sm text-gray-500">
        <span className="text-gray-400 font-medium">{name}</span> integration is not yet implemented.
      </p>
    </div>
  );
}
