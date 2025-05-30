package models

type PlaceOrderRequest struct {
	Symbol   string  `json:"symbol" validate:"required"`
	Side     string  `json:"side" validate:"required,oneof=buy sell"`
	Type     string  `json:"type" validate:"required,oneof=limit market"`
	Price    float64 `json:"price,omitempty" validate:"omitempty,gt=0"` 
	Quantity int     `json:"quantity" validate:"required,gt=0"`
}

type CancelOrderRequest struct {
	OrderID int64 `json:"order_id" validate:"required"`
}
