package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/Puneet-Vishnoi/order-matching-engine/models"
	"github.com/Puneet-Vishnoi/order-matching-engine/repository"
)

type OrderService struct {
	OrderRepo      *repository.OrderRepository
	TradeRepo      *repository.TradeRepository
	MatchingEngine *MatchingEngine
}

func NewOrderService(orderRepo *repository.OrderRepository, tradeRepo *repository.TradeRepository) *OrderService {
	return &OrderService{
		OrderRepo:      orderRepo,
		TradeRepo:      tradeRepo,
		MatchingEngine: NewMatchingEngine(),
	}
}

func (s *OrderService) PlaceOrder(ctx context.Context, req *models.PlaceOrderRequest) (*models.PlaceOrderResponse, error) {
	tx, err := s.OrderRepo.DBHelper.PostgresClient.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	order := models.Order{
		Symbol:       req.Symbol,
		Side:         req.Side,
		Type:         req.Type,
		Price:        req.Price,
		Quantity:     req.Quantity,
		RemainingQty: req.Quantity,
		Status:       "open",
		CreatedAt:    time.Now(),
	}
	// Step 1: Insert Order
	orderID, err := s.OrderRepo.CreateOrder(ctx, tx, &order)
	if err != nil {
		return nil, err
	}
	order.ID = orderID

	// Step 2: Match Order (get counter-orders and execute trades)
	trades, updatedOrders, err := s.MatchingEngine.Match(ctx, tx, &order, s.OrderRepo)
	if err != nil {
		return nil, err
	}
	// Step 3: Save Trades
	for _, trade := range trades {
		if err := s.TradeRepo.CreateTrade(ctx, tx, &trade); err != nil {
			return nil, err
		}
	}

	// Step 4: Update All Affected Orders
	for _, u := range updatedOrders {
		if err := s.OrderRepo.UpdateOrder(ctx, tx, &u); err != nil {
			return nil, err
		}
	}

	// Step 5: Update This Order
	if err := s.OrderRepo.UpdateOrder(ctx, tx, &order); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.PlaceOrderResponse{
		OrderID:           order.ID,
		Status:            order.Status,
		RemainingQuantity: order.RemainingQty,
		Message:           "Order placed successfully",
	}, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderIDStr string) (*models.CancelOrderResponse, error) {
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		return nil, errors.New("invalid order ID")
	}

	tx, err := s.OrderRepo.DBHelper.PostgresClient.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	order, err := s.OrderRepo.GetOrderByID(ctx, tx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == "filled" || order.Status == "canceled" {
		return nil, errors.New("order cannot be canceled")
	}

	order.Status = "canceled"
	order.RemainingQty = 0

	if err := s.OrderRepo.UpdateOrder(ctx, tx, order); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.CancelOrderResponse{
		Message: fmt.Sprintf("Order %d canceled", orderID),
	}, nil
}

func (s *OrderService) GetOrderStatus(ctx context.Context, orderID string) (*models.OrderStatusResponse, error) {
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid order ID")
	}

	order, err := s.OrderRepo.GetOrderByID(ctx, nil, id)
	if err != nil {
		return nil, err
	}

	executedQty := order.Quantity - order.RemainingQty

	return &models.OrderStatusResponse{
		OrderID:           order.ID,
		Status:            order.Status,
		ExecutedQuantity:  executedQty,
		RemainingQuantity: order.RemainingQty,
	}, nil
}

func (s *OrderService) ListTrades(ctx context.Context, symbol string) ([]models.Trade, error) {
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}

	trades, err := s.TradeRepo.ListTradesBySymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (s *OrderService) GetOrderBook(ctx context.Context, symbol string) (*models.OrderBookResponse, error) {
	tx, err := s.OrderRepo.DBHelper.PostgresClient.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	buyOrders, err := s.OrderRepo.FetchOpenBuyOrders(ctx, tx, symbol)
	if err != nil {
		return nil, err
	}

	sellOrders, err := s.OrderRepo.FetchOpenSellOrders(ctx, tx, symbol)
	if err != nil {
		return nil, err
	}

	// Group orders by price level
	bidMap := make(map[float64]int)
	for _, o := range buyOrders {
		bidMap[o.Price] += o.RemainingQty
	}

	askMap := make(map[float64]int)
	for _, o := range sellOrders {
		askMap[o.Price] += o.RemainingQty
	}

	// Convert to sorted slices
	bids := flattenAndSortOrderBook(bidMap, true)  // DESC
	asks := flattenAndSortOrderBook(askMap, false) // ASC

	return &models.OrderBookResponse{
		Symbol: symbol,
		Bids:   bids,
		Asks:   asks,
	}, nil
}

func flattenAndSortOrderBook(book map[float64]int, desc bool) []models.OrderBookEntry {
	var prices []float64
	for price := range book {
		prices = append(prices, price)
	}

	if desc {
		sort.Slice(prices, func(i, j int) bool { return prices[i] > prices[j] })
	} else {
		sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	}

	var entries []models.OrderBookEntry
	for _, price := range prices {
		entries = append(entries, models.OrderBookEntry{
			Price:    price,
			Quantity: book[price],
		})
	}
	return entries
}
