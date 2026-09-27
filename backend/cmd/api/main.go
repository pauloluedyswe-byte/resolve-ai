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

	"resolveai/internal/config"
	"resolveai/internal/database"
	"resolveai/internal/httpapi"
	"resolveai/internal/repository"
	"resolveai/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("falha ao iniciar", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	files, err := repository.NewLocalFileStore(cfg.UploadDir)
	if err != nil {
		return err
	}
	users := repository.NewUserRepo(pool)
	authSvc := service.NewAuthService(users, cfg.JWTSecret, cfg.JWTTTL)
	occSvc := service.NewOccurrenceService(
		repository.NewOccurrenceRepo(pool), repository.NewCategoryRepo(pool), users, files)

	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := authSvc.EnsureGestor(ctx, cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(authSvc, occSvc, files.Dir(), cfg.CORSOrigins).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("API Resolve Aí ouvindo", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
