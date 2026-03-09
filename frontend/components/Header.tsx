'use client';

interface Props {
  wsConnected: boolean;
}

export function Header({ wsConnected }: Props) {
  const now = new Date();
  const h = now.getHours();
  const greeting = h < 12 ? 'Good morning' : h < 17 ? 'Good afternoon' : 'Good evening';
  const dateStr = now.toLocaleDateString('en-US', {
    weekday: 'long', month: 'long', day: 'numeric',
  });

  return (
    <header className="sticky top-0 z-40 bg-gray-950/90 backdrop-blur border-b border-gray-800/80">
      {/* Gradient accent line */}
      <div className="h-px bg-gradient-to-r from-transparent via-cyan-500/40 to-transparent" />

      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-3">
          {/* App mark */}
          <div className="w-6 h-6 rounded-md bg-gradient-to-br from-cyan-500 to-cyan-700 flex items-center justify-center flex-shrink-0">
            <span className="text-white text-xs font-bold leading-none">B</span>
          </div>
          <div>
            <span className="font-semibold text-white">{greeting}!</span>
            <span className="text-gray-500 ml-2 text-sm">{dateStr}</span>
          </div>
        </div>

        <div className="flex items-center gap-2 text-xs">
          <span className={`w-1.5 h-1.5 rounded-full transition-colors ${wsConnected ? 'bg-emerald-400 animate-pulse' : 'bg-gray-600'}`} />
          <span className={wsConnected ? 'text-emerald-400/80' : 'text-gray-500'}>
            {wsConnected ? 'Live' : 'Connecting…'}
          </span>
        </div>
      </div>
    </header>
  );
}
