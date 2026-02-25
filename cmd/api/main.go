package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tomimandalaputra/e-commerce-go/internal/config"
	"github.com/tomimandalaputra/e-commerce-go/internal/database"
	"github.com/tomimandalaputra/e-commerce-go/internal/logger"
	"github.com/tomimandalaputra/e-commerce-go/internal/server"
	"github.com/tomimandalaputra/e-commerce-go/internal/services"
)

func main() {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	mainDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get database connection")
	}

	defer func() {
		if err := mainDB.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}
	}()

	gin.SetMode(cfg.Server.GinMode)

	authService := services.NewAuthService(db, cfg)
	productService := services.NewProductService(db)
	userService := services.NewUserService(db)

	srv := server.New(cfg, db, &log, authService, productService, userService)
	router := srv.SetupRoutes()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Starting http server")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start http server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown http server")
	}

	log.Info().Msg("Shutting down database")
}
