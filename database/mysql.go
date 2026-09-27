package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"task-management-backend/config"
)

// NewMySQL opens a connection pool to MySQL and verifies it with a ping.
func NewMySQL(cfg *config.Config) *sql.DB {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open mysql connection: %v", err)
	}

	// Reasonable pool defaults for a small service.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	log.Println("connected to mysql")
	return db
}
