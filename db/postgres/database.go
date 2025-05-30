package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type Db struct {
	PostgresClient *sql.DB
}

// ConnectDB establishes a connection to the PostgreSQL database
func ConnectDB() *Db {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	var db *sql.DB
	var err error
	maxRetries, _ := strconv.Atoi(os.Getenv("MAX_DB_ATTEMPTS"))
	if maxRetries == 0 {
		maxRetries = 10
	}

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Attempt %d: failed to open database connection: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = db.Ping()
		if err == nil {
			fmt.Println("Connected to PostgreSQL database successfully!")
			return &Db{PostgresClient: db}
		}

		log.Printf("Attempt %d: failed to ping PostgreSQL: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Exceeded max retries. Could not connect to PostgreSQL: %v", err)
	return nil
}

// Stop gracefully closes the PostgreSQL connection
func (db *Db) Stop() {
	if db.PostgresClient != nil {
		err := db.PostgresClient.Close()
		if err != nil {
			log.Printf("Error closing PostgreSQL connection: %v", err)
		} else {
			fmt.Println("PostgreSQL connection closed successfully!")
		}
	}
}

func (db *Db) InitSchema() error {
	schemaPath := filepath.Join("db", "postgres", "schema.sql")
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


// InitSchema creates the necessary tables in the PostgreSQL database
// func (db *Db) InitSchema() error {
// 	schema := fmt.Sprintf(`
// 	CREATE TYPE usage_type AS ENUM ('%s', '%s');
// 	CREATE TYPE discount_type AS ENUM ('%s', '%s');
// 	CREATE TYPE discount_target AS ENUM ('%s', '%s', '%s');

// 	CREATE TABLE IF NOT EXISTS coupons (
// 		coupon_code TEXT PRIMARY KEY,
// 		expiry_date TIMESTAMPTZ NOT NULL,
// 		usage_type usage_type NOT NULL DEFAULT '%s',
// 		applicable_medicine_ids JSONB NOT NULL DEFAULT '[]',
// 		applicable_categories JSONB NOT NULL DEFAULT '[]',
// 		min_order_value DOUBLE PRECISION NOT NULL DEFAULT 0,
// 		valid_start TIMESTAMPTZ NOT NULL,
// 		valid_end TIMESTAMPTZ NOT NULL,
// 		terms_and_conditions TEXT NOT NULL DEFAULT '',
// 		discount_type discount_type NOT NULL DEFAULT '%s',
// 		discount_value DOUBLE PRECISION NOT NULL DEFAULT 0,
// 		max_usage_per_user INTEGER NOT NULL DEFAULT 1,
// 		discount_target discount_target NOT NULL DEFAULT '%s',
// 		max_discount_amount DOUBLE PRECISION NOT NULL DEFAULT 0
// 	);

// 	CREATE TABLE IF NOT EXISTS coupon_usages (
// 		id SERIAL PRIMARY KEY,
// 		user_id TEXT NOT NULL,
// 		coupon_code TEXT NOT NULL REFERENCES coupons(coupon_code) ON DELETE CASCADE,
// 		used_at TIMESTAMPTZ DEFAULT NOW()
// 	);`,
// 		models.UsageTypeSingleUse,
// 		models.UsageTypeMultiUse,
// 		models.DiscountTypeFlat,
// 		models.DiscountTypePercentage,
// 		models.DiscountTargetMedicine,
// 		models.DiscountTargetDelivery,
// 		models.DiscountTargetOrder,
// 		models.UsageTypeSingleUse,
// 		models.DiscountTypeFlat,
// 		models.DiscountTargetOrder)

// 	_, err := db.PostgresClient.Exec(schema)
// 	if err != nil {
// 		return fmt.Errorf("failed to create schema: %w", err)
// 	}

// 	fmt.Println("Database schema initialized successfully.")
// 	return nil
// }

