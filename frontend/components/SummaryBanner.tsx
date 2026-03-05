export function SummaryBanner({ summary }: { summary: string }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 mb-4">
      <div className="flex items-center gap-2 mb-2">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">AI Briefing Summary</h3>
        <span className="text-xs bg-cyan-900 text-cyan-300 px-2 py-0.5 rounded font-medium">AI</span>
      </div>
      <p className="text-sm text-gray-300 leading-relaxed">{summary}</p>
    </div>
  );
}
