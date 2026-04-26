package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"rio-notify/internal/crypto"
	"rio-notify/internal/logger"
	appMiddleware "rio-notify/internal/middleware"
)

func generateTestToken(cpf, secret string) string {
	claims := jwt.MapClaims{
		"sub":                cpf,
		"preferred_username": cpf,
		"name":               "Test User",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func setupAuthTest() (*crypto.Hasher, string) {
	log := logger.New("error")
	hasher, _ := crypto.NewHasher("test-pepper", log)
	secret := "test-jwt-secret"
	return hasher, secret
}

func TestJwtAuthMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher, secret := setupAuthTest()
	log := logger.New("error")
	r := gin.New()
	r.GET("/test", appMiddleware.JwtAuth(secret, hasher, log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestJwtAuthInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher, secret := setupAuthTest()
	log := logger.New("error")
	r := gin.New()
	r.GET("/test", appMiddleware.JwtAuth(secret, hasher, log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestJwtAuthValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher, secret := setupAuthTest()
	log := logger.New("error")
	r := gin.New()
	r.GET("/test", appMiddleware.JwtAuth(secret, hasher, log), func(c *gin.Context) {
		userHash := c.GetString("user_hash")
		if userHash == "" {
			t.Error("user_hash should be set")
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	token := generateTestToken("52998224725", secret)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestJwtAuthExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher, secret := setupAuthTest()
	log := logger.New("error")
	r := gin.New()
	r.GET("/test", appMiddleware.JwtAuth(secret, hasher, log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	claims := jwt.MapClaims{
		"sub":                "12345678901",
		"preferred_username": "12345678901",
		"exp":                time.Now().Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestJwtAuthInvalidCPF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher, secret := setupAuthTest()
	log := logger.New("error")
	r := gin.New()
	r.GET("/test", appMiddleware.JwtAuth(secret, hasher, log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	token := generateTestToken("123", secret)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}
