package models

type PlaceOrderResponse struct {
	OrderID          int64  `json:"order_id"`
	Status           string `json:"status"`             
	RemainingQuantity int    `json:"remaining_quantity"` 
	Message          string `json:"message,omitempty"`
}

type CancelOrderResponse struct {
	Message string `json:"message"`
}

type OrderStatusResponse struct {
	OrderID           int64   `json:"order_id"`
	Status            string  `json:"status"`
	ExecutedQuantity  int     `json:"executed_quantity"`
	RemainingQuantity int     `json:"remaining_quantity"`
}

type OrderBookEntry struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type OrderBookResponse struct {
	Symbol string            `json:"symbol"`
	Bids   []OrderBookEntry  `json:"bids"`
	Asks   []OrderBookEntry  `json:"asks"`
}
