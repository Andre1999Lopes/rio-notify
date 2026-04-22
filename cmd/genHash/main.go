package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func main() {
	body := `{"chamado_id":"CH-2024-001234","tipo":"status_change","cpf":"12345678901","status_anterior":"em_analise","status_novo":"em_execucao","titulo":"Buraco na Rua — Atualização","descricao":"Equipe designada para reparo na Rua das Laranjeiras, 100","timestamp":"2024-11-15T14:30:00Z"}`

	secret := "dev-secret-change-me"

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(body))
	hash := hex.EncodeToString(h.Sum(nil))

	fmt.Printf("sha256=%s\n", hash)
}
