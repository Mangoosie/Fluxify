package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/tobia/Fluxify/internal/config"
	fluxerdiscord "github.com/tobia/Fluxify/internal/discord"
	"github.com/tobia/Fluxify/internal/fluxer"
	"github.com/tobia/Fluxify/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	log.Printf("store: %s ready", cfg.DBPath)

	discordBot, err := fluxerdiscord.New(cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord: %v", err)
	}
	fluxerBot := fluxer.New(cfg.FluxerToken, cfg.FluxerOrigin)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("discord: starting gateway")
		errCh <- discordBot.Run(ctx, func(m *discordgo.MessageCreate) {
			handleDiscordMessage(ctx, discordBot, m)
		})
	}()

	go func() {
		defer wg.Done()
		log.Printf("fluxer: starting gateway (%s)", cfg.FluxerOrigin)
		errCh <- fluxerBot.Run(ctx, func(m fluxer.Message) {
			handleFluxerMessage(ctx, fluxerBot, m)
		})
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Printf("bot stopped: %v", err)
		}
		stop()
	}

	wg.Wait()
	log.Printf("stopped")
}

func handleDiscordMessage(ctx context.Context, bot *fluxerdiscord.Bot, m *discordgo.MessageCreate) {
	if m.Content != "!ping" {
		return
	}
	if err := bot.SendMessage(ctx, m.ChannelID, "pong"); err != nil {
		log.Printf("discord: reply failed: %v", err)
	}
}

func handleFluxerMessage(ctx context.Context, bot *fluxer.Bot, m fluxer.Message) {
	if m.Content != "!ping" {
		return
	}
	if err := bot.SendMessage(ctx, m.ChannelID, "pong"); err != nil {
		log.Printf("fluxer: reply failed: %v", err)
	}
}
