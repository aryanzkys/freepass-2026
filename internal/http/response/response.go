package response

import "github.com/gin-gonic/gin"

type errorResponse struct {
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

func Error(c *gin.Context, status int, message string, details interface{}) {
	c.AbortWithStatusJSON(status, errorResponse{Message: message, Details: details})
}
