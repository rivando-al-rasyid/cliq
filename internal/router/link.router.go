package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq-backend/internal/controller"
	"github.com/rivando-al-rasyid/cliq-backend/internal/middleware"
	"github.com/rivando-al-rasyid/cliq-backend/internal/repository"
	"github.com/rivando-al-rasyid/cliq-backend/internal/service"
)

func LinkRouter(router *gin.Engine, db *pgxpool.Pool, rdb *redis.Client) {
	linkRepo := repository.NewLinkRepo(db)
	linkServ := service.NewLinkService(linkRepo, rdb)
	linkServ.StartClickFlushWorker(context.Background(), 10*time.Minute)
	linkCont := controller.NewLinkController(linkServ)

	cliq := router.Group("/link", middleware.AuthRequired(db))
	cliq.POST("/create", linkCont.CreateSlug)
	cliq.GET("/dashboard", linkCont.GetDashboard)
	cliq.DELETE("/:id", linkCont.DeleteLink)

	// Keep this at the end so system routes like /auth, /profile, /link, and
	router.GET("/:slug", linkCont.RedirectBySlug)
}
