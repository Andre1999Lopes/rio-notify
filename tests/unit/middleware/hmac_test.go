package unit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"rio-notify/internal/logger"
	appMiddleware "rio-notify/internal/middleware"
)

func generateHmac(body string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func TestHmacValidationMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("error")
	r := gin.New()
	r.POST("/webhook", appMiddleware.HmacValidation("secret", log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestHmacValidationInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("error")
	r := gin.New()
	r.POST("/webhook", appMiddleware.HmacValidation("secret", log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	body := `{"test":"data"}`
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256=invalidhash")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestHmacValidationValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("error")
	r := gin.New()
	r.POST("/webhook", appMiddleware.HmacValidation("secret", log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	body := `{"test":"data"}`
	hash := generateHmac(body, "secret")

	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+hash)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHmacValidationDifferentSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("error")
	r := gin.New()
	r.POST("/webhook", appMiddleware.HmacValidation("correct-secret", log), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	body := `{"test":"data"}`
	hash := generateHmac(body, "wrong-secret")

	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", "sha256="+hash)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}
