package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}

	body := `{"chamado_id":"CH-2026-001234","tipo":"status_change","cpf":"12345678901","status_anterior":"aberto","status_novo":"em_execucao","titulo":"Buraco na Rua — Atualização","descricao":"Equipe designada para reparo na Rua das Laranjeiras, 100","timestamp":"2026-04-23T14:30:00Z"}`

	if len(os.Args) > 1 {
		body = os.Args[1]
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body))
	hash := hex.EncodeToString(h.Sum(nil))

	fmt.Printf("sha256=%s\n", hash)
}
