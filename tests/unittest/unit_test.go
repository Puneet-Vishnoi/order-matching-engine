package unittest

import (
	"context"
	"strconv"
	"testing"

	"github.com/Puneet-Vishnoi/order-matching-engine/models"
	"github.com/Puneet-Vishnoi/order-matching-engine/tests/mockdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var test = mockdb.GetTestInstance()

func TestPlaceOrder(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() []int64 // Returns order IDs for cleanup
		request     models.PlaceOrderRequest
		wantStatus  string
		wantRemQty  int
		wantErr     string
		checkTrades bool
		tradeCount  int
	}{
		{
			name: "Buy Limit Order No Match",
			setup: func() []int64 {
				return []int64{} // No setup needed
			},
			request: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    100.0,
				Quantity: 10,
			},
			wantStatus:  "open",
			wantRemQty:  10,
			wantErr:     "",
			checkTrades: false,
			tradeCount:  0,
		},
		{
			name: "Sell Limit Order No Match",
			setup: func() []int64 {
				return []int64{}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "sell",
				Type:     "limit",
				Price:    200.0,
				Quantity: 5,
			},
			wantStatus:  "open",
			wantRemQty:  5,
			wantErr:     "",
			checkTrades: false,
			tradeCount:  0,
		},
		{
			name: "Buy Order Matches Existing Sell - Full Fill",
			setup: func() []int64 {
				// Create a sell order first
				sellReq := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "sell",
					Type:     "limit",
					Price:    150.0,
					Quantity: 10,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &sellReq)
				require.NoError(t, err)
				return []int64{resp.OrderID}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    150.0,
				Quantity: 10,
			},
			wantStatus:  "filled",
			wantRemQty:  0,
			wantErr:     "",
			checkTrades: true,
			tradeCount:  1,
		},
		{
			name: "Buy Order Matches Existing Sell - Partial Fill",
			setup: func() []int64 {
				sellReq := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "sell",
					Type:     "limit",
					Price:    140.0,
					Quantity: 5,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &sellReq)
				require.NoError(t, err)
				return []int64{resp.OrderID}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    140.0,
				Quantity: 10,
			},
			wantStatus:  "partial",
			wantRemQty:  5,
			wantErr:     "",
			checkTrades: true,
			tradeCount:  1,
		},
		{
			name: "Market Buy Order Matches Multiple Sells",
			setup: func() []int64 {
				var orderIDs []int64

				// Create multiple sell orders at different prices
				sellOrders := []models.PlaceOrderRequest{
					{Symbol: "MSFT", Side: "sell", Type: "limit", Price: 100.0, Quantity: 5},
					{Symbol: "MSFT", Side: "sell", Type: "limit", Price: 101.0, Quantity: 3},
					{Symbol: "MSFT", Side: "sell", Type: "limit", Price: 102.0, Quantity: 2},
				}

				for _, order := range sellOrders {
					resp, err := test.Service.PlaceOrder(context.Background(), &order)
					require.NoError(t, err)
					orderIDs = append(orderIDs, resp.OrderID)
				}
				return orderIDs
			},
			request: models.PlaceOrderRequest{
				Symbol:   "MSFT",
				Side:     "buy",
				Type:     "market",
				Quantity: 8,
			},
			wantStatus:  "filled",
			wantRemQty:  0,
			wantErr:     "",
			checkTrades: true,
			tradeCount:  2, // Should match first two sell orders
		},
		{
			name: "Invalid Order Side",
			setup: func() []int64 {
				return []int64{}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "GOOGL",
				Side:     "invalid",
				Type:     "limit",
				Price:    100.0,
				Quantity: 5,
			},
			wantStatus: "",
			wantRemQty: 0,
			wantErr: "pq: new row for relation \"orders\" violates check constraint \"orders_side_check\"",
		},
		{
			name: "Buy Order Higher Price Than Sell",
			setup: func() []int64 {
				sellReq := models.PlaceOrderRequest{
					Symbol:   "TSLA",
					Side:     "sell",
					Type:     "limit",
					Price:    200.0,
					Quantity: 8,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &sellReq)
				require.NoError(t, err)
				return []int64{resp.OrderID}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "TSLA",
				Side:     "buy",
				Type:     "limit",
				Price:    210.0, // Higher than sell price, should match
				Quantity: 8,
			},
			wantStatus:  "filled",
			wantRemQty:  0,
			wantErr:     "",
			checkTrades: true,
			tradeCount:  1,
		},
		{
			name: "Sell Order Lower Price Than Buy",
			setup: func() []int64 {
				buyReq := models.PlaceOrderRequest{
					Symbol:   "NVDA",
					Side:     "buy",
					Type:     "limit",
					Price:    300.0,
					Quantity: 6,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &buyReq)
				require.NoError(t, err)
				return []int64{resp.OrderID}
			},
			request: models.PlaceOrderRequest{
				Symbol:   "NVDA",
				Side:     "sell",
				Type:     "limit",
				Price:    290.0, // Lower than buy price, should match
				Quantity: 6,
			},
			wantStatus:  "filled",
			wantRemQty:  0,
			wantErr:     "",
			checkTrades: true,
			tradeCount:  1,
		},
	}

	t.Cleanup(func() { test.Cleanup() })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupOrderIDs := tc.setup()

			resp, err := test.Service.PlaceOrder(context.Background(), &tc.request)

			if tc.wantErr != "" {
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)
			assert.Equal(t, tc.wantRemQty, resp.RemainingQuantity)

			if tc.checkTrades {
				trades, err := test.Service.ListTrades(context.Background(), tc.request.Symbol)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(trades), tc.tradeCount)
			}

			// Cleanup: Cancel any remaining orders
			allOrderIDs := append(setupOrderIDs, resp.OrderID)
			for _, orderID := range allOrderIDs {
				test.Service.CancelOrder(context.Background(), strconv.FormatInt(orderID, 10))
			}
		})
	}
}

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() string // Returns order ID as string
		orderID string
		wantErr string
		wantMsg string
	}{
		{
			name: "Cancel Valid Open Order",
			setup: func() string {
				req := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "buy",
					Type:     "limit",
					Price:    100.0,
					Quantity: 10,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &req)
				require.NoError(t, err)
				return strconv.FormatInt(resp.OrderID, 10)
			},
			wantErr: "",
			wantMsg: "canceled",
		},
		{
			name: "Cancel Non-Existent Order",
			setup: func() string {
				return "99999"
			},
			wantErr: "order with ID 99999 not found",
		},
		{
			name: "Cancel Already Filled Order",
			setup: func() string {
				// Create matching orders to fill completely
				sellReq := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "sell",
					Type:     "limit",
					Price:    150.0,
					Quantity: 5,
				}
				test.Service.PlaceOrder(context.Background(), &sellReq)

				buyReq := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "buy",
					Type:     "limit",
					Price:    150.0,
					Quantity: 5,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &buyReq)
				require.NoError(t, err)
				return strconv.FormatInt(resp.OrderID, 10)
			},
			wantErr: "order cannot be canceled",
		},
		{
			name: "Invalid Order ID Format",
			setup: func() string {
				return "invalid_id"
			},
			orderID: "invalid_id",
			wantErr: "invalid order ID",
		},
	}

	t.Cleanup(func() { test.Cleanup() })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var orderID string
			if tc.orderID != "" {
				orderID = tc.orderID
			} else {
				orderID = tc.setup()
			}

			resp, err := test.Service.CancelOrder(context.Background(), orderID)

			if tc.wantErr != "" {
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, resp.Message, tc.wantMsg)
		})
	}
}

func TestGetOrderStatus(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() string
		orderID     string
		wantStatus  string
		wantExecQty int
		wantRemQty  int
		wantErr     string
	}{
		{
			name: "Get Status of Open Order",
			setup: func() string {
				req := models.PlaceOrderRequest{
					Symbol:   "AAPL",
					Side:     "buy",
					Type:     "limit",
					Price:    100.0,
					Quantity: 10,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &req)
				require.NoError(t, err)
				return strconv.FormatInt(resp.OrderID, 10)
			},
			wantStatus:  "open",
			wantExecQty: 0,
			wantRemQty:  10,
			wantErr:     "",
		},
		{
			name: "Get Status of Filled Order",
			setup: func() string {
				// Create sell order first
				sellReq := models.PlaceOrderRequest{
					Symbol:   "MSFT",
					Side:     "sell",
					Type:     "limit",
					Price:    200.0,
					Quantity: 5,
				}
				test.Service.PlaceOrder(context.Background(), &sellReq)

				// Create matching buy order
				buyReq := models.PlaceOrderRequest{
					Symbol:   "MSFT",
					Side:     "buy",
					Type:     "limit",
					Price:    200.0,
					Quantity: 5,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &buyReq)
				require.NoError(t, err)
				return strconv.FormatInt(resp.OrderID, 10)
			},
			wantStatus:  "filled",
			wantExecQty: 5,
			wantRemQty:  0,
			wantErr:     "",
		},
		{
			name: "Get Status of Partially Filled Order",
			setup: func() string {
				// Create smaller sell order first
				sellReq := models.PlaceOrderRequest{
					Symbol:   "GOOGL",
					Side:     "sell",
					Type:     "limit",
					Price:    300.0,
					Quantity: 3,
				}
				test.Service.PlaceOrder(context.Background(), &sellReq)

				// Create larger buy order
				buyReq := models.PlaceOrderRequest{
					Symbol:   "GOOGL",
					Side:     "buy",
					Type:     "limit",
					Price:    300.0,
					Quantity: 10,
				}
				resp, err := test.Service.PlaceOrder(context.Background(), &buyReq)
				require.NoError(t, err)
				return strconv.FormatInt(resp.OrderID, 10)
			},
			wantStatus:  "partial",
			wantExecQty: 3,
			wantRemQty:  7,
			wantErr:     "",
		},
		{
			name: "Invalid Order ID",
			setup: func() string {
				return "invalid"
			},
			wantErr: "invalid order ID",
		},
		{
			name: "Non-Existent Order ID",
			setup: func() string {
				return "99999"
			},
			wantErr: "order with ID 99999 not found",
		},
	}

	t.Cleanup(func() { test.Cleanup() })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orderID := tc.setup()

			resp, err := test.Service.GetOrderStatus(context.Background(), orderID)

			if tc.wantErr != "" {
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.Status)
			assert.Equal(t, tc.wantExecQty, resp.ExecutedQuantity)
			assert.Equal(t, tc.wantRemQty, resp.RemainingQuantity)
		})
	}
}

func TestListTrades(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		symbol    string
		wantCount int
		wantErr   string
	}{
		{
			name: "List Trades for Symbol with Trades",
			setup: func() {
				// Create matching orders to generate trades
				sellReq := models.PlaceOrderRequest{
					Symbol:   "TRADE_TEST",
					Side:     "sell",
					Type:     "limit",
					Price:    100.0,
					Quantity: 5,
				}
				test.Service.PlaceOrder(context.Background(), &sellReq)

				buyReq := models.PlaceOrderRequest{
					Symbol:   "TRADE_TEST",
					Side:     "buy",
					Type:     "limit",
					Price:    100.0,
					Quantity: 5,
				}
				test.Service.PlaceOrder(context.Background(), &buyReq)
			},
			symbol:    "TRADE_TEST",
			wantCount: 1,
			wantErr:   "",
		},
		{
			name: "List Trades for Symbol with No Trades",
			setup: func() {
				// No setup needed
			},
			symbol:    "NO_TRADES",
			wantCount: 0,
			wantErr:   "",
		},
		{
			name: "Empty Symbol",
			setup: func() {
				// No setup needed
			},
			symbol:  "",
			wantErr: "symbol is required",
		},
	}

	t.Cleanup(func() { test.Cleanup() })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			trades, err := test.Service.ListTrades(context.Background(), tc.symbol)

			if tc.wantErr != "" {
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantCount, len(trades))
		})
	}
}

func TestGetOrderBook(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		symbol   string
		wantBids int
		wantAsks int
		wantErr  string
	}{
		{
			name: "Order Book with Bids and Asks",
			setup: func() {
				// Create buy orders (bids)
				buyOrders := []models.PlaceOrderRequest{
					{Symbol: "BOOK_TEST", Side: "buy", Type: "limit", Price: 95.0, Quantity: 10},
					{Symbol: "BOOK_TEST", Side: "buy", Type: "limit", Price: 94.0, Quantity: 5},
				}

				// Create sell orders (asks)
				sellOrders := []models.PlaceOrderRequest{
					{Symbol: "BOOK_TEST", Side: "sell", Type: "limit", Price: 105.0, Quantity: 8},
					{Symbol: "BOOK_TEST", Side: "sell", Type: "limit", Price: 106.0, Quantity: 3},
				}

				for _, order := range buyOrders {
					test.Service.PlaceOrder(context.Background(), &order)
				}
				for _, order := range sellOrders {
					test.Service.PlaceOrder(context.Background(), &order)
				}
			},
			symbol:   "BOOK_TEST",
			wantBids: 2,
			wantAsks: 2,
			wantErr:  "",
		},
		{
			name: "Empty Order Book",
			setup: func() {
				// No setup needed
			},
			symbol:   "EMPTY_BOOK",
			wantBids: 0,
			wantAsks: 0,
			wantErr:  "",
		},
		{
			name: "Order Book with Only Bids",
			setup: func() {
				buyReq := models.PlaceOrderRequest{
					Symbol:   "BIDS_ONLY",
					Side:     "buy",
					Type:     "limit",
					Price:    100.0,
					Quantity: 10,
				}
				test.Service.PlaceOrder(context.Background(), &buyReq)
			},
			symbol:   "BIDS_ONLY",
			wantBids: 1,
			wantAsks: 0,
			wantErr:  "",
		},
	}

	t.Cleanup(func() { test.Cleanup() })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			book, err := test.Service.GetOrderBook(context.Background(), tc.symbol)

			if tc.wantErr != "" {
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.symbol, book.Symbol)
			assert.Equal(t, tc.wantBids, len(book.Bids))
			assert.Equal(t, tc.wantAsks, len(book.Asks))

			// Verify bid ordering (highest price first)
			for i := 1; i < len(book.Bids); i++ {
				assert.GreaterOrEqual(t, book.Bids[i-1].Price, book.Bids[i].Price)
			}

			// Verify ask ordering (lowest price first)
			for i := 1; i < len(book.Asks); i++ {
				assert.LessOrEqual(t, book.Asks[i-1].Price, book.Asks[i].Price)
			}
		})
	}
}
