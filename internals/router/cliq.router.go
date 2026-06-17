package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq/internals/controller"
	"github.com/rivando-al-rasyid/cliq/internals/middleware"
	"github.com/rivando-al-rasyid/cliq/internals/repository"
	"github.com/rivando-al-rasyid/cliq/internals/service"
)

func CliqRouter(router *gin.Engine, db *pgxpool.Pool, rdb *redis.Client) {
	linkRepo := repository.NewCliqRepo(db)
	linkServ := service.NewCliqService(linkRepo, rdb)
	linkCont := controller.NewCliqController(linkServ)
	router.GET("/:slug", linkCont.RedirectBySlug)
	cliq := router.Group("/link", middleware.VerifyTokenWithDB(db))
	cliq.POST("create", linkCont.CreateSlug)

}
