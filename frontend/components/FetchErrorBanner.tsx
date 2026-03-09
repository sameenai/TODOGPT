export function FetchErrorBanner({ error }: { error: string }) {
  void error; // displayed in tooltip via title
  return (
    <span
      title={error}
      className="inline-flex items-center gap-1 text-xs text-amber-300 bg-amber-950/40 border border-amber-700/40 px-2 py-0.5 rounded-full"
    >
      <span aria-hidden="true">⚠</span>
      <span>cached</span>
    </span>
  );
}
