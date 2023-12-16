package db

import (
	"database/sql"
	"testing"
)

func TestDatabase(t *testing.T) {
	// BEGIN: TestDatabase
	db := &Database{
		conn: &sql.DB{},
	}

	err := db.conn.Ping()
	if err != nil {
		t.Errorf("Failed to connect to the database: %v", err)
	}
	// END: TestDatabase
}
