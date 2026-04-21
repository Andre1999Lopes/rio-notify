package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"rio-notify/internal/logger"
)

type Hasher struct {
	pepper string
	logger *logger.Logger
}

func NewHasher(pepper string, logger *logger.Logger) (*Hasher, error) {
	if pepper == "" {
		logger.Error("Pepper não pode ser vazio")
		return nil, errors.New("pepper is required")
	}

	logger.Info("Hasher inicializado com sucesso")
	return &Hasher{
		pepper: pepper,
		logger: logger,
	}, nil
}

func (h *Hasher) HashCPF(cpf string) string {
	cpf = cleanCPF(cpf)

	data := cpf + ":" + h.pepper
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

func cleanCPF(cpf string) string {
	re := regexp.MustCompile(`[^0-9]`)
	return re.ReplaceAllString(cpf, "")
}

func (h *Hasher) ValidateCPF(cpf string) bool {
	cleaned := cleanCPF(cpf)

	if len(cleaned) != 11 {
		h.logger.Debug("CPF com tamanho inválido",
			"tamanho", len(cleaned),
			"esperado", 11,
		)
		return false
	}

	if isAllSameDigits(cleaned) {
		h.logger.Debug("CPF com todos dígitos iguais")
		return false
	}

	return true
}

func isAllSameDigits(s string) bool {
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

func MaskCPF(cpf string) string {
	cleaned := cleanCPF(cpf)
	if len(cleaned) != 11 {
		return "***"
	}
	return "***" + cleaned[7:11]
}

func (h *Hasher) VerifyHash(cpf, hash string) bool {
	return h.HashCPF(cpf) == hash
}
