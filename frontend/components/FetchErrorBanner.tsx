export function FetchErrorBanner({ error }: { error: string }) {
  return (
    <span className="flex items-center gap-1 text-xs text-amber-400 bg-amber-950/30 border border-amber-800/40 px-2 py-0.5 rounded">
      <span>⚠</span>
      <span>Fetch failed · cached data</span>
    </span>
  );
}
