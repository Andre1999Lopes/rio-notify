package unit

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"rio-notify/internal/database"
	"rio-notify/internal/logger"
	"rio-notify/internal/notification"
	"rio-notify/internal/webhook"
)

func setupTestContainer(t *testing.T) (*database.PostgresDB, func()) {
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

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")

	if err != nil {
		t.Fatalf("Falha ao obter connection string: %v", err)
	}

	log := logger.New("error")
	db, err := database.NewPostgresDb(ctx, connStr, log)

	if err != nil {
		t.Fatalf("Falha ao conectar no PostgreSQL: %v", err)
	}

	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("Falha ao executar migrations: %v", err)
	}

	cleanup := func() {
		db.Close()
		pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

func createTestNotification(t *testing.T, db *database.PostgresDB, userHash, callId, status string) {
	t.Helper()
	repository := webhook.NewWebhookRepository(db, logger.New("error"))
	statusOld := "aberto"
	_, err := repository.Create(context.Background(), webhook.CreateNotificationParams{
		UserHash:       userHash,
		CallId:         callId,
		Title:          "Notificação de Teste",
		Description:    "Descrição da notificação de teste",
		StatusOld:      &statusOld,
		StatusNew:      status,
		EventTimestamp: time.Now(),
	})

	if err != nil {
		t.Fatalf("Falha ao criar notificação de teste: %v", err)
	}
}

func TestFindByUserHash(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	userHash := "test-hash-find"

	for i := 1; i <= 3; i++ {
		createTestNotification(t, db, userHash, "TEST-FIND-00"+string(rune('0'+i)), "aberto")
	}

	notifications, total, err := repository.FindByUserHash(ctx, userHash, 1, 10)

	if err != nil {
		t.Errorf("FindByUserHash() error = %v", err)
	}

	if total != 3 {
		t.Errorf("Expected 3, got %d", total)
	}

	if len(notifications) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(notifications))
	}
}

func TestCountUnread(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	userHash := "test-hash-count"
	createTestNotification(t, db, userHash, "TEST-COUNT-001", "aberto")
	count, err := repository.CountUnread(ctx, userHash)

	if err != nil {
		t.Errorf("CountUnread() error = %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestMarkAsRead(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	userHash := "test-hash-read"
	createTestNotification(t, db, userHash, "TEST-READ-001", "aberto")
	notifications, _, _ := repository.FindByUserHash(ctx, userHash, 1, 1)

	if len(notifications) == 0 {
		t.Fatal("Nenhuma notificação criada")
	}

	id := notifications[0].Id
	updated, err := repository.MarkAsRead(ctx, id, userHash)

	if err != nil {
		t.Errorf("MarkAsRead() error = %v", err)
	}

	if !updated {
		t.Error("Expected true")
	}

	count, _ := repository.CountUnread(ctx, userHash)

	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

func TestMarkAsReadWrongUser(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	createTestNotification(t, db, "user-a", "TEST-WRONG-001", "aberto")
	notifications, _, _ := repository.FindByUserHash(ctx, "user-a", 1, 1)

	if len(notifications) == 0 {
		t.Fatal("Nenhuma notificação criada")
	}

	id := notifications[0].Id
	updated, err := repository.MarkAsRead(ctx, id, "user-b")

	if err != nil {
		t.Errorf("MarkAsRead() error = %v", err)
	}

	if updated {
		t.Error("User B should not update User A's notification")
	}
}

func TestFindByUserHashPagination(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	userHash := "test-hash-page"

	for i := 1; i <= 5; i++ {
		createTestNotification(t, db, userHash, "TEST-PAGE-00"+string(rune('0'+i)), "aberto")
		time.Sleep(10 * time.Millisecond)
	}

	notifications, total, err := repository.FindByUserHash(ctx, userHash, 1, 2)

	if err != nil {
		t.Errorf("FindByUserHash() error = %v", err)
	}

	if len(notifications) != 2 {
		t.Errorf("Expected 2, got %d", len(notifications))
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	notifications, total, _ = repository.FindByUserHash(ctx, userHash, 3, 2)

	if len(notifications) != 1 {
		t.Errorf("Expected 1, got %d", len(notifications))
	}
}

func TestFindByUserHashEmptyResult(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	notifications, total, err := repository.FindByUserHash(ctx, "no-user", 1, 10)

	if err != nil {
		t.Errorf("FindByUserHash() error = %v", err)
	}

	if total != 0 {
		t.Errorf("Expected 0, got %d", total)
	}

	if len(notifications) != 0 {
		t.Errorf("Expected empty, got %d", len(notifications))
	}
}

func TestCountUnreadZeroWhenAllRead(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()
	repository := notification.NewNotificationRepository(db, logger.New("error"))
	ctx := context.Background()
	userHash := "test-hash-allread"
	createTestNotification(t, db, userHash, "TEST-ALLREAD-001", "aberto")
	notifications, _, _ := repository.FindByUserHash(ctx, userHash, 1, 1)

	if len(notifications) > 0 {
		repository.MarkAsRead(ctx, notifications[0].Id, userHash)
	}

	count, err := repository.CountUnread(ctx, userHash)

	if err != nil {
		t.Errorf("CountUnread() error = %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}
