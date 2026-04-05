import { useRef, useEffect } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import './Modal.css';

interface ModalProps {
  open: boolean;
  onClose?: () => void;
  dismissable?: boolean;
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  children: ComponentChildren;
  class?: string;
}

export function Modal({
  open,
  onClose,
  dismissable = true,
  ariaLabelledBy,
  ariaDescribedBy,
  children,
  class: className,
}: ModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const previousFocusRef = useRef<Element | null>(null);

  function restoreFocus() {
    if (previousFocusRef.current instanceof HTMLElement) {
      previousFocusRef.current.focus();
      previousFocusRef.current = null;
    }
  }

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open) {
      previousFocusRef.current = document.activeElement;
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      if (dialog.open) {
        dialog.close();
      }
      restoreFocus();
    }

    // Restore focus when component unmounts while open
    return () => {
      restoreFocus();
    };
  }, [open]);

  function handleCancel(e: Event) {
    // Always prevent native close — we manage open/close via the open prop
    e.preventDefault();
    if (dismissable) {
      onClose?.();
    }
  }

  function handleClick(e: MouseEvent) {
    // Detect backdrop click: the click target is the dialog element itself
    if (dismissable && e.target === dialogRef.current) {
      onClose?.();
    }
  }

  const classes = ['modal', className].filter(Boolean).join(' ');

  return (
    <dialog
      ref={dialogRef}
      class={classes}
      aria-labelledby={ariaLabelledBy}
      aria-describedby={ariaDescribedBy}
      onCancel={handleCancel}
      onClick={handleClick}
    >
      {children}
    </dialog>
  );
}
