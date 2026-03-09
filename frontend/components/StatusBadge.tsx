export function StatusBadge({ live }: { live?: boolean }) {
  if (live === undefined) return null;
  return live ? (
    <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-400 bg-emerald-950/50 border border-emerald-800/60 px-2 py-0.5 rounded-full">
      <span aria-hidden="true" className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
      LIVE
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 text-xs font-semibold text-amber-400/90 bg-amber-950/40 border border-amber-800/50 px-2 py-0.5 rounded-full">
      <span aria-hidden="true" className="w-1.5 h-1.5 rounded-full bg-amber-500/70" />
      DEMO
    </span>
  );
}
