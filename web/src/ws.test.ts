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
import { connect, disconnect, retry, generateRoomName, send } from './ws';

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

describe('message queue', () => {
  it('discards queued messages on reconnect', () => {
    connect('test-room');
    const ws1 = latestWs();
    ws1.onopen?.(new Event('open'));

    // Simulate disconnect
    ws1.onclose?.({} as CloseEvent);

    // Queue messages while disconnected
    send({ type: 'vote', payload: { value: '5' } });
    send({ type: 'vote', payload: { value: '8' } });

    // Trigger reconnect
    vi.advanceTimersByTime(1500);
    const ws2 = latestWs();
    ws2.onopen?.(new Event('open'));

    // Only the join message should be sent, no stale queued messages
    const sentMessages = ws2.send.mock.calls.map((call: string[]) => JSON.parse(call[0]));
    expect(sentMessages).toHaveLength(1);
    expect(sentMessages[0].type).toBe('join');
  });

  it('caps queue size to prevent memory leaks', () => {
    connect('test-room');
    const ws = latestWs();
    // Do not open — messages go to queue
    ws.readyState = MockWebSocket.CLOSED;

    // Send more messages than the cap
    for (let i = 0; i < 30; i++) {
      send({ type: 'vote', payload: { value: String(i) } });
    }

    // Now open — queue should have been cleared, only join sent
    ws.readyState = MockWebSocket.OPEN;
    ws.onopen?.(new Event('open'));
    const sentMessages = ws.send.mock.calls.map((call: string[]) => JSON.parse(call[0]));
    // Only the join message (queue was cleared on open)
    expect(sentMessages).toHaveLength(1);
    expect(sentMessages[0].type).toBe('join');
  });
});

describe('generateRoomName', () => {
  it('strips trailing hex suffix and replaces dashes with spaces', () => {
    expect(generateRoomName('sprint-42-a3f1c9b2d4e6')).toBe('sprint 42');
  });

  it('handles room id with long hex suffix', () => {
    expect(generateRoomName('my-room-abcdef0123456789')).toBe('my room');
  });

  it('replaces dashes with spaces when no hex suffix', () => {
    expect(generateRoomName('my-cool-room')).toBe('my cool room');
  });

  it('returns single-word id unchanged', () => {
    expect(generateRoomName('planning')).toBe('planning');
  });

  it('does not strip short hex-like suffixes (less than 8 chars)', () => {
    expect(generateRoomName('sprint-abc')).toBe('sprint abc');
  });
});

describe('join payload includes roomName', () => {
  it('sends roomName in join message on connect', () => {
    connect('sprint-42-a3f1c9b2');
    const ws = latestWs();
    ws.onopen?.(new Event('open'));

    // First send call after open is the join message
    const joinCall = ws.send.mock.calls.find((call: string[]) => {
      const msg = JSON.parse(call[0]);
      return msg.type === 'join';
    });
    expect(joinCall).toBeDefined();
    const joinMsg = JSON.parse(joinCall![0]);
    expect(joinMsg.payload.roomName).toBe('sprint 42');
  });
});
