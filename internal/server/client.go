package server

import (
	"context"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

const (
	sendBufferSize = 32
	writeTimeout   = 5 * time.Second
	pingInterval   = 5 * time.Second
	pingTimeout    = 10 * time.Second
)

// Client represents a single WebSocket connection to a room.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	mu        sync.Mutex // protects sessionID
	sessionID string
	roomID    string
	manager   *RoomManager
	closeOnce sync.Once
	done      chan struct{}
}

// SessionID returns the current session ID (thread-safe).
func (c *Client) SessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// SetSessionID updates the session ID (thread-safe).
func (c *Client) SetSessionID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}

// NewClient creates a client bound to a WebSocket connection.
func NewClient(conn *websocket.Conn, roomID string, manager *RoomManager) *Client {
	return &Client{
		conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		roomID:  roomID,
		manager: manager,
		done:    make(chan struct{}),
	}
}

// Send queues a message for sending. Non-blocking; drops message if buffer full.
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		log.Printf("client %s: send buffer full, dropping message", c.SessionID())
	}
}

// SendError sends an error message to the client.
func (c *Client) SendError(code, message string) {
	msg, err := MakeEnvelope("error", ErrorPayload{Code: code, Message: message})
	if err != nil {
		log.Printf("client %s: failed to create error message: %v", c.SessionID(), err)
		return
	}
	c.Send(msg)
}

// Close shuts down the client connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

// WritePump sends messages from the send channel to the WebSocket.
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				log.Printf("client %s: write error: %v", c.SessionID(), err)
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Printf("client %s: ping error: %v", c.SessionID(), err)
				return
			}
			// Update last ping time for the participant.
			if sid := c.SessionID(); sid != "" {
				c.manager.UpdatePingTime(c.roomID, sid)
			}
		}
	}
}
