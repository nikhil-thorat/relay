package relay

import (
	"context"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (relay *Relay) Shutdown(ctx context.Context) error {

	if relay.Logger != nil {
		relay.Logger.Info("shutting down")
	}

	if relay.cancel != nil {
		relay.cancel()
	}

	if relay.server != nil {
		err := relay.server.Shutdown(ctx)
		if err != nil {
			if relay.Logger != nil {
				relay.Logger.Error("server shutdown error", "error", err)
			}
			return err
		}
	}

	if relay.metricsEnabled && relay.metricsServer != nil {
		err := relay.metricsServer.Shutdown(ctx)
		if err != nil {
			if relay.Logger != nil {
				relay.Logger.Error("metrics server shutdown error", "error", err)
			}
			return err
		}
	}

	if relay.Logger != nil {
		relay.Logger.Info("relay shutdown complete...")
	}

	return nil
}
