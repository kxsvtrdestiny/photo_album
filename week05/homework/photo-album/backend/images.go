package main

import (
	"bytes"
	"crypto/rand"
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

	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp"
)

const maxImageBytes = 5 << 20

var imageFilename = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp)$`)
var imageExtensions = map[string]string{"jpeg": ".jpg", "png": ".png", "webp": ".webp"}

func parseUpload(c *gin.Context) bool {
	const maxRequestBytes = maxImageBytes + (64 << 10)
	if c.Request.ContentLength > maxRequestBytes {
		fail(c, http.StatusRequestEntityTooLarge, "request is too large")
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
	if err := c.Request.ParseMultipartForm(maxRequestBytes); err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			fail(c, http.StatusRequestEntityTooLarge, "request is too large")
		} else {
			fail(c, http.StatusBadRequest, "multipart form required")
		}
		return false
	}
	return true
}

func uploadedImage(c *gin.Context, field string) ([]byte, string, bool) {
	files := c.Request.MultipartForm.File[field]
	if len(files) != 1 {
		fail(c, http.StatusBadRequest, "one "+field+" file is required")
		return nil, "", false
	}
	file, err := files[0].Open()
	if err != nil {
		fail(c, http.StatusBadRequest, "cannot read "+field)
		return nil, "", false
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		fail(c, http.StatusBadRequest, "cannot read "+field)
		return nil, "", false
	}
	if len(content) > maxImageBytes {
		fail(c, http.StatusRequestEntityTooLarge, field+" exceeds 5 MiB")
		return nil, "", false
	}
	extension, ok := imageExtension(content)
	if !ok {
		fail(c, http.StatusUnsupportedMediaType, field+" must be a valid JPEG, PNG or WebP up to 16 megapixels")
		return nil, "", false
	}
	return content, extension, true
}

func imageExtension(content []byte) (string, bool) {
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	extension := imageExtensions[format]
	if err != nil || extension == "" || config.Width <= 0 || config.Height <= 0 || config.Width > 16_000_000/config.Height {
		return "", false
	}
	_, _, err = image.Decode(bytes.NewReader(content))
	return extension, err == nil
}

func storeImage(directory, extension string, content []byte) (filename string, discard func(*gin.Context), err error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", nil, err
	}
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", nil, err
	}
	filename = hex.EncodeToString(randomBytes[:]) + extension
	path := filepath.Join(directory, filename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", nil, err
	}
	_, writeErr := file.Write(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return "", nil, errors.Join(err, os.Remove(path))
	}
	return filename, func(c *gin.Context) {
		if err := os.Remove(path); err != nil {
			_ = c.Error(err)
		}
	}, nil
}
