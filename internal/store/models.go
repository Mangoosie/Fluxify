package store

type BridgeLink struct {
	ID               int64
	DiscordGuildID   string
	DiscordChannelID string
	FluxerGuildID    string
	FluxerChannelID  string
	CreatedAt        string
}
