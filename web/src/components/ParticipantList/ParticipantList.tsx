import { roomState } from '../../state';
import { ParticipantCard } from '../ParticipantCard/ParticipantCard';
import './ParticipantList.css';

export function ParticipantList() {
  const state = roomState.value;
  if (!state) return null;

  return (
    <div class="participant-list">
      {state.participants.map((p, i) => (
        <ParticipantCard key={p.sessionId} participant={p} index={i} />
      ))}
    </div>
  );
}
