package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"autodataops.local/control-plane/internal/database"
	"autodataops.local/control-plane/internal/httpapi"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)

	if err := run(logger); err != nil {
		logger.Error(
			"control plane terminated",
			"error", err,
		)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	address := strings.TrimSpace(
		os.Getenv("AUTODATAOPS_HTTP_ADDR"),
	)

	if address == "" {
		address = "127.0.0.1:8080"
	}

	pool, err := database.Open(
		context.Background(),
		os.Getenv("AUTODATAOPS_DATABASE_URL"),
	)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer pool.Close()

	server := &http.Server{
		Addr:              address,
		Handler:           httpapi.NewHandler(pool.Ping),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("start listener: %w", err)
	}
	defer listener.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- server.Serve(listener)
	}()

	logger.Info(
		"control plane started",
		"address", listener.Addr().String(),
		"service", "autodataops-control-plane",
	)

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")

	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("http server failure: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("control plane stopped successfully")

	return nil
}
