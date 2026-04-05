/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import { NameEntryModal } from './NameEntryModal';

// Mock state module
vi.mock('../../state', () => ({
  setUserName: vi.fn(),
}));

import { setUserName } from '../../state';

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

describe('NameEntryModal', () => {
  it('renders the "What should we call you?" label', () => {
    render(<NameEntryModal />);
    expect(screen.getByText('What should we call you?')).toBeInTheDocument();
  });

  it('Join button disabled when input is empty', () => {
    render(<NameEntryModal />);
    const joinBtn = screen.getByText('Join');
    expect(joinBtn).toBeDisabled();
  });

  it('Join button enabled when input has text', () => {
    render(<NameEntryModal />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: 'Alice' } });
    const joinBtn = screen.getByText('Join');
    expect(joinBtn).not.toBeDisabled();
  });

  it('on submit: calls setUserName with trimmed name', () => {
    render(<NameEntryModal />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: '  Alice  ' } });
    const form = screen.getByText('Join').closest('form')!;
    fireEvent.submit(form);
    expect(setUserName).toHaveBeenCalledWith('Alice');
  });

  it('does not submit when name is empty', () => {
    render(<NameEntryModal />);
    const form = screen.getByText('Join').closest('form')!;
    fireEvent.submit(form);
    expect(setUserName).not.toHaveBeenCalled();
  });

  it('is not dismissable (dismissable=false passed to Modal)', () => {
    render(<NameEntryModal />);
    const dialog = screen.getByRole('dialog', { hidden: true });
    // When not dismissable, cancel event should NOT close the dialog
    // The onClose callback is not passed, so the dialog cancel handler
    // should prevent default but not call onClose
    const cancelEvent = new Event('cancel', { cancelable: true });
    dialog.dispatchEvent(cancelEvent);
    // Dialog should still be open (cancel was prevented)
    expect(cancelEvent.defaultPrevented).toBe(true);
    // Dialog should still have the open attribute
    expect(dialog).toHaveAttribute('open');
  });

  it('has aria-labelledby pointing to label id', () => {
    render(<NameEntryModal />);
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-labelledby', 'name-entry-label');
    expect(screen.getByText('What should we call you?')).toHaveAttribute('id', 'name-entry-label');
  });

  it('does NOT close on Escape (no onClose passed to Modal)', () => {
    render(<NameEntryModal />);
    const dialog = screen.getByRole('dialog', { hidden: true });
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    // Dialog should remain open since dismissable=false
    expect(dialog).toHaveAttribute('open');
  });
});
