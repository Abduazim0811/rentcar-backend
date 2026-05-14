package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"car-rental-system/internal/cache"
	"car-rental-system/internal/config"
	"car-rental-system/internal/database"
	"car-rental-system/internal/handler"
	"car-rental-system/internal/middleware"
	"car-rental-system/internal/repository"
	"car-rental-system/internal/router"
	"car-rental-system/internal/service"

	"github.com/gin-gonic/gin"
)

func Run(logger *slog.Logger) error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	if cfg.Env == "production" {
		ginSetReleaseMode()
	}

	db, err := database.Connect(context.Background(), database.Config{
		URL:      cfg.DatabaseURL,
		MaxConns: cfg.DBMaxConns,
		MinConns: cfg.DBMinConns,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	userRepo := repository.NewUserPostgresRepository(db)
	authTokenRepo := repository.NewAuthTokenPostgresRepository(db)
	carRepo := repository.NewCarPostgresRepository(db)
	rentalRepo := repository.NewRentalPostgresRepository(db)
	paymentRepo := repository.NewPaymentPostgresRepository(db)
	maintenanceRepo := repository.NewMaintenancePostgresRepository(db)
	invoiceRepo := repository.NewInvoicePostgresRepository(db)
	notificationRepo := repository.NewNotificationPostgresRepository(db)
	reportRepo := repository.NewReportPostgresRepository(db)
	auditRepo := repository.NewAuditPostgresRepository(db)
	permissionRepo := repository.NewPermissionPostgresRepository(db)
	emailSender := service.NewEmailSender(service.EmailSenderConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		FromName: cfg.SMTPFromName,
		UseTLS:   cfg.SMTPUseTLS,
	}, logger)

	userService := service.NewUserService(userRepo, authTokenRepo, emailSender, cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.EmailVerifyTTL)
	carService := service.NewCarService(carRepo)
	maintenanceService := service.NewMaintenanceService(maintenanceRepo, carRepo)
	rentalService := service.NewRentalService(rentalRepo, carRepo, maintenanceRepo)
	invoiceService := service.NewInvoiceService(invoiceRepo, rentalRepo)
	paymentService := service.NewPaymentService(paymentRepo, rentalRepo)
	notificationService := service.NewNotificationService(notificationRepo)
	reportService := service.NewReportService(reportRepo)
	auditService := service.NewAuditService(auditRepo)
	imageStorage, err := service.NewImageStorage(context.Background(), service.ImageStorageConfig{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
		UseSSL:    cfg.MinIOUseSSL,
		PublicURL: cfg.ImagePublicURL,
	})
	if err != nil {
		return err
	}

	userHandler := handler.NewUserHandler(userService, auditService)
	carHandler := handler.NewCarHandler(carService, auditService)
	rentalHandler := handler.NewRentalHandler(rentalService, notificationService, auditService)
	paymentHandler := handler.NewPaymentHandler(paymentService, invoiceService, auditService)
	uploadHandler := handler.NewUploadHandler(imageStorage)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	maintenanceHandler := handler.NewMaintenanceHandler(maintenanceService, auditService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	reportHandler := handler.NewReportHandler(reportService)
	auditHandler := handler.NewAuditHandler(auditService)

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret)
	rateStore, err := cache.NewRedisRateLimitStore(cfg.RedisURL, 500*time.Millisecond)
	if err != nil {
		logger.Warn("redis_rate_limiter_disabled", slog.String("error", err.Error()))
	}

	lifecycleCtx, stopLifecycle := context.WithCancel(context.Background())
	defer stopLifecycle()
	go runRentalLifecycleWorker(lifecycleCtx, logger, rentalService)

	r := router.New(router.Dependencies{
		UserHandler:         userHandler,
		CarHandler:          carHandler,
		RentalHandler:       rentalHandler,
		PaymentHandler:      paymentHandler,
		UploadHandler:       uploadHandler,
		InvoiceHandler:      invoiceHandler,
		MaintenanceHandler:  maintenanceHandler,
		NotificationHandler: notificationHandler,
		ReportHandler:       reportHandler,
		AuditHandler:        auditHandler,
		Auth:                authMiddleware,
		Permissions:         permissionRepo,
		HealthChecker:       db,
		Logger:              logger,
		MaxBodyBytes:        cfg.MaxBodyBytes,
		TrustedProxies:      cfg.TrustedProxies,
		RateLimit: middleware.RateLimitConfig{
			MaxRequests: cfg.RateLimitMax,
			Window:      cfg.RateLimitWindow,
			Store:       rateStore,
		},
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server_started", slog.String("addr", server.Addr), slog.String("env", cfg.Env))
		errCh <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case sig := <-stop:
		logger.Info("shutdown_signal_received", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(ctx)
}

func ginSetReleaseMode() {
	// Kept behind a function so app startup remains framework-light outside router wiring.
	// Gin reads this global before engine creation.
	gin.SetMode(gin.ReleaseMode)
}

func runRentalLifecycleWorker(ctx context.Context, logger *slog.Logger, rentals *service.RentalService) {
	sync := func() {
		syncCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		result, err := rentals.SyncLifecycle(syncCtx, time.Now())
		if err != nil {
			logger.Error("rental_lifecycle_sync_failed", slog.String("error", err.Error()))
			return
		}
		if result.Cancelled > 0 || result.Activated > 0 || result.Completed > 0 || result.FailedPayments > 0 || result.VoidedInvoices > 0 {
			logger.Info(
				"rental_lifecycle_synced",
				slog.Int64("cancelled", result.Cancelled),
				slog.Int64("activated", result.Activated),
				slog.Int64("completed", result.Completed),
				slog.Int64("failed_payments", result.FailedPayments),
				slog.Int64("voided_invoices", result.VoidedInvoices),
			)
		}
	}

	sync()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sync()
		}
	}
}
