export function StatusBadge({ live }: { live?: boolean }) {
  if (live === undefined) return null;
  return live ? (
    <span className="text-xs font-semibold text-green-400 bg-green-950 border border-green-800 px-1.5 py-0.5 rounded">
      LIVE
    </span>
  ) : (
    <span className="text-xs font-semibold text-amber-400 bg-amber-950 border border-amber-800 px-1.5 py-0.5 rounded">
      DEMO
    </span>
  );
}
