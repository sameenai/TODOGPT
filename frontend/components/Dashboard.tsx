'use client';

import { useState, useEffect } from 'react';
import type { Briefing, TodoItem, DashboardUpdate } from '@/lib/types';
import { fetchBriefing } from '@/lib/api';
import { useWebSocket } from '@/lib/useWebSocket';

import { Header } from './Header';
import { ScoreRow } from './ScoreRow';
import { SummaryBanner } from './SummaryBanner';
import { BriefingChat } from './BriefingChat';
import { NewsSection } from './sections/NewsSection';
import { WeatherSection } from './sections/WeatherSection';
import { CalendarSection } from './sections/CalendarSection';
import { EmailSection } from './sections/EmailSection';
import { SlackSection } from './sections/SlackSection';
import { GitHubSection } from './sections/GitHubSection';
import { JiraSection } from './sections/JiraSection';
import { NotionSection } from './sections/NotionSection';
import { PomodoroTimer } from './PomodoroTimer';
import { TodoList } from './TodoList';
import { InboxZeroProgress } from './InboxZeroProgress';
import { SettingsPanel } from './SettingsPanel';
import { DailyReview } from './DailyReview';
import { TimeBlocking } from './TimeBlocking';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? 'ws://localhost:8080/ws';

interface Props {
  initialBriefing: Briefing | null;
}

const DEMO_INTEGRATION_LABELS: Record<string, string> = {
  calendar: 'Calendar',
  github: 'GitHub',
  jira: 'Jira',
  notion: 'Notion',
  email: 'Email',
  slack: 'Slack',
};

export function Dashboard({ initialBriefing }: Props) {
  const [briefing, setBriefing] = useState<Briefing | null>(initialBriefing);
  const [wsConnected, setWsConnected] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [bannerDismissed, setBannerDismissed] = useState(false);

  useWebSocket(
    WS_URL,
    (data) => {
      const msg = data as DashboardUpdate;
      if (msg.type === 'full_refresh') {
        setBriefing(msg.payload as Briefing);
      } else if (msg.type === 'todos_updated') {
        setBriefing(prev => prev ? { ...prev, todos: msg.payload as TodoItem[] } : prev);
      }
    },
    setWsConnected,
  );

  // If SSR failed to fetch (backend down), try client-side
  useEffect(() => {
    if (!initialBriefing) {
      fetchBriefing().then(setBriefing).catch(() => { /* ignore — show loading */ });
    }
  }, [initialBriefing]);

  // Handle OAuth redirect: ?google=connected|denied|error
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const googleParam = params.get('google');
    if (googleParam) {
      // Remove query param from URL without triggering a reload
      const url = new URL(window.location.href);
      url.searchParams.delete('google');
      window.history.replaceState({}, '', url.toString());
      // Open settings so user can see connection status
      if (googleParam === 'connected' || googleParam === 'denied' || googleParam === 'error') {
        setSettingsOpen(true);
      }
    }
  }, []);

  if (!briefing) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="text-gray-500 text-lg animate-pulse">Loading briefing&hellip;</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <Header wsConnected={wsConnected} />

      {/* Toolbar: settings + daily review */}
      <div className="max-w-[1600px] mx-auto px-4 pt-3 flex gap-2 justify-end">
        <button
          onClick={() => setReviewOpen(true)}
          className="flex items-center gap-1.5 text-xs text-gray-400 hover:text-white bg-gray-900 hover:bg-gray-800 border border-gray-800 px-3 py-1.5 rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500"
          aria-label="Open end-of-day review"
        >
          🌅 <span>Daily Review</span>
        </button>
        <button
          onClick={() => setSettingsOpen(true)}
          className="flex items-center gap-1.5 text-xs text-gray-400 hover:text-white bg-gray-900 hover:bg-gray-800 border border-gray-800 px-3 py-1.5 rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500"
          aria-label="Open settings"
        >
          ⚙️ <span>Settings</span>
        </button>
      </div>

      <main className="max-w-[1600px] mx-auto px-4 py-4">
        {/* Demo data warning banner */}
        {!bannerDismissed && (() => {
          const statuses = briefing.integration_statuses ?? {};
          const demoIntegrations = Object.keys(DEMO_INTEGRATION_LABELS).filter(k => !statuses[k as keyof typeof statuses]);
          if (demoIntegrations.length === 0) return null;
          return (
            <div className="mb-4 flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-300">
              <span className="mt-0.5 text-base leading-none">⚠</span>
              <div className="flex-1">
                <span className="font-semibold text-amber-200">Sample data shown</span>
                {' — '}
                <span className="font-medium">{demoIntegrations.map(k => DEMO_INTEGRATION_LABELS[k]).join(', ')}</span>
                {demoIntegrations.length === 1 ? ' is' : ' are'} not connected. Numbers and content for {demoIntegrations.length === 1 ? 'this integration are' : 'these integrations are'} placeholder data, not your real information.{' '}
                <button
                  onClick={() => setSettingsOpen(true)}
                  className="underline underline-offset-2 hover:text-amber-100 transition-colors"
                >
                  Connect in Settings
                </button>
              </div>
              <button
                onClick={() => setBannerDismissed(true)}
                className="text-amber-400/60 hover:text-amber-200 transition-colors text-base leading-none"
                aria-label="Dismiss"
              >
                ✕
              </button>
            </div>
          );
        })()}

        <ScoreRow briefing={briefing} />

        {briefing.summary && <SummaryBanner summary={briefing.summary} />}

        <NewsSection news={briefing.news} isLive={briefing.integration_statuses?.news} fetchError={briefing.integration_errors?.news} />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mt-4">
          {/* Column 1: Weather · Calendar · Notion */}
          <div className="space-y-4">
            <WeatherSection weather={briefing.weather} isLive={briefing.integration_statuses?.weather} fetchError={briefing.integration_errors?.weather} />
            <CalendarSection
              events={briefing.events}
              isLive={briefing.integration_statuses?.calendar}
              isAvailable={briefing.integration_available?.calendar}
              fetchError={briefing.integration_errors?.calendar}
            />
            <NotionSection pages={briefing.notion_pages} isLive={briefing.integration_statuses?.notion} fetchError={briefing.integration_errors?.notion} />
          </div>

          {/* Column 2: Email · Slack · GitHub · Jira */}
          <div className="space-y-4">
            <EmailSection
              emails={briefing.unread_emails}
              isLive={briefing.integration_statuses?.email}
              isAvailable={briefing.integration_available?.email}
              fetchError={briefing.integration_errors?.email}
            />
            <SlackSection
              messages={briefing.slack_messages}
              isLive={briefing.integration_statuses?.slack}
              isAvailable={briefing.integration_available?.slack}
              fetchError={briefing.integration_errors?.slack}
            />
            <GitHubSection notifications={briefing.github_notifications} isLive={briefing.integration_statuses?.github} fetchError={briefing.integration_errors?.github} />
            <JiraSection tickets={briefing.jira_tickets} isLive={briefing.integration_statuses?.jira} fetchError={briefing.integration_errors?.jira} />
          </div>

          {/* Column 3: Inbox Zero · Pomodoro · Time Blocking · Todos */}
          <div className="space-y-4">
            <InboxZeroProgress briefing={briefing} />
            <PomodoroTimer />
            <TimeBlocking />
            <TodoList
              todos={briefing.todos}
              onTodosChange={todos => setBriefing(prev => prev ? { ...prev, todos } : prev)}
            />
          </div>
        </div>
      </main>

      <BriefingChat open={chatOpen} onToggle={() => setChatOpen(o => !o)} />
      <SettingsPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <DailyReview open={reviewOpen} onClose={() => setReviewOpen(false)} />
    </div>
  );
}
