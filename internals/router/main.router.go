package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	_ "github.com/rivando-al-rasyid/cliq-backend/docs"
	"github.com/rivando-al-rasyid/cliq-backend/internals/middleware"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func MainRouter(router *gin.Engine, db *pgxpool.Pool, rdb *redis.Client) {
	router.Use(middleware.CORSMiddleware)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	router.Static("/img", "public/img")
	AuthRouter(router, db, rdb)
	ProfileRouter(router, db)
	CliqRouter(router, db, rdb)
}
