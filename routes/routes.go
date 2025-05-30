package routes

import (
	"github.com/Puneet-Vishnoi/order-matching-engine/handlers"
	"github.com/Puneet-Vishnoi/order-matching-engine/service"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, service *service.OrderService) {
	orderHandler := handlers.NewOrderHandler(service)

	api := router.Group("/api")
	{
		api.POST("/orders", orderHandler.PlaceOrder)
		api.DELETE("/orders/:id", orderHandler.CancelOrder)
		api.GET("/orderbook", orderHandler.GetOrderBook)

		api.GET("/orders/:id", orderHandler.GetOrderStatus)
		api.GET("/trades", orderHandler.ListTrades)
	}
}
