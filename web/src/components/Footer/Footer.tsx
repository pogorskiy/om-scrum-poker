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
  const deployTime = typeof __BUILD_TIMESTAMP__ !== 'undefined' ? __BUILD_TIMESTAMP__ : '';
  const formatted = formatDeployTime(deployTime);
  const isValid = formatted !== 'Development build';

  return (
    <footer class="footer">
      <span class="footer__text">
        {isValid ? (
          <>
            Deployed:{' '}
            <time class="footer__time" dateTime={deployTime}>
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
