package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type Database struct {
	conn *sql.DB
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func NewDatabase(config Config) (*Database, error) {
	connStr := "postgresql://%s:%s@%s/%s?sslmode=disable"
	db, err := sql.Open("postgres", fmt.Sprintf(connStr, config.User, config.Password, config.Host, config.Port, config.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Database{
		conn: db,
	}, nil
}

func (d *Database) Close() error {
	return d.conn.Close()
}
