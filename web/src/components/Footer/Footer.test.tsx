import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/preact';
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

describe('Footer component', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders a footer element', () => {
    const { container } = render(<Footer />);
    const footer = container.querySelector('footer');
    expect(footer).toBeTruthy();
    expect(footer?.classList.contains('footer')).toBe(true);
  });

  it('renders with footer__text class', () => {
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text).toBeTruthy();
  });

  it('renders <time> element with correct datetime attribute when BUILD_TIMESTAMP is valid', () => {
    vi.stubGlobal('__BUILD_TIMESTAMP__', '2024-01-15T14:30:00Z');
    const { container } = render(<Footer />);
    const time = container.querySelector('time');
    expect(time).toBeTruthy();
    expect(time?.getAttribute('dateTime')).toBe('2024-01-15T14:30:00Z');
    expect(time?.classList.contains('footer__time')).toBe(true);
    expect(time?.textContent).toContain('UTC');
  });

  it('renders "Deployed:" prefix when BUILD_TIMESTAMP is valid', () => {
    vi.stubGlobal('__BUILD_TIMESTAMP__', '2024-06-20T09:15:30Z');
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toContain('Deployed:');
  });

  it('renders "Development build" without <time> when BUILD_TIMESTAMP is invalid', () => {
    vi.stubGlobal('__BUILD_TIMESTAMP__', 'not-a-date');
    const { container } = render(<Footer />);
    const time = container.querySelector('time');
    expect(time).toBeNull();
    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toBe('Development build');
  });

  it('renders "Development build" when BUILD_TIMESTAMP is empty', () => {
    vi.stubGlobal('__BUILD_TIMESTAMP__', '');
    const { container } = render(<Footer />);
    const time = container.querySelector('time');
    expect(time).toBeNull();
    expect(container.querySelector('.footer__text')?.textContent).toBe('Development build');
  });

  it('displays text content in the footer', () => {
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toBeTruthy();
    expect(text?.textContent?.length).toBeGreaterThan(0);
  });
});
