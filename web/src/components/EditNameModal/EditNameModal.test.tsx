/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import { userName } from '../../state';
import { EditNameModal } from './EditNameModal';

// Mock state and ws modules
vi.mock('../../state', async () => {
  const { signal } = await import('@preact/signals');
  const userNameSignal = signal('Alice');
  return {
    userName: userNameSignal,
    setUserName: vi.fn(),
  };
});

vi.mock('../../ws', () => ({
  send: vi.fn(),
}));

import { setUserName } from '../../state';
import { send } from '../../ws';

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
  userName.value = 'Alice';
});

describe('EditNameModal', () => {
  it('renders with current username pre-filled', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const input = screen.getByPlaceholderText('Your name...') as HTMLInputElement;
    expect(input.value).toBe('Alice');
  });

  it('input updates on typing', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const input = screen.getByPlaceholderText('Your name...') as HTMLInputElement;
    fireEvent.input(input, { target: { value: 'Bob' } });
    expect(input.value).toBe('Bob');
  });

  it('Save button disabled when input is empty/whitespace', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: '   ' } });
    const saveBtn = screen.getByText('Save');
    expect(saveBtn).toBeDisabled();
  });

  it('Save button enabled when input has text', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const saveBtn = screen.getByText('Save');
    expect(saveBtn).not.toBeDisabled();
  });

  it('on submit: calls setUserName with trimmed name', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: '  Bob  ' } });
    const form = screen.getByText('Save').closest('form')!;
    fireEvent.submit(form);
    expect(setUserName).toHaveBeenCalledWith('Bob');
  });

  it('on submit: calls send with update_name message', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: 'Bob' } });
    const form = screen.getByText('Save').closest('form')!;
    fireEvent.submit(form);
    expect(send).toHaveBeenCalledWith({ type: 'update_name', payload: { userName: 'Bob' } });
  });

  it('on submit: calls onClose', () => {
    const onClose = vi.fn();
    render(<EditNameModal onClose={onClose} />);
    const form = screen.getByText('Save').closest('form')!;
    fireEvent.submit(form);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not submit when name is empty', () => {
    const onClose = vi.fn();
    render(<EditNameModal onClose={onClose} />);
    const input = screen.getByPlaceholderText('Your name...');
    fireEvent.input(input, { target: { value: '' } });
    const form = screen.getByText('Save').closest('form')!;
    fireEvent.submit(form);
    expect(setUserName).not.toHaveBeenCalled();
    expect(send).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('Cancel button calls onClose', () => {
    const onClose = vi.fn();
    render(<EditNameModal onClose={onClose} />);
    fireEvent.click(screen.getByText('Cancel'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('has aria-labelledby pointing to label', () => {
    render(<EditNameModal onClose={vi.fn()} />);
    const dialog = screen.getByRole('dialog', { hidden: true });
    expect(dialog).toHaveAttribute('aria-labelledby', 'edit-name-label');
    expect(screen.getByText('Change your name')).toHaveAttribute('id', 'edit-name-label');
  });

  it('closes on Escape (via dialog cancel event)', () => {
    const onClose = vi.fn();
    render(<EditNameModal onClose={onClose} />);
    const dialog = screen.getByRole('dialog', { hidden: true });
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
