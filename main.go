package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"task-management-backend/config"
	"task-management-backend/database"
	"task-management-backend/handlers"
	"task-management-backend/middleware"
	"task-management-backend/models"
	"task-management-backend/routes"
)

func main() {
	cfg := config.Load()

	db := database.NewMySQL(cfg)
	defer db.Close()

	cache := database.NewRedis(cfg)
	defer cache.Close()

	taskRepo := models.NewTaskRepository(db)
	taskHandler := handlers.NewTaskHandler(taskRepo, cache)

	r := gin.Default()
	r.Use(middleware.Recovery())

	routes.Register(r, taskHandler)

	log.Printf("server listening on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
