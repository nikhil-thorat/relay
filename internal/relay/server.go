package relay

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (relay *Relay) Shutdown(ctx context.Context) error {
	relay.lifecycleMu.Lock()
	defer relay.lifecycleMu.Unlock()

	if relay.Logger != nil {
		relay.Logger.Info("shutting down")
	}

	relay.started = false

	if relay.cancel != nil {
		relay.cancel()
	}

	relay.stopHealth()

	var errs []error

	if err := relay.shutdownServer(ctx, relay.server, "http server"); err != nil {
		if relay.Logger != nil {
			relay.Logger.Error("server shutdown error", "error", err)
		}
		errs = append(errs, err)
	}

	if relay.metricsEnabled {
		if err := relay.shutdownServer(ctx, relay.metricsServer, "metrics server"); err != nil {
			if relay.Logger != nil {
				relay.Logger.Error("metrics server shutdown error", "error", err)
			}
			errs = append(errs, err)
		}
	}

	if relay.proxy != nil {
		relay.proxy.Close()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if relay.Logger != nil {
		relay.Logger.Info("relay shutdown complete...")
	}

	return nil
}

func (relay *Relay) shutdownServer(ctx context.Context, server Server, name string) error {
	if server == nil {
		return nil
	}

	timeout := relay.cfg.Server.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("%s shutdown: %w", name, err)
	}

	return nil
}
