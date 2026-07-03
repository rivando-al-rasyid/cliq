package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/cliq-backend/internals/controller"
	"github.com/rivando-al-rasyid/cliq-backend/internals/middleware"
	"github.com/rivando-al-rasyid/cliq-backend/internals/repository"
	"github.com/rivando-al-rasyid/cliq-backend/internals/service"
)

func ProfileRouter(router *gin.Engine, db *pgxpool.Pool) {
	profRepo := repository.NewProfileRepo(db)
	profServ := service.NewProfileService(profRepo)
	profCont := controller.NewProfileController(profServ)

	profileRouter := router.Group("/profile", middleware.AuthRequired(db))

	// Header info — lightweight, called on every page load
	profileRouter.GET("/info", profCont.GetUserInfo)

	// Full profile CRUD
	profileRouter.GET("/", profCont.GetProfile)
	profileRouter.PATCH("/edit", profCont.EditProfile)
	profileRouter.PATCH("/change/password", profCont.EditPassword)
}
