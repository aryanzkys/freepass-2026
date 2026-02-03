package context

import "github.com/gin-gonic/gin"

func UserID(c *gin.Context) (string, bool) {
	value, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	id, ok := value.(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

func UserRole(c *gin.Context) (string, bool) {
	value, ok := c.Get("user_role")
	if !ok {
		return "", false
	}
	role, ok := value.(string)
	if !ok || role == "" {
		return "", false
	}
	return role, true
}
