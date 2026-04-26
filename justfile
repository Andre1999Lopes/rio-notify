set dotenv-load := true
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

default:
    @just --list

run:
    docker compose up --build -d

restart:
    docker compose down
    docker compose up -d

restart-clean:
    docker compose down -v
    docker compose up --build -d

down:
    docker compose down

clean:
    docker compose down -v

logs:
    docker compose logs api

psql:
    docker compose exec postgres psql -U postgres -d rionotify

redis:
    docker compose exec redis redis-cli

test:
    go test -v ./tests/...

k6-test:
    just --dotenv-path .env _k6-run

_k6-run:
    k6 run tests/load/k6.js

status:
    docker compose ps

token cpf="12345678901":
    go run ./cmd/gentoken/main.go {{cpf}}

hash body='{"chamado_id":"CH-2026-001234","tipo":"status_change","cpf":"12345678901","status_anterior":"aberto","status_novo":"em_execucao","titulo":"Buraco na Rua — Atualização","descricao":"Equipe designada para reparo na Rua das Laranjeiras, 100","timestamp":"2026-04-23T14:30:00Z"}':
    go run ./cmd/genhash/main.go '{{body}}'

health:
    docker compose exec api sh -c "wget -q -O - http://localhost:8080/health 2>/dev/null || curl -s http://localhost:8080/health"