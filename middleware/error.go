package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery catches panics in any handler and returns the same consistent
// error JSON shape used everywhere else, instead of Gin's default plain-text
// 500 or a crashed process.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("panic recovered: %v", recovered)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "an unexpected error occurred",
			},
		})
		c.Abort()
	})
}

// NotFound handles requests to routes that don't exist at all.
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "resource not found",
			},
		})
	}
}
