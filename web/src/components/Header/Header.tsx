import { roomState, addToast } from '../../state';
import './Header.css';

export function Header() {
  const state = roomState.value;

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
        {state && <span class="header__room-name">{state.roomName}</span>}
      </div>
      <button class="header__copy-btn" onClick={handleCopyLink}>
        Copy Link
      </button>
    </header>
  );
}
