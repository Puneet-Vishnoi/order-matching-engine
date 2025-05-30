package providers

import (
	"database/sql"
	"fmt"
)

type DBHelper struct {
	PostgresClient *sql.DB
}

func NewDbProvider(postgresDBClient *sql.DB) (*DBHelper, error) {
	if postgresDBClient == nil {
		return nil, fmt.Errorf("invalid postgres client: nil pointer provided")
	}
	return &DBHelper{PostgresClient: postgresDBClient}, nil
}
 
