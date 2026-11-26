package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL,
	avatar_url TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS albums (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	description TEXT,
	is_public INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS photos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	album_id INTEGER NOT NULL REFERENCES albums(id),
	file_path TEXT NOT NULL,
	file_size INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func setting(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func initDatabase(ctx context.Context, path string) (*sql.DB, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return nil, err
	}
	db, err := driver.Open(absolutePath, func(conn *sqlite3.Conn) error {
		return conn.Exec("PRAGMA foreign_keys = ON")
	})
	if err != nil {
		return nil, err
	}
	if err := createTables(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func createTables(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return err
	}
	return tx.Commit()
}

func run() error {
	key := os.Getenv("PHOTO_ALBUM_JWT_SECRET")
	if len(key) < 32 {
		return fmt.Errorf("PHOTO_ALBUM_JWT_SECRET must be set to at least 32 bytes")
	}

	uploadDir := setting("PHOTO_ALBUM_UPLOAD_DIR", "uploads")
	if err := os.MkdirAll(filepath.Join(uploadDir, "avatars"), 0o755); err != nil {
		return fmt.Errorf("initialize uploads: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := initDatabase(ctx, setting("PHOTO_ALBUM_DB_PATH", filepath.Join("data", "photo-album.db")))
	cancel()
	if err != nil {
		return fmt.Errorf("initialize SQLite: %w", err)
	}
	defer db.Close()

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return err
	}
	router.Use(gin.Logger(), gin.Recovery())
	api := router.Group("/api/v1")
	app := application{db: db, uploadDir: uploadDir, jwtKey: []byte(key), secureCookies: os.Getenv("PHOTO_ALBUM_COOKIE_SECURE") == "true"}
	app.routes(router, api)
	api.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "database unavailable", "data": gin.H{"status": "unavailable"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "ok"}})
	})

	addr := setting("PHOTO_ALBUM_ADDR", "127.0.0.1:8080")
	log.Printf("Photo album API listening on %s", addr)
	return router.Run(addr)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
