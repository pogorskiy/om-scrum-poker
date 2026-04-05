import { useEffect } from 'preact/hooks';
import { useState } from 'preact/hooks';
import {
  roomState,
  connectionStatus,
  userName,
  isRevealed,
  voteCount,
  selectedCard,
  myParticipant,
} from '../../state';
import { connect, disconnect, send } from '../../ws';
import { parseRoomId } from '../../utils/room-url';
import { usePresence } from '../../hooks/usePresence';
import { Header } from '../Header/Header';
import { ParticipantList } from '../ParticipantList/ParticipantList';
import { CardDeck } from '../CardDeck/CardDeck';
import { NameEntryModal } from '../NameEntryModal/NameEntryModal';
import { ConnectionBanner } from '../ConnectionBanner/ConnectionBanner';
import { ConfirmDialog } from '../ConfirmDialog/ConfirmDialog';
import './RoomPage.css';

interface Props {
  path: string;
}

export function RoomPage({ path }: Props) {
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const name = userName.value;
  const roomId = parseRoomId(path);

  // Track presence (idle/active)
  usePresence();

  // Connect WebSocket when we have a name and room ID
  useEffect(() => {
    if (!roomId || !name) return;
    connect(roomId);
    return () => {
      disconnect();
      roomState.value = null;
      selectedCard.value = '';
    };
  }, [roomId, name]);

  // Show name modal if no name
  if (!name) {
    return <NameEntryModal />;
  }

  if (!roomId) {
    return <div class="room__connecting">Invalid room URL</div>;
  }

  const status = connectionStatus.value;
  const state = roomState.value;
  const revealed = isRevealed.value;
  const counts = voteCount.value;
  const observerCount = state
    ? state.participants.filter((p) => p.status !== 'disconnected' && p.role === 'observer').length
    : 0;

  if (status === 'connecting' || (!state && status === 'connected')) {
    return <div class="room__connecting">Connecting...</div>;
  }

  function handleReveal() {
    send({ type: 'reveal', payload: {} });
  }

  function handleNewRound() {
    send({ type: 'new_round', payload: {} });
  }

  function handleClearRoom() {
    setShowClearConfirm(true);
  }

  function confirmClear() {
    send({ type: 'clear_room', payload: {} });
    setShowClearConfirm(false);
  }

  return (
    <div class="room">
      <Header />
      <ConnectionBanner />

      <ParticipantList />

      {/* Vote results */}
      {revealed && state?.result && (
        <div
          class={`room__result${state.result.hasConsensus ? ' room__result--consensus' : ''}`}
        >
          <div class="room__result-stats">
            <div class="room__result-stat">
              <div class="room__result-value">
                {state.result.average !== null ? state.result.average.toFixed(1) : '—'}
              </div>
              <div class="room__result-label">Average</div>
            </div>
            <div class="room__result-stat">
              <div class="room__result-value">
                {state.result.median !== null ? state.result.median : '—'}
              </div>
              <div class="room__result-label">Median</div>
            </div>
            {state.result.hasConsensus && (
              <div class="room__result-stat">
                <div class="room__result-value">Consensus</div>
                <div class="room__result-label">Agreement</div>
              </div>
            )}
            {state.result.uncertainCount > 0 && (
              <div class="room__result-stat">
                <div class="room__result-value">{state.result.uncertainCount}</div>
                <div class="room__result-label">Uncertain</div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Action buttons */}
      <div class="room__actions">
        {!revealed && (
          <button
            class="room__action-btn room__action-btn--primary"
            onClick={handleReveal}
            disabled={counts.voted === 0}
          >
            Show Votes ({counts.voted} of {counts.total} voted{observerCount > 0 ? `, ${observerCount} observing` : ''})
          </button>
        )}
        {revealed && (
          <button
            class="room__action-btn room__action-btn--secondary"
            onClick={handleNewRound}
          >
            New Round
          </button>
        )}
        <button
          class="room__action-btn room__action-btn--danger"
          onClick={handleClearRoom}
        >
          Clear Room
        </button>
      </div>

      <CardDeck />

      {showClearConfirm && (
        <ConfirmDialog
          title="Clear Room?"
          message="This will remove all participants."
          onConfirm={confirmClear}
          onCancel={() => setShowClearConfirm(false)}
        />
      )}
    </div>
  );
}
