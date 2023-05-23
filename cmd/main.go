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

	"github.com/begenov/student-servcie/internal/config"
	"github.com/begenov/student-servcie/internal/handlers"
	"github.com/begenov/student-servcie/internal/server"
	"github.com/begenov/student-servcie/internal/services"
	"github.com/begenov/student-servcie/internal/storage"
	"github.com/begenov/student-servcie/pkg/auth"
	"github.com/begenov/student-servcie/pkg/postgresql"
)

const (
	path_config = "./.env"
)

func main() {
	cfg, err := config.NewConfig(path_config)
	if err != nil {
		log.Fatalf("can't load config: %v", err)
		return
	}

	db, err := postgresql.NewPostgreSQLDB(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("error creating database object: %v", err)
		return
	}

	tokenManager, err := auth.NewManager(cfg.JWT.SigningKey)

	if err != nil {
		log.Printf("Error while creating token manager: %v", err)
		return
	}

	storage := storage.NewStorage(db)

	services := services.NewService(storage, tokenManager, cfg)

	handlers := handlers.NewHandler(services, tokenManager)

	srv := server.NewServer(cfg, handlers.Init(cfg))

	go func() {
		if err := srv.Run(); errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server closed with error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit
	const timeout = 5 * time.Second

	ctx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("error stopping HTTP server: %v", err)
	}

}