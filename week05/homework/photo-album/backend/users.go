package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/ncruces/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	_ "golang.org/x/image/webp"
)

const maxAvatarBytes = 5 << 20

var avatarFilename = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp)$`)
var errAvatarSize = errors.New("avatar exceeds 5 MiB")
var errAvatarFormat = errors.New("avatar must be a valid JPEG, PNG or WebP up to 16 megapixels")

type user struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

func (app application) register(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarBytes+(64<<10))
	if err := c.Request.ParseMultipartForm(maxAvatarBytes + (64 << 10)); err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			fail(c, http.StatusRequestEntityTooLarge, "request is too large")
		} else {
			fail(c, http.StatusBadRequest, "multipart form required")
		}
		return
	}
	form := c.Request.MultipartForm
	defer form.RemoveAll()
	username := strings.TrimSpace(c.Request.PostFormValue("username"))
	password := c.Request.PostFormValue("password")
	if username == "" || utf8.RuneCountInString(username) > 64 || len(password) < 8 || len(password) > 72 {
		fail(c, http.StatusBadRequest, "username must be 1–64 characters and password 8–72 bytes")
		return
	}
	if len(form.File["avatar"]) != 1 {
		fail(c, http.StatusBadRequest, "one avatar file is required")
		return
	}
	file, err := form.File["avatar"][0].Open()
	if err != nil {
		fail(c, http.StatusBadRequest, "cannot read avatar")
		return
	}
	defer file.Close()
	content, extension, err := readAvatar(file)
	switch {
	case errors.Is(err, errAvatarSize):
		fail(c, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, errAvatarFormat):
		fail(c, http.StatusUnsupportedMediaType, err.Error())
	case err != nil:
		fail(c, http.StatusBadRequest, "cannot read avatar")
	}
	if err != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot register user")
		return
	}
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		fail(c, http.StatusInternalServerError, "cannot store avatar")
		return
	}
	filename := hex.EncodeToString(randomBytes[:]) + extension
	path := filepath.Join(app.uploadDir, "avatars", filename)
	destination, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot store avatar")
		return
	}
	keepFile := false
	defer func() {
		if !keepFile {
			if err := os.Remove(path); err != nil {
				_ = c.Error(err)
			}
		}
	}()
	_, writeErr := destination.Write(content)
	closeErr := destination.Close()
	if writeErr != nil || closeErr != nil {
		fail(c, http.StatusInternalServerError, "cannot store avatar")
		return
	}

	item := user{Username: username, AvatarURL: "/uploads/avatars/" + filename}
	err = app.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO users (username, password, avatar_url) VALUES (?, ?, ?) RETURNING id
	`, username, string(hash), item.AvatarURL).Scan(&item.ID)
	if err != nil {
		var dbError *sqlite3.Error
		if errors.As(err, &dbError) && dbError.ExtendedCode() == sqlite3.CONSTRAINT_UNIQUE {
			fail(c, http.StatusConflict, "username already exists")
		} else {
			_ = c.Error(err)
			fail(c, http.StatusInternalServerError, "cannot register user")
		}
		return
	}
	keepFile = true
	success(c, http.StatusCreated, item)
}

func readAvatar(source io.Reader) ([]byte, string, error) {
	content, err := io.ReadAll(io.LimitReader(source, maxAvatarBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(content) > maxAvatarBytes {
		return nil, "", errAvatarSize
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 16_000_000/config.Height {
		return nil, "", errAvatarFormat
	}
	var extension string
	switch format {
	case "jpeg":
		extension = ".jpg"
	case "png":
		extension = ".png"
	case "webp":
		extension = ".webp"
	default:
		return nil, "", errAvatarFormat
	}
	if _, _, err := image.Decode(bytes.NewReader(content)); err != nil {
		return nil, "", errAvatarFormat
	}
	return content, extension, nil
}

func (app application) avatar(c *gin.Context) {
	filename := c.Param("filename")
	if !avatarFilename.MatchString(filename) {
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
