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

	protected := api.Group("", app.requireAuth)
	protected.POST("/albums", app.createAlbum)
	protected.GET("/albums", app.myAlbums)
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
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		fail(c, http.StatusUnauthorized, "valid Bearer token required")
		return
	}
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(parts[1], claims, func(*jwt.Token) (any, error) {
		return app.jwtKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(tokenIssuer))
	if err != nil || claims.UserID <= 0 {
		fail(c, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	c.Set("userID", claims.UserID)
	c.Next()
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
