package relay

import (
	"context"
	"log"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (relay *Relay) Shutdown(ctx context.Context) error {
	log.Println("Relay shutting down...")

	relay.cancel()
	err := relay.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	if relay.metricsEnabled {
		err = relay.metricsServer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	log.Println("Relay shutdown complete...")

	return nil
}
