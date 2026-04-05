/// <reference types="vitest/globals" />
import { render, screen } from '@testing-library/preact';
import { selectedCard, roomState, connectionStatus } from '../../state';
import { CardDeck } from './CardDeck';

// Mock the ws module to prevent actual WebSocket calls
vi.mock('../../ws', () => ({
  send: vi.fn(),
}));

beforeEach(() => {
  selectedCard.value = '';
  connectionStatus.value = 'connected';
  roomState.value = {
    roomId: 'test-room',
    roomName: 'Test Room',
    phase: 'voting',
    participants: [],
    result: null,
  };
});

describe('CardDeck', () => {
  it('renders container with role="group"', () => {
    render(<CardDeck />);
    expect(screen.getByRole('group')).toBeInTheDocument();
  });

  it('renders container with aria-label="Select your vote"', () => {
    render(<CardDeck />);
    expect(screen.getByRole('group')).toHaveAttribute('aria-label', 'Select your vote');
  });

  it('renders all 12 card values', () => {
    render(<CardDeck />);
    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(12);
  });

  it('selected card has aria-pressed=true', () => {
    selectedCard.value = '5';
    render(<CardDeck />);
    const selected = screen.getByText('5');
    expect(selected).toHaveAttribute('aria-pressed', 'true');
  });

  it('non-selected cards have aria-pressed=false', () => {
    selectedCard.value = '5';
    render(<CardDeck />);
    const other = screen.getByText('3');
    expect(other).toHaveAttribute('aria-pressed', 'false');
  });

  it('cards are disabled during reveal phase', () => {
    roomState.value = {
      roomId: 'test-room',
      roomName: 'Test Room',
      phase: 'reveal',
      participants: [],
      result: null,
    };
    render(<CardDeck />);
    const buttons = screen.getAllByRole('button');
    buttons.forEach((btn) => {
      expect(btn).toBeDisabled();
    });
  });
});
