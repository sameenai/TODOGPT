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

const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? 'ws://localhost:8080/ws';

interface Props {
  initialBriefing: Briefing | null;
}

export function Dashboard({ initialBriefing }: Props) {
  const [briefing, setBriefing] = useState<Briefing | null>(initialBriefing);
  const [wsConnected, setWsConnected] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);

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

      <main className="max-w-[1600px] mx-auto px-4 py-4">
        <ScoreRow briefing={briefing} />

        {briefing.summary && <SummaryBanner summary={briefing.summary} />}

        <NewsSection news={briefing.news} isLive={briefing.integration_statuses?.news} />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mt-4">
          {/* Column 1: Weather · Calendar · Notion */}
          <div className="space-y-4">
            <WeatherSection weather={briefing.weather} isLive={briefing.integration_statuses?.weather} />
            <CalendarSection
              events={briefing.events}
              isLive={briefing.integration_statuses?.calendar}
              isAvailable={briefing.integration_available?.calendar}
            />
            <NotionSection pages={briefing.notion_pages} isLive={briefing.integration_statuses?.notion} />
          </div>

          {/* Column 2: Email · Slack · GitHub · Jira */}
          <div className="space-y-4">
            <EmailSection
              emails={briefing.unread_emails}
              isLive={briefing.integration_statuses?.email}
              isAvailable={briefing.integration_available?.email}
            />
            <SlackSection
              messages={briefing.slack_messages}
              isLive={briefing.integration_statuses?.slack}
              isAvailable={briefing.integration_available?.slack}
            />
            <GitHubSection notifications={briefing.github_notifications} isLive={briefing.integration_statuses?.github} />
            <JiraSection tickets={briefing.jira_tickets} isLive={briefing.integration_statuses?.jira} />
          </div>

          {/* Column 3: Inbox Zero · Pomodoro · Todos */}
          <div className="space-y-4">
            <InboxZeroProgress briefing={briefing} />
            <PomodoroTimer />
            <TodoList
              todos={briefing.todos}
              onTodosChange={todos => setBriefing(prev => prev ? { ...prev, todos } : prev)}
            />
          </div>
        </div>
      </main>

      <BriefingChat open={chatOpen} onToggle={() => setChatOpen(o => !o)} />
    </div>
  );
}
