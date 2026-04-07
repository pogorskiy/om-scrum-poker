import { toasts } from '../../state';
import './Toast.css';

export function Toast() {
  const items = toasts.value;
  if (items.length === 0) return null;

  return (
    <div class="toast-container">
      {items.map((t) => (
        <div
          key={t.id}
          class={`toast${t.type === 'error' ? ' toast--error' : ''}`}
          role={t.type === 'error' ? 'alert' : 'status'}
          aria-live={t.type === 'error' ? 'assertive' : 'polite'}
        >
          {t.message}
        </div>
      ))}
    </div>
  );
}
