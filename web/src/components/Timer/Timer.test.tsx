/// <reference types="vitest/globals" />
import { render, screen } from '@testing-library/preact';
import { timerState } from '../../state';
import { Timer } from './Timer';

// Mock ws module
vi.mock('../../ws', () => ({
  send: vi.fn(),
}));

import { send } from '../../ws';

const defaultTimer = { duration: 30, state: 'idle' as const, startedAt: null, remaining: 30 };

beforeEach(() => {
  timerState.value = { ...defaultTimer };
  vi.clearAllMocks();
});

describe('Timer', () => {
  describe('idle state', () => {
    it('renders formatted duration', () => {
      render(<Timer />);
      expect(screen.getByText('0:30')).toBeTruthy();
    });

    it('renders start button', () => {
      render(<Timer />);
      expect(screen.getByTitle('Start timer')).toBeTruthy();
    });

    it('renders duration arrows', () => {
      render(<Timer />);
      expect(screen.getByTitle('Decrease duration')).toBeTruthy();
      expect(screen.getByTitle('Increase duration')).toBeTruthy();
    });

    it('disables decrease arrow at minimum (30s)', () => {
      timerState.value = { ...defaultTimer, duration: 30, remaining: 30 };
      render(<Timer />);
      expect(screen.getByTitle('Decrease duration')).toBeDisabled();
    });

    it('disables increase arrow at maximum (600s)', () => {
      timerState.value = { ...defaultTimer, duration: 600, remaining: 600 };
      render(<Timer />);
      expect(screen.getByTitle('Increase duration')).toBeDisabled();
    });

    it('sends timer_start on start click', () => {
      render(<Timer />);
      screen.getByTitle('Start timer').click();
      expect(send).toHaveBeenCalledWith({ type: 'timer_start', payload: {} });
    });

    it('sends timer_set_duration on increase click', () => {
      render(<Timer />);
      screen.getByTitle('Increase duration').click();
      expect(send).toHaveBeenCalledWith({ type: 'timer_set_duration', payload: { duration: 60 } });
    });

    it('sends timer_set_duration on decrease click', () => {
      timerState.value = { ...defaultTimer, duration: 60, remaining: 60 };
      render(<Timer />);
      screen.getByTitle('Decrease duration').click();
      expect(send).toHaveBeenCalledWith({ type: 'timer_set_duration', payload: { duration: 30 } });
    });

    it('formats multi-minute durations correctly', () => {
      timerState.value = { ...defaultTimer, duration: 120, remaining: 120 };
      render(<Timer />);
      expect(screen.getByText('2:00')).toBeTruthy();
    });
  });

  describe('running state', () => {
    it('renders countdown display with running style', () => {
      timerState.value = { duration: 30, state: 'running', startedAt: Date.now(), remaining: 25 };
      render(<Timer />);
      expect(screen.getByText('0:25')).toBeTruthy();
    });

    it('renders reset button instead of start', () => {
      timerState.value = { duration: 30, state: 'running', startedAt: Date.now(), remaining: 25 };
      render(<Timer />);
      expect(screen.getByTitle('Reset timer')).toBeTruthy();
      expect(screen.queryByTitle('Start timer')).toBeNull();
    });

    it('does not render duration arrows', () => {
      timerState.value = { duration: 30, state: 'running', startedAt: Date.now(), remaining: 25 };
      render(<Timer />);
      expect(screen.queryByTitle('Decrease duration')).toBeNull();
      expect(screen.queryByTitle('Increase duration')).toBeNull();
    });

    it('sends timer_reset on reset click', () => {
      timerState.value = { duration: 30, state: 'running', startedAt: Date.now(), remaining: 25 };
      render(<Timer />);
      screen.getByTitle('Reset timer').click();
      expect(send).toHaveBeenCalledWith({ type: 'timer_reset', payload: {} });
    });
  });

  describe('expired state', () => {
    it('renders bell icon (no time display)', () => {
      timerState.value = { duration: 30, state: 'expired', startedAt: null, remaining: 0 };
      render(<Timer />);
      // Bell icon is an SVG, no time text should be visible
      expect(screen.queryByText('0:00')).toBeNull();
      expect(screen.queryByText('0:30')).toBeNull();
    });

    it('renders reset button', () => {
      timerState.value = { duration: 30, state: 'expired', startedAt: null, remaining: 0 };
      render(<Timer />);
      expect(screen.getByTitle('Reset timer')).toBeTruthy();
    });

    it('does not render duration arrows', () => {
      timerState.value = { duration: 30, state: 'expired', startedAt: null, remaining: 0 };
      render(<Timer />);
      expect(screen.queryByTitle('Decrease duration')).toBeNull();
      expect(screen.queryByTitle('Increase duration')).toBeNull();
    });

    it('does not render start button', () => {
      timerState.value = { duration: 30, state: 'expired', startedAt: null, remaining: 0 };
      render(<Timer />);
      expect(screen.queryByTitle('Start timer')).toBeNull();
    });

    it('sends timer_reset on reset click', () => {
      timerState.value = { duration: 30, state: 'expired', startedAt: null, remaining: 0 };
      render(<Timer />);
      screen.getByTitle('Reset timer').click();
      expect(send).toHaveBeenCalledWith({ type: 'timer_reset', payload: {} });
    });
  });
});
