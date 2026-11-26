package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func (app application) deleteAlbum(c *gin.Context) {
	ctx := c.Request.Context()
	tx, err := app.db.BeginTx(ctx, nil)
	if err != nil {
		photoError(c, err, "cannot delete album")
		return
	}
	defer tx.Rollback()

	item, ok := findAlbum(c, tx)
	if !ok {
		return
	}
	if item.UserID != c.GetInt64("userID") {
		fail(c, http.StatusForbidden, "only the album owner can delete it")
		return
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT p.file_path FROM photos AS p
		WHERE p.album_id = ? AND NOT EXISTS (
			SELECT 1 FROM photos AS other
			WHERE other.file_path = p.file_path AND other.album_id <> p.album_id
		)
	`, item.ID)
	if err != nil {
		photoError(c, err, "cannot delete album")
		return
	}
	defer rows.Close()
	var filenames []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			photoError(c, err, "cannot delete album")
			return
		}
		name, matched := strings.CutPrefix(path, photoURLPrefix)
		if matched && imageFilename.MatchString(name) {
			filenames = append(filenames, name)
		}
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		photoError(c, err, "cannot delete album")
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM photos WHERE album_id = ?", item.ID); err != nil {
		photoError(c, err, "cannot delete album")
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM albums WHERE id = ? AND user_id = ?", item.ID, item.UserID); err != nil {
		photoError(c, err, "cannot delete album")
		return
	}
	if err := tx.Commit(); err != nil {
		photoError(c, err, "cannot delete album")
		return
	}

	cleanupPending := false
	for _, name := range filenames {
		if err := os.Remove(filepath.Join(app.uploadDir, "photos", name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupPending = true
			_ = c.Error(fmt.Errorf("clean up deleted album %d: %w", item.ID, err))
		}
	}
	success(c, http.StatusOK, gin.H{"id": item.ID, "cleanup_pending": cleanupPending})
}
