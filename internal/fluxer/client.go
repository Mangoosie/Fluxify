package fluxer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is a hand-rolled Fluxer HTTP + gateway client.
//
// Endpoints are discovered from the instance via GET /.well-known/fluxer;
// authenticated requests use the "Bot <token>" authorization scheme. The
// gateway state (session ID and sequence) is retained across reconnects so a
// dropped socket can resume.
type Client struct {
	origin string
	token  string
	http   *http.Client

	apiBase string
	gwURL   string

	mu        sync.Mutex
	sessionID string
	seq       int
	hasSeq    bool
	botUser   User

	writeMu sync.Mutex
}

// NewClient returns a client for the given instance origin and bot token.
func NewClient(origin, token string) *Client {
	return &Client{
		origin: strings.TrimRight(origin, "/"),
		token:  token,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

type endpointDocument struct {
	Endpoints struct {
		APIPublic string `json:"api_public"`
		Gateway   string `json:"gateway"`
	} `json:"endpoints"`
}

type gatewayBot struct {
	URL string `json:"url"`
}

// Discover reads the instance discovery document and records the public API
// base URL and the advertised gateway URL.
func (c *Client) Discover(ctx context.Context) error {
	var doc endpointDocument
	if err := c.getJSON(ctx, c.origin+"/.well-known/fluxer", &doc); err != nil {
		return fmt.Errorf("instance discovery: %w", err)
	}
	if doc.Endpoints.APIPublic == "" {
		return fmt.Errorf("instance discovery: missing endpoints.api_public")
	}
	c.apiBase = strings.TrimRight(doc.Endpoints.APIPublic, "/")
	if c.gwURL == "" {
		c.gwURL = doc.Endpoints.Gateway
	}
	return nil
}

// GatewayURL returns the gateway WebSocket URL, preferring the value from the
// bot endpoint and falling back to the discovery document.
func (c *Client) GatewayURL(ctx context.Context) (string, error) {
	if c.apiBase == "" {
		if err := c.Discover(ctx); err != nil {
			return "", err
		}
	}
	var g gatewayBot
	if err := c.getJSON(ctx, c.apiBase+"/v1/gateway/bot", &g); err != nil {
		return "", fmt.Errorf("get gateway: %w", err)
	}
	if g.URL != "" {
		c.gwURL = g.URL
	}
	if c.gwURL == "" {
		return "", fmt.Errorf("gateway URL is empty")
	}
	return c.gwURL, nil
}

// CreateMessage posts a message to a Fluxer channel.
func (c *Client) CreateMessage(ctx context.Context, channelID, content string) (*Message, error) {
	if c.apiBase == "" {
		if err := c.Discover(ctx); err != nil {
			return nil, err
		}
	}

	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiBase+"/v1/channels/"+channelID+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("create message: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	var msg Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// TriggerTyping shows the typing indicator in a Fluxer channel.
func (c *Client) TriggerTyping(ctx context.Context, channelID string) error {
	if c.apiBase == "" {
		if err := c.Discover(ctx); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiBase+"/v1/channels/"+channelID+"/typing", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("trigger typing: %s", resp.Status)
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
