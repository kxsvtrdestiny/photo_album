package main

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/ncruces/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

func (app application) register(c *gin.Context) {
	if !parseUpload(c) {
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	username := strings.TrimSpace(c.Request.PostFormValue("username"))
	password := c.Request.PostFormValue("password")
	if username == "" || utf8.RuneCountInString(username) > 64 || len(password) < 8 || len(password) > 72 {
		fail(c, http.StatusBadRequest, "username must be 1–64 characters and password 8–72 bytes")
		return
	}
	content, extension, ok := uploadedImage(c, "avatar")
	if !ok {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot register user")
		return
	}
	filename, discard, err := storeImage(filepath.Join(app.uploadDir, "avatars"), extension, content)
	if err != nil {
		_ = c.Error(err)
		fail(c, http.StatusInternalServerError, "cannot store avatar")
		return
	}

	item := user{Username: username, AvatarURL: "/uploads/avatars/" + filename}
	err = app.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO users (username, password, avatar_url) VALUES (?, ?, ?) RETURNING id
	`, username, string(hash), item.AvatarURL).Scan(&item.ID)
	if err != nil {
		discard(c)
		var dbError *sqlite3.Error
		if errors.As(err, &dbError) && dbError.ExtendedCode() == sqlite3.CONSTRAINT_UNIQUE {
			fail(c, http.StatusConflict, "username already exists")
		} else {
			_ = c.Error(err)
			fail(c, http.StatusInternalServerError, "cannot register user")
		}
		return
	}
	success(c, http.StatusCreated, item)
}

func (app application) avatar(c *gin.Context) {
	filename := c.Param("filename")
	if !imageFilename.MatchString(filename) {
		fail(c, http.StatusNotFound, "avatar not found")
		return
	}
	var registered int
	err := app.db.QueryRowContext(c.Request.Context(), "SELECT 1 FROM users WHERE avatar_url = ? LIMIT 1",
		"/uploads/avatars/"+filename).Scan(&registered)
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, "avatar not found")
		return
	}
	if err != nil {
		_ = c.Error(err)
		fail(c, http.StatusInternalServerError, "cannot read avatar")
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(filepath.Join(app.uploadDir, "avatars", filename))
}

func (app application) login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&credentials); err != nil {
		fail(c, http.StatusBadRequest, "JSON credentials required")
		return
	}
	var item user
	var hash string
	err := app.db.QueryRowContext(c.Request.Context(), `
		SELECT id, username, COALESCE(avatar_url, ''), password FROM users WHERE username = ?
	`, strings.TrimSpace(credentials.Username)).Scan(&item.ID, &item.Username, &item.AvatarURL, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_ = c.Error(err)
		fail(c, http.StatusInternalServerError, "cannot log in")
		return
	}
	if err != nil || len(credentials.Password) > 72 || bcrypt.CompareHashAndPassword([]byte(hash), []byte(credentials.Password)) != nil {
		fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	token, expiry, err := app.signToken(item.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot log in")
		return
	}
	app.setImageCookie(c, token, expiry, int(time.Until(expiry).Seconds()))
	success(c, http.StatusOK, gin.H{"token": token, "user": item})
}
