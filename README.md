# Go Order Matching System

A high-performance order matching engine built with Go, PostgreSQL, and REST API architecture. This system provides real-time order matching capabilities for trading applications with support for limit orders, market orders, and trade execution.

## 🏗️ Architecture

```
order-matching-system/
├── db/postgres/                        # PostgreSQL DB setup
│   ├── provider/providers.go
│   └── database.go
├── cmd/app/                            # Application entry point
│   └── main.go
├── handlers/                           # HTTP handlers
│   ├── handler.go
├── service/                            # Core business logic
│   ├── order_service.go
│   └── matching_engine.go
├── repository/                         # Database queries (raw SQL)
│   ├── order_repo.go
│   └── trade_repo.go
├── models/                             # Data models
│   ├── order.go
│   ├── trade.go
│   ├── request.go
│   └── response.go
├── routes/                             # API routes
│   └── router.go
├── tests/                              # Test coverage
│   ├── mockdb/connection.go
│   ├── unittest/unit_test.go
│   └── integration/integration_test.go
├── Dockerfile                          # Docker configuration
├── docker-compose.yml                  # Docker Compose setup
├── go.mod / go.sum                     # Go module files
└── README.md                           # This file
```

## 🚀 Features

- **Real-time Order Matching**: High-performance matching engine with price-time priority
- **Multiple Order Types**: Support for limit orders and market orders
- **RESTful API**: Clean API endpoints for order management and trade tracking
- **PostgreSQL Integration**: Robust data persistence with raw SQL queries
- **Graceful Shutdown**: Proper server lifecycle management
- **Docker Support**: Containerized deployment with Docker Compose
- **Test Coverage**: Unit and integration tests with mock database
- **Order Book Management**: Real-time order book viewing and management

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin HTTP framework
- **Database**: PostgreSQL 15
- **Containerization**: Docker & Docker Compose
- **Testing**: Go testing package with mock database
- **Environment**: dotenv for configuration management

## 📋 Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- PostgreSQL 15 (if running locally without Docker)

## ⚡ Quick Start

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/Puneet-Vishnoi/order-matching-engine.git
   cd order-matching-engine
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start the application**
   ```bash
   docker-compose up --build
   ```

4. **Access the API**
   - Application: http://localhost:8080
   - Health check: http://localhost:8080/api/orderbook

### Local Development

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Set up PostgreSQL**
   ```bash
   # Start PostgreSQL service
   docker run --name postgres-order-matching \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=Puneet \
     -e POSTGRES_DB=order-matching-engine \
     -p 5432:5432 -d postgres:15
   ```

3. **Run the application**
   ```bash
   go run cmd/app/main.go
   ```

## 🔗 API Endpoints

### Orders

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | Place a new order |
| DELETE | `/api/orders/:id` | Cancel an existing order |
| GET | `/api/orders/:id` | Get order status |
| GET | `/api/orderbook` | Get current order book |

### Trades

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/trades` | List all trades |

### Request/Response Examples

**Place Order Request:**
```json
{
  "symbol": "BTCUSD",
  "side": "buy",
  "type": "limit",
  "quantity": 1.5,
  "price": 45000.00
}
```

**Order Response:**
```json
{
  "id": "uuid-order-id",
  "symbol": "BTCUSD",
  "side": "buy",
  "type": "limit",
  "quantity": 1.5,
  "price": 45000.00,
  "status": "open",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

**Order Book Response:**
```json
{
  "symbol": "BTCUSD",
  "bids": [
    {"price": 44999.00, "quantity": 2.5},
    {"price": 44998.00, "quantity": 1.0}
  ],
  "asks": [
    {"price": 45001.00, "quantity": 1.5},
    {"price": 45002.00, "quantity": 3.0}
  ]
}
```

## 🧪 Testing

### Run Unit Tests
```bash
go test ./tests/unittest/... -v
```

### Run Integration Tests
```bash
# Start test database
docker-compose up test-postgres -d

# Run integration tests
go test ./tests/integration/... -v
```

### Run All Tests
```bash
go test ./... -v
```

## 🔧 Configuration

Environment variables configuration:

```bash
# Application
PORT=8080
APP_ENV=development

# PostgreSQL (Main)
POSTGRES_USER=postgres
POSTGRES_PASSWORD=Puneet
POSTGRES_DB=order-matching-engine
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DSN=postgres://postgres:Puneet@postgres:5432/order-matching-engine?sslmode=disable

# PostgreSQL (Test DB)
TEST_POSTGRES_USER=test_user
TEST_POSTGRES_PASSWORD=test_pass
TEST_POSTGRES_DB=order-matching-test
TEST_POSTGRES_HOST=test-postgres
TEST_POSTGRES_PORT=5432
TEST_POSTGRES_DSN=postgres://test_user:test_pass@test-postgres:5432/order-matching-test?sslmode=disable

# Database Configuration
MAX_DB_ATTEMPTS=5
```

## 🐳 Docker Usage

### Build Image
```bash
docker build -t order-matching-engine .
```

### Run Container
```bash
docker run -p 8080:8080 --env-file .env order-matching-engine
```

### Docker Compose Services
- **app**: Main application service
- **postgres**: Primary PostgreSQL database
- **test-postgres**: Test database for integration tests

## 📊 Monitoring & Health Checks

The application includes built-in health checks:

- **Docker Health Check**: Automatically monitors application health
- **Graceful Shutdown**: Handles SIGINT and SIGTERM signals properly
- **Database Connection Monitoring**: Tracks database connection status

## 🔀 Order Matching Logic

The matching engine implements a **price-time priority** algorithm:

1. **Price Priority**: Better prices are matched first
2. **Time Priority**: Earlier orders are matched first among same-price orders
3. **Partial Fills**: Large orders can be partially filled by multiple smaller orders
4. **Immediate Execution**: Market orders execute immediately at best available price

### Matching Examples

**Scenario 1: Exact Match**
- Buy Order: 1.0 BTC at $45,000
- Sell Order: 1.0 BTC at $45,000
- Result: Complete fill for both orders

**Scenario 2: Partial Fill**
- Buy Order: 2.0 BTC at $45,000
- Sell Order: 1.0 BTC at $45,000
- Result: Sell order completely filled, buy order partially filled (1.0 BTC remaining)

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

For questions, issues, or contributions:

- **GitHub Issues**: [Create an issue](https://github.com/Puneet-Vishnoi/order-matching-engine)
- **Email**: puneetvishnoi2017@gmail.com.com
- **LinkedIn**: [Puneet Vishnoi](https://www.linkedin.com/in/puneetvishnoi2017)

## 🙏 Acknowledgments

- Go community for excellent tooling and libraries
- PostgreSQL team for robust database system
- Gin framework contributors for lightweight HTTP framework

---

**Built with ❤️ by [Puneet Vishnoi](https://github.com/Puneet-Vishnoi)**