import { useState, useEffect } from 'preact/hooks';
import { timerState } from '../../state';
import { send } from '../../ws';
import './Timer.css';

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export function Timer() {
  const timer = timerState.value;
  const [displayRemaining, setDisplayRemaining] = useState(timer.remaining);

  // Sync displayRemaining from signal
  useEffect(() => {
    setDisplayRemaining(timerState.value.remaining);
  }, [timer.remaining]);

  // Local countdown when running — auto-expire on client side when reaching zero
  useEffect(() => {
    if (timer.state !== 'running') return;
    const id = setInterval(() => {
      setDisplayRemaining((prev) => {
        const next = prev - 1;
        if (next <= 0) {
          // Auto-expire on client: update signal so UI shows expired state immediately
          timerState.value = { ...timerState.value, state: 'expired', remaining: 0 };
          clearInterval(id);
          return 0;
        }
        return next;
      });
    }, 1000);
    return () => clearInterval(id);
  }, [timer.state]);

  function handleDecrease() {
    send({ type: 'timer_set_duration', payload: { duration: timer.duration - 30 } });
  }

  function handleIncrease() {
    send({ type: 'timer_set_duration', payload: { duration: timer.duration + 30 } });
  }

  function handleStart() {
    send({ type: 'timer_start', payload: {} });
  }

  function handleReset() {
    send({ type: 'timer_reset', payload: {} });
  }

  if (timer.state === 'expired') {
    return (
      <div class="timer">
        <span class="timer__display timer__display--expired">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/></svg>
        </span>
        <button class="timer__action-btn" onClick={handleReset} title="Reset timer">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><rect x="4" y="4" width="16" height="16" rx="2"/></svg>
        </button>
      </div>
    );
  }

  if (timer.state === 'running') {
    return (
      <div class="timer">
        <span class="timer__display timer__display--running">
          {formatTime(displayRemaining)}
        </span>
        <button class="timer__action-btn" onClick={handleReset} title="Reset timer">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><rect x="4" y="4" width="16" height="16" rx="2"/></svg>
        </button>
      </div>
    );
  }

  // Idle state
  return (
    <div class="timer">
      <button
        class="timer__arrow-btn"
        onClick={handleDecrease}
        disabled={timer.duration <= 30}
        title="Decrease duration"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
      </button>
      <span class="timer__display">{formatTime(timer.duration)}</span>
      <button
        class="timer__arrow-btn"
        onClick={handleIncrease}
        disabled={timer.duration >= 600}
        title="Increase duration"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="m18 15-6-6-6 6"/></svg>
      </button>
      <button class="timer__action-btn" onClick={handleStart} title="Start timer">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><polygon points="6,3 20,12 6,21"/></svg>
      </button>
    </div>
  );
}
