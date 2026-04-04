import { roomState } from '../../state';
import { ParticipantCard } from '../ParticipantCard/ParticipantCard';
import './ParticipantList.css';

// Sort order: active first, then idle, then disconnected
const STATUS_ORDER: Record<string, number> = {
  active: 0,
  idle: 1,
  disconnected: 2,
};

export function ParticipantList() {
  const state = roomState.value;
  if (!state) return null;

  const sorted = [...state.participants].sort(
    (a, b) => (STATUS_ORDER[a.status] ?? 2) - (STATUS_ORDER[b.status] ?? 2)
  );

  return (
    <div class="participant-list">
      {sorted.map((p) => (
        <ParticipantCard key={p.sessionId} participant={p} />
      ))}
    </div>
  );
}
