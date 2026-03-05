import type { Briefing } from '@/lib/types';
import { Dashboard } from '@/components/Dashboard';

const GO_BACKEND = process.env.GO_BACKEND_URL || 'http://localhost:8080';

async function getInitialBriefing(): Promise<Briefing | null> {
  try {
    const res = await fetch(`${GO_BACKEND}/api/briefing`, {
      next: { revalidate: 0 },
      signal: AbortSignal.timeout(10_000),
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export default async function Page() {
  const initialBriefing = await getInitialBriefing();
  return <Dashboard initialBriefing={initialBriefing} />;
}
