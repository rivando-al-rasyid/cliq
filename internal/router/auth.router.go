package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/cliq-backend/internals/controller"
	"github.com/rivando-al-rasyid/cliq-backend/internals/middleware"
	"github.com/rivando-al-rasyid/cliq-backend/internals/repository"
	"github.com/rivando-al-rasyid/cliq-backend/internals/service"
)

func AuthRouter(router *gin.Engine, db *pgxpool.Pool, rdb *redis.Client) {
	authRepo := repository.NewAuthRepo(db)
	authServ := service.NewAuthService(authRepo, rdb)
	authCont := controller.NewAuthController(authServ)

	auth := router.Group("/auth")

	// Public — no token required
	auth.POST("/register", authCont.Register)
	auth.POST("/login", authCont.Login)
	auth.POST("/reset", authCont.ResetPassword)
	auth.POST("/reset/confirm", authCont.ConfirmResetPassword)

	// Protected with normal access token
	protected := auth.Group("/", middleware.AuthRequired(db))
	protected.GET("/me", authCont.Me)
	protected.POST("/logout", authCont.Logout)

	// Protected with password-reset JWT (sub="password-reset", 10 min)
	// Client must attach the JWT returned by POST /auth/reset/confirm
	resetProtected := auth.Group("/", middleware.PasswordResetRequired())
	resetProtected.POST("/change-password", authCont.ChangePassword)
}
