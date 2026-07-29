package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redisClient,
	}
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// Health - Shallow health check (rapide)
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status: "ok",
		Services: map[string]string{
			"api": "up",
		},
	})
}

// HealthDeep - Deep health check (vérifie DB et Redis)
func (h *HealthHandler) HealthDeep(c *fiber.Ctx) error {
	services := make(map[string]string)
	status := "ok"

	// Check PostgreSQL
	sqlDB, err := h.db.DB()
	if err != nil {
		services["postgres"] = "down"
		status = "degraded"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			services["postgres"] = "down"
			status = "degraded"
		} else {
			services["postgres"] = "up"
		}
	}

	// Check Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "down"
		status = "degraded"
	} else {
		services["redis"] = "up"
	}

	services["api"] = "up"

	httpStatus := fiber.StatusOK
	if status == "degraded" {
		httpStatus = fiber.StatusServiceUnavailable
	}

	return c.Status(httpStatus).JSON(HealthResponse{
		Status:   status,
		Services: services,
	})
}
