import { useState, useEffect } from 'preact/hooks';
import { userName, setUserName } from '../../state';
import { send } from '../../ws';
import './EditNameModal.css';

interface Props {
  onClose: () => void;
}

export function EditNameModal({ onClose }: Props) {
  const [name, setName] = useState(userName.value);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    setUserName(trimmed);
    send({ type: 'update_name', payload: { userName: trimmed } });
    onClose();
  }

  return (
    <div class="edit-name-modal__overlay" onClick={onClose}>
      <form
        class="edit-name-modal"
        onSubmit={handleSubmit}
        onClick={(e) => e.stopPropagation()}
      >
        <label class="edit-name-modal__label">Change your name</label>
        <input
          class="edit-name-modal__input"
          type="text"
          maxLength={30}
          placeholder="Your name..."
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          autoFocus
        />
        <div class="edit-name-modal__actions">
          <button
            class="edit-name-modal__btn edit-name-modal__btn--cancel"
            type="button"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            class="edit-name-modal__btn edit-name-modal__btn--save"
            type="submit"
            disabled={!name.trim()}
          >
            Save
          </button>
        </div>
      </form>
    </div>
  );
}
