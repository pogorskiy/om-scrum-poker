import { isRevealed, sessionId, type Participant } from '../../state';
import './ParticipantCard.css';

interface Props {
  participant: Participant;
  index: number;
}

export function ParticipantCard({ participant, index }: Props) {
  const revealed = isRevealed.value;
  const isSelf = participant.sessionId === sessionId.value;

  const cardClasses = [
    'participant-card',
    participant.status === 'disconnected' && 'participant-card--disconnected',
    isSelf && 'participant-card--self',
  ]
    .filter(Boolean)
    .join(' ');

  const hasVote = participant.hasVoted && participant.vote !== undefined;
  const showFlip = revealed && hasVote;

  // Front face content (before reveal)
  let frontContent: string;
  let frontClass = 'participant-card__vote-face participant-card__vote-face--front';
  if (participant.status === 'disconnected') {
    frontContent = '-';
    frontClass += ' participant-card__vote--empty';
  } else if (participant.hasVoted) {
    frontContent = '\u2713';
    frontClass += ' participant-card__vote--check';
  } else {
    frontContent = '';
    frontClass += ' participant-card__vote--empty';
  }

  // Back face content (after reveal)
  const backContent = participant.vote ?? '';

  const flipperClasses = [
    'participant-card__vote-flipper',
    showFlip && 'participant-card__vote-flipper--revealed',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div class={cardClasses} aria-label={`${participant.userName} - ${participant.status}`}>
      <span
        class={`participant-card__status participant-card__status--${participant.status}`}
        role="img"
        aria-label={`Status: ${participant.status}`}
      />
      <span class="participant-card__name">{participant.userName}</span>
      <div
        class={flipperClasses}
        style={`--flip-delay: ${index * 0.08}s`}
      >
        <span class={frontClass}>{frontContent}</span>
        <span class="participant-card__vote-face participant-card__vote-face--back">
          {backContent}
        </span>
      </div>
    </div>
  );
}
