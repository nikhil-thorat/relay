package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/relay"
	"github.com/prometheus/client_golang/prometheus"
)

func run() error {
	configPath := flag.String(
		"config",
		"relay.yml",
		"path to Relay configuration file.",
	)

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", *configPath, err)
	}

	registry := prometheus.NewRegistry()

	relay, err := relay.New(cfg, registry)
	if err != nil {
		return fmt.Errorf("create relay: %w", err)
	}

	log.Printf("loaded config %s, listening on %s, metrics %s", *configPath, cfg.Server.Address, cfg.Metrics.Address)

	go func() {
		err := relay.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Relay stopped : %v", err)
		}
	}()

	hupCh := make(chan os.Signal, 1)
	signal.Notify(hupCh, syscall.SIGHUP)
	defer signal.Stop(hupCh)

	go func() {
		for range hupCh {
			log.Printf("reloading config %s", *configPath)
			if err := relay.Reload(*configPath); err != nil {
				log.Printf("reload failed: %v", err)
			}
		}
	}()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.Server.ShutdownTimeout,
	)
	defer cancel()

	return relay.Shutdown(shutdownCtx)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
