package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"rio-notify/internal/crypto"
	"rio-notify/internal/logger"
)

func JwtAuth(secret string, hasher *crypto.Hasher, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			log.Warn("Token JWT ausente",
				"path", c.Request.URL.Path,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Token de autorização é obrigatório",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Warn("Formato do header Authorization inválido",
				"header", authHeader,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Formato do header deve ser: Bearer <token>",
			})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				log.Warn("Algoritmo JWT inesperado",
					"algorithm", token.Header["alg"],
				)
				return nil, jwt.ErrSignatureInvalid
			}

			return []byte(secret), nil
		})

		if err != nil {
			log.Warn("Token JWT inválido",
				"error", err,
				"client_ip", c.ClientIP(),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Token inválido ou expirado",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok || !token.Valid {
			log.Warn("Claims do JWT inválidas")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "Dados do token inválidos",
			})
			return
		}

		cpf, ok := claims["preferred_username"].(string)

		if !ok || cpf == "" {
			log.Warn("Campo preferred_username ausente ou inválido no JWT")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "CPF não encontrado no token",
			})
			return
		}

		if !hasher.ValidateCpf(cpf) {
			log.Warn("CPF inválido no token",
				"cpf_masked", crypto.MaskCpf(cpf),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"erro": "CPF inválido no token",
			})
			return
		}

		userHash := hasher.HashCpf(cpf)
		c.Set("user_hash", userHash)
		c.Set("cpf_masked", crypto.MaskCpf(cpf))
		log.Debug("Cidadão autenticado com sucesso",
			"user_hash", userHash[:8]+"...",
			"path", c.Request.URL.Path,
		)
		c.Next()
	}
}

func ValidateJwtToken(tokenString, secret string, hasher *crypto.Hasher) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	cpf, ok := claims["preferred_username"].(string)

	if !ok || cpf == "" {
		return "", jwt.ErrSignatureInvalid
	}

	if !hasher.ValidateCpf(cpf) {
		return "", jwt.ErrSignatureInvalid
	}

	return cpf, nil
}
