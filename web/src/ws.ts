import {
  roomState,
  connectionStatus,
  sessionId,
  userName,
  selectedCard,
  addToast,
  reconnectInfo,
  type ClientMessage,
  type ServerMessage,
  type Participant,
} from './state';

let socket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let reconnectStartTime = 0;
let currentRoomId = '';
const messageQueue: ClientMessage[] = [];

const MAX_RECONNECT_DELAY = 10000;
const BASE_DELAY = 500;
const RECONNECT_TIMEOUT = 30000;

// Build WebSocket URL from current page protocol
function buildWsUrl(roomId: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws/${roomId}`;
}

// Send a message, queue if not connected
export function send(msg: ClientMessage): void {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(msg));
  } else {
    messageQueue.push(msg);
  }
}

// Flush queued messages after reconnect
function flushQueue(): void {
  while (messageQueue.length > 0) {
    const msg = messageQueue.shift()!;
    socket?.send(JSON.stringify(msg));
  }
}

// Calculate reconnect delay with jitter
function getReconnectDelay(): number {
  const base = Math.min(BASE_DELAY * Math.pow(2, reconnectAttempt), MAX_RECONNECT_DELAY);
  const jitter = base * 0.3 * Math.random();
  return base + jitter;
}

// Handle incoming server messages
function handleMessage(event: MessageEvent): void {
  let msg: ServerMessage;
  try {
    msg = JSON.parse(event.data as string) as ServerMessage;
  } catch {
    return;
  }

  switch (msg.type) {
    case 'room_state': {
      // Map vote values onto participants during reveal phase
      if (msg.payload.phase === 'reveal' && msg.payload.result) {
        const votesMap = new Map(
          msg.payload.result.votes.map((v: { sessionId: string; value: string }) => [v.sessionId, v.value])
        );
        msg.payload.participants = msg.payload.participants.map((p: Participant) => ({
          ...p,
          vote: votesMap.get(p.sessionId),
        }));
      }
      roomState.value = msg.payload;
      // Restore selected card if we have a vote
      if (msg.payload.phase === 'voting') {
        const me = msg.payload.participants.find((p: Participant) => p.sessionId === sessionId.value);
        if (me && me.vote) {
          selectedCard.value = me.vote;
        }
      }
      break;
    }

    case 'participant_joined': {
      if (!roomState.value) break;
      const exists = roomState.value.participants.some(
        (p) => p.sessionId === msg.payload.sessionId
      );
      if (!exists) {
        const newParticipant: Participant = {
          sessionId: msg.payload.sessionId,
          userName: msg.payload.userName,
          status: msg.payload.status as Participant['status'],
          hasVoted: false,
        };
        roomState.value = {
          ...roomState.value,
          participants: [...roomState.value.participants, newParticipant],
        };
        if (msg.payload.sessionId !== sessionId.value) {
          addToast(`${msg.payload.userName} joined`);
        }
      }
      break;
    }

    case 'participant_left': {
      if (!roomState.value) break;
      const leaving = roomState.value.participants.find(
        (p) => p.sessionId === msg.payload.sessionId
      );
      roomState.value = {
        ...roomState.value,
        participants: roomState.value.participants.filter(
          (p) => p.sessionId !== msg.payload.sessionId
        ),
      };
      if (leaving) {
        addToast(`${leaving.userName} left`);
      }
      break;
    }

    case 'vote_cast': {
      if (!roomState.value) break;
      roomState.value = {
        ...roomState.value,
        participants: roomState.value.participants.map((p) =>
          p.sessionId === msg.payload.sessionId ? { ...p, hasVoted: true } : p
        ),
      };
      break;
    }

    case 'vote_retracted': {
      if (!roomState.value) break;
      roomState.value = {
        ...roomState.value,
        participants: roomState.value.participants.map((p) =>
          p.sessionId === msg.payload.sessionId ? { ...p, hasVoted: false } : p
        ),
      };
      break;
    }

    case 'votes_revealed': {
      if (!roomState.value) break;
      // Update participants with their votes
      const votesMap = new Map(
        msg.payload.votes.map((v) => [v.sessionId, v.value])
      );
      roomState.value = {
        ...roomState.value,
        phase: 'reveal',
        result: msg.payload,
        participants: roomState.value.participants.map((p) => ({
          ...p,
          vote: votesMap.get(p.sessionId),
        })),
      };
      break;
    }

    case 'round_reset': {
      if (!roomState.value) break;
      selectedCard.value = '';
      roomState.value = {
        ...roomState.value,
        phase: 'voting',
        result: null,
        participants: roomState.value.participants.map((p) => ({
          ...p,
          hasVoted: false,
          vote: undefined,
        })),
      };
      addToast('New round started');
      break;
    }

    case 'room_cleared': {
      selectedCard.value = '';
      roomState.value = null;
      addToast('Room cleared');
      // Re-join so the server re-creates our participant entry.
      // Dead clients won't re-join and are effectively removed.
      send({
        type: 'join',
        payload: { sessionId: sessionId.value, userName: userName.value },
      });
      break;
    }

    case 'presence_changed': {
      if (!roomState.value) break;
      roomState.value = {
        ...roomState.value,
        participants: roomState.value.participants.map((p) =>
          p.sessionId === msg.payload.sessionId
            ? { ...p, status: msg.payload.status as Participant['status'] }
            : p
        ),
      };
      break;
    }

    case 'name_updated': {
      if (!roomState.value) break;
      roomState.value = {
        ...roomState.value,
        participants: roomState.value.participants.map((p) =>
          p.sessionId === msg.payload.sessionId
            ? { ...p, userName: msg.payload.userName }
            : p
        ),
      };
      break;
    }

    case 'error':
      addToast(msg.payload.message, 'error');
      break;
  }
}

// Slow polling interval after RECONNECT_TIMEOUT exceeded
const SLOW_POLL_INTERVAL = 10000;

// Schedule a reconnection attempt
function scheduleReconnect(): void {
  if (reconnectTimer) return;

  const elapsed = Date.now() - reconnectStartTime;
  const maxReached = elapsed > RECONNECT_TIMEOUT;

  // Use exponential backoff normally, switch to slow polling after timeout
  const delay = maxReached ? SLOW_POLL_INTERVAL : getReconnectDelay();
  reconnectAttempt++;

  reconnectInfo.value = { attempt: reconnectAttempt, maxReached };

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connect(currentRoomId);
  }, delay);
}

// Close socket without resetting reconnect state (used internally)
function closeSocket(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    socket.onclose = null;
    socket.onerror = null;
    socket.onmessage = null;
    socket.close();
    socket = null;
  }
}

// Connect to a room
export function connect(roomId: string): void {
  closeSocket();
  currentRoomId = roomId;
  connectionStatus.value = 'connecting';

  if (reconnectAttempt === 0) {
    reconnectStartTime = Date.now();
  }

  const url = buildWsUrl(roomId);
  socket = new WebSocket(url);

  socket.onopen = () => {
    connectionStatus.value = 'connected';
    reconnectAttempt = 0;
    reconnectInfo.value = { attempt: 0, maxReached: false };

    // Send join message
    send({
      type: 'join',
      payload: { sessionId: sessionId.value, userName: userName.value },
    });

    flushQueue();
  };

  socket.onmessage = handleMessage;

  socket.onclose = () => {
    connectionStatus.value = 'disconnected';
    socket = null;
    scheduleReconnect();
  };

  socket.onerror = () => {
    // onclose will fire after this
  };
}

// Clean disconnect
export function disconnect(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    // Send leave before closing so the server removes the participant immediately
    if (socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'leave', payload: {} }));
    }
    socket.onclose = null;
    socket.onerror = null;
    socket.onmessage = null;
    socket.close();
    socket = null;
  }
  connectionStatus.value = 'disconnected';
  reconnectAttempt = 0;
  reconnectInfo.value = { attempt: 0, maxReached: false };
  messageQueue.length = 0;
}

// Retry connection manually (after timeout)
export function retry(): void {
  reconnectAttempt = 0;
  reconnectStartTime = Date.now();
  reconnectInfo.value = { attempt: 0, maxReached: false };
  connect(currentRoomId);
}
