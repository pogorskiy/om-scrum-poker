/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import { Modal } from './Modal';

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

describe('Modal', () => {
  it('calls showModal() when open=true', () => {
    render(<Modal open={true}>Content</Modal>);
    expect(HTMLDialogElement.prototype.showModal).toHaveBeenCalled();
  });

  it('calls close() when open changes to false', () => {
    const { rerender } = render(<Modal open={true}>Content</Modal>);
    vi.clearAllMocks();
    rerender(<Modal open={false}>Content</Modal>);
    expect(HTMLDialogElement.prototype.close).toHaveBeenCalled();
  });

  it('does not call showModal() when open=false', () => {
    render(<Modal open={false}>Content</Modal>);
    expect(HTMLDialogElement.prototype.showModal).not.toHaveBeenCalled();
  });

  it('renders children content', () => {
    render(<Modal open={true}><span>Hello World</span></Modal>);
    expect(screen.getByText('Hello World')).toBeInTheDocument();
  });

  it('sets aria-labelledby attribute', () => {
    render(<Modal open={true} ariaLabelledBy="my-label">Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-labelledby', 'my-label');
  });

  it('sets aria-describedby attribute', () => {
    render(<Modal open={true} ariaDescribedBy="my-desc">Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-describedby', 'my-desc');
  });

  it('applies custom class alongside modal class', () => {
    render(<Modal open={true} class="custom-class">Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveClass('modal');
    expect(dialog).toHaveClass('custom-class');
  });

  it('calls onClose when cancel event fires and dismissable=true', () => {
    const onClose = vi.fn();
    render(<Modal open={true} onClose={onClose} dismissable={true}>Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does NOT call onClose when cancel event fires and dismissable=false', () => {
    const onClose = vi.fn();
    render(<Modal open={true} onClose={onClose} dismissable={false}>Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    expect(onClose).not.toHaveBeenCalled();
  });

  it('prevents default on cancel event always', () => {
    render(<Modal open={true} dismissable={false}>Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    const event = new Event('cancel', { cancelable: true });
    dialog.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
  });

  it('calls onClose on backdrop click (click on dialog element itself)', () => {
    const onClose = vi.fn();
    render(<Modal open={true} onClose={onClose}>Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    // Simulate clicking the dialog element itself (backdrop)
    fireEvent.click(dialog);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does NOT call onClose on content click (click on child element)', () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose}>
        <div>Inner content</div>
      </Modal>
    );
    fireEvent.click(screen.getByText('Inner content'));
    expect(onClose).not.toHaveBeenCalled();
  });

  it('does NOT call onClose on backdrop click when dismissable=false', () => {
    const onClose = vi.fn();
    render(<Modal open={true} onClose={onClose} dismissable={false}>Content</Modal>);
    const dialog = screen.getByRole('dialog', { hidden: true });
    fireEvent.click(dialog);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('restores focus to previously focused element when closing', () => {
    // Create a button and focus it
    const button = document.createElement('button');
    button.textContent = 'Focus me';
    document.body.appendChild(button);
    button.focus();
    expect(document.activeElement).toBe(button);

    const { rerender } = render(<Modal open={true}>Content</Modal>);

    // Close the modal
    rerender(<Modal open={false}>Content</Modal>);

    expect(document.activeElement).toBe(button);
    document.body.removeChild(button);
  });

  it('restores focus on unmount via cleanup', () => {
    const button = document.createElement('button');
    button.textContent = 'Focus me';
    document.body.appendChild(button);
    button.focus();
    expect(document.activeElement).toBe(button);

    const { unmount } = render(<Modal open={true}>Content</Modal>);
    unmount();

    expect(document.activeElement).toBe(button);
    document.body.removeChild(button);
  });
});
