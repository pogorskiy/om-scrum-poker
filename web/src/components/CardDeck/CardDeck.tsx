import { selectedCard, isRevealed, connectionStatus } from '../../state';
import { send } from '../../ws';
import { Card } from '../Card/Card';
import './CardDeck.css';

const CARD_VALUES = ['?', '0', '0.5', '1', '2', '3', '5', '8', '13', '20', '40', '100'];

export function CardDeck() {
  const current = selectedCard.value;
  const disabled = isRevealed.value;
  const offline = connectionStatus.value !== 'connected';

  function handleClick(value: string) {
    if (disabled) return;

    if (current === value) {
      // Deselect — retract vote
      selectedCard.value = '';
      send({ type: 'vote', payload: { value: '' } });
    } else {
      // Select — cast vote
      selectedCard.value = value;
      send({ type: 'vote', payload: { value } });
    }
  }

  return (
    <div class={`card-deck${offline ? ' card-deck--disabled' : ''}`} role="group" aria-label="Select your vote">
      {CARD_VALUES.map((v) => (
        <Card
          key={v}
          value={v}
          selected={current === v}
          disabled={disabled}
          onClick={() => handleClick(v)}
        />
      ))}
    </div>
  );
}
