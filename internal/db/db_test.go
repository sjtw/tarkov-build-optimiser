package db

import (
	"testing"
)

func TestBuildOptimiserDbConn(t *testing.T) {
	db, err := CreateBuildOptimiserDBClient()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Conn.Ping()
	if err != nil {
		t.Errorf("Failed to connect to the database: %v", err)
	}
}
