# Fluxify
This repo is meant to allow gradual transition from a discord community to Fluxer.

## What this is

A Go scaffold that connects to **both** [Discord](https://discord.com/developers)
and [Fluxer](https://docs.fluxer.app) as bots, backed by a local **SQLite**
database. Today it proves connectivity: each bot replies `pong` to a `!ping`
message, and the database is opened and migrated on startup. The
`bridge_links` and `seen_messages` tables are seeded for the future
message-bridge work.

- Discord: [`github.com/bwmarrin/discordgo`](https://github.com/bwmarrin/discordgo)
- Fluxer: hand-rolled REST + Gateway client (`internal/fluxer`), using the
  instance discovery document and `Authorization: Bot <token>` scheme.
- SQLite: [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)
  (pure Go, no C compiler required).

## Requirements

- Go 1.27.2 (declared in `go.mod`; `GOTOOLCHAIN=auto` downloads it on first build)
- A Discord bot token, with the **Message Content** intent enabled in the
  Developer Portal.
- A Fluxer bot token from the Fluxer dashboard.

## Setup

```sh
cp .env.example .env
# edit .env and fill in DISCORD_BOT_TOKEN and FLUXER_BOT_TOKEN
go mod tidy
go run .
```

On startup each bot connects; send `!ping` in a channel the bot can read and it
replies `pong`. Press `Ctrl+C` to stop.

## Layout

```
main.go                     wires config, store, and both bots
internal/config/            environment configuration
internal/store/             SQLite open, migrations, and channel-link queries
internal/discord/           Discord bot (discordgo)
internal/fluxer/            Fluxer REST client + gateway client
```

