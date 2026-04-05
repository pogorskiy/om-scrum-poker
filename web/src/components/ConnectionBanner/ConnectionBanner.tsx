import { connectionStatus, reconnectInfo } from '../../state';
import { retry } from '../../ws';
import './ConnectionBanner.css';

export function ConnectionBanner() {
  const status = connectionStatus.value;
  const info = reconnectInfo.value;

  // Nothing to show when connected
  if (status === 'connected') return null;

  // Connection lost: maxReached and disconnected
  if (info.maxReached) {
    return (
      <div class="connection-banner connection-banner--lost" role="alert">
        <span class="connection-banner__text">Connection lost</span>
        <button class="connection-banner__retry" onClick={() => retry()}>
          Retry
        </button>
      </div>
    );
  }

  // Reconnecting: still attempting (connecting or disconnected, not maxReached)
  if (info.attempt > 0) {
    return (
      <div class="connection-banner connection-banner--reconnecting" role="status">
        <span class="connection-banner__spinner" />
        <span class="connection-banner__text">
          Reconnecting... (attempt {info.attempt})
        </span>
      </div>
    );
  }

  return null;
}
