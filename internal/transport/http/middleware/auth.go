package middleware

import (
	"net/http"

	authmodule "github.com/cnxianyi/xy_wealth/internal/modules/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func XToken(service *authmodule.Service, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		valid, err := service.Validate(c.Request.Context(), c.GetHeader("x-token"))
		if err != nil {
			log.Error("validate x-token failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{"code": "authentication_unavailable", "message": "authentication service is unavailable"},
			})
			return
		}
		if !valid {
			c.Header("WWW-Authenticate", "X-Token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "unauthorized", "message": "a valid x-token header is required"},
			})
			return
		}
		c.Next()
	}
}
