package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Strangebrewer/go-budget/account"
	"github.com/Strangebrewer/go-budget/app"
	"github.com/Strangebrewer/go-budget/bill"
	"github.com/Strangebrewer/go-budget/category"
	"github.com/Strangebrewer/go-budget/config"
	"github.com/Strangebrewer/go-budget/db_connection"
	"github.com/Strangebrewer/go-budget/middleware"
	"github.com/Strangebrewer/go-budget/server"
	"github.com/Strangebrewer/go-budget/tracer"
	"github.com/Strangebrewer/go-budget/transaction"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx := context.Background()
	client, db, err := db_connection.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			slog.Error("failed to disconnect from database", "error", err)
		}
	}()

	authMiddleware, err := middleware.RequireAuth(cfg.JWTPublicKey)
	if err != nil {
		slog.Error("failed to parse JWT public key", "error", err)
		os.Exit(1)
	}

	var tracerClient *tracer.Client
	if cfg.TracerURL != "" && cfg.TracerServiceKey != "" {
		tracerClient = tracer.NewClient(cfg.TracerURL, cfg.TracerServiceKey, "go-budget")
	}

	application := &app.Application{
		AccountStore:     account.NewStore(db),
		BillStore:        bill.NewStore(db),
		CategoryStore:    category.NewStore(db),
		TransactionStore: transaction.NewStore(db),
		Tracer:           tracerClient,
		RubeOwidNextURL:  cfg.RubeOwidNextURL,
	}

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	srv := server.New(":"+port, cfg.AllowedOrigins, application, authMiddleware)

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.HTTPServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
