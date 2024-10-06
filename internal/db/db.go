package db

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog/log"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type Database struct {
	Conn *sql.DB
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func NewDatabase(config Config) (*Database, error) {
	connStrTemplate := "postgresql://%s:%s@%s:%s/%s?sslmode=disable"
	connStr := fmt.Sprintf(connStrTemplate, config.User, config.Password, config.Host, config.Port, config.Name)
	log.Debug().Msgf("Connecting to database: %s:%s", config.Host, config.Port)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Debug().Msg("Connected to database")

	db.SetMaxIdleConns(50)
	db.SetMaxOpenConns(50)

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		Conn: db,
	}, nil
}

func (d *Database) Close() error {
	log.Debug().Msg("Closing database connection")
	return d.Conn.Close()
}
