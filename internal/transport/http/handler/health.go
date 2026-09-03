package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Health struct {
	postgres *sql.DB
	redis    *redis.Client
}

func NewHealth(postgres *sql.DB, redisClient *redis.Client) *Health {
	return &Health{postgres: postgres, redis: redisClient}
}

func (h *Health) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Health) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{"postgres": "ok", "redis": "ok"}
	ready := true
	if err := h.postgres.PingContext(ctx); err != nil {
		checks["postgres"] = "error"
		ready = false
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "error"
		ready = false
	}

	status := http.StatusOK
	state := "ok"
	if !ready {
		status = http.StatusServiceUnavailable
		state = "not_ready"
	}
	c.JSON(status, gin.H{"status": state, "checks": checks})
}
