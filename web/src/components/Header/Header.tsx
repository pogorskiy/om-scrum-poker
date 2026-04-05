import { useState } from 'preact/hooks';
import { roomState, userName, addToast } from '../../state';
import { EditNameModal } from '../EditNameModal/EditNameModal';
import './Header.css';

export function Header() {
  const state = roomState.value;
  const [showEditName, setShowEditName] = useState(false);

  function handleCopyLink() {
    const url = window.location.href;
    navigator.clipboard.writeText(url).then(
      () => addToast('Link copied!'),
      () => addToast('Failed to copy link', 'error')
    );
  }

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
      </div>
      {showEditName && <EditNameModal onClose={() => setShowEditName(false)} />}
    </header>
  );
}
