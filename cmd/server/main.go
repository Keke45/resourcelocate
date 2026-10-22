// Command server runs the ResourceLocate API and serves the web app.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KeKe45/resourcelocate/internal/api"
	"github.com/KeKe45/resourcelocate/internal/store"
	"github.com/KeKe45/resourcelocate/internal/vision"
	"github.com/KeKe45/resourcelocate/internal/webui"
)

// env returns the environment variable key, or fallback when it is unset.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := ":" + env("PORT", "8080")
	dbPath := env("DB_PATH", "resource.db")

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer st.Close()

	// Photo reading is optional: without a key the server still runs and the
	// Give screen falls back to the manual form.
	var reader *vision.Reader
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		reader = vision.New(key, env("CLAUDE_MODEL", "claude-opus-5-5"))
	} else {
		log.Print("ANTHROPIC_API_KEY not set: photo-to-listing is off, the manual form still works")
	}

	mux := http.NewServeMux()
	api.New(st, reader).Routes(mux)
	mux.Handle("/", webui.Handler())

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("ResourceLocate listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// Finish in-flight requests before exiting on Ctrl+C or a platform stop.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
