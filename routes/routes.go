package routes

import (
	"github.com/gin-gonic/gin"

	"task-management-backend/handlers"
	"task-management-backend/middleware"
)

func Register(r *gin.Engine, taskHandler *handlers.TaskHandler) {
	r.NoRoute(middleware.NotFound())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		tasks := api.Group("/tasks")
		{
			tasks.GET("", taskHandler.List)
			tasks.GET("/:id", taskHandler.Get)
			tasks.POST("", taskHandler.Create)
			tasks.PUT("/:id", taskHandler.Update)
			tasks.DELETE("/:id", taskHandler.Delete)
		}
	}
}
