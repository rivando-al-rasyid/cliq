package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/rivando-al-rasyid/cliq-backend/internals/config"
	"github.com/rivando-al-rasyid/cliq-backend/internals/router"
)

// @title						Cliq
// @version						1.0
// @description					Backend Cliq  using Gin

// @license.name				MIT

// @host						localhost:8080
// @BasePath					/

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and your JWT. Example: "Bearer eyJhbGci..."
func main() {
	app := gin.Default()
	// connect ke db
	db, err := config.ConnectPsql()
	if err != nil {
		log.Fatalf("DB connection error. \ncause: %s", err.Error())
	}
	defer db.Close()
	log.Println("DB Connected")
	// connect ke redis
	rc, err := config.ConnectRedis()
	if err != nil {
		log.Fatalf("Redis connection error. \ncause: %s", err.Error())
	}
	defer rc.Close()
	log.Println("Redis Connected")
	// install router
	router.MainRouter(app, db, rc)

	if err := app.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("Failed to start server %v", err)
	}

}
