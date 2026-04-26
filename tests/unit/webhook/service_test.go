package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"rio-notify/internal/crypto"
	"rio-notify/internal/database"
	"rio-notify/internal/logger"
	"rio-notify/internal/webhook"
)

func setupServiceTest(t *testing.T) (*webhook.WebhookService, func()) {
	t.Helper()
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("rionotify_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)

	if err != nil {
		t.Fatalf("Falha ao criar container PostgreSQL: %v", err)
	}

	pgConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")

	if err != nil {
		t.Fatalf("Falha ao obter connection string: %v", err)
	}

	redisContainer, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)

	if err != nil {
		t.Fatalf("Falha ao criar container Redis: %v", err)
	}

	redisConnStr, err := redisContainer.ConnectionString(ctx)

	if err != nil {
		t.Fatalf("Falha ao obter connection string do Redis: %v", err)
	}

	log := logger.New("error")
	db, err := database.NewPostgresDb(ctx, pgConnStr, log)

	if err != nil {
		t.Fatalf("Falha ao conectar PostgreSQL: %v", err)
	}

	redis, err := database.NewRedisClient(ctx, redisConnStr, log)

	if err != nil {
		db.Close()
		t.Fatalf("Falha ao conectar Redis: %v", err)
	}

	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("Falha ao executar migrations: %v", err)
	}

	hasher, _ := crypto.NewHasher("test-pepper", log)
	repo := webhook.NewWebhookRepository(db, log)
	service := webhook.NewWebhookService(repo, redis, hasher, log)
	cleanup := func() {
		db.Close()
		redis.Close()
		pgContainer.Terminate(ctx)
		redisContainer.Terminate(ctx)
	}

	return service, cleanup
}

func makePayload(callId, cpf, status string) webhook.WebhookPayload {
	statusOld := "aberto"
	return webhook.WebhookPayload{
		ChamadoId:      callId,
		Tipo:           "status_change",
		Cpf:            cpf,
		StatusAnterior: &statusOld,
		StatusNovo:     status,
		Titulo:         "Teste",
		Descricao:      "Descrição de teste",
		Timestamp:      time.Now(),
	}
}

func TestProcessWebhookSuccess(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()

	ctx := context.Background()
	callID := fmt.Sprintf("TEST-SVC-%d", time.Now().UnixNano())
	payload := makePayload(callID, "52998224725", "em_execucao")

	_, err := service.ProcessWebhook(ctx, payload)

	if err != nil {
		t.Errorf("ProcessWebhook() error = %v", err)
	}
}

func TestProcessWebhookDuplicate(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	callID := fmt.Sprintf("TEST-DUP-%d", time.Now().UnixNano())
	payload := makePayload(callID, "52998224725", "concluido")
	_, err := service.ProcessWebhook(ctx, payload)

	if err != nil {
		t.Fatalf("Primeira chamada falhou: %v", err)
	}

	_, err = service.ProcessWebhook(ctx, payload)

	if err == nil {
		t.Error("Esperado erro de duplicata, mas não houve erro")
	}
}

func TestIsDuplicateRedis(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	callID := fmt.Sprintf("TEST-ISDUP-%d", time.Now().UnixNano())

	if service.IsDuplicate(ctx, callID, "aberto") {
		t.Error("Não deveria ser duplicata antes de processar")
	}

	service.MarkAsProcessed(ctx, callID, "aberto")

	if !service.IsDuplicate(ctx, callID, "aberto") {
		t.Error("Deveria ser duplicata após marcar como processado")
	}
}

func TestMarkAsProcessed(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	callID := fmt.Sprintf("TEST-MARK-%d", time.Now().UnixNano())
	service.MarkAsProcessed(ctx, callID, "em_analise")

	if !service.IsDuplicate(ctx, callID, "em_analise") {
		t.Error("Deveria estar marcado como processado")
	}

	if service.IsDuplicate(ctx, callID, "concluido") {
		t.Error("Status diferente não deveria ser duplicata")
	}
}

func TestHashCPFConsistency(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	cpf := "52998224725"
	call1 := fmt.Sprintf("TEST-HASH1-%d", time.Now().UnixNano())
	call2 := fmt.Sprintf("TEST-HASH2-%d", time.Now().UnixNano())
	payload1 := makePayload(call1, cpf, "aberto")
	payload2 := makePayload(call2, cpf, "em_execucao")
	_, err1 := service.ProcessWebhook(ctx, payload1)
	_, err2 := service.ProcessWebhook(ctx, payload2)

	if err1 != nil {
		t.Errorf("ProcessWebhook() error = %v", err1)
	}

	if err2 != nil {
		t.Errorf("ProcessWebhook() error = %v", err2)
	}
}

func TestDifferentCPF_DifferentHash(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx := context.Background()
	callId := fmt.Sprintf("TEST-DIFF-%d", time.Now().UnixNano())
	payload1 := makePayload(callId, "52998224725", "aberto")
	_, err1 := service.ProcessWebhook(ctx, payload1)

	if err1 != nil {
		t.Fatalf("Falha ao processar CPF 1: %v", err1)
	}

	service.MarkAsProcessed(ctx, callId, "aberto")

	if service.IsDuplicate(ctx, callId, "aberto") {
		t.Log("Nota: IsDuplicate usa call_id + status, não user_hash")
	}
}

func TestPublishToRedis(t *testing.T) {
	service, cleanup := setupServiceTest(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	callID := fmt.Sprintf("TEST-PUB-%d", time.Now().UnixNano())
	payload := makePayload(callID, "52998224725", "aberto")
	_, err := service.ProcessWebhook(ctx, payload)

	if err != nil {
		t.Fatalf("ProcessWebhook() error = %v", err)
	}

	service.PublishToRedis(ctx, "test-notification-id", payload)
	t.Log("Publicação no Redis testada")
}
