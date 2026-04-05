import { describe, it, expect, beforeEach } from 'vitest';
import { roomState, voteCount, myParticipant, sessionId, userRole, type Participant, type RoomState } from './state';

function makeParticipant(overrides: Partial<Participant> = {}): Participant {
  return {
    sessionId: 'test-session',
    userName: 'Test User',
    status: 'active',
    hasVoted: false,
    role: 'voter',
    ...overrides,
  };
}

function makeRoomState(participants: Participant[]): RoomState {
  return {
    roomId: 'test-room',
    roomName: 'Test Room',
    createdBy: 'Someone',
    phase: 'voting',
    participants,
    result: null,
  };
}

describe('Observer mode', () => {
  beforeEach(() => {
    roomState.value = null;
    userRole.value = 'voter';
    sessionId.value = 'my-session';
  });

  describe('voteCount', () => {
    it('excludes observers from total count', () => {
      roomState.value = makeRoomState([
        makeParticipant({ sessionId: 'a', role: 'voter', hasVoted: true }),
        makeParticipant({ sessionId: 'b', role: 'voter', hasVoted: false }),
        makeParticipant({ sessionId: 'c', role: 'observer', hasVoted: false }),
      ]);
      const counts = voteCount.value;
      expect(counts.total).toBe(2);
      expect(counts.voted).toBe(1);
    });

    it('excludes disconnected participants', () => {
      roomState.value = makeRoomState([
        makeParticipant({ sessionId: 'a', role: 'voter', status: 'active', hasVoted: true }),
        makeParticipant({ sessionId: 'b', role: 'voter', status: 'disconnected', hasVoted: false }),
      ]);
      const counts = voteCount.value;
      expect(counts.total).toBe(1);
      expect(counts.voted).toBe(1);
    });

    it('returns zeros when no room state', () => {
      roomState.value = null;
      const counts = voteCount.value;
      expect(counts.total).toBe(0);
      expect(counts.voted).toBe(0);
    });
  });

  describe('myParticipant', () => {
    it('detects observer role for current user', () => {
      roomState.value = makeRoomState([
        makeParticipant({ sessionId: 'my-session', role: 'observer' }),
      ]);
      expect(myParticipant.value?.role).toBe('observer');
    });

    it('detects voter role for current user', () => {
      roomState.value = makeRoomState([
        makeParticipant({ sessionId: 'my-session', role: 'voter' }),
      ]);
      expect(myParticipant.value?.role).toBe('voter');
    });
  });

  describe('userRole signal', () => {
    it('defaults to voter', () => {
      expect(userRole.value).toBe('voter');
    });

    it('can be set to observer', () => {
      userRole.value = 'observer';
      expect(userRole.value).toBe('observer');
    });
  });

  describe('role_updated handler logic', () => {
    it('updates participant role in room state', () => {
      roomState.value = makeRoomState([
        makeParticipant({ sessionId: 'a', role: 'voter' }),
        makeParticipant({ sessionId: 'b', role: 'voter' }),
      ]);

      // Simulate what the role_updated handler does
      const targetId = 'a';
      const newRole = 'observer';
      roomState.value = {
        ...roomState.value!,
        participants: roomState.value!.participants.map((p) =>
          p.sessionId === targetId ? { ...p, role: newRole as Participant['role'] } : p
        ),
      };

      expect(roomState.value.participants[0].role).toBe('observer');
      expect(roomState.value.participants[1].role).toBe('voter');
    });
  });
});
