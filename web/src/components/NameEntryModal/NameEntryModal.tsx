import { useState } from 'preact/hooks';
import { setUserName, userRole } from '../../state';
import { Modal } from '../Modal/Modal';
import './NameEntryModal.css';

export function NameEntryModal() {
  const [name, setName] = useState('');
  const [role, setRole] = useState<'voter' | 'observer'>('voter');

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    userRole.value = role;
    setUserName(trimmed);
  }

  return (
    <Modal open={true} dismissable={false} ariaLabelledBy="name-entry-label">
      <form class="name-modal" onSubmit={handleSubmit}>
        <label class="name-modal__label" id="name-entry-label">What should we call you?</label>
        <input
          class="name-modal__input"
          type="text"
          maxLength={30}
          placeholder="Your name..."
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          autoFocus
        />
        <label class="name-modal__observer">
          <input
            type="checkbox"
            checked={role === 'observer'}
            onChange={(e) => setRole((e.target as HTMLInputElement).checked ? 'observer' : 'voter')}
          />
          Join as observer (no voting)
        </label>
        <button class="name-modal__btn" type="submit" disabled={!name.trim()}>
          Join
        </button>
      </form>
    </Modal>
  );
}
