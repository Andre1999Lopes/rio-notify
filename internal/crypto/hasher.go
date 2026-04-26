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

func (hasher *Hasher) HashCpf(cpf string) string {
	cpf = cleanCpf(cpf)
	data := cpf + ":" + hasher.pepper
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

func cleanCpf(cpf string) string {
	re := regexp.MustCompile(`[^0-9]`)
	return re.ReplaceAllString(cpf, "")
}

func (hasher *Hasher) ValidateCpf(cpf string) bool {
	cleaned := cleanCpf(cpf)

	if len(cleaned) != 11 {
		return false
	}

	if isAllSameDigits(cleaned) {
		return false
	}

	return validateDigits(cleaned)
}

func validateDigits(cpf string) bool {
	sum := 0

	for i := range 9 {
		sum += int(cpf[i]-'0') * (10 - i)
	}

	d1 := (sum * 10) % 11

	if d1 == 10 {
		d1 = 0
	}

	sum = 0

	for i := range 10 {
		sum += int(cpf[i]-'0') * (11 - i)
	}

	d2 := (sum * 10) % 11

	if d2 == 10 {
		d2 = 0
	}

	return int(cpf[9]-'0') == d1 && int(cpf[10]-'0') == d2
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

func MaskCpf(cpf string) string {
	cleaned := cleanCpf(cpf)

	if len(cleaned) != 11 {
		return "***"
	}

	return "***" + cleaned[7:11]
}

func (hasher *Hasher) VerifyHash(cpf, hash string) bool {
	return hasher.HashCpf(cpf) == hash
}
