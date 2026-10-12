package fluxer

import "encoding/json"

// Gateway opcodes used by the Fluxer gateway.
const (
	opDispatch       = 0
	opHeartbeat      = 1
	opIdentify       = 2
	opResume         = 6
	opReconnect      = 7
	opInvalidSession = 9
	opHello          = 10
	opHeartbeatACK   = 11
)

// Gateway event names handled by the bridge.
const (
	eventReady         = "READY"
	eventMessageCreate = "MESSAGE_CREATE"
)

// GatewayPayload is the envelope for every gateway message.
type GatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  int             `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

// Hello is the payload of opcode 10.
type Hello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// IdentifyProperties describes the connecting client to the gateway.
type IdentifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

// Identify is the opcode 2 payload.
type Identify struct {
	Token      string             `json:"token"`
	Properties IdentifyProperties `json:"properties"`
}

// Resume is the opcode 6 payload.
type Resume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       *int   `json:"seq"`
}

// User is the subset of a Fluxer user used by the bridge.
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot"`
}

// Ready is the payload of the READY dispatch.
type Ready struct {
	SessionID string `json:"session_id"`
	User      User   `json:"user"`
}

// Message is the subset of a Fluxer message used by the bridge.
type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Author    User   `json:"author"`
}
