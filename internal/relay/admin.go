package relay

import (
	"encoding/json"
	"net/http"

	"github.com/nikhil-thorat/relay/internal/target"
)

type targetStatus struct {
	ID                  string `json:"id"`
	Address             string `json:"address"`
	Weight              int    `json:"weight"`
	Healthy             bool   `json:"healthy"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	LastChecked         string `json:"last_checked"`
}

func adminTargetsHandler(pool *target.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		list := pool.List()

		out := make([]targetStatus, 0, len(list))
		for _, t := range list {
			state, err := pool.GetState(t.ID)
			if err != nil {
				continue
			}
			out = append(out, targetStatus{
				ID:                  t.ID,
				Address:             t.Address,
				Weight:              t.Weight,
				Healthy:             state.Healthy,
				ConsecutiveFailures: state.ConsecutiveFailures,
				LastChecked:         state.LastChecked.Format("2006-01-02T15:04:05Z07:00"),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}

func healthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func readyzHandler(pool *target.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(pool.Healthy()) == 0 {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
}

func buildHandler(pool *target.Pool, proxy http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/admin/targets", adminTargetsHandler(pool))
	mux.Handle("/healthz", healthzHandler())
	mux.Handle("/readyz", readyzHandler(pool))
	mux.Handle("/", proxy)
	return mux
}
