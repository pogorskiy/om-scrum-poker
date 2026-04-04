import { isRevealed, sessionId, type Participant } from '../../state';
import './ParticipantCard.css';

interface Props {
  participant: Participant;
}

export function ParticipantCard({ participant }: Props) {
  const revealed = isRevealed.value;
  const isSelf = participant.sessionId === sessionId.value;

  const cardClasses = [
    'participant-card',
    participant.status === 'disconnected' && 'participant-card--disconnected',
    isSelf && 'participant-card--self',
  ]
    .filter(Boolean)
    .join(' ');

  // Determine vote indicator
  let voteContent: string;
  let voteClass = 'participant-card__vote';

  if (revealed && participant.vote !== undefined) {
    voteContent = participant.vote;
  } else if (participant.status === 'disconnected') {
    voteContent = '-';
    voteClass += ' participant-card__vote--empty';
  } else if (participant.hasVoted) {
    voteContent = '\u2713';
    voteClass += ' participant-card__vote--check';
  } else {
    voteContent = '';
    voteClass += ' participant-card__vote--empty';
  }

  return (
    <div class={cardClasses}>
      <span class={`participant-card__status participant-card__status--${participant.status}`} />
      <span class="participant-card__name">{participant.userName}</span>
      <span class={voteClass}>{voteContent}</span>
    </div>
  );
}
