import { useState } from 'preact/hooks';
import { setUserName } from '../../state';
import './NameEntryModal.css';

export function NameEntryModal() {
  const [name, setName] = useState('');

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    setUserName(trimmed);
  }

  return (
    <div class="name-modal-overlay">
      <form class="name-modal" onSubmit={handleSubmit}>
        <label class="name-modal__label">What should we call you?</label>
        <input
          class="name-modal__input"
          type="text"
          maxLength={30}
          placeholder="Your name..."
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          autoFocus
        />
        <button
          class="name-modal__btn"
          type="submit"
          disabled={!name.trim()}
        >
          Join
        </button>
      </form>
    </div>
  );
}
