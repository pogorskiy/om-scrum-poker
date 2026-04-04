import { useEffect, useRef } from 'preact/hooks';
import { send } from '../ws';

const IDLE_TIMEOUT = 2 * 60 * 1000; // 2 minutes

export function usePresence(): void {
  const statusRef = useRef<'active' | 'idle'>('active');
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    function sendStatus(status: 'active' | 'idle'): void {
      if (statusRef.current === status) return;
      statusRef.current = status;
      send({ type: 'presence', payload: { status } });
    }

    function resetIdleTimer(): void {
      sendStatus('active');
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => sendStatus('idle'), IDLE_TIMEOUT);
    }

    function handleVisibility(): void {
      if (document.hidden) {
        sendStatus('idle');
        if (timerRef.current) clearTimeout(timerRef.current);
      } else {
        resetIdleTimer();
      }
    }

    // Activity events
    const events: Array<keyof WindowEventMap> = [
      'mousemove',
      'keydown',
      'touchstart',
      'scroll',
    ];

    events.forEach((evt) => window.addEventListener(evt, resetIdleTimer, { passive: true }));
    document.addEventListener('visibilitychange', handleVisibility);

    // Start idle timer
    resetIdleTimer();

    return () => {
      events.forEach((evt) => window.removeEventListener(evt, resetIdleTimer));
      document.removeEventListener('visibilitychange', handleVisibility);
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);
}
