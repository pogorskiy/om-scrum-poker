// Slugify a room name for use in URLs
function slugify(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 48);
}

// Generate 12 random hex characters
function randomHex(): string {
  const arr = new Uint8Array(6);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('');
}

// Generate a room URL path from a human-readable name
export function generateRoomUrl(name: string): string {
  const slug = slugify(name);
  const hex = randomHex();
  const roomId = slug ? `${slug}-${hex}` : hex;
  return `/room/${roomId}`;
}

// Parse room ID from a URL path like /room/:id
export function parseRoomId(path: string): string | null {
  const match = path.match(/^\/room\/([a-z0-9-]+)$/);
  return match ? match[1] : null;
}
