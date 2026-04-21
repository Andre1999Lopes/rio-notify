## 🔒 Segurança e Credenciais

### Ambiente de Desenvolvimento (Avaliação)

As credenciais no `docker-compose.yml` estão **hardcoded intencionalmente** para:
- Facilitar a avaliação do teste técnico
- Garantir que `docker compose up` funcione sem configuração adicional
- Permitir que o avaliador execute o projeto imediatamente

### Ambiente de Produção (Hipotético)

Em um cenário real de produção, as seguintes práticas seriam implementadas:

| Recurso | Solução Proposta |
|---------|------------------|
| **Secrets** | HashiCorp Vault ou AWS Secrets Manager |
| **Variáveis de Ambiente** | Injetadas via CI/CD (GitHub Actions Secrets) |
| **Rotação de Credenciais** | Automatizada a cada 90 dias |
| **Banco de Dados** | Credenciais efêmeras via Vault Dynamic Secrets |
| **Webhook Secret** | Armazenado em KMS com auditoria de acesso |

### Mitigações já implementadas

Apesar das credenciais hardcoded para avaliação:
- ✅ Serviços **não são expostos** externamente (apenas `localhost`)
- ✅ Rede Docker isolada (`rionotify-network`)
- ✅ Senha do PostgreSQL é apenas para desenvolvimento local
- ✅ CPF dos cidadãos é **sempre anonimizado** com SHA-256 + Pepper