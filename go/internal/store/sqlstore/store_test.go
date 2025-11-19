package sqlstore

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

var testStore *SQLStore

func SetupTestDB(t *testing.T) {
	var err error
	testStore, err = New("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
}

func TeardownTestDB() {
	testStore.db.Close()
}
