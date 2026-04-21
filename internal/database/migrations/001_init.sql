CREATE TYPE CALL_STATUS AS ENUM('aberto', 'em_analise', 'em_execucao', 'concluido');

CREATE TABLE IF NOT EXISTS notifications (
  id UUID primary key default gen_random_uuid(),
  user_hash VARCHAR(64) NOT NULL,
  call_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  status_old CALL_STATUS NULL,
  status_new CALL_STATUS NOT NULL,
  event_timestamp TIMESTAMPTZ NOT NULL,
  read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_notifications_user_hash_created ON notifications(user_hash, created_at DESC);

CREATE UNIQUE INDEX idx_notifications_idempotency ON notifications (user_hash, call_id, status_new);

CREATE INDEX idx_notifications_unread ON notifications(user_hash, read) WHERE read = FALSE;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_notifications_updated_at
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);