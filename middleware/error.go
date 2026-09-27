package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)


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
