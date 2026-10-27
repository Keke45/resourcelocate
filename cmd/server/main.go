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

	// TRUST_PROXY=1 when running behind a proxy that sets X-Forwarded-For
	// (Render, Fly.io), so rate limits apply per visitor, not per proxy.
	srvAPI := api.New(st, os.Getenv("TRUST_PROXY") == "1")
	// DEMO_LOGINS=1 adds one-click demo giver and seeker logins, for judges.
	if os.Getenv("DEMO_LOGINS") == "1" {
		if err := srvAPI.EnableDemoLogins(context.Background()); err != nil {
			log.Fatalf("demo logins: %v", err)
		}
		log.Print("demo logins are on")
	}

	mux := http.NewServeMux()
	srvAPI.Routes(mux)
	// A listing's share link: the app, with link-preview tags for that listing.
	mux.Handle("GET /l/{id}", srvAPI.SharePage(webui.Index()))
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
