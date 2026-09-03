package handler

import (
	"errors"
	"net/http"

	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const maxLoginBodyBytes = 4 << 10

type Auth struct {
	service *authmodule.Service
	log     *zap.Logger
}

func NewAuth(service *authmodule.Service, log *zap.Logger) *Auth {
	return &Auth{service: service, log: log}
}

func (h *Auth) Login(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLoginBodyBytes)
	var request struct {
		Secret string `json:"secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": "secret is required"}})
		return
	}
	session, err := h.service.Login(c.Request.Context(), request.Secret)
	if errors.Is(err, authmodule.ErrInvalidSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "invalid_credentials", "message": "secret is invalid"}})
		return
	}
	if err != nil {
		h.log.Error("create authentication session failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "authentication_unavailable", "message": "authentication service is unavailable"}})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *Auth) Session(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"authenticated": true})
}

func (h *Auth) Logout(c *gin.Context) {
	if err := h.service.Logout(c.Request.Context(), c.GetHeader("x-token")); err != nil {
		h.log.Error("delete authentication session failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "authentication_unavailable", "message": "authentication service is unavailable"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
