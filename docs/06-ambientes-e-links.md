# 6. Ambientes e links

Onde cada parte da aplicação está hospedada e como acessá-la.

## 6.1 Produção (Render)

| Componente | Serviço no Render | URL pública | Painel |
|------------|-------------------|-------------|--------|
| **Frontend** (React) | `resolve-ai-web` — Static Site | https://resolve-ai-web.onrender.com | [dashboard](https://dashboard.render.com/static/srv-dasks5g473hc738p80q0) |
| **Backend / API** (Go) | `resolve-ai-api` — Web Service (Docker) | https://resolve-ai-api-3zs7.onrender.com — health check: [`/api/health`](https://resolve-ai-api-3zs7.onrender.com/api/health) | [dashboard](https://dashboard.render.com/web/srv-dasksf0473hc738p97vg) · [variáveis de ambiente](https://dashboard.render.com/web/srv-dasksf0473hc738p97vg/env) · [logs](https://dashboard.render.com/web/srv-dasksf0473hc738p97vg/logs) |
| **Banco de dados** (PostgreSQL) | `resolve-ai-db` — PostgreSQL (free) · banco `resolveai_xs6d`, usuário `resolveai` | Sem página pública; conectado à API pela variável `DATABASE_URL`. Para acesso externo (DBeaver/pgAdmin/psql), use a *External Database URL* do painel, com SSL | Acessível pelo [Blueprint](https://dashboard.render.com/blueprint/exs-dasks2vpn0mc738s37og) → `resolve-ai-db` |
| **Blueprint** (infraestrutura como código) | Definido em [`render.yaml`](../render.yaml) | — | [dashboard](https://dashboard.render.com/blueprint/exs-dasks2vpn0mc738s37og) |

- Os links de **painel** só funcionam para quem tem acesso à conta/workspace do Render.
- Região: **Virginia (EUA)**. Plano: **gratuito**.
- O frontend encaminha `/api/*` e `/uploads/*` para a API; usuários acessam apenas `resolve-ai-web.onrender.com`.

## 6.2 Código-fonte e integração contínua (GitHub)

| Item | Link |
|------|------|
| Repositório (público) | https://github.com/pauloluedyswe-byte/resolve-ai |
| Execuções do CI (GitHub Actions) | https://github.com/pauloluedyswe-byte/resolve-ai/actions |
| Documentação | https://github.com/pauloluedyswe-byte/resolve-ai/tree/main/docs |

Cada push na branch `main` dispara o CI (lint, testes e build das imagens Docker) e o **deploy automático** no Render.

## 6.3 Acesso à aplicação

| Perfil | Como obter acesso |
|--------|-------------------|
| **Solicitante** | Criar conta em https://resolve-ai-web.onrender.com/cadastro |
| **Gestor** | Criado automaticamente pela API a partir das variáveis `ADMIN_EMAIL` e `ADMIN_PASSWORD` do serviço `resolve-ai-api` ([ver/editar](https://dashboard.render.com/web/srv-dasksf0473hc738p97vg/env)) |

> As credenciais **não** ficam registradas no repositório. Para recuperá-las, consulte as variáveis de ambiente no painel do Render. O gestor só é criado se o e-mail ainda não existir; para criar outro, altere `ADMIN_EMAIL` (com um e-mail ainda não cadastrado) e `ADMIN_PASSWORD` e salve — o Render reinicia a API.

## 6.4 Ambiente local

| Componente | URL |
|------------|-----|
| Frontend (Docker Compose) | http://localhost:3000 |
| Frontend (desenvolvimento, `npm run dev`) | http://localhost:5173 |
| API | http://localhost:8080/api/health |
| PostgreSQL | `localhost:5432` (usuário/banco `resolveai`) |

Instruções em [README principal](../README.md#como-rodar).

## 6.5 Limitações do plano gratuito

| Recurso | Limitação | Impacto |
|---------|-----------|---------|
| API (Web Service free) | Hiberna após ~15 min sem acesso | Primeira requisição após a pausa leva ~50 s |
| PostgreSQL free | **Expira 30 dias após a criação** (criado em 27/09/2026 → expira por volta de **27/10/2026**) | Os dados são apagados; antes da data de entrega, migrar para um plano pago ou recriar o banco |
| Static Site | Sem limitação relevante | — |
