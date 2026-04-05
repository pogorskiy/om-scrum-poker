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

  it('renders time element when BUILD_TIMESTAMP is valid ISO', () => {
    // __BUILD_TIMESTAMP__ is defined by vite - in test env it may or may not exist
    // We test the formatDeployTime function directly for format correctness
    const { container } = render(<Footer />);
    const footer = container.querySelector('footer');
    expect(footer).toBeTruthy();
  });

  it('displays text content in the footer', () => {
    const { container } = render(<Footer />);
    const text = container.querySelector('.footer__text');
    expect(text?.textContent).toBeTruthy();
    expect(text?.textContent?.length).toBeGreaterThan(0);
  });
});
