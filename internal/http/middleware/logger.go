package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		requestID := c.GetString("request_id")
		log.Printf("method=%s path=%s status=%d latency=%s request_id=%s", method, path, status, latency, requestID)
	}
}
