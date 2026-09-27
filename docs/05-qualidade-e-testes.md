# 5. Qualidade e testes

## 5.1 Estratégia

```mermaid
flowchart TB
    E2E["Verificação ponta a ponta<br/>fluxo completo na interface e na API"]
    HTTP["Testes da camada HTTP<br/>autenticação, perfis, validação, CORS"]
    SVC["Testes de serviço<br/>regras de negócio com repositórios em memória"]
    DOM["Testes de domínio<br/>máquina de estados, validadores"]
    FE["Testes do frontend (Vitest)<br/>regras de transição espelhadas"]

    E2E --- HTTP --- SVC --- DOM
    E2E --- FE
```

| Nível | Onde | O que cobre |
|-------|------|-------------|
| Domínio | `backend/internal/domain/status_test.go` | Todas as transições válidas e inválidas; estados finais; validação de status e prioridade |
| Serviço | `backend/internal/service/*_test.go` | RN-01 a RN-12: criação e validação, visibilidade por perfil, ciclo completo com histórico, transições inválidas, cancelamento pelo solicitante, operações exclusivas do gestor, avaliação, upload, comentários, cadastro/login/token |
| HTTP | `backend/internal/httpapi/router_test.go` | Rotas públicas × protegidas, token inválido, restrição por perfil (403), parâmetros inválidos (400), preflight CORS |
| Frontend | `frontend/src/lib/labels.test.ts` | Transições disponíveis por perfil, estados finais |

Os testes de serviço usam implementações em memória das interfaces de repositório (`fakes_test.go`), por isso rodam em milissegundos e sem banco.

## 5.2 Integração contínua

O workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) roda a cada push e pull request:

1. **backend** — `go vet` e `go test -race`
2. **frontend** — `oxlint`, `vitest` e build de produção (inclui checagem de tipos)
3. **docker** — build das imagens do `docker-compose.yml`

O deploy no Render acontece automaticamente a cada push na `main`.

## 5.3 Verificações ponta a ponta realizadas

Contra um PostgreSQL real (API executando localmente):

- Cadastro, e-mail duplicado (409), login do gestor.
- Criação de ocorrência, upload de imagem e download idêntico em `/uploads/…`; arquivo inexistente → 404.
- Filtros por status, prioridade, categoria e busca com acentuação.
- Dashboard bloqueado para solicitante (403).
- Prioridade, atribuição, transição inválida (422), ciclo Aberta → Em análise → Em atendimento → Resolvida.
- Histórico com 4 registros (criação + 3 transições), cada um com o usuário correto.
- Comentário, avaliação, dashboard com indicadores.
- Cancelamento pelo solicitante sem motivo (422) e com motivo (200).

Em produção (Render):

- `GET /api/health` via proxy do frontend → 200.
- Login inválido → 401 com mensagem da API (confirma a conexão API ↔ banco).
- Rota protegida sem token → 401.
- Rotas do SPA (ex.: `/ocorrencias/1`) servidas pelo `index.html`.

## 5.4 Como executar

```bash
cd backend  && go test ./...
cd frontend && npm test
```
