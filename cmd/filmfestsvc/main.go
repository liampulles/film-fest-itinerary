package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Setup zerolog
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Delegate to run func.
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("application failed")
	}
}

func run() error {
	// Setup context and errgroup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	g, ctx := errgroup.WithContext(ctx)

	// Setup HTTP server
	serve(ctx, g)

	// Wait until all goroutines on the errgroup close
	// (which will happen if they all fail to start, or
	// the context is closed).
	if err := g.Wait(); err != nil {
		return err
	}
	log.Info().Msg("server gracefully stopped")
	return nil
}

func serve(ctx context.Context, g *errgroup.Group) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the HTTP server in the workgroup.
	g.Go(func() error {
		log.Info().Msgf("starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server ListenAndServe: %w", err)
		}
		return nil
	})

	// Handle graceful shutdown in the workgroup.
	g.Go(func() error {
		// Wait for the context to be closed
		<-ctx.Done()
		log.Info().Msg("shutting down server")

		// Give a 25 second deadline for active connections
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		// Shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		return nil
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
