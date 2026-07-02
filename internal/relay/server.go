package relay

import (
	"context"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (relay *Relay) Shutdown(ctx context.Context) error {

	relay.Logger.Info("shutting down")

	relay.cancel()
	err := relay.server.Shutdown(ctx)
	if err != nil {
		relay.Logger.Error("server shutdown error", "error", err)
		return err
	}

	if relay.metricsEnabled {
		err = relay.metricsServer.Shutdown(ctx)
		if err != nil {
			relay.Logger.Error("metrics server shutdown error", "error", err)
			return err
		}
	}

	relay.Logger.Info("relay shutdown complete...")

	return nil
}
