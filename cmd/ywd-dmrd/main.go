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

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/httpapi"
)

var (
	version = "0.0.0-dev"
	commit  = "unknown"
	branch  = "unknown"
)

func main() {
	listen := envOr("YWD_DMR_LISTEN", config.DefaultListen)
	webRoot := envOr("YWD_DMR_WEB_ROOT", "web")
	docsRoot := envOr("YWD_DMR_DOCS_ROOT", "docs")
	stateDir := envOr("YWD_DMR_STATE_DIR", config.DefaultStateDir)

	state := core.NewState(version, commit, branch)
	store := config.NewFileStore(stateDir)
	if loaded, err := store.Load(); err == nil {
		state.SetKnownGoodConfiguration(loaded.Config.Revision, loaded.RecoveredFromPrevious)
		if loaded.RecoveredFromPrevious {
			log.Printf("WARNING: known-good configuration recovered from previous snapshot (revision %d)", loaded.Config.Revision)
		}
	} else if !errors.Is(err, config.ErrNoKnownGoodConfig) {
		state.SetConfigurationLoadError()
		log.Printf("WARNING: known-good configuration could not be loaded: %v", err)
	}

	handler := httpapi.New(state, webRoot, docsRoot)

	srv := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("YWD-DMR %s starting on http://%s", version, listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
