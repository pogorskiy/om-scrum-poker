import { Modal } from '../Modal/Modal';
import './ConfirmDialog.css';

interface Props {
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({ title, message, onConfirm, onCancel }: Props) {
  return (
    <Modal open={true} onClose={onCancel} ariaLabelledBy="confirm-title" ariaDescribedBy="confirm-message">
      <div class="confirm-dialog">
        <div class="confirm-dialog__title" id="confirm-title">{title}</div>
        <div class="confirm-dialog__message" id="confirm-message">{message}</div>
        <div class="confirm-dialog__actions">
          <button class="confirm-dialog__btn confirm-dialog__btn--cancel" onClick={onCancel}>
            Cancel
          </button>
          <button class="confirm-dialog__btn confirm-dialog__btn--confirm" onClick={onConfirm}>
            Confirm
          </button>
        </div>
      </div>
    </Modal>
  );
}
