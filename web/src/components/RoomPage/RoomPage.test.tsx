/// <reference types="vitest/globals" />
import { render, screen, fireEvent } from '@testing-library/preact';
import {
  roomState,
  connectionStatus,
  userName,
  selectedCard,
  actionPending,
} from '../../state';
import { RoomPage } from './RoomPage';

const mockSend = vi.fn();

vi.mock('../../ws', () => ({
  connect: vi.fn(),
  disconnect: vi.fn(),
  send: (...args: unknown[]) => mockSend(...args),
  retry: vi.fn(),
}));

vi.mock('../../hooks/usePresence', () => ({
  usePresence: vi.fn(),
}));

function setVotingState(voted = 1) {
  connectionStatus.value = 'connected';
  userName.value = 'Alice';
  selectedCard.value = '';
  actionPending.value = false;
  roomState.value = {
    roomId: 'test-room',
    roomName: 'Test Room',
    createdBy: 'Alice',
    phase: 'voting',
    participants: [
      { sessionId: 's1', userName: 'Alice', status: 'active', hasVoted: voted > 0, role: 'voter' },
      { sessionId: 's2', userName: 'Bob', status: 'active', hasVoted: voted > 1, role: 'voter' },
    ],
    result: null,
    timer: { duration: 30, state: 'idle', startedAt: null, remaining: 30 },
  };
}

function setRevealState() {
  connectionStatus.value = 'connected';
  userName.value = 'Alice';
  selectedCard.value = '';
  actionPending.value = false;
  roomState.value = {
    roomId: 'test-room',
    roomName: 'Test Room',
    createdBy: 'Alice',
    phase: 'reveal',
    participants: [
      { sessionId: 's1', userName: 'Alice', status: 'active', hasVoted: true, vote: '5', role: 'voter' },
      { sessionId: 's2', userName: 'Bob', status: 'active', hasVoted: true, vote: '8', role: 'voter' },
    ],
    result: {
      votes: [
        { sessionId: 's1', name: 'Alice', value: '5' },
        { sessionId: 's2', name: 'Bob', value: '8' },
      ],
      average: 6.5,
      median: 6.5,
      uncertainCount: 0,
      totalVoters: 2,
      hasConsensus: false,
      spread: [5, 8],
    },
    timer: { duration: 30, state: 'idle', startedAt: null, remaining: 30 },
  };
}

beforeEach(() => {
  mockSend.mockClear();
  actionPending.value = false;
});

describe('RoomPage — double-click protection', () => {
  it('sends reveal on first click', () => {
    setVotingState();
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText(/Show Votes/);
    fireEvent.click(btn);
    expect(mockSend).toHaveBeenCalledWith({ type: 'reveal', payload: {} });
    expect(mockSend).toHaveBeenCalledTimes(1);
  });

  it('disables Show Votes button when actionPending is true', () => {
    setVotingState();
    actionPending.value = true;
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText(/Show Votes/);
    expect(btn).toBeDisabled();
  });

  it('sets actionPending to true after clicking Show Votes', () => {
    setVotingState();
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText(/Show Votes/);
    fireEvent.click(btn);
    expect(actionPending.value).toBe(true);
  });

  it('does not send duplicate reveal when actionPending', () => {
    setVotingState();
    actionPending.value = true;
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText(/Show Votes/);
    fireEvent.click(btn);
    expect(mockSend).not.toHaveBeenCalled();
  });

  it('sends new_round on first click', () => {
    setRevealState();
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText('New Round');
    fireEvent.click(btn);
    expect(mockSend).toHaveBeenCalledWith({ type: 'new_round', payload: {} });
    expect(mockSend).toHaveBeenCalledTimes(1);
  });

  it('disables New Round button when actionPending is true', () => {
    setRevealState();
    actionPending.value = true;
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText('New Round');
    expect(btn).toBeDisabled();
  });

  it('sets actionPending to true after clicking New Round', () => {
    setRevealState();
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText('New Round');
    fireEvent.click(btn);
    expect(actionPending.value).toBe(true);
  });

  it('does not send duplicate new_round when actionPending', () => {
    setRevealState();
    actionPending.value = true;
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText('New Round');
    fireEvent.click(btn);
    expect(mockSend).not.toHaveBeenCalled();
  });

  it('Show Votes is disabled when no votes cast', () => {
    setVotingState(0);
    render(<RoomPage path="/room/test-room" />);
    const btn = screen.getByText(/Show Votes/);
    expect(btn).toBeDisabled();
  });
});
