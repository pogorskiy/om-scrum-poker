import { useState } from 'preact/hooks';
import { navigate } from '../../state';
import { generateRoomUrl } from '../../utils/room-url';
import './HomePage.css';

export function HomePage() {
  const [roomName, setRoomName] = useState('');

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmed = roomName.trim();
    if (!trimmed) return;
    const url = generateRoomUrl(trimmed);
    navigate(url);
  }

  return (
    <div class="home">
      <div class="home__card">
        <h1 class="home__title">om-scrum-poker</h1>
        <p class="home__subtitle">Simple. Self-hosted. No signup.</p>
        <form class="home__form" onSubmit={handleSubmit}>
          <input
            class="home__input"
            type="text"
            maxLength={60}
            placeholder="Enter room name..."
            value={roomName}
            onInput={(e) => setRoomName((e.target as HTMLInputElement).value)}
            autoFocus
          />
          <button class="home__btn" type="submit" disabled={!roomName.trim()}>
            Create Room
          </button>
        </form>
      </div>
    </div>
  );
}
