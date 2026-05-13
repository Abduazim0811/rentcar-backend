package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"car-rental-system/internal/docs"
	"car-rental-system/internal/handler"
	"car-rental-system/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	UserHandler         *handler.UserHandler
	CarHandler          *handler.CarHandler
	RentalHandler       *handler.RentalHandler
	PaymentHandler      *handler.PaymentHandler
	UploadHandler       *handler.UploadHandler
	InvoiceHandler      *handler.InvoiceHandler
	MaintenanceHandler  *handler.MaintenanceHandler
	NotificationHandler *handler.NotificationHandler
	ReportHandler       *handler.ReportHandler
	AuditHandler        *handler.AuditHandler
	Auth                *middleware.AuthMiddleware
	Permissions         middleware.PermissionChecker
	HealthChecker       HealthChecker
	Logger              *slog.Logger
	MaxBodyBytes        int64
	RateLimit           middleware.RateLimitConfig
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func New(dep Dependencies) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(middleware.BodyLimit(dep.MaxBodyBytes))
	r.Use(middleware.RateLimiter(dep.RateLimit))
	r.Use(middleware.Logger(dep.Logger))
	r.Use(middleware.Recovery(dep.Logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		if dep.HealthChecker == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := dep.HealthChecker.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "database": "down"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "up"})
	})
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", []byte(docs.OpenAPIYAML))
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.DocsHTML))
	})

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", dep.UserHandler.Register)
			auth.POST("/verify-email", dep.UserHandler.VerifyEmail)
			auth.POST("/resend-verification", dep.UserHandler.ResendEmailVerification)
			auth.POST("/login", dep.UserHandler.Login)
			auth.POST("/refresh", dep.UserHandler.Refresh)
			auth.POST("/logout", dep.UserHandler.Logout)
		}

		api.GET("/availability/cars/:id", dep.RentalHandler.Availability)
		api.GET("/availability/cars/:id/calendar", dep.RentalHandler.Calendar)
		api.GET("/cars", dep.CarHandler.List)
		api.GET("/cars/:id", dep.CarHandler.GetByID)
		api.GET("/uploads/images/:object", dep.UploadHandler.GetImage)
		api.HEAD("/uploads/images/:object", dep.UploadHandler.GetImage)

		protected := api.Group("")
		protected.Use(dep.Auth.RequireAuth())
		{
			users := protected.Group("/users")
			{
				users.GET("/me", dep.UserHandler.Me)
				users.PATCH("/me", dep.UserHandler.UpdateProfile)
				users.POST("/logout-all", dep.UserHandler.LogoutAll)
			}

			notifications := protected.Group("/notifications")
			{
				notifications.GET("", dep.NotificationHandler.List)
				notifications.POST("/:id/read", dep.NotificationHandler.MarkRead)
			}

			rentals := protected.Group("/rentals")
			{
				rentals.POST("", dep.RentalHandler.Create)
				rentals.GET("/me", dep.RentalHandler.MyRentals)
				rentals.GET("/:id", dep.RentalHandler.GetByID)
				rentals.GET("/:id/invoice", dep.InvoiceHandler.GetByRental)
				rentals.POST("/:id/cancel", dep.RentalHandler.Cancel)
			}

			payments := protected.Group("/payments")
			{
				payments.POST("", dep.PaymentHandler.Create)
			}

			admin := protected.Group("/admin")
			admin.Use(middleware.RequirePermission(dep.Permissions, "admin:access"))
			{
				admin.POST("/uploads/images", middleware.RequirePermission(dep.Permissions, "cars:manage"), dep.UploadHandler.UploadImage)

				admin.POST("/cars", middleware.RequirePermission(dep.Permissions, "cars:manage"), dep.CarHandler.Create)
				admin.PUT("/cars/:id", middleware.RequirePermission(dep.Permissions, "cars:manage"), dep.CarHandler.Update)
				admin.DELETE("/cars/:id", middleware.RequirePermission(dep.Permissions, "cars:manage"), dep.CarHandler.Delete)

				admin.GET("/rentals", middleware.RequirePermission(dep.Permissions, "rentals:manage"), dep.RentalHandler.ListAll)
				admin.POST("/rentals/:id/approve", middleware.RequirePermission(dep.Permissions, "rentals:manage"), dep.RentalHandler.Approve)
				admin.POST("/rentals/:id/reject", middleware.RequirePermission(dep.Permissions, "rentals:manage"), dep.RentalHandler.Reject)
				admin.PATCH("/rentals/:id/status", middleware.RequirePermission(dep.Permissions, "rentals:manage"), dep.RentalHandler.UpdateStatus)

				admin.POST("/payments/:id/confirm", middleware.RequirePermission(dep.Permissions, "payments:manage"), dep.PaymentHandler.Confirm)
				admin.POST("/payments/:id/fail", middleware.RequirePermission(dep.Permissions, "payments:manage"), dep.PaymentHandler.Fail)
				admin.POST("/payments/:id/refund", middleware.RequirePermission(dep.Permissions, "payments:manage"), dep.PaymentHandler.Refund)

				admin.GET("/maintenance", middleware.RequirePermission(dep.Permissions, "maintenance:manage"), dep.MaintenanceHandler.List)
				admin.POST("/maintenance", middleware.RequirePermission(dep.Permissions, "maintenance:manage"), dep.MaintenanceHandler.Create)
				admin.PUT("/maintenance/:id", middleware.RequirePermission(dep.Permissions, "maintenance:manage"), dep.MaintenanceHandler.Update)
				admin.DELETE("/maintenance/:id", middleware.RequirePermission(dep.Permissions, "maintenance:manage"), dep.MaintenanceHandler.Delete)

				admin.GET("/reports/summary", middleware.RequirePermission(dep.Permissions, "reports:view"), dep.ReportHandler.Summary)
				admin.GET("/audit-logs", middleware.RequirePermission(dep.Permissions, "audit:view"), dep.AuditHandler.List)

				admin.GET("/users", middleware.RequirePermission(dep.Permissions, "users:view"), dep.UserHandler.List)
				admin.PATCH("/users/:id/role", middleware.RequirePermission(dep.Permissions, "users:manage"), dep.UserHandler.UpdateRole)
			}
		}
	}

	return r
}
