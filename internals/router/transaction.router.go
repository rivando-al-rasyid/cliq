package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/controller"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/middleware"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/repository"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/service"
)

func TransactionRouter(router *gin.Engine, db *pgxpool.Pool, rdb *redis.Client) {
	txRepo := repository.NewTransactionRepo(db)
	txServ := service.NewTransactionService(txRepo, rdb)
	txCont := controller.NewTransactionController(txServ)

	g := router.Group("/transaction", middleware.VerifyTokenWithDB(db))

	// Read-only
	g.GET("/receiver", txCont.FindReceivers)
	g.GET("/summary", txCont.GetSummary)
	g.GET("/report", txCont.GetTransactionReport)
	g.GET("/history", txCont.GetHistory) // unified paginated history feed

	// Mutations — all require PIN in the request body
	g.POST("/topup", txCont.CreateTopup)
	g.PATCH("/topup/:id/confirm", txCont.ConfirmTopup)
	g.POST("/withdrawal", txCont.CreateWithdrawal)
	g.POST("/transfer", txCont.CreateTransfer)
	g.POST("/expense", txCont.CreateExpense)
}
