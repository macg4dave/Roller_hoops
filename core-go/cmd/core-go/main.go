package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"roller_hoops/core-go/internal/db"
	"roller_hoops/core-go/internal/discoveryworker"
	"roller_hoops/core-go/internal/httpapi"
)

func main() {
	addr := envOr("HTTP_ADDR", ":8081")
	logLevel := envOr("LOG_LEVEL", "info")
	databaseURL := envOr("DATABASE_URL", "")

	logger := httpapi.NewLogger(logLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var pool *db.Pool
	if databaseURL != "" {
		p, err := db.Open(ctx, databaseURL)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer p.Close()
		pool = p
	}

	if pool != nil {
		worker := discoveryworker.New(logger, pool.Queries(), discoveryworker.Options{})
		go worker.Run(ctx)
	}

	h := httpapi.NewHandler(logger, pool)
	srv := &http.Server{
		Addr:              addr,
		Handler:           h.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", addr).Msg("core-go listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("http server error")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	logger.Info().Msg("shutdown complete")
}

func envOr(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
