package main

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type album struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	CreatedAt   string `json:"created_at"`
	Creator     *user  `json:"creator,omitempty"`
}

func (app application) createAlbum(c *gin.Context) {
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "valid album JSON required")
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(request.Description) > 2000 {
		fail(c, http.StatusBadRequest, "name must be 1–120 characters and description at most 2000 characters")
		return
	}
	item := album{
		UserID: c.GetInt64("userID"), Name: name,
		Description: request.Description, IsPublic: request.IsPublic,
	}
	err := app.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO albums (user_id, name, description, is_public) VALUES (?, ?, ?, ?)
		RETURNING id, CAST(created_at AS TEXT)
	`, item.UserID, item.Name, item.Description, item.IsPublic).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		_ = c.Error(err)
		fail(c, http.StatusInternalServerError, "cannot create album")
		return
	}
	success(c, http.StatusCreated, item)
}

func (app application) myAlbums(c *gin.Context) {
	app.listAlbums(c, false)
}

func (app application) publicAlbums(c *gin.Context) {
	app.listAlbums(c, true)
}

func (app application) listAlbums(c *gin.Context, public bool) {
	page, pageSize, ok := pagination(c)
	if !ok {
		return
	}

	filter := " WHERE a.user_id = ?"
	args := []any{c.GetInt64("userID")}
	if public {
		filter = " WHERE a.is_public = 1"
		args = nil
	}
	const from = " FROM albums a JOIN users u ON u.id = a.user_id"
	ctx := c.Request.Context()
	tx, err := app.db.BeginTx(ctx, nil)
	if err != nil {
		albumListError(c, err)
		return
	}
	defer tx.Rollback()
	var total int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*)"+from+filter, args...).Scan(&total); err != nil {
		albumListError(c, err)
		return
	}

	items := make([]album, 0)
	if total > 0 && page-1 <= (total-1)/pageSize {
		rows, err := tx.QueryContext(ctx, `
			SELECT a.id, a.user_id, a.name, COALESCE(a.description, ''), COALESCE(a.is_public, 0),
				COALESCE(CAST(a.created_at AS TEXT), ''), u.username, COALESCE(u.avatar_url, '')
		`+from+filter+" ORDER BY a.id DESC LIMIT ? OFFSET ?", append(args, pageSize, (page-1)*pageSize)...)
		if err != nil {
			albumListError(c, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var item album
			var creator user
			if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.Description, &item.IsPublic,
				&item.CreatedAt, &creator.Username, &creator.AvatarURL); err != nil {
				albumListError(c, err)
				return
			}
			if public {
				creator.ID = item.UserID
				item.Creator = &creator
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			albumListError(c, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		albumListError(c, err)
		return
	}
	success(c, http.StatusOK, gin.H{"items": items, "page": page, "page_size": pageSize, "total": total})
}

func albumListError(c *gin.Context, err error) {
	_ = c.Error(err)
	fail(c, http.StatusInternalServerError, "cannot list albums")
}

func pagination(c *gin.Context) (page, pageSize int64, ok bool) {
	page, pageErr := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, sizeErr := strconv.ParseInt(c.DefaultQuery("page_size", "12"), 10, 64)
	if pageErr != nil || sizeErr != nil || page < 1 || pageSize < 1 || pageSize > 100 {
		fail(c, http.StatusBadRequest, "page must be positive and page_size between 1 and 100")
		return 0, 0, false
	}

	return page, pageSize, true
}
