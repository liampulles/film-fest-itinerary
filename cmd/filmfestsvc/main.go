package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Setup zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Delegate to run func.
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("application failed")
	}
}

func run() error {
	// Create a context that is canceled when an interrupt signal is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create an errgroup to manage goroutines and their shared lifecycle.
	g, ctx := errgroup.WithContext(ctx)

	// Setup and start the HTTP server.
	if err := serve(ctx, g); err != nil {
		return err
	}

	// Wait for all goroutines in the group to finish.
	if err := g.Wait(); err != nil {
		return err
	}

	log.Info().Msg("server gracefully stopped")
	return nil
}

func serve(ctx context.Context, g *errgroup.Group) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Create a listener so we can return an error if the server cannot start (e.g. port in use).
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", srv.Addr, err)
	}

	// Start the HTTP server in the workgroup.
	g.Go(func() error {
		log.Info().Msgf("starting server on %s", srv.Addr)
		// Serve blocks until the server is closed or an error occurs.
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server Serve: %w", err)
		}
		return nil
	})

	// Handle graceful shutdown in the workgroup.
	g.Go(func() error {
		// Wait for the context to be canceled (via signal or error in another goroutine).
		<-ctx.Done()
		log.Info().Msg("shutting down server")

		// Use a separate context with a timeout for the shutdown process.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		return nil
	})

	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
