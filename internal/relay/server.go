package relay

import "context"

type Server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func (r *Relay) Shutdown(ctx context.Context) error {
	if err := r.server.Shutdown(ctx); err != nil {
		return err
	}

	if r.metricsEnabled {
		if err := r.metricsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
