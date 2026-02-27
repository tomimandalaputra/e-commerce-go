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
	"github.com/tomimandalaputra/e-commerce-go/internal/interfaces"
	"github.com/tomimandalaputra/e-commerce-go/internal/logger"
	"github.com/tomimandalaputra/e-commerce-go/internal/providers"
	"github.com/tomimandalaputra/e-commerce-go/internal/server"
	"github.com/tomimandalaputra/e-commerce-go/internal/services"
)

// @title E-Commerce API
// @version 1.0
// @description A modern e-commerce API built with Go, Gin, and GORM
// @termsOfService http://swagger.io/terms/

// @contact.name   Tomi Mandala Putra
// @contact.url   https://www.linkedin.com/in/tomimandalaputra
// @contact.email  no-email@no-email

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemas http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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
	cartService := services.NewCartService(db)
	orderService := services.NewOrderService(db)

	var uploadProvider interfaces.UploadProvider
	if cfg.Upload.UploadProvider == "s3" {
		uploadProvider = providers.NewS3Provider(cfg)
	} else {
		uploadProvider = providers.NewLocalUploadProvider(cfg.Upload.Path)
	}

	uploadService := services.NewUploadService(uploadProvider)

	srv := server.New(
		cfg,
		db,
		&log,
		authService,
		productService,
		userService,
		uploadService,
		cartService,
		orderService,
	)

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
