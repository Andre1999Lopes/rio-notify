# Rio Notify

Serviço de notificações em tempo real para chamados de manutenção urbana da Prefeitura do Rio de Janeiro.

## 🚀 Como executar

### Pré-requisitos

- Docker e Docker Compose
- Go 1.24+
- [just](https://github.com/casey/just) (task runner)

### Iniciar

```bash
# Com just
just run

# Sem just
docker compose up
```

A aplicação estará disponível em `http://localhost:8080`.

### Comandos úteis

```bash
just run                  # Sobe todos os serviços
just down                 # Para os serviços
just logs                 # Logs da API
just psql                 # Acessa o PostgreSQL
just redis                # Acessa o Redis
just test                 # Executa os testes
just health               # Health check da API
just restart              # Reinicia toda a aplicação
just restart-clean        # Reinicia Toda a aplicação, limpando os volumes e reconstruindo os containers
just clean                # Derruba a aplicação e limpa os volumes
```

## 📡 Endpoints

### Webhook (Prefeitura → Serviço)

| Método | Rota | Autenticação |
|--------|------|-------------|
| `POST` | `/webhook` | HMAC-SHA256 (`X-Signature-256`) |

Payload esperado:

```json
{
  "chamado_id": "CH-2026-001234",
  "tipo": "status_change",
  "cpf": "12345678901",
  "status_anterior": "em_analise",
  "status_novo": "em_execucao",
  "titulo": "Buraco na Rua — Atualização",
  "descricao": "Equipe designada para reparo na Rua das Laranjeiras, 100",
  "timestamp": "2026-04-23T14:30:00Z"
}
```
O CPF deve ser válido. Caso não seja, a chamada será recusada. O CPF utilizado no README é apenas para exemplificar
o uso.

### API REST (App do Cidadão)

| Método | Rota | Autenticação |
|--------|------|-------------|
| `GET` | `/api/v1/notifications?pagina=1&limite=20` | JWT Bearer |
| `PATCH` | `/api/v1/notifications/:id/read` | JWT Bearer |
| `GET` | `/api/v1/notifications/unread-count` | JWT Bearer |

O token JWT deve conter o CPF do cidadão no campo `preferred_username`.

### WebSocket

| Método | Rota | Autenticação |
|--------|------|-------------|
| `GET` | `/ws?token=<JWT>` | JWT via query param |

A conexão recebe notificações em tempo real sempre que um webhook é processado para o CPF autenticado.

### Health Check

| Método | Rota |
|--------|------|
| `GET` | `/health` |

Retorna o status do serviço, PostgreSQL e Redis.

## 🧪 Testes

```bash
# Testes unitários
just test

# Gerar token JWT para testar a API
just token 12345678901

# Gerar assinatura HMAC para testar o webhook
just hash '{"chamado_id":"CH-001234",tipo":"status_change","cpf":"12345678901","status_anterior":"em_analise","status_novo":"em_execucao","titulo":"Teste","descricao":"Teste","timestamp":"2026-04-23T14:30:00Z"}'

# Testar WebSocket manualmente
wscat -c "ws://localhost:8080/ws?token=<JWT>"
```

## 🏗️ Decisões Técnicas

### pgx em vez de database/sql + lib/pq

O desafio pedia queries SQL diretas sem ORM. Escolhi `pgx/v5` com pool nativo em vez do padrão `database/sql` com `lib/pq`. Oferece melhor performance, suporte nativo a tipos PostgreSQL e pool de conexões integrado.

### Migrations idempotentes com tabela de controle

Em vez de executar migrations manualmente, implementei uma tabela `schema_migrations` que controla quais já foram aplicadas. As migrations rodam automaticamente na inicialização e cada uma executa apenas uma vez, sem risco de duplicação.

### Índice parcial para notificações não lidas

Criei um índice com `WHERE read = FALSE` em vez de um índice composto comum. Como a maioria das notificações eventualmente é lida, o índice parcial é menor e mais rápido para a contagem de não lidas.

### UUID em vez de SERIAL para chave primária

Usei `UUID DEFAULT gen_random_uuid()` em vez de `SERIAL`. Evita colisões em cenários de escalabilidade horizontal e permite gerar IDs sem depender do banco de dados.

### Trigger automático para updated_at

Criei uma função `plpgsql` com trigger `BEFORE UPDATE` para atualizar automaticamente o campo `updated_at`, eliminando a necessidade da aplicação lembrar desse detalhe em cada query.

### Timeout explícito no Redis

Configurei timeout de 5 segundos no `Ping` e nas operações Redis, evitando que a aplicação trave indefinidamente se o Redis estiver indisponível.

### Graceful shutdown ordenado

O servidor HTTP para de aceitar novas requisições primeiro, depois as conexões com banco são fechadas. Isso evita que requisições em andamento falhem por falta de banco durante o desligamento.

### hmac.Equal para comparação segura

Usei `hmac.Equal` em vez de `==` para comparar hashes HMAC, prevenindo timing attacks na validação de assinaturas.

### Docker multi-stage build

Separei o Dockerfile em estágio de build (Go) e estágio de runtime (apenas Alpine + binário), reduzindo o tamanho da imagem final.

### TTL de 5 minutos na idempotência do Redis

As chaves de idempotência expiram automaticamente após 5 minutos, equilibrando proteção contra duplicatas com liberação de memória sem intervenção manual.

### Credenciais Hardcoded

As credenciais no `docker-compose.yml` estão hardcoded **intencionalmente** para facilitar a avaliação do teste técnico. Em um ambiente de produção, seriam utilizados:

- HashiCorp Vault ou AWS Secrets Manager para secrets
- Variáveis de ambiente injetadas via CI/CD
- Rotação automática de credenciais

### Logger Estruturado

Usamos `log/slog` com saída em JSON e remoção automática de campos sensíveis (CPF, webhook secret, JWT token) via `ReplaceAttr`.

## 📁 Estrutura do Projeto

```
cmd/
├── server/           # Entrypoint da aplicação
├── genhash/          # Gerador de assinatura HMAC para testes
├── gentoken/         # Gerador de token JWT para testes
internal/
├── config/           # Configurações via variáveis de ambiente
├── crypto/           # Hash de CPF com SHA256 + Pepper
├── database/         # Conexões PostgreSQL (pgx) e Redis
│   └── migrations/   # Migrations SQL
├── middleware/        # HMAC e JWT
├── notification/     # API REST de notificações
├── webhook/          # Recepção e processamento de webhooks
└── ws/               # WebSocket Hub e Client
pkg/
└── logger/           # Logger estruturado com slog
migrations/           # Arquivos SQL de migração
```

## 🔮 Melhorias Futuras

Com mais tempo, implementaria:

- **Testes de integração** com `testcontainers-go` para PostgreSQL e Redis
- **Testes de carga** com k6 para validar performance sob alto volume de webhooks
- **Circuit Breaker** para PostgreSQL e Redis usando `gobreaker`
- **Dead Letter Queue** no Redis para webhooks que falharam na persistência
- **OpenTelemetry** para tracing distribuído entre webhook, API e WebSocket
- **Rate limiting** no endpoint de webhook por IP
- **Validação completa de CPF** com dígitos verificadores
- **Criptografia AES** no banco para dados sensíveis em vez de apenas hash
- **Manifests Kubernetes** para deploy em cluster
- **Graceful degradation** se Redis estiver indisponível (fallback apenas PostgreSQL)