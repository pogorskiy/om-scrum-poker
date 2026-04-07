import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/preact';
import { Footer, formatDeployTime } from './Footer';

describe('formatDeployTime', () => {
  it('formats a valid ISO timestamp', () => {
    const result = formatDeployTime('2024-01-15T14:30:00Z');
    expect(result).toContain('Jan');
    expect(result).toContain('15');
    expect(result).toContain('2024');
    expect(result).toContain('14:30');
    expect(result).toContain('UTC');
  });

  it('returns "Development build" for invalid date', () => {
    expect(formatDeployTime('not-a-date')).toBe('Development build');
  });

  it('returns "Development build" for empty string', () => {
    expect(formatDeployTime('')).toBe('Development build');
  });

  it('handles ISO timestamps with milliseconds', () => {
    const result = formatDeployTime('2024-06-20T09:15:30.123Z');
    expect(result).toContain('Jun');
    expect(result).toContain('20');
    expect(result).toContain('2024');
    expect(result).toContain('UTC');
  });

  it('handles midnight UTC correctly', () => {
    const result = formatDeployTime('2024-12-31T00:00:00Z');
    expect(result).toContain('Dec');
    expect(result).toContain('31');
    expect(result).toContain('00:00');
    expect(result).toContain('UTC');
  });
});

function mockFetch(buildTime: string) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    json: () => Promise.resolve({ build_time: buildTime }),
  } as Response);
}

function mockFetchError() {
  return vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network'));
}

describe('Footer component', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders a footer element', () => {
    mockFetchError();
    const { container } = render(<Footer />);
    const footer = container.querySelector('footer');
    expect(footer).toBeTruthy();
    expect(footer?.classList.contains('footer')).toBe(true);
  });

  it('renders with footer__text class', () => {
    mockFetchError();
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text).toBeTruthy();
  });

  it('fetches build_time from /health and renders <time> element', async () => {
    mockFetch('2024-01-15T14:30:00Z');
    const { container } = render(<Footer />);

    await waitFor(() => {
      const time = container.querySelector('time');
      expect(time).toBeTruthy();
      expect(time?.getAttribute('dateTime')).toBe('2024-01-15T14:30:00Z');
      expect(time?.classList.contains('footer__time')).toBe(true);
      expect(time?.textContent).toContain('UTC');
    });
  });

  it('renders "Deployed:" prefix when health returns valid build_time', async () => {
    mockFetch('2024-06-20T09:15:30Z');
    const { container } = render(<Footer />);

    await waitFor(() => {
      const text = container.querySelector('.footer__text');
      expect(text?.textContent).toContain('Deployed:');
    });
  });

  it('renders "Development build" when health returns "dev"', async () => {
    mockFetch('dev');
    const { container } = render(<Footer />);

    await waitFor(() => {
      const text = container.querySelector('.footer__text');
      expect(text?.textContent).toBe('Development build');
    });

    const time = container.querySelector('time');
    expect(time).toBeNull();
  });

  it('renders "Development build" when /health fetch fails', async () => {
    mockFetchError();
    const { container } = render(<Footer />);

    // Wait a tick for the failed fetch to resolve.
    await new Promise((r) => setTimeout(r, 10));

    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toBe('Development build');
    expect(container.querySelector('time')).toBeNull();
  });

  it('renders "Development build" initially before fetch completes', () => {
    mockFetch('2024-01-15T14:30:00Z');
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toBe('Development build');
  });
});
