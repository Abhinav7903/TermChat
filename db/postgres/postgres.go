package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

type Postgres struct {
	DbConn *sql.DB // Capitalized to export the field
}

var once = sync.Once{}

// NewPostgres initializes a singleton instance of Postgres and returns it.
// If connection creation fails, it returns an error.
func NewPostgres() (*Postgres, error) {
	var db *sql.DB
	var err error
	once.Do(func() { db, err = createConnection() })
	if err != nil {
		return nil, err
	}
	return &Postgres{
		DbConn: db, // Use the exported field
	}, nil
}

// createConnection establishes a connection to the PostgreSQL database
// and returns the connection object.
// createConnection establishes a connection to a PostgreSQL database using the connection
// string provided in the "postgresURL" configuration key. It configures the connection pool
// with optional tuning parameters and verifies the connection by pinging the database.
//
// Returns:
//   - *sql.DB: A pointer to the database connection object.
//   - error: An error object if the connection fails or the database is unreachable.
//
// Connection Pool Settings:
//   - MaxIdleConns: Sets the maximum number of idle connections (default: 10).
//   - MaxOpenConns: Sets the maximum number of open connections (default: 50).
//   - ConnMaxLifetime: Sets the maximum lifetime of a connection (default: 0, no limit).
//
// Logs:
//   - Logs a success message with the database URL if the connection is established successfully.
//
// Example:
//
//	db, err := createConnection()
//	if err != nil {
//	    log.Fatalf("Failed to connect to database: %v", err)
//	}
//	defer db.Close()
func createConnection() (*sql.DB, error) {
	postgresURL := viper.GetString("postgres_url")
	if postgresURL == "" {
		// Try fallback to the old key or env var
		postgresURL = viper.GetString("postgresURL")
	}
	if postgresURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL is not set. Please provide it in config or as an environment variable")
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, err
	}

	// Set connection pool limits (optional tuning)
	db.SetMaxIdleConns(10)   // maximum number of idle connections
	db.SetMaxOpenConns(50)   // maximum number of open connections
	db.SetConnMaxLifetime(0) // connection lifetime (set to 0 for no limit)

	// Ping the database to ensure it's reachable
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	slog.Info("Connected to postgres", "URL", postgresURL)
	return db, nil
}

// Close closes the database connection gracefully
func (p *Postgres) Close() error {
	return p.DbConn.Close() // Use the exported field
}
