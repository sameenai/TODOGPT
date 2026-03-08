import { render, screen } from '@testing-library/react';
import { ScoreRow } from '@/components/ScoreRow';
import type { Briefing } from '@/lib/types';

const baseBriefing: Briefing = {
  date: '2026-03-08T00:00:00Z',
  events: [],
  news: [],
  unread_emails: [],
  slack_messages: [],
  github_notifications: [],
  jira_tickets: [],
  notion_pages: [],
  todos: [],
  email_count: 0,
  slack_unread: 0,
  generated_at: '2026-03-08T00:00:00Z',
  integration_statuses: {},
  integration_available: {},
};

describe('ScoreRow', () => {
  it('renders all 6 score cards', () => {
    render(<ScoreRow briefing={baseBriefing} />);
    expect(screen.getByText('Unread Emails')).toBeInTheDocument();
    expect(screen.getByText('Slack')).toBeInTheDocument();
    expect(screen.getByText('GitHub')).toBeInTheDocument();
    expect(screen.getByText('Events Today')).toBeInTheDocument();
    expect(screen.getByText('Action Items')).toBeInTheDocument();
    expect(screen.getByText('Focus Score')).toBeInTheDocument();
  });

  it('shows "—" for unconfigured integrations', () => {
    render(<ScoreRow briefing={baseBriefing} />);
    // email, slack, github, calendar are all not live → show "—"
    const dashes = screen.getAllByText('—');
    expect(dashes.length).toBeGreaterThanOrEqual(3);
  });

  it('shows real count when github is live with unread notifications', () => {
    const briefing: Briefing = {
      ...baseBriefing,
      integration_statuses: { github: true },
      github_notifications: [
        { id: '1', title: 'PR merged', repo: 'org/repo', type: 'PullRequest', url: '', reason: 'author', unread: true, updated_at: '' },
        { id: '2', title: 'Issue opened', repo: 'org/repo', type: 'Issue', url: '', reason: 'mention', unread: false, updated_at: '' },
      ],
    };
    render(<ScoreRow briefing={briefing} />);
    // GitHub shows count "1" (only unread)
    expect(screen.getByText('1')).toBeInTheDocument();
  });

  it('shows 100% focus score when no todos', () => {
    render(<ScoreRow briefing={baseBriefing} />);
    expect(screen.getByText('100%')).toBeInTheDocument();
  });
});
