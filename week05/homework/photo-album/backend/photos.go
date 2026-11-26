package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

const photoURLPrefix = "/uploads/photos/"

type photo struct {
	ID        int64  `json:"id"`
	AlbumID   int64  `json:"album_id"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`
	CreatedAt string `json:"created_at"`
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findAlbum(c *gin.Context, db rowQuerier) (album, bool) {
	var item album
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid album ID")
		return item, false
	}
	err = db.QueryRowContext(c.Request.Context(), `
		SELECT id, user_id, name, COALESCE(description, ''), COALESCE(is_public, 0),
			COALESCE(CAST(created_at AS TEXT), '') FROM albums WHERE id = ?
	`, id).Scan(&item.ID, &item.UserID, &item.Name, &item.Description, &item.IsPublic, &item.CreatedAt)
	return item, foundPhotoResource(c, err, "album")
}

func foundPhotoResource(c *gin.Context, err error, resource string) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, resource+" not found")
	} else {
		photoError(c, err, "cannot read "+resource)
	}
	return false
}

func photoError(c *gin.Context, err error, message string) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, message)
}

func canReadAlbum(c *gin.Context, owner, viewer int64, public bool) bool {
	switch {
	case public, viewer > 0 && owner == viewer:
		return true
	case viewer == 0:
		fail(c, http.StatusUnauthorized, "login required for private album")
	default:
		fail(c, http.StatusForbidden, "private album belongs to another user")
	}
	return false
}

func (app application) uploadPhoto(c *gin.Context) {
	item, ok := findAlbum(c, app.db)
	if !ok {
		return
	}
	if c.GetInt64("userID") != item.UserID {
		fail(c, http.StatusForbidden, "only the album owner can upload photos")
		return
	}
	if !parseUpload(c) {
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	content, extension, ok := uploadedImage(c, "photo")
	if !ok {
		return
	}
	filename, discard, err := storeImage(filepath.Join(app.uploadDir, "photos"), extension, content)
	if err != nil {
		photoError(c, err, "cannot store photo")
		return
	}
	result := photo{AlbumID: item.ID, FilePath: photoURLPrefix + filename, FileSize: int64(len(content))}
	err = app.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO photos (album_id, file_path, file_size) VALUES (?, ?, ?)
		RETURNING id, CAST(created_at AS TEXT)
	`, result.AlbumID, result.FilePath, result.FileSize).Scan(&result.ID, &result.CreatedAt)
	if err != nil {
		discard(c)
		photoError(c, err, "cannot save photo record")
		return
	}
	success(c, http.StatusCreated, result)
}

func (app application) listPhotos(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	page, pageSize, ok := pagination(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	tx, err := app.db.BeginTx(ctx, nil)
	if err != nil {
		photoError(c, err, "cannot list photos")
		return
	}
	defer tx.Rollback()
	item, ok := findAlbum(c, tx)
	viewer := c.GetInt64("userID")
	if !ok || !canReadAlbum(c, item.UserID, viewer, item.IsPublic) {
		return
	}
	var total int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM photos WHERE album_id = ?", item.ID).Scan(&total); err != nil {
		photoError(c, err, "cannot list photos")
		return
	}
	items := make([]photo, 0)
	if total > 0 && page-1 <= (total-1)/pageSize {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, album_id, file_path, COALESCE(file_size, 0), COALESCE(CAST(created_at AS TEXT), '')
			FROM photos WHERE album_id = ? ORDER BY id DESC LIMIT ? OFFSET ?
		`, item.ID, pageSize, (page-1)*pageSize)
		if err != nil {
			photoError(c, err, "cannot list photos")
			return
		}
		defer rows.Close()
		for rows.Next() {
			var p photo
			if err := rows.Scan(&p.ID, &p.AlbumID, &p.FilePath, &p.FileSize, &p.CreatedAt); err != nil {
				photoError(c, err, "cannot list photos")
				return
			}
			items = append(items, p)
		}
		if err := rows.Err(); err != nil {
			photoError(c, err, "cannot list photos")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		photoError(c, err, "cannot list photos")
		return
	}
	success(c, http.StatusOK, gin.H{
		"album": item, "can_upload": viewer > 0 && viewer == item.UserID,
		"items": items, "page": page, "page_size": pageSize, "total": total,
	})
}

func (app application) servePhoto(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	filename := c.Param("filename")
	if !imageFilename.MatchString(filename) {
		fail(c, http.StatusNotFound, "photo not found")
		return
	}

	var owner int64
	var public bool
	err := app.db.QueryRowContext(c.Request.Context(), `
		SELECT a.user_id, COALESCE(a.is_public, 0)
		FROM photos p JOIN albums a ON a.id = p.album_id
		WHERE p.file_path = ?
	`, photoURLPrefix+filename).Scan(&owner, &public)
	if !foundPhotoResource(c, err, "photo") {
		return
	}
	if !public {
		token, _ := c.Cookie(apiCookie)
		if !canReadAlbum(c, owner, app.tokenUser(token), false) {
			return
		}
	}

	file, err := os.Open(filepath.Join(app.uploadDir, "photos", filename))
	if errors.Is(err, os.ErrNotExist) {
		fail(c, http.StatusNotFound, "photo not found")
		return
	}
	if err != nil {
		photoError(c, err, "cannot open photo")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		photoError(c, err, "cannot read photo")
		return
	}
	if !info.Mode().IsRegular() {
		fail(c, http.StatusNotFound, "photo not found")
		return
	}
	http.ServeContent(c.Writer, c.Request, filename, info.ModTime(), file)
}
