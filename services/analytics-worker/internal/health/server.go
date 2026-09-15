package health

import (
	"log"
	"net/http"
)

// Serve starts a minimal HTTP server exposing /health for the k8s liveness
// probe, backed by the tracker's view of whether recent click events are
// actually being persisted.
func Serve(addr string, tracker *Tracker) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !tracker.Healthy() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("health: server stopped: %v", err)
		}
	}()
}
