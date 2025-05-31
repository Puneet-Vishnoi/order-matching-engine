package mockdb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/Puneet-Vishnoi/order-matching-engine/db/postgres"
	providers "github.com/Puneet-Vishnoi/order-matching-engine/db/postgres/providers"
	"github.com/Puneet-Vishnoi/order-matching-engine/repository"
	"github.com/Puneet-Vishnoi/order-matching-engine/service"

	_ "github.com/lib/pq"
)

type TestDeps struct {
	Service        *service.OrderService
	OrderRepo      *repository.OrderRepository
	TradeRepo      *repository.TradeRepository
	PostgresClient *postgres.Db
	Cleanup        func()
}

// ConnectTestDB connects to the test PostgreSQL DB using TEST_POSTGRES_* env vars
func ConnectTestDB() *postgres.Db {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("TEST_POSTGRES_HOST"),
		os.Getenv("TEST_POSTGRES_PORT"),
		os.Getenv("TEST_POSTGRES_USER"),
		os.Getenv("TEST_POSTGRES_PASSWORD"),
		os.Getenv("TEST_POSTGRES_DB"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open test database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to test PostgreSQL DB: %v", err)
	}

	log.Println("Connected to test PostgreSQL database successfully!")
	return &postgres.Db{PostgresClient: db}
}

func InitSchema(db *postgres.Db) error {

	schemaPath := filepath.Join("../../", "db", "postgres", "schema.sql")
	fmt.Println(schemaPath, "scpth")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = db.PostgresClient.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	fmt.Println("Database schema initialized successfully from file.")
	return nil
}

// Returns an instance of initialized test services and clients
func GetTestInstance() *TestDeps {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Failed to load env file: ", err)
	}

	// 1. Connect to Postgres (you can point this to a dedicated test DB)
	pgClient := ConnectTestDB()
	err = InitSchema(pgClient)
	if err != nil {
		log.Fatalf("failed to init test schema: %v", err)
	}

	// 2. Setup Postgres Provider
	dbHelper, err := providers.NewDbProvider(pgClient.PostgresClient)
	if err != nil {
		log.Fatalf("failed to get dbHelper: %v", err)
	}
	orderRepo := repository.NewOrderRepository(dbHelper)
	tradeRepo := repository.NewTradeRepository(dbHelper)

	// 4. Build service
	svc := service.NewOrderService(orderRepo, tradeRepo)

	return &TestDeps{
		Service:        svc,
		OrderRepo:      orderRepo,
		TradeRepo:      tradeRepo,
		PostgresClient: pgClient,
		Cleanup: func() {
			pgClient.Stop()
		},
	}
}
