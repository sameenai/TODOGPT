import { render, screen } from '@testing-library/react';
import { GitHubSection } from '@/components/sections/GitHubSection';
import type { GitHubNotification } from '@/lib/types';

const mockNotification: GitHubNotification = {
  id: '1',
  title: 'Fix: null pointer in auth flow',
  repo: 'org/backend',
  type: 'PullRequest',
  url: 'https://github.com/org/backend/pull/42',
  reason: 'review_requested',
  unread: true,
  updated_at: '2026-03-08T10:00:00Z',
};

describe('GitHubSection', () => {
  it('shows ConnectPrompt when not live', () => {
    render(<GitHubSection notifications={[]} isLive={false} />);
    // ConnectPrompt renders setup instructions
    expect(screen.getByText(/Personal access tokens/i)).toBeInTheDocument();
  });

  it('shows DEMO badge when not live', () => {
    render(<GitHubSection notifications={[]} isLive={false} />);
    expect(screen.getByText('DEMO')).toBeInTheDocument();
  });

  it('shows LIVE badge and notification list when live', () => {
    render(<GitHubSection notifications={[mockNotification]} isLive={true} />);
    expect(screen.getByText('LIVE')).toBeInTheDocument();
    expect(screen.getByText('Fix: null pointer in auth flow')).toBeInTheDocument();
    expect(screen.getByText(/org\/backend/)).toBeInTheDocument();
  });

  it('shows "No unread notifications" when live with zero unread', () => {
    const readNotif = { ...mockNotification, unread: false };
    render(<GitHubSection notifications={[readNotif]} isLive={true} />);
    expect(screen.getByText('No unread notifications')).toBeInTheDocument();
  });

  it('shows PR type badge', () => {
    render(<GitHubSection notifications={[mockNotification]} isLive={true} />);
    expect(screen.getByText('PR')).toBeInTheDocument();
  });

  it('shows IS badge for Issue type', () => {
    const issue = { ...mockNotification, type: 'Issue' };
    render(<GitHubSection notifications={[issue]} isLive={true} />);
    expect(screen.getByText('IS')).toBeInTheDocument();
  });
});
