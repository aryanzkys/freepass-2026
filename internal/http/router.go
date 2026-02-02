package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	h := NewHandler()
	r.GET("/health", h.Health)
}
