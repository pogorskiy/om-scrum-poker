import './ConfirmDialog.css';

interface Props {
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({ title, message, onConfirm, onCancel }: Props) {
  return (
    <div class="confirm-overlay" onClick={onCancel}>
      <div class="confirm-dialog" onClick={(e) => e.stopPropagation()}>
        <div class="confirm-dialog__title">{title}</div>
        <div class="confirm-dialog__message">{message}</div>
        <div class="confirm-dialog__actions">
          <button class="confirm-dialog__btn confirm-dialog__btn--cancel" onClick={onCancel}>
            Cancel
          </button>
          <button class="confirm-dialog__btn confirm-dialog__btn--confirm" onClick={onConfirm}>
            Confirm
          </button>
        </div>
      </div>
    </div>
  );
}
