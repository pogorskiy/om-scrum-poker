/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import { connectionStatus, reconnectInfo } from '../../state';
import { ConnectionBanner } from './ConnectionBanner';

// Mock the ws module so we can spy on retry()
vi.mock('../../ws', () => ({
  retry: vi.fn(),
}));

import { retry } from '../../ws';

function resetSignals() {
  connectionStatus.value = 'disconnected';
  reconnectInfo.value = { attempt: 0, maxReached: false };
}

beforeEach(() => {
  resetSignals();
  vi.clearAllMocks();
});

describe('ConnectionBanner', () => {
  it('renders nothing when connected', () => {
    connectionStatus.value = 'connected';
    reconnectInfo.value = { attempt: 0, maxReached: false };

    const { container } = render(<ConnectionBanner />);
    expect(container.innerHTML).toBe('');
  });

  it('shows "Reconnecting..." banner with attempt count when connecting and attempt > 0', () => {
    connectionStatus.value = 'connecting';
    reconnectInfo.value = { attempt: 3, maxReached: false };

    render(<ConnectionBanner />);
    expect(screen.getByText(/Reconnecting/)).toBeInTheDocument();
    expect(screen.getByText(/attempt 3/)).toBeInTheDocument();
  });

  it('shows "Reconnecting..." banner when disconnected and maxReached is false', () => {
    connectionStatus.value = 'disconnected';
    reconnectInfo.value = { attempt: 2, maxReached: false };

    render(<ConnectionBanner />);
    expect(screen.getByText(/Reconnecting/)).toBeInTheDocument();
    expect(screen.getByText(/attempt 2/)).toBeInTheDocument();
  });

  it('shows "Connection lost" banner with Retry button when maxReached is true', () => {
    connectionStatus.value = 'disconnected';
    reconnectInfo.value = { attempt: 8, maxReached: true };

    render(<ConnectionBanner />);
    expect(screen.getByText('Connection lost')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('calls retry() when Retry button is clicked', () => {
    connectionStatus.value = 'disconnected';
    reconnectInfo.value = { attempt: 8, maxReached: true };

    render(<ConnectionBanner />);
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(retry).toHaveBeenCalledTimes(1);
  });

  it('has correct CSS class for reconnecting state', () => {
    connectionStatus.value = 'disconnected';
    reconnectInfo.value = { attempt: 1, maxReached: false };

    render(<ConnectionBanner />);
    const banner = screen.getByRole('status');
    expect(banner).toHaveClass('connection-banner');
    expect(banner).toHaveClass('connection-banner--reconnecting');
  });

  it('has correct CSS class for lost state', () => {
    connectionStatus.value = 'disconnected';
    reconnectInfo.value = { attempt: 8, maxReached: true };

    render(<ConnectionBanner />);
    const banner = screen.getByRole('alert');
    expect(banner).toHaveClass('connection-banner');
    expect(banner).toHaveClass('connection-banner--lost');
  });

  it('shows correct attempt number in reconnecting message', () => {
    connectionStatus.value = 'connecting';
    reconnectInfo.value = { attempt: 5, maxReached: false };

    render(<ConnectionBanner />);
    expect(screen.getByText(/attempt 5/)).toBeInTheDocument();
  });
});
