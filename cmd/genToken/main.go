package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		secret = "dev-jwt-secret"
	}

	cpf := "12345678901"

	if len(os.Args) > 1 {
		cpf = os.Args[1]
	}

	claims := jwt.MapClaims{
		"sub":                cpf,
		"preferred_username": cpf,
		"name":               "Nome Sobrenome",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))

	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Token JWT para CPF %s:\n\n", cpf)
	fmt.Printf("Authorization: Bearer %s\n\n", tokenString)
	fmt.Printf("Comando curl:\n")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/api/v1/notifications\n", tokenString)
}
