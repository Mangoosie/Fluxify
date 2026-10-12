package discord

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Bot wraps a discordgo session with the intents the bridge needs.
type Bot struct {
	session *discordgo.Session
}

// New creates a Discord bot session for the given token.
func New(token string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent
	return &Bot{session: session}, nil
}

// Run opens the Discord gateway and invokes onMessage for every non-bot
// message until ctx is cancelled.
func (b *Bot) Run(ctx context.Context, onMessage func(*discordgo.MessageCreate)) error {
	b.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}
		if onMessage != nil {
			onMessage(m)
		}
	})

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord gateway: %w", err)
	}
	defer b.session.Close()

	if b.session.State != nil && b.session.State.User != nil {
		log.Printf("discord: connected as %s", b.session.State.User.Username)
	}

	<-ctx.Done()
	return nil
}

// SendMessage posts a plain text message to a channel.
func (b *Bot) SendMessage(_ context.Context, channelID, content string) error {
	_, err := b.session.ChannelMessageSend(channelID, content)
	return err
}
