CREATE TABLE IF NOT EXISTS bridge_links (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    discord_guild_id   TEXT NOT NULL,
    discord_channel_id TEXT NOT NULL UNIQUE,
    fluxer_guild_id    TEXT NOT NULL,
    fluxer_channel_id  TEXT NOT NULL UNIQUE,
    created_at         TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS seen_messages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    platform   TEXT NOT NULL CHECK (platform IN ('discord', 'fluxer')),
    message_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (platform, message_id)
);

CREATE INDEX IF NOT EXISTS idx_seen_messages_channel ON seen_messages (platform, channel_id);
