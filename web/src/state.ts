import { signal, computed } from '@preact/signals';

// --- Types ---

export interface Participant {
  sessionId: string;
  userName: string;
  status: 'active' | 'idle' | 'disconnected';
  hasVoted: boolean;
  vote?: string;
}

export interface VoteResult {
  votes: Array<{ sessionId: string; name: string; value: string }>;
  average: number | null;
  median: number | null;
  uncertainCount: number;
  totalVoters: number;
  hasConsensus: boolean;
  spread: [number, number] | null;
}

export interface RoomState {
  roomId: string;
  roomName: string;
  createdBy: string;
  phase: 'voting' | 'reveal';
  participants: Participant[];
  result: VoteResult | null;
}

export interface Toast {
  id: number;
  message: string;
  type: 'info' | 'error';
}

// Server message types (discriminated union)
export type ServerMessage =
  | { type: 'room_state'; payload: RoomState }
  | { type: 'participant_joined'; payload: { sessionId: string; userName: string; status: string } }
  | { type: 'participant_left'; payload: { sessionId: string } }
  | { type: 'vote_cast'; payload: { sessionId: string } }
  | { type: 'vote_retracted'; payload: { sessionId: string } }
  | { type: 'votes_revealed'; payload: VoteResult }
  | { type: 'round_reset'; payload: Record<string, never> }
  | { type: 'room_cleared'; payload: Record<string, never> }
  | { type: 'presence_changed'; payload: { sessionId: string; status: string } }
  | { type: 'name_updated'; payload: { sessionId: string; userName: string } }
  | { type: 'error'; payload: { code: string; message: string } };

// Client message types
export type ClientMessage =
  | { type: 'join'; payload: { sessionId: string; userName: string; roomName?: string } }
  | { type: 'vote'; payload: { value: string } }
  | { type: 'reveal'; payload: Record<string, never> }
  | { type: 'new_round'; payload: Record<string, never> }
  | { type: 'clear_room'; payload: Record<string, never> }
  | { type: 'update_name'; payload: { userName: string } }
  | { type: 'presence'; payload: { status: string } }
  | { type: 'leave'; payload: Record<string, never> };

// --- localStorage helpers ---

function safeGet(key: string): string {
  try {
    return localStorage.getItem(key) ?? '';
  } catch {
    return '';
  }
}

function safeSet(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Private browsing mode — silently ignore
  }
}

function generateHex(length: number): string {
  const arr = new Uint8Array(length / 2);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('');
}

// --- Initialize session ID ---

function getOrCreateSessionId(): string {
  const existing = safeGet('om-poker-session');
  if (existing) return existing;
  const id = generateHex(32);
  safeSet('om-poker-session', id);
  return id;
}

// --- Reconnect info ---

export interface ReconnectInfo {
  attempt: number;
  maxReached: boolean;
}

// --- Signals ---

export const roomState = signal<RoomState | null>(null);
export const reconnectInfo = signal<ReconnectInfo>({ attempt: 0, maxReached: false });
export const connectionStatus = signal<'connecting' | 'connected' | 'disconnected'>('disconnected');
export const userName = signal<string>(safeGet('om-poker-name'));
export const sessionId = signal<string>(getOrCreateSessionId());
export const selectedCard = signal<string>('');
export const toasts = signal<Toast[]>([]);
export const currentPath = signal<string>(window.location.pathname);

// --- Computed ---

export const isRevealed = computed(() => roomState.value?.phase === 'reveal');

export const myParticipant = computed(() =>
  roomState.value?.participants.find((p) => p.sessionId === sessionId.value) ?? null
);

export const voteCount = computed(() => {
  if (!roomState.value) return { voted: 0, total: 0 };
  const active = roomState.value.participants.filter((p) => p.status !== 'disconnected');
  const voted = active.filter((p) => p.hasVoted).length;
  return { voted, total: active.length };
});

// --- Actions ---

let toastCounter = 0;

export function setUserName(name: string): void {
  userName.value = name;
  safeSet('om-poker-name', name);
}

export function addToast(message: string, type: 'info' | 'error' = 'info'): void {
  const id = ++toastCounter;
  const current = toasts.value;
  // Keep max 3, newest first
  toasts.value = [{ id, message, type }, ...current].slice(0, 3);

  setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }, 2300);
}

export function navigate(path: string): void {
  window.history.pushState(null, '', path);
  currentPath.value = path;
}
