import { useState } from 'preact/hooks';
import { roomState, userName, userRole, myParticipant, addToast } from '../../state';
import { send } from '../../ws';
import { EditNameModal } from '../EditNameModal/EditNameModal';
import { ConfirmDialog } from '../ConfirmDialog/ConfirmDialog';
import './Header.css';

export function Header() {
  const state = roomState.value;
  const [showEditName, setShowEditName] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);

  function handleCopyLink() {
    const url = window.location.href;
    navigator.clipboard.writeText(url).then(
      () => addToast('Link copied!'),
      () => addToast('Failed to copy link', 'error')
    );
  }

  function handleToggleRole() {
    const newRole = myParticipant.value?.role === 'observer' ? 'voter' : 'observer';
    userRole.value = newRole;
    send({ type: 'update_role', payload: { role: newRole } });
  }

  const currentRole = myParticipant.value?.role ?? 'voter';

  return (
    <header class="header">
      <div class="header__left">
        <span class="header__logo">om</span>
        {state && (
          <span class="header__room-name">
            {state.roomName}
            {state.createdBy && (
              <span class="header__created-by"> by {state.createdBy}</span>
            )}
          </span>
        )}
      </div>
      <div class="header__right">
        <button
          class={`header__role-btn${currentRole === 'observer' ? ' header__role-btn--observer' : ''}`}
          onClick={handleToggleRole}
          title={currentRole === 'observer' ? 'Switch to voter' : 'Switch to observer'}
        >
          {currentRole === 'observer' ? 'Observing' : 'Voting'}
        </button>
        <span class="header__user-name">{userName.value}</span>
        <button
          class="header__edit-btn"
          onClick={() => setShowEditName(true)}
          title="Change name"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M17 3a2.83 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
            <path d="m15 5 4 4" />
          </svg>
        </button>
        <button class="header__copy-btn" onClick={handleCopyLink}>
          Copy Link
        </button>
        <button
          class="header__clear-btn"
          onClick={() => setShowClearConfirm(true)}
          title="Clear room"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 6h18"/>
            <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
          </svg>
        </button>
      </div>
      {showEditName && <EditNameModal onClose={() => setShowEditName(false)} />}
      {showClearConfirm && (
        <ConfirmDialog
          title="Clear Room?"
          message="This will remove all participants."
          onConfirm={() => { send({ type: 'clear_room', payload: {} }); setShowClearConfirm(false); }}
          onCancel={() => setShowClearConfirm(false)}
        />
      )}
    </header>
  );
}
