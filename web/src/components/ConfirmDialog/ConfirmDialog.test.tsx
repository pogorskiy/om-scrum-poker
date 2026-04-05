/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import { ConfirmDialog } from './ConfirmDialog';

// Mock HTMLDialogElement methods not implemented in jsdom
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = vi.fn(function (this: HTMLDialogElement) {
    this.setAttribute('open', '');
  });
  HTMLDialogElement.prototype.close = vi.fn(function (this: HTMLDialogElement) {
    this.removeAttribute('open');
  });
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('ConfirmDialog', () => {
  const defaultProps = {
    title: 'Delete item?',
    message: 'This action cannot be undone.',
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  function renderDialog(overrides = {}) {
    return render(<ConfirmDialog {...defaultProps} {...overrides} />);
  }

  it('renders title and message', () => {
    renderDialog();
    expect(screen.getByText('Delete item?')).toBeInTheDocument();
    expect(screen.getByText('This action cannot be undone.')).toBeInTheDocument();
  });

  it('calls onConfirm when Confirm button clicked', () => {
    renderDialog();
    fireEvent.click(screen.getByText('Confirm'));
    expect(defaultProps.onConfirm).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel when Cancel button clicked', () => {
    renderDialog();
    fireEvent.click(screen.getByText('Cancel'));
    expect(defaultProps.onCancel).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel on Escape key (via cancel event on dialog)', () => {
    renderDialog();
    const dialog = screen.getByRole('dialog', { hidden: true });
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    expect(defaultProps.onCancel).toHaveBeenCalledTimes(1);
  });

  it('has aria-labelledby pointing to title id', () => {
    renderDialog();
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-labelledby', 'confirm-title');
    const title = screen.getByText('Delete item?');
    expect(title).toHaveAttribute('id', 'confirm-title');
  });

  it('has aria-describedby pointing to message id', () => {
    renderDialog();
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-describedby', 'confirm-message');
    const message = screen.getByText('This action cannot be undone.');
    expect(message).toHaveAttribute('id', 'confirm-message');
  });

  it('renders inside a dialog element (uses Modal)', () => {
    renderDialog();
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toBeInTheDocument();
    expect(dialog.tagName).toBe('DIALOG');
  });
});
