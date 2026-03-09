export function SummaryBanner({ summary }: { summary: string }) {
  return (
    <div className="relative bg-gray-900 border border-gray-800 rounded-xl p-4 mb-4 overflow-hidden">
      {/* Subtle background glow */}
      <div className="absolute inset-0 bg-gradient-to-r from-cyan-950/20 to-transparent pointer-events-none" />
      <div className="relative flex items-start gap-3">
        <span className="flex-shrink-0 text-xs font-bold bg-cyan-950 text-cyan-300 border border-cyan-800/50 px-2 py-0.5 rounded-full mt-0.5">
          AI
        </span>
        <div>
          <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-widest mb-1.5">
            Briefing Summary
          </h3>
          <p className="text-sm text-gray-300 leading-relaxed">{summary}</p>
        </div>
      </div>
    </div>
  );
}
