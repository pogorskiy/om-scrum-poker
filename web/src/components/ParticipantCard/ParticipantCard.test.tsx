/// <reference types="vitest/globals" />
import { render } from '@testing-library/preact';
import { roomState, sessionId, type Participant } from '../../state';
import { ParticipantCard } from './ParticipantCard';

// Helper to set isRevealed via roomState (isRevealed is a computed signal)
function setRevealed(revealed: boolean) {
  roomState.value = {
    roomId: 'test-room',
    roomName: 'Test Room',
    phase: revealed ? 'reveal' : 'voting',
    participants: [],
    result: null,
  };
}

const activeVoter: Participant = {
  sessionId: 'user-1',
  userName: 'Alice',
  status: 'active',
  hasVoted: true,
  vote: '5',
};

const activeNonVoter: Participant = {
  sessionId: 'user-2',
  userName: 'Bob',
  status: 'active',
  hasVoted: false,
};

const disconnectedUser: Participant = {
  sessionId: 'user-3',
  userName: 'Charlie',
  status: 'disconnected',
  hasVoted: false,
};

const idleUser: Participant = {
  sessionId: 'user-4',
  userName: 'Diana',
  status: 'idle',
  hasVoted: false,
};

beforeEach(() => {
  setRevealed(false);
  sessionId.value = 'my-session';
});

describe('ParticipantCard', () => {
  // --- Structure tests ---

  it('renders participant name correctly', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const name = container.querySelector('.participant-card__name');
    expect(name).toHaveTextContent('Alice');
  });

  it('renders status indicator with correct class for active', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const status = container.querySelector('.participant-card__status');
    expect(status).toHaveClass('participant-card__status--active');
  });

  it('renders status indicator with correct class for idle', () => {
    const { container } = render(<ParticipantCard participant={idleUser} index={0} />);
    const status = container.querySelector('.participant-card__status');
    expect(status).toHaveClass('participant-card__status--idle');
  });

  it('renders status indicator with correct class for disconnected', () => {
    const { container } = render(<ParticipantCard participant={disconnectedUser} index={0} />);
    const status = container.querySelector('.participant-card__status');
    expect(status).toHaveClass('participant-card__status--disconnected');
  });

  it('has self class when participant is current session user', () => {
    sessionId.value = 'user-1';
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const card = container.querySelector('.participant-card');
    expect(card).toHaveClass('participant-card--self');
  });

  it('has disconnected class when participant is disconnected', () => {
    const { container } = render(<ParticipantCard participant={disconnectedUser} index={0} />);
    const card = container.querySelector('.participant-card');
    expect(card).toHaveClass('participant-card--disconnected');
  });

  // --- Flip animation tests ---

  it('contains flip container with front and back faces', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).toBeInTheDocument();
    expect(flipper!.querySelector('.participant-card__vote-face--front')).toBeInTheDocument();
    expect(flipper!.querySelector('.participant-card__vote-face--back')).toBeInTheDocument();
  });

  it('front face shows checkmark when participant has voted and not revealed', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const front = container.querySelector('.participant-card__vote-face--front');
    expect(front).toHaveTextContent('\u2713');
  });

  it('front face shows dash when participant is disconnected', () => {
    const { container } = render(<ParticipantCard participant={disconnectedUser} index={0} />);
    const front = container.querySelector('.participant-card__vote-face--front');
    expect(front).toHaveTextContent('-');
  });

  it('front face is empty when participant has not voted', () => {
    const { container } = render(<ParticipantCard participant={activeNonVoter} index={0} />);
    const front = container.querySelector('.participant-card__vote-face--front');
    expect(front).toHaveTextContent('');
  });

  it('back face shows vote value when available', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const back = container.querySelector('.participant-card__vote-face--back');
    expect(back).toHaveTextContent('5');
  });

  it('flipper has --revealed class when isRevealed is true AND participant has voted', () => {
    setRevealed(true);
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).toHaveClass('participant-card__vote-flipper--revealed');
  });

  it('flipper does NOT have --revealed class when isRevealed is false', () => {
    setRevealed(false);
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).not.toHaveClass('participant-card__vote-flipper--revealed');
  });

  it('flipper does NOT have --revealed class when participant has not voted even if revealed', () => {
    setRevealed(true);
    const { container } = render(<ParticipantCard participant={activeNonVoter} index={0} />);
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).not.toHaveClass('participant-card__vote-flipper--revealed');
  });

  it('CSS custom property --flip-delay is set based on index prop', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={2} />);
    const flipper = container.querySelector('.participant-card__vote-flipper') as HTMLElement;
    expect(flipper.style.getPropertyValue('--flip-delay')).toBe('0.16s');
  });

  it('--flip-delay for index=0 is "0s"', () => {
    const { container } = render(<ParticipantCard participant={activeVoter} index={0} />);
    const flipper = container.querySelector('.participant-card__vote-flipper') as HTMLElement;
    expect(flipper.style.getPropertyValue('--flip-delay')).toBe('0s');
  });

  // --- Edge cases ---

  it('disconnected participant with no vote: front shows dash, no flip on reveal', () => {
    setRevealed(true);
    const { container } = render(<ParticipantCard participant={disconnectedUser} index={0} />);
    const front = container.querySelector('.participant-card__vote-face--front');
    expect(front).toHaveTextContent('-');
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).not.toHaveClass('participant-card__vote-flipper--revealed');
  });

  it('participant with vote retracted: front is empty, no flip', () => {
    setRevealed(true);
    const retractedVoter: Participant = {
      sessionId: 'user-5',
      userName: 'Eve',
      status: 'active',
      hasVoted: false,
    };
    const { container } = render(<ParticipantCard participant={retractedVoter} index={0} />);
    const front = container.querySelector('.participant-card__vote-face--front');
    expect(front).toHaveTextContent('');
    const flipper = container.querySelector('.participant-card__vote-flipper');
    expect(flipper).not.toHaveClass('participant-card__vote-flipper--revealed');
  });
});
