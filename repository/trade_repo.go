package repository

import (
	"context"
	"database/sql"

	"github.com/Puneet-Vishnoi/order-matching-engine/db/postgres/providers"
	"github.com/Puneet-Vishnoi/order-matching-engine/models"
)

type TradeRepository struct {
	DBHelper *providers.DBHelper
}

func NewTradeRepository(db *providers.DBHelper) *TradeRepository {
	return &TradeRepository{DBHelper: db}
}

// CreateTrade saves a trade in the DB and retrieves its ID
func (r *TradeRepository) CreateTrade(ctx context.Context, tx *sql.Tx, trade *models.Trade) error {
	query := `
		INSERT INTO trades (buy_order_id, sell_order_id, price, quantity, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	return tx.QueryRowContext(ctx, query,
		trade.BuyOrderID,
		trade.SellOrderID,
		trade.Price,
		trade.Quantity,
		trade.CreatedAt,
	).Scan(&trade.ID)
}

// ListTradesBySymbol fetches trades for a symbol
func (r *TradeRepository) ListTradesBySymbol(ctx context.Context, symbol string) ([]models.Trade, error) {
	query := `
		SELECT DISTINCT ON (t.id) t.id, t.buy_order_id, t.sell_order_id, t.price, t.quantity, t.created_at
		FROM trades t
		JOIN orders o ON o.id = t.buy_order_id OR o.id = t.sell_order_id
		WHERE o.symbol = $1
		ORDER BY t.id, t.created_at DESC`

	rows, err := r.DBHelper.PostgresClient.QueryContext(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []models.Trade
	for rows.Next() {
		var t models.Trade
		if err := rows.Scan(&t.ID, &t.BuyOrderID, &t.SellOrderID, &t.Price, &t.Quantity, &t.CreatedAt); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, nil
}