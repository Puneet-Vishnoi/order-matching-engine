package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Puneet-Vishnoi/order-matching-engine/db/postgres/providers"
	"github.com/Puneet-Vishnoi/order-matching-engine/models"
)

// type CouponRepository struct {
// 	DBHelper *providers.DBHelper
// }

// func NewCouponRepository(db *providers.DBHelper) *CouponRepository {
// 	return &CouponRepository{DBHelper: db}
// }

type OrderRepository struct {
	DBHelper *providers.DBHelper
}

func NewOrderRepository(db *providers.DBHelper) *OrderRepository {
	return &OrderRepository{DBHelper: db}
}

// CreateOrder inserts a new order into the DB.
func (r *OrderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, order *models.Order) (int64, error) {
	query := `
		INSERT INTO orders (symbol, side, type, price, quantity, remaining_quantity, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	err := tx.QueryRowContext(ctx, query,
		order.Symbol, order.Side, order.Type, order.Price,
		order.Quantity, order.RemainingQty, order.Status, order.CreatedAt,
	).Scan(&order.ID)
	return order.ID, err
}

// UpdateOrder updates status and remaining quantity
func (r *OrderRepository) UpdateOrder(ctx context.Context, tx *sql.Tx, order *models.Order) error {
	query := `
		UPDATE orders
		SET remaining_quantity = $1, status = $2
		WHERE id = $3`
	_, err := tx.ExecContext(ctx, query, order.RemainingQty, order.Status, order.ID)
	return err
}

// Fetch open SELL orders for a symbol, ordered by price ASC, time ASC (used when buying)
func (r *OrderRepository) FetchOpenSellOrders(ctx context.Context, tx *sql.Tx, symbol string) ([]models.Order, error) {
	query := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at
		FROM orders
		WHERE symbol = $1 AND side = 'sell' AND status IN ('open', 'partial')
		ORDER BY price ASC, created_at ASC`
	rows, err := tx.QueryContext(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.RemainingQty, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// Fetch open BUY orders for a symbol, ordered by price DESC, time ASC (used when selling)
func (r *OrderRepository) FetchOpenBuyOrders(ctx context.Context, tx *sql.Tx, symbol string) ([]models.Order, error) {
	query := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at
		FROM orders
		WHERE symbol = $1 AND side = 'buy' AND status IN ('open', 'partial')
		ORDER BY price DESC, created_at ASC`
	rows, err := tx.QueryContext(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.RemainingQty, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// GetOrderByID fetches one order by ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, tx *sql.Tx, id int64) (*models.Order, error) {
	query := `
		SELECT id, symbol, side, type, price, quantity, remaining_quantity, status, created_at
		FROM orders WHERE id = $1`
	var o models.Order

	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, id)
	} else {
		row = r.DBHelper.PostgresClient.QueryRowContext(ctx, query, id)
	}

	err := row.Scan(&o.ID, &o.Symbol, &o.Side, &o.Type, &o.Price, &o.Quantity, &o.RemainingQty, &o.Status, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get order by ID %d: %w", id, err)
	}

	return &o, nil
}
