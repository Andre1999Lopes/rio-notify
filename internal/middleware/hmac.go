package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"rio-notify/internal/logger"

	"github.com/gin-gonic/gin"
)

func HmacValidation(secret string, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		signatureHeader := c.GetHeader("X-Signature-256")

		if signatureHeader == "" {
			log.Warn("Webhook sem assinatura",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Header de assinatura faltando",
			})
			return
		}

		if !strings.HasPrefix(signatureHeader, "sha256=") {
			log.Warn("Formato de assinatura inválido",
				"header", signatureHeader,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Formato de assinatura inválido",
			})
			return
		}

		receivedHash := strings.TrimPrefix(signatureHeader, "sha256=")

		if len(receivedHash) != 64 {
			log.Warn("Hash com tamanho inválido",
				"length", len(receivedHash),
				"expected", 64,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Hash com tamanho inválido",
			})
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)

		if err != nil {
			log.Error("Falha ao ler body do webhook",
				"erro", err,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"erro": "Falha ao ler corpo da requisição",
			})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(bodyBytes)
		expectedHash := hex.EncodeToString(h.Sum(nil))
		log.Warn("🔍 DEBUG HMAC",
			"received", receivedHash,
			"expected", expectedHash,
			"body_string", string(bodyBytes),
			"body_length", len(bodyBytes),
			"secret", secret,
		)

		if !hmac.Equal([]byte(receivedHash), []byte(expectedHash)) {
			log.Warn("Assinatura HMAC inválida",
				"path", c.Request.URL.Path,
				"client_ip", c.ClientIP(),
				"received_prefix", receivedHash[:8]+"...",
				"expected_prefix", expectedHash[:8]+"...",
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Assinatura HMAC inválida",
			})
			return
		}

		log.Debug("Assinatura HMAC validada com sucesso",
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
		)

		c.Next()
	}
}
