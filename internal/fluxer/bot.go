package fluxer

import (
	"context"
	"log"
)

// Bot is a high-level Fluxer bot built on the Client.
type Bot struct {
	client *Client
}

// New returns a bot for the given token and instance origin.
func New(token, origin string) *Bot {
	return &Bot{client: NewClient(origin, token)}
}

// Run discovers the instance, connects to the gateway, and invokes onMessage
// for every inbound MESSAGE_CREATE until ctx is cancelled.
func (b *Bot) Run(ctx context.Context, onMessage func(Message)) error {
	if err := b.client.Discover(ctx); err != nil {
		return err
	}
	log.Printf("fluxer: api base %s", b.client.apiBase)
	return b.client.Run(ctx, onMessage)
}

// SendMessage posts a plain text message to a channel.
func (b *Bot) SendMessage(ctx context.Context, channelID, content string) error {
	_, err := b.client.CreateMessage(ctx, channelID, content)
	return err
}

// TriggerTyping shows the typing indicator in a channel.
func (b *Bot) TriggerTyping(ctx context.Context, channelID string) error {
	return b.client.TriggerTyping(ctx, channelID)
}
