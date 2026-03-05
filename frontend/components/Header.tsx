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
    <header className="sticky top-0 z-40 flex items-center justify-between px-4 py-3 bg-gray-900 border-b border-gray-800">
      <div>
        <span className="font-bold text-white">{greeting}!</span>
        <span className="text-gray-400 ml-2 text-sm">{dateStr}</span>
      </div>
      <div className="flex items-center gap-2 text-sm">
        <span className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-green-400' : 'bg-gray-600'}`} />
        <span className="text-gray-400">{wsConnected ? 'Live' : 'Connecting...'}</span>
      </div>
    </header>
  );
}
