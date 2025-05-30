package handlers

import (
	"net/http"

	"github.com/Puneet-Vishnoi/order-matching-engine/models"
	"github.com/Puneet-Vishnoi/order-matching-engine/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type OrderHandler struct {
	Service   *service.OrderService
	Validator *validator.Validate
}

func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{
		Service:   s,
		Validator: validator.New(),
	}
}

func formatValidationError(err error) map[string]string {
	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		errors[e.Field()] = "failed on tag '" + e.Tag() + "'"
	}
	return errors
}

// POST /orders
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	var req models.PlaceOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.Validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"validation_errors": formatValidationError(err)})
		return
	}

	resp, err := h.Service.PlaceOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DELETE /orders/:id
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	resp, err := h.Service.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /orderbook?symbol=XYZ
func (h *OrderHandler) GetOrderBook(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing symbol query parameter"})
		return
	}

	resp, err := h.Service.GetOrderBook(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /orders/:id
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	resp, err := h.Service.GetOrderStatus(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /trades?symbol=XYZ
func (h *OrderHandler) ListTrades(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'symbol' query parameter"})
		return
	}

	resp, err := h.Service.ListTrades(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
