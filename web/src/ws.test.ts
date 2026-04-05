/// <reference types="vitest/globals" />
import { connectionStatus, reconnectInfo, toasts } from './state';

// Track all created MockWebSocket instances
const wsInstances: MockWebSocket[] = [];

class MockWebSocket {
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  static readonly CONNECTING = 0;
  readyState = MockWebSocket.OPEN;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  send = vi.fn();
  close = vi.fn();

  constructor(_url: string) {
    wsInstances.push(this);
  }
}

vi.stubGlobal('WebSocket', MockWebSocket);

// Import after mocking WebSocket
import { connect, disconnect, retry } from './ws';

/** Get the most recently created WebSocket instance */
function latestWs(): MockWebSocket {
  return wsInstances[wsInstances.length - 1];
}

function resetState() {
  connectionStatus.value = 'disconnected';
  reconnectInfo.value = { attempt: 0, maxReached: false };
  toasts.value = [];
}

beforeEach(() => {
  wsInstances.length = 0;
  resetState();
  vi.useFakeTimers();
  disconnect();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('ws.ts reconnect logic', () => {
  it('updates reconnectInfo on each reconnect attempt', () => {
    connect('test-room');
    const ws1 = latestWs();
    ws1.onopen?.(new Event('open'));
    expect(reconnectInfo.value.attempt).toBe(0);

    // Simulate disconnect — scheduleReconnect is called synchronously,
    // which increments attempt and sets the timer
    ws1.onclose?.({} as CloseEvent);
    expect(connectionStatus.value).toBe('disconnected');
    expect(reconnectInfo.value.attempt).toBe(1);
    expect(reconnectInfo.value.maxReached).toBe(false);

    // Advance timer so reconnect fires — creates a new WS
    vi.advanceTimersByTime(1500);
    const ws2 = latestWs();
    expect(ws2).not.toBe(ws1);
    // Simulate second failure
    ws2.onclose?.({} as CloseEvent);
    expect(reconnectInfo.value.attempt).toBe(2);
  });

  it('sets maxReached to true after RECONNECT_TIMEOUT', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    // Keep failing reconnects until past 30s
    for (let i = 0; i < 20; i++) {
      vi.advanceTimersByTime(5000);
      const ws = latestWs();
      if (ws.onclose) {
        ws.onclose({} as CloseEvent);
      }
    }

    expect(reconnectInfo.value.maxReached).toBe(true);
  });

  it('resets reconnectInfo on retry()', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    // scheduleReconnect already bumped attempt to 1
    expect(reconnectInfo.value.attempt).toBe(1);

    retry();
    expect(reconnectInfo.value).toEqual({ attempt: 0, maxReached: false });
  });

  it('resets reconnectInfo on successful connection', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    expect(reconnectInfo.value.attempt).toBe(1);

    // Advance timer to trigger reconnect
    vi.advanceTimersByTime(1500);
    // New socket opens successfully
    latestWs().onopen?.(new Event('open'));
    expect(reconnectInfo.value).toEqual({ attempt: 0, maxReached: false });
    expect(connectionStatus.value).toBe('connected');
  });

  it('resets reconnectInfo on disconnect()', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    expect(reconnectInfo.value.attempt).toBe(1);

    disconnect();
    expect(reconnectInfo.value).toEqual({ attempt: 0, maxReached: false });
  });

  it('sends leave message before closing on disconnect()', () => {
    connect('test-room');
    const ws = latestWs();
    ws.onopen?.(new Event('open'));

    expect(ws.readyState).toBe(MockWebSocket.OPEN);

    disconnect();

    // Should have sent a leave message before closing
    expect(ws.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'leave', payload: {} }),
    );
    // send should be called before close
    const sendOrder = ws.send.mock.invocationCallOrder[0];
    const closeOrder = ws.close.mock.invocationCallOrder[0];
    expect(sendOrder).toBeLessThan(closeOrder);
  });

  it('does not send leave message if socket is not open', () => {
    connect('test-room');
    const ws = latestWs();
    // Socket never opened (readyState stays OPEN by default in mock,
    // so set it to CLOSED to simulate)
    ws.readyState = MockWebSocket.CLOSED;

    disconnect();

    expect(ws.send).not.toHaveBeenCalled();
  });

  it('continues reconnecting after RECONNECT_TIMEOUT (does NOT stop)', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    // Fast-forward past RECONNECT_TIMEOUT with repeated failures
    for (let i = 0; i < 20; i++) {
      vi.advanceTimersByTime(5000);
      const ws = latestWs();
      if (ws.onclose) {
        ws.onclose({} as CloseEvent);
      }
    }

    expect(reconnectInfo.value.maxReached).toBe(true);
    const attemptAfterTimeout = reconnectInfo.value.attempt;

    // Advance more — should still be reconnecting with slow poll (10s interval)
    vi.advanceTimersByTime(11000);
    latestWs().onclose?.({} as CloseEvent);

    expect(reconnectInfo.value.attempt).toBeGreaterThan(attemptAfterTimeout);
  });

  it('does NOT show "Connection lost" toast (removed from scheduleReconnect)', () => {
    connect('test-room');
    latestWs().onopen?.(new Event('open'));
    latestWs().onclose?.({} as CloseEvent);

    // Fast-forward past RECONNECT_TIMEOUT
    for (let i = 0; i < 20; i++) {
      vi.advanceTimersByTime(5000);
      const ws = latestWs();
      if (ws.onclose) {
        ws.onclose({} as CloseEvent);
      }
    }

    const lostToast = toasts.value.find((t) => t.message.includes('Connection lost'));
    expect(lostToast).toBeUndefined();
  });
});
