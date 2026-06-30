package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		return err
	}

	registry := prometheus.NewRegistry()

	relay, err := relay.New(cfg, registry)
	if err != nil {
		return err
	}

	go func() {
		err := relay.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Relay stopped : %v", err)
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
		time.Second*5,
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
