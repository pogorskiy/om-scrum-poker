/// <reference types="vitest/globals" />
import { render, screen } from '@testing-library/preact';
import { Card } from './Card';

describe('Card', () => {
  it('renders the card value', () => {
    render(<Card value="5" selected={false} disabled={false} onClick={() => {}} />);
    expect(screen.getByRole('button')).toHaveTextContent('5');
  });

  it('has aria-pressed=true when selected', () => {
    render(<Card value="5" selected={true} disabled={false} onClick={() => {}} />);
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'true');
  });

  it('has aria-pressed=false when not selected', () => {
    render(<Card value="5" selected={false} disabled={false} onClick={() => {}} />);
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'false');
  });

  it('is disabled when disabled prop is true', () => {
    render(<Card value="5" selected={false} disabled={true} onClick={() => {}} />);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('calls onClick when clicked and not disabled', () => {
    const onClick = vi.fn();
    render(<Card value="5" selected={false} disabled={false} onClick={onClick} />);
    screen.getByRole('button').click();
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('does not call onClick when disabled', () => {
    const onClick = vi.fn();
    render(<Card value="5" selected={false} disabled={true} onClick={onClick} />);
    screen.getByRole('button').click();
    expect(onClick).not.toHaveBeenCalled();
  });

  it('has card--selected class when selected', () => {
    render(<Card value="5" selected={true} disabled={false} onClick={() => {}} />);
    expect(screen.getByRole('button')).toHaveClass('card--selected');
  });

  it('does not have card--selected class when not selected', () => {
    render(<Card value="5" selected={false} disabled={false} onClick={() => {}} />);
    expect(screen.getByRole('button')).not.toHaveClass('card--selected');
  });
});
