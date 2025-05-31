package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Puneet-Vishnoi/order-matching-engine/models"
	"github.com/Puneet-Vishnoi/order-matching-engine/tests/mockdb"
	"github.com/go-playground/assert"
)

const baseURL = "http://localhost:8080/api"

func TestOrderMatchingEngineIntegration(t *testing.T) {
	// Initialize the test dependencies
	testDeps := mockdb.GetTestInstance()
	defer testDeps.Cleanup()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{"TestPlaceOrderIntegration", testPlaceOrderIntegration},
		{"TestOrderMatchingIntegration", testOrderMatchingIntegration},
		{"TestCancelOrderIntegration", testCancelOrderIntegration},
		{"TestOrderStatusIntegration", testOrderStatusIntegration},
		{"TestOrderBookIntegration", testOrderBookIntegration},
		{"TestTradeHistoryIntegration", testTradeHistoryIntegration},
		{"TestComplexMatchingScenarios", testComplexMatchingScenarios},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up database before each test
			cleanupDatabase(t, testDeps)
			tt.testFunc(t)
		})
	}
}

func testPlaceOrderIntegration(t *testing.T) {
	testCases := []struct {
		name           string
		order          models.PlaceOrderRequest
		expectedStatus int
		expectedMsg    string
		validateFunc   func(t *testing.T, response *models.PlaceOrderResponse)
	}{
		{
			name: "Valid limit buy order",
			order: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    150.0,
				Quantity: 100,
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Order placed successfully",
			validateFunc: func(t *testing.T, response *models.PlaceOrderResponse) {
				assert.Equal(t, response.Status, "open")
				assert.Equal(t, response.RemainingQuantity, 100)
			},
		},
		{
			name: "Valid market sell order",
			order: models.PlaceOrderRequest{
				Symbol:   "GOOGL",
				Side:     "sell",
				Type:     "market",
				Quantity: 50,
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "Order placed successfully",
			validateFunc: func(t *testing.T, response *models.PlaceOrderResponse) {
				assert.Equal(t, response.Status, "canceled")
			},
		},
		{
			name: "Invalid order - missing symbol",
			order: models.PlaceOrderRequest{
				Side:     "buy",
				Type:     "limit",
				Price:    150.0,
				Quantity: 100,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid order - negative quantity",
			order: models.PlaceOrderRequest{
				Symbol:   "TSLA",
				Side:     "buy",
				Type:     "limit",
				Price:    200.0,
				Quantity: -10,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid order - invalid side",
			order: models.PlaceOrderRequest{
				Symbol:   "MSFT",
				Side:     "invalid",
				Type:     "limit",
				Price:    100.0,
				Quantity: 50,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orderJSON, err := json.Marshal(tc.order)
			if err != nil {
				t.Fatalf("failed to marshal order: %v", err)
			}

			resp, err := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
			if err != nil {
				t.Fatalf("failed to place order: %v", err)
			}
			defer resp.Body.Close()

			assert.Equal(t, resp.StatusCode, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK {
				var result models.PlaceOrderResponse
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				assert.Equal(t, result.Message, tc.expectedMsg)
				if tc.validateFunc != nil {
					tc.validateFunc(t, &result)
				}
			}
		})
	}
}

func testOrderMatchingIntegration(t *testing.T) {
	testCases := []struct {
		name              string
		setupOrders       []models.PlaceOrderRequest
		incomingOrder     models.PlaceOrderRequest
		expectedTrades    int
		expectedStatus    string
		expectedRemaining int
	}{
		{
			name: "Full match - buy matches sell exactly",
			setupOrders: []models.PlaceOrderRequest{
				{
					Symbol:   "AAPL",
					Side:     "sell",
					Type:     "limit",
					Price:    100.0,
					Quantity: 50,
				},
			},
			incomingOrder: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    100.0,
				Quantity: 50,
			},
			expectedTrades:    1,
			expectedStatus:    "filled",
			expectedRemaining: 0,
		},
		{
			name: "Partial match - larger buy order",
			setupOrders: []models.PlaceOrderRequest{
				{
					Symbol:   "GOOGL",
					Side:     "sell",
					Type:     "limit",
					Price:    200.0,
					Quantity: 30,
				},
			},
			incomingOrder: models.PlaceOrderRequest{
				Symbol:   "GOOGL",
				Side:     "buy",
				Type:     "limit",
				Price:    200.0,
				Quantity: 50,
			},
			expectedTrades:    1,
			expectedStatus:    "partial",
			expectedRemaining: 20,
		},
		{
			name: "Multiple matches across price levels",
			setupOrders: []models.PlaceOrderRequest{
				{
					Symbol:   "TSLA",
					Side:     "sell",
					Type:     "limit",
					Price:    300.0,
					Quantity: 25,
				},
				{
					Symbol:   "TSLA",
					Side:     "sell",
					Type:     "limit",
					Price:    301.0,
					Quantity: 30,
				},
			},
			incomingOrder: models.PlaceOrderRequest{
				Symbol:   "TSLA",
				Side:     "buy",
				Type:     "limit",
				Price:    301.0,
				Quantity: 40,
			},
			expectedTrades:    2,
			expectedStatus:    "filled",
			expectedRemaining: 0, // 40 - 25 - 15 = 0
		},
		{
			name: "No match - price too low",
			setupOrders: []models.PlaceOrderRequest{
				{
					Symbol:   "NVDA",
					Side:     "sell",
					Type:     "limit",
					Price:    400.0,
					Quantity: 50,
				},
			},
			incomingOrder: models.PlaceOrderRequest{
				Symbol:   "NVDA",
				Side:     "buy",
				Type:     "limit",
				Price:    390.0,
				Quantity: 25,
			},
			expectedTrades:    0,
			expectedStatus:    "open",
			expectedRemaining: 25,
		},
		{
			name: "Market order matches best available prices",
			setupOrders: []models.PlaceOrderRequest{
				{
					Symbol:   "AMD",
					Side:     "sell",
					Type:     "limit",
					Price:    80.0,
					Quantity: 20,
				},
				{
					Symbol:   "AMD",
					Side:     "sell",
					Type:     "limit",
					Price:    82.0,
					Quantity: 15,
				},
			},
			incomingOrder: models.PlaceOrderRequest{
				Symbol:   "AMD",
				Side:     "buy",
				Type:     "market",
				Quantity: 30,
			},
			expectedTrades:    2,
			expectedStatus:    "filled",
			expectedRemaining: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup orders first
			var setupOrderIDs []int64
			for _, setupOrder := range tc.setupOrders {
				orderJSON, _ := json.Marshal(setupOrder)
				resp, err := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
				if err != nil {
					t.Fatalf("failed to setup order: %v", err)
				}

				var result models.PlaceOrderResponse
				json.NewDecoder(resp.Body).Decode(&result)
				setupOrderIDs = append(setupOrderIDs, result.OrderID)
				resp.Body.Close()
			}

			// Place incoming order
			incomingJSON, _ := json.Marshal(tc.incomingOrder)
			resp, err := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(incomingJSON))
			if err != nil {
				t.Fatalf("failed to place incoming order: %v", err)
			}
			defer resp.Body.Close()

			var result models.PlaceOrderResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Validate the response
			assert.Equal(t, result.Status, tc.expectedStatus)
			assert.Equal(t, result.RemainingQuantity, tc.expectedRemaining)

			// Check trades were created
			if tc.expectedTrades > 0 {
				tradesResp, err := http.Get(fmt.Sprintf("%s/trades?symbol=%s", baseURL, tc.incomingOrder.Symbol))
				if err != nil {
					t.Fatalf("failed to get trades: %v", err)
				}
				defer tradesResp.Body.Close()

				var trades []models.Trade
				json.NewDecoder(tradesResp.Body).Decode(&trades)
				assert.Equal(t, len(trades), tc.expectedTrades)
			}
		})
	}
}

func testCancelOrderIntegration(t *testing.T) {
	testCases := []struct {
		name           string
		setupOrder     models.PlaceOrderRequest
		cancelAfter    time.Duration
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "Cancel open order successfully",
			setupOrder: models.PlaceOrderRequest{
				Symbol:   "AAPL",
				Side:     "buy",
				Type:     "limit",
				Price:    150.0,
				Quantity: 100,
			},
			cancelAfter:    time.Millisecond * 100,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Cancel non-existent order",
			setupOrder: models.PlaceOrderRequest{
				Symbol:   "GOOGL",
				Side:     "sell",
				Type:     "limit",
				Price:    200.0,
				Quantity: 50,
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup order
			orderJSON, _ := json.Marshal(tc.setupOrder)
			resp, err := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
			if err != nil {
				t.Fatalf("failed to setup order: %v", err)
			}

			var orderResult models.PlaceOrderResponse
			json.NewDecoder(resp.Body).Decode(&orderResult)
			resp.Body.Close()

			if tc.cancelAfter > 0 {
				time.Sleep(tc.cancelAfter)
			}

			// Cancel order
			var cancelOrderID int64 = orderResult.OrderID
			if tc.name == "Cancel non-existent order" {
				cancelOrderID = 99999 // Non-existent ID
			}

			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/orders/%d", baseURL, cancelOrderID), nil)
			if err != nil {
				t.Fatalf("failed to create DELETE request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			cancelResp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to cancel order: %v", err)
			}
			defer cancelResp.Body.Close()

			assert.Equal(t, cancelResp.StatusCode, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK {
				var cancelResult models.CancelOrderResponse
				json.NewDecoder(cancelResp.Body).Decode(&cancelResult)
				assert.NotEqual(t, cancelResult.Message, "")
			}
		})
	}
}

func testOrderStatusIntegration(t *testing.T) {
	// Setup a test order
	order := models.PlaceOrderRequest{
		Symbol:   "AAPL",
		Side:     "buy",
		Type:     "limit",
		Price:    150.0,
		Quantity: 100,
	}

	orderJSON, _ := json.Marshal(order)
	resp, err := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
	if err != nil {
		t.Fatalf("failed to setup order: %v", err)
	}

	var orderResult models.PlaceOrderResponse
	json.NewDecoder(resp.Body).Decode(&orderResult)
	resp.Body.Close()

	testCases := []struct {
		name           string
		orderID        string
		expectedStatus int
		validateFunc   func(t *testing.T, response *models.OrderStatusResponse)
	}{
		{
			name:           "Get status of existing order",
			orderID:        strconv.FormatInt(orderResult.OrderID, 10),
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *models.OrderStatusResponse) {
				assert.Equal(t, response.OrderID, orderResult.OrderID)
				assert.Equal(t, response.Status, "open")
				assert.Equal(t, response.RemainingQuantity, 100)
				assert.Equal(t, response.ExecutedQuantity, 0)
			},
		},
		{
			name:           "Get status of non-existent order",
			orderID:        "99999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid order ID format",
			orderID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/orders/%s", baseURL, tc.orderID))
			if err != nil {
				t.Fatalf("failed to get order status: %v", err)
			}
			defer resp.Body.Close()

			assert.Equal(t, resp.StatusCode, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK && tc.validateFunc != nil {
				var result models.OrderStatusResponse
				json.NewDecoder(resp.Body).Decode(&result)
				tc.validateFunc(t, &result)
			}
		})
	}
}

func testOrderBookIntegration(t *testing.T) {
	symbol := "AAPL"

	// Setup orders to create order book
	setupOrders := []models.PlaceOrderRequest{
		{Symbol: symbol, Side: "buy", Type: "limit", Price: 149.0, Quantity: 100},
		{Symbol: symbol, Side: "buy", Type: "limit", Price: 148.5, Quantity: 50},
		{Symbol: symbol, Side: "sell", Type: "limit", Price: 151.0, Quantity: 75},
		{Symbol: symbol, Side: "sell", Type: "limit", Price: 152.0, Quantity: 25},
	}

	for _, order := range setupOrders {
		orderJSON, _ := json.Marshal(order)
		resp, _ := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
		resp.Body.Close()
	}

	testCases := []struct {
		name           string
		symbol         string
		expectedStatus int
		validateFunc   func(t *testing.T, response *models.OrderBookResponse)
	}{
		{
			name:           "Get order book for existing symbol",
			symbol:         symbol,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *models.OrderBookResponse) {
				assert.Equal(t, response.Symbol, symbol)
				assert.Equal(t, len(response.Bids), 2) // 2 buy price levels
				assert.Equal(t, len(response.Asks), 2) // 2 sell price levels

				// Check bid ordering (highest first)
				assert.Equal(t, response.Bids[0].Price, 149.0)
				assert.Equal(t, response.Bids[1].Price, 148.5)

				// Check ask ordering (lowest first)
				assert.Equal(t, response.Asks[0].Price, 151.0)
				assert.Equal(t, response.Asks[1].Price, 152.0)
			},
		},
		{
			name:           "Get order book for non-existent symbol",
			symbol:         "NONEXISTENT",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *models.OrderBookResponse) {
				assert.Equal(t, len(response.Bids), 0)
				assert.Equal(t, len(response.Asks), 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/orderbook?symbol=%s", baseURL, tc.symbol))
			if err != nil {
				t.Fatalf("failed to get order book: %v", err)
			}
			defer resp.Body.Close()

			assert.Equal(t, resp.StatusCode, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK && tc.validateFunc != nil {
				var result models.OrderBookResponse
				json.NewDecoder(resp.Body).Decode(&result)
				tc.validateFunc(t, &result)
			}
		})
	}
}

func testTradeHistoryIntegration(t *testing.T) {
	symbol := "MSFT"

	// Create matching orders to generate trades
	sellOrder := models.PlaceOrderRequest{
		Symbol: symbol, Side: "sell", Type: "limit", Price: 100.0, Quantity: 50,
	}
	buyOrder := models.PlaceOrderRequest{
		Symbol: symbol, Side: "buy", Type: "limit", Price: 100.0, Quantity: 50,
	}

	// Place sell order first
	sellJSON, _ := json.Marshal(sellOrder)
	resp, _ := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(sellJSON))
	resp.Body.Close()

	// Place buy order to create trade
	buyJSON, _ := json.Marshal(buyOrder)
	resp, _ = http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(buyJSON))
	resp.Body.Close()

	testCases := []struct {
		name           string
		symbol         string
		expectedStatus int
		validateFunc   func(t *testing.T, trades []models.Trade)
	}{
		{
			name:           "Get trades for symbol with trades",
			symbol:         symbol,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, trades []models.Trade) {
				assert.Equal(t, len(trades), 1)
				assert.Equal(t, trades[0].Price, 100.0)
				assert.Equal(t, trades[0].Quantity, 50)
			},
		},
		{
			name:           "Get trades for symbol without trades",
			symbol:         "NOTRADE",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, trades []models.Trade) {
				assert.Equal(t, len(trades), 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/trades?symbol=%s", baseURL, tc.symbol))
			if err != nil {
				t.Fatalf("failed to get trades: %v", err)
			}
			defer resp.Body.Close()

			assert.Equal(t, resp.StatusCode, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK && tc.validateFunc != nil {
				var trades []models.Trade
				json.NewDecoder(resp.Body).Decode(&trades)
				tc.validateFunc(t, trades)
			}
		})
	}
}

func testComplexMatchingScenarios(t *testing.T) {
	testCases := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "Price-time priority matching",
			scenario: func(t *testing.T) {
				symbol := "PRIORITY"

				// Place orders with same price but different times
				orders := []models.PlaceOrderRequest{
					{Symbol: symbol, Side: "sell", Type: "limit", Price: 100.0, Quantity: 30},
					{Symbol: symbol, Side: "sell", Type: "limit", Price: 100.0, Quantity: 20}, // Should match second due to time priority
				}

				for _, order := range orders {
					orderJSON, _ := json.Marshal(order)
					resp, _ := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(orderJSON))
					resp.Body.Close()
					time.Sleep(time.Millisecond * 10) // Ensure different timestamps
				}

				// Place buy order that should match first sell order
				buyOrder := models.PlaceOrderRequest{
					Symbol: symbol, Side: "buy", Type: "limit", Price: 100.0, Quantity: 25,
				}
				buyJSON, _ := json.Marshal(buyOrder)
				resp, _ := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(buyJSON))

				var result models.PlaceOrderResponse
				json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()

				assert.Equal(t, result.Status, "filled")
				assert.Equal(t, result.RemainingQuantity, 0)
			},
		},
		{
			name: "Partial fill and remaining order",
			scenario: func(t *testing.T) {
				symbol := "PARTIAL"

				// Place large sell order
				sellOrder := models.PlaceOrderRequest{
					Symbol: symbol, Side: "sell", Type: "limit", Price: 200.0, Quantity: 100,
				}
				sellJSON, _ := json.Marshal(sellOrder)
				resp, _ := http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(sellJSON))
				var sellResult models.PlaceOrderResponse
				json.NewDecoder(resp.Body).Decode(&sellResult)
				resp.Body.Close()

				// Place smaller buy order
				buyOrder := models.PlaceOrderRequest{
					Symbol: symbol, Side: "buy", Type: "limit", Price: 200.0, Quantity: 30,
				}
				buyJSON, _ := json.Marshal(buyOrder)
				resp, _ = http.Post(fmt.Sprintf("%s/orders", baseURL), "application/json", bytes.NewBuffer(buyJSON))
				var buyResult models.PlaceOrderResponse
				json.NewDecoder(resp.Body).Decode(&buyResult)
				resp.Body.Close()

				// Check sell order is partially filled
				statusResp, _ := http.Get(fmt.Sprintf("%s/orders/%d", baseURL, sellResult.OrderID))
				var sellStatus models.OrderStatusResponse
				json.NewDecoder(statusResp.Body).Decode(&sellStatus)
				statusResp.Body.Close()

				assert.Equal(t, sellStatus.Status, "partial")
				assert.Equal(t, sellStatus.ExecutedQuantity, 30)
				assert.Equal(t, sellStatus.RemainingQuantity, 70)

				// Check buy order is filled
				assert.Equal(t, buyResult.Status, "filled")
				assert.Equal(t, buyResult.RemainingQuantity, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.scenario(t)
		})
	}
}

// Helper function to clean up database before each test
func cleanupDatabase(t *testing.T, testDeps *mockdb.TestDeps) {
	ctx := context.Background()

	// You might want to add cleanup methods to your repository
	// For now, we'll assume there are methods like DeleteAllOrders, DeleteAllTrades
	_, err := testDeps.PostgresClient.PostgresClient.ExecContext(ctx, "DELETE FROM trades")
	if err != nil {
		t.Logf("Warning: failed to clean trades: %v", err)
	}

	_, err = testDeps.PostgresClient.PostgresClient.ExecContext(ctx, "DELETE FROM orders")
	if err != nil {
		t.Logf("Warning: failed to clean orders: %v", err)
	}
}
