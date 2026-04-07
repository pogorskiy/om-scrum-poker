import { useSignal } from '@preact/signals';
import { useEffect } from 'preact/hooks';
import './Footer.css';

function formatDeployTime(isoString: string): string {
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return 'Development build';

  const options: Intl.DateTimeFormatOptions = {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'UTC',
  };

  return date.toLocaleDateString('en-US', options) + ' UTC';
}

export function Footer() {
  const deployTime = useSignal('');

  useEffect(() => {
    fetch('/health')
      .then((r) => r.json())
      .then((data) => {
        if (data.build_time) {
          deployTime.value = data.build_time;
        }
      })
      .catch(() => {});
  }, []);

  const formatted = formatDeployTime(deployTime.value);
  const isValid = formatted !== 'Development build';

  return (
    <footer class="footer">
      <span class="footer__text">
        {isValid ? (
          <>
            Deployed:{' '}
            <time class="footer__time" dateTime={deployTime.value}>
              {formatted}
            </time>
          </>
        ) : (
          formatted
        )}
      </span>
    </footer>
  );
}

export { formatDeployTime };
