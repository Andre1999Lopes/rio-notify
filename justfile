set dotenv-load := true
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

default:
    @just --list

run:
    docker compose up --build -d

restart:
    docker compose down
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
    go test -v ./...

fmt:
    go fmt ./...

tidy:
    go mod tidy

status:
    docker compose ps

health:
    docker compose exec api sh -c "wget -q -O - http://localhost:8080/health 2>/dev/null || curl -s http://localhost:8080/health"