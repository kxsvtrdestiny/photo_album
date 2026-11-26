package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInitDatabasePreservesDataAndEnablesForeignKeys(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "album data", "photo-album.db")
	db, err := initDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, "INSERT INTO users (username, password) VALUES (?, ?)", "existing-user", "test-hash")
	closeErr := db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	db, err = initDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var username string
	if err := db.QueryRowContext(ctx, "SELECT username FROM users").Scan(&username); err != nil {
		t.Fatal(err)
	}
	if username != "existing-user" {
		t.Fatalf("existing username = %q", username)
	}

	first, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	second, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	var enabled int
	if err := second.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("foreign_keys on new connection = %d, want 1", enabled)
	}
}
