package controller

import (
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/service"
)

type QliqController struct {
	QliqService *service.QliqService
}

func NewQliqController(QliqService *service.QliqService) *QliqController {
	return &QliqController{QliqService: QliqService}
}
