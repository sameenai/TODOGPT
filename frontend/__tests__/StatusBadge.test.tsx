import { render, screen } from '@testing-library/react';
import { StatusBadge } from '@/components/StatusBadge';

describe('StatusBadge', () => {
  it('renders LIVE badge when live=true', () => {
    render(<StatusBadge live={true} />);
    expect(screen.getByText('LIVE')).toBeInTheDocument();
    expect(screen.queryByText('DEMO')).not.toBeInTheDocument();
  });

  it('renders DEMO badge when live=false', () => {
    render(<StatusBadge live={false} />);
    expect(screen.getByText('DEMO')).toBeInTheDocument();
    expect(screen.queryByText('LIVE')).not.toBeInTheDocument();
  });

  it('renders nothing when live is undefined', () => {
    const { container } = render(<StatusBadge />);
    expect(container.firstChild).toBeNull();
  });
});
