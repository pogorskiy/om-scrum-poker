import './Card.css';

interface Props {
  value: string;
  selected: boolean;
  disabled: boolean;
  onClick: () => void;
}

export function Card({ value, selected, disabled, onClick }: Props) {
  const classes = [
    'card',
    selected && 'card--selected',
    disabled && 'card--disabled',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <button class={classes} onClick={disabled ? undefined : onClick} disabled={disabled}>
      {value}
    </button>
  );
}
