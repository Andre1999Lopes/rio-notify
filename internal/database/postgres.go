package database

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rio-notify/internal/logger"
)

type PostgresDB struct {
	*pgxpool.Pool
	logger *logger.Logger // ✅ Seu logger personalizado
}

type PostgresConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func DefaultConfig() PostgresConfig {
	return PostgresConfig{
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   2 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}
}

func NewPostgresDB(ctx context.Context, databaseURL string, log *logger.Logger) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Error("Falha ao fazer parse da URL do banco", "error", err, "url", databaseURL)
		return nil, errors.New("failed to parse database URL")
	}

	poolConfig := DefaultConfig()
	config.MaxConns = poolConfig.MaxConns
	config.MinConns = poolConfig.MinConns
	config.MaxConnLifetime = poolConfig.MaxConnLifetime
	config.MaxConnIdleTime = poolConfig.MaxConnIdleTime
	config.HealthCheckPeriod = poolConfig.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Error("Falha ao criar pool de conexões", "error", err)
		return nil, errors.New("failed to create connection pool")
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Error("Falha ao pingar banco de dados", "error", err)
		return nil, errors.New("failed to ping database")
	}

	log.Info("PostgreSQL conectado com sucesso",
		"max_conns", config.MaxConns,
		"min_conns", config.MinConns,
	)

	return &PostgresDB{
		Pool:   pool,
		logger: log,
	}, nil
}

func (db *PostgresDB) Close() {
	db.Pool.Close()
	db.logger.Info("Pool PostgreSQL fechado")
}

func (db *PostgresDB) Health(ctx context.Context) error {
	if err := db.Pool.Ping(ctx); err != nil {
		db.logger.Error("Health check do PostgreSQL falhou", "error", err)
		return errors.New("database health check failed")
	}
	return nil
}

//go:embed migrations/*.sql
var migrationFiles embed.FS

func RunMigrations(ctx context.Context, db *PostgresDB) error {
	if err := createMigrationsTable(ctx, db); err != nil {
		db.logger.Error("Falha ao criar tabela de controle de migrations", "error", err)
		return errors.New("failed to create migrations table")
	}

	files, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		db.logger.Error("Falha ao ler diretório de migrations", "error", err)
		return errors.New("failed to read migrations directory")
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	executed := 0
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := file.Name()

		alreadyExecuted, err := isMigrationExecuted(ctx, db, version)
		if err != nil {
			db.logger.Error("Falha ao verificar status da migration",
				"version", version,
				"error", err,
			)
			return errors.New("failed to check migration status")
		}

		if alreadyExecuted {
			db.logger.Debug("Migration já executada, pulando", "version", version)
			continue
		}

		content, err := migrationFiles.ReadFile("migrations/" + version)
		if err != nil {
			db.logger.Error("Falha ao ler arquivo de migration",
				"version", version,
				"error", err,
			)
			return errors.New("failed to read migration file")
		}

		if err := executeMigration(ctx, db, version, string(content)); err != nil {
			db.logger.Error("Falha ao executar migration",
				"version", version,
				"error", err,
			)
			return errors.New("failed to execute migration")
		}

		db.logger.Info("Migration executada com sucesso", "version", version)
		executed++
	}

	if executed > 0 {
		db.logger.Info("Migrations concluídas", "executed", executed)
	} else {
		db.logger.Info("Nenhuma migration pendente")
	}

	return nil
}

func createMigrationsTable(ctx context.Context, db *PostgresDB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.Exec(ctx, query)
	if err != nil {
		db.logger.Error("Falha ao executar query de criação da tabela schema_migrations", "error", err)
		return err
	}
	return nil
}

func isMigrationExecuted(ctx context.Context, db *PostgresDB, version string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM schema_migrations WHERE version = $1`
	err := db.QueryRow(ctx, query, version).Scan(&count)
	if err != nil {
		db.logger.Error("Falha ao consultar schema_migrations",
			"version", version,
			"error", err,
		)
		return false, err
	}
	return count > 0, nil
}

func executeMigration(ctx context.Context, db *PostgresDB, version, sqlContent string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		db.logger.Error("Falha ao iniciar transação para migration",
			"version", version,
			"error", err,
		)
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, sqlContent); err != nil {
		db.logger.Error("Falha ao executar SQL da migration",
			"version", version,
			"error", err,
		)
		return err
	}

	insertQuery := `INSERT INTO schema_migrations (version) VALUES ($1)`
	if _, err := tx.Exec(ctx, insertQuery, version); err != nil {
		db.logger.Error("Falha ao registrar migration executada",
			"version", version,
			"error", err,
		)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		db.logger.Error("Falha ao commitar transação da migration",
			"version", version,
			"error", err,
		)
		return err
	}

	return nil
}

func (db *PostgresDB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		db.logger.Error("Falha ao iniciar transação", "error", err)
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		db.logger.Error("Falha ao executar função em transação", "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		db.logger.Error("Falha ao commitar transação", "error", err)
		return err
	}

	return nil
}

var ErrNotFound = errors.New("record not found")
