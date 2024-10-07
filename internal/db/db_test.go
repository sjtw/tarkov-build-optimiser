package db

import (
	"tarkov-build-optimiser/internal/env"
	"testing"
)

func TestBuildOptimiserDbConn(t *testing.T) {
	environment, err := env.Get()
	if err != nil {
		t.Fatal(err)
	}

	db, err := CreateBuildOptimiserDBClient(environment)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Conn.Ping()
	if err != nil {
		t.Errorf("Failed to connect to the database: %v", err)
	}
}
