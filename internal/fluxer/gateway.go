package fluxer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	neturl "net/url"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
	healthyAfter   = 30 * time.Second
)

// Run connects to the Fluxer gateway and dispatches MESSAGE_CREATE events to
// onMessage until ctx is cancelled. It reconnects automatically, resuming the
// session where possible.
func (c *Client) Run(ctx context.Context, onMessage func(Message)) error {
	url, err := c.GatewayURL(ctx)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(url, "ws") {
		if strings.HasPrefix(url, "https") {
			url = "wss" + strings.TrimPrefix(url, "https")
		} else if strings.HasPrefix(url, "http") {
			url = "ws" + strings.TrimPrefix(url, "http")
		}
	}
	
	parsed, err := neturl.Parse(url)
	if (err != nil || parsed.Scheme != "wss" || parsed.Host == "") {
		return fmt.Errorf("gateway URL must use wss")
	}

	backoff := initialBackoff
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		start := time.Now()
		err := c.connectOnce(ctx, url, onMessage)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if time.Since(start) > healthyAfter {
			backoff = initialBackoff
		}
		if err != nil {
			log.Printf("fluxer: gateway connection ended: %v", err)
		}

		log.Printf("fluxer: reconnecting in %s", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *Client) connectOnce(ctx context.Context, base string, onMessage func(Message)) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, base+"?v=1&encoding=json", nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	if err := c.awaitHello(conn); err != nil {
		return err
	}

	hello, err := c.readHello(conn)
	if err != nil {
		return err
	}

	if err := c.identify(ctx, conn); err != nil {
		return err
	}

	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	go c.heartbeatLoop(hbCtx, conn, time.Duration(hello.HeartbeatInterval)*time.Millisecond)

	return c.readLoop(ctx, conn, onMessage)
}

// awaitHello sets a short read deadline for the initial Hello frame.
func (c *Client) awaitHello(conn *websocket.Conn) error {
	return conn.SetReadDeadline(time.Now().Add(30 * time.Second))
}

func (c *Client) readHello(conn *websocket.Conn) (*Hello, error) {
	var payload GatewayPayload
	if err := conn.ReadJSON(&payload); err != nil {
		return nil, fmt.Errorf("read hello: %w", err)
	}
	if payload.Op != opHello {
		return nil, fmt.Errorf("expected hello (op 10), got op %d", payload.Op)
	}
	var hello Hello
	if err := json.Unmarshal(payload.D, &hello); err != nil {
		return nil, fmt.Errorf("decode hello: %w", err)
	}
	return &hello, nil
}

func (c *Client) identify(ctx context.Context, conn *websocket.Conn) error {
	_ = ctx

	c.mu.Lock()
	sessionID := c.sessionID
	seq := c.seq
	c.mu.Unlock()

	if sessionID != "" {
		log.Printf("fluxer: resuming session %s", sessionID)
		return c.writeJSON(conn, GatewayPayload{
			Op: opResume,
			D:  mustJSON(Resume{Token: c.token, SessionID: sessionID, Seq: &seq}),
		})
	}

	log.Printf("fluxer: identifying")
	return c.writeJSON(conn, GatewayPayload{
		Op: opIdentify,
		D: mustJSON(Identify{
			Token: c.token,
			Properties: IdentifyProperties{
				OS:      runtime.GOOS,
				Browser: "Fluxify",
				Device:  "fluxify",
			},
		}),
	})
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, onMessage func(Message)) error {
	for {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

		var payload GatewayPayload
		if err := conn.ReadJSON(&payload); err != nil {
			return err
		}

		switch payload.Op {
		case opDispatch:
			c.handleDispatch(payload, onMessage)
		case opHeartbeat:
			// The server asks for an immediate heartbeat.
			c.sendHeartbeat(conn)
		case opHeartbeatACK:
			// Nothing to do.
		case opReconnect:
			return errors.New("gateway requested reconnect")
		case opInvalidSession:
			c.resetSession()
			return errors.New("invalid session")
		}
	}
}

func (c *Client) handleDispatch(payload GatewayPayload, onMessage func(Message)) {
	c.mu.Lock()
	c.seq = payload.S
	c.hasSeq = true
	c.mu.Unlock()

	switch payload.T {
	case eventReady:
		var ready Ready
		if err := json.Unmarshal(payload.D, &ready); err != nil {
			log.Printf("fluxer: decode ready: %v", err)
			return
		}
		c.mu.Lock()
		c.sessionID = ready.SessionID
		c.botUser = ready.User
		c.mu.Unlock()
		log.Printf("fluxer: ready as %s (%s)", ready.User.Username, ready.User.ID)
	case eventMessageCreate:
		var msg Message
		if err := json.Unmarshal(payload.D, &msg); err != nil {
			log.Printf("fluxer: decode message: %v", err)
			return
		}
		if c.isSelf(msg.Author) {
			return
		}
		if onMessage != nil {
			onMessage(msg)
		}
	}
}

func (c *Client) heartbeatLoop(ctx context.Context, conn *websocket.Conn, interval time.Duration) {
	if interval <= 0 {
		interval = 40 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sendHeartbeat(conn)
		}
	}
}

func (c *Client) sendHeartbeat(conn *websocket.Conn) {
	c.mu.Lock()
	var d any
	if c.hasSeq {
		d = c.seq
	}
	c.mu.Unlock()

	if err := c.writeJSON(conn, GatewayPayload{Op: opHeartbeat, D: mustJSON(d)}); err != nil {
		log.Printf("fluxer: heartbeat failed: %v", err)
	}
}

func (c *Client) writeJSON(conn *websocket.Conn, v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

func (c *Client) isSelf(u User) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.botUser.ID != "" && u.ID == c.botUser.ID
}

func (c *Client) resetSession() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = ""
	c.hasSeq = false
	c.seq = 0
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
