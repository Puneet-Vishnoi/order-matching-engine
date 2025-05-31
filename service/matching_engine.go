package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Puneet-Vishnoi/order-matching-engine/models"
	"github.com/Puneet-Vishnoi/order-matching-engine/repository"
)

type MatchingEngine struct{}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{}
}

// Match performs order matching logic and returns:
// - trades to be created
// - updated counterparty orders
// - the updated current order
func (e *MatchingEngine) Match(
	ctx context.Context,
	tx *sql.Tx,
	incoming *models.Order,
	repo *repository.OrderRepository,
) ([]models.Trade, []models.Order, error) {

	var counterOrders []models.Order
	var err error

	if incoming.Side == "buy" {
		counterOrders, err = repo.FetchOpenSellOrders(ctx, tx, incoming.Symbol)
	} else if incoming.Side == "sell" {
		counterOrders, err = repo.FetchOpenBuyOrders(ctx, tx, incoming.Symbol)
	} else {
		return nil, nil, errors.New("invalid order side")
	}
	if err != nil {
		return nil, nil, err
	}

	var trades []models.Trade
	var updatedOrders []models.Order
	remaining := incoming.RemainingQty

	for i := 0; i < len(counterOrders) && remaining > 0; i++ {
		resting := &counterOrders[i]

		// Match rules
		if incoming.Type == "market" || // market order matches any price
			(incoming.Side == "buy" && incoming.Price >= resting.Price) ||
			(incoming.Side == "sell" && incoming.Price <= resting.Price) {

			matchQty := min(remaining, resting.RemainingQty)
			tradePrice := resting.Price
			remaining -= matchQty
			resting.RemainingQty -= matchQty

			if resting.RemainingQty == 0 {
				resting.Status = "filled"
			} else {
				resting.Status = "partial"
			}
			updatedOrders = append(updatedOrders, *resting)

			trades = append(trades, models.Trade{
				BuyOrderID:  ifBuy(incoming, resting),
				SellOrderID: ifSell(incoming, resting),
				Price:       tradePrice,
				Quantity:    matchQty,
				CreatedAt:   time.Now(),
			})
		}
	}

	// Update incoming order
	incoming.RemainingQty = remaining
	switch {
	case remaining == 0:
		incoming.Status = "filled" // Order is completely filled
	case remaining < incoming.Quantity:
		incoming.Status = "partial" // Order is partially filled
	case incoming.Type == "market":
		incoming.Status = "canceled" // Market order that couldn't be filled should be canceled
	case incoming.Type == "limit":
		incoming.Status = "open" // Limit order that hasn't matched yet remains open
	}

	return trades, updatedOrders, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ifBuy(a, b *models.Order) int64 {
	if a.Side == "buy" {
		return a.ID
	}
	return b.ID
}

func ifSell(a, b *models.Order) int64 {
	if a.Side == "sell" {
		return a.ID
	}
	return b.ID
}
