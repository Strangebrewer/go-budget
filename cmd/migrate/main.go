package main

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Strangebrewer/go-budget/config"
)

func main() {
	if len(os.Args) < 2 {
		slog.Error("usage: migrate [up|down]")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	dbURL := strings.NewReplacer("postgres://", "pgx5://", "postgresql://", "pgx5://").Replace(cfg.DatabaseURL)

	m, err := migrate.New("file://db/migrations", dbURL)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch os.Args[1] {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")
	case "down":
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migration rolled back one step")
	case "force":
		if len(os.Args) < 3 {
			slog.Error("usage: migrate force <version>")
			os.Exit(1)
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			slog.Error("invalid version", "version", os.Args[2])
			os.Exit(1)
		}
		if err := m.Force(v); err != nil {
			slog.Error("migration force failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migration forced", "version", v)
	default:
		slog.Error("unknown command, use up, down, or force", "command", os.Args[1])
		os.Exit(1)
	}
}
