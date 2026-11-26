package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	apiCookie   = "photo_album_token"
	tokenTTL    = 24 * time.Hour
	tokenIssuer = "photo-album"
)

type application struct {
	db            *sql.DB
	uploadDir     string
	jwtKey        []byte
	secureCookies bool
}

func (app application) routes(router *gin.Engine, api *gin.RouterGroup) {
	api.POST("/register", app.register)
	api.POST("/login", app.login)
	api.POST("/logout", app.logout)
	api.GET("/albums/public", app.publicAlbums)
	router.GET("/uploads/avatars/:filename", app.avatar)
	router.GET("/uploads/photos/:filename", app.servePhoto)
	api.Group("", app.optionalAuth).GET("/albums/:id/photos", app.listPhotos)

	protected := api.Group("", app.requireAuth)
	protected.POST("/albums", app.createAlbum)
	protected.GET("/albums", app.myAlbums)
	protected.POST("/albums/:id/photos", app.uploadPhoto)
}

func success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"code": 0, "message": "ok", "data": data})
}

func fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": status, "message": message, "data": gin.H{}})
}

type tokenClaims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

func (app application) signToken(userID int64) (string, time.Time, error) {
	now := time.Now()
	expiry := now.Add(tokenTTL).Truncate(time.Second)
	claims := tokenClaims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: tokenIssuer, ExpiresAt: jwt.NewNumericDate(expiry), IssuedAt: jwt.NewNumericDate(now),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(app.jwtKey)
	return token, expiry, err
}

func (app application) requireAuth(c *gin.Context) {
	app.bearerAuth(c, true)
}

func (app application) optionalAuth(c *gin.Context) {
	app.bearerAuth(c, false)
}

func (app application) bearerAuth(c *gin.Context, required bool) {
	header := c.GetHeader("Authorization")
	if header == "" && !required {
		c.Next()
		return
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		fail(c, http.StatusUnauthorized, "valid Bearer token required")
		return
	}
	userID := app.tokenUser(parts[1])
	if userID == 0 {
		fail(c, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	c.Set("userID", userID)
	c.Next()
}

func (app application) tokenUser(token string) int64 {
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return app.jwtKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(tokenIssuer))
	if err != nil || claims.UserID <= 0 {
		return 0
	}
	return claims.UserID
}

func (app application) setImageCookie(c *gin.Context, token string, expiry time.Time, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: apiCookie, Value: token, Path: "/uploads/", Expires: expiry, MaxAge: maxAge,
		HttpOnly: true, Secure: app.secureCookies, SameSite: http.SameSiteLaxMode,
	})
}

func (app application) logout(c *gin.Context) {
	app.setImageCookie(c, "", time.Unix(0, 0), -1)
	success(c, http.StatusOK, gin.H{})
}
