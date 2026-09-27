# Resolve Aí — Plataforma de Gestão de Ocorrências

MVP Full Stack do Hackathon POSTECH FSDT (Fase 5). Usuários registram ocorrências (iluminação, vazamentos, limpeza, segurança…) e acompanham todo o processo até a resolução; gestores analisam, priorizam, atribuem e resolvem.

| Camada    | Tecnologia                                              |
|-----------|---------------------------------------------------------|
| Backend   | Go 1.27 · chi · pgx · JWT (HS256) · bcrypt              |
| Banco     | PostgreSQL 17 (migrações SQL embutidas no binário)      |
| Frontend  | React 19 · TypeScript · Vite · React Router             |
| Testes    | `go test` (domínio, serviços, HTTP) · Vitest            |
| Infra     | Docker · Docker Compose · Nginx · GitHub Actions (CI)   |

## Como rodar

### Tudo com Docker (recomendado)

```bash
cp .env.example .env          # opcional: ajuste segredos
docker compose up --build
```

- Frontend: http://localhost:3000
- API: http://localhost:8080/api/health
- Gestor inicial: `gestor@resolveai.com` / `gestor123` (definido por `ADMIN_EMAIL`/`ADMIN_PASSWORD`)
- Solicitantes criam a própria conta pela tela de cadastro.

### Desenvolvimento local

```bash
# 1. Banco
docker compose up -d db

# 2. Backend (porta 8080)
cd backend
JWT_SECRET=dev-secret-com-16-chars ADMIN_EMAIL=gestor@resolveai.com ADMIN_PASSWORD=gestor123 go run ./cmd/api

# 3. Frontend (porta 5173, com proxy de /api e /uploads para :8080)
cd frontend
npm install
npm run dev
```

### Testes

```bash
cd backend  && go test ./...
cd frontend && npm test
```

## Arquitetura

```
resolve-ai/
├── backend/
│   ├── cmd/api/              # main: config, conexão, migração, seed do gestor, servidor HTTP
│   └── internal/
│       ├── domain/           # entidades, máquina de estados, erros de domínio
│       ├── service/          # regras de negócio e permissões (independente de HTTP/SQL)
│       ├── repository/       # PostgreSQL (pgx) e armazenamento de imagens
│       ├── httpapi/          # rotas chi, middlewares (auth, papel, CORS), handlers
│       ├── database/         # conexão + migrações em migrations/*.sql
│       └── config/           # variáveis de ambiente
├── frontend/
│   └── src/
│       ├── lib/              # cliente da API, tipos, rótulos/regras de transição
│       ├── auth/             # contexto de autenticação (JWT em localStorage)
│       ├── components/       # layout, badges
│       └── pages/            # login/cadastro, lista, nova, detalhe, dashboard
└── docker-compose.yml
```

Arquitetura em camadas (**handler → service → repository**). Os serviços dependem de interfaces (`service/ports.go`), o que permite testar as regras de negócio com repositórios em memória, sem banco.

### Ciclo de vida da ocorrência

```
Aberta ──► Em análise ──► Em atendimento ──► Resolvida ──► Avaliação (1–5)
   │            │                │
   └────────────┴────────────────┴──► Cancelada
```

- Toda mudança de status grava em `status_history`: **status anterior, novo status, data/hora, usuário responsável e observação** — na mesma transação da alteração.
- A atualização usa controle otimista (`WHERE status = <anterior>`): duas mudanças concorrentes não geram histórico inconsistente (a segunda recebe `409`).
- Resolver exige solução aplicada registrada; cancelar exige motivo.
- O solicitante só pode cancelar a própria ocorrência enquanto ela está **Aberta**; demais transições são do gestor.

### Perfis

| Solicitante                                   | Gestor                                                    |
|-----------------------------------------------|-----------------------------------------------------------|
| Criar conta / autenticar-se                   | Visualizar todas as ocorrências                           |
| Registrar ocorrência (título, descrição, categoria, localização, imagem) | Filtrar por categoria, status e prioridade |
| Acompanhar andamento e histórico              | Alterar prioridade · atribuir responsável                 |
| Comentar                                      | Atualizar status · comentar · registrar solução           |
| Avaliar a resolução                           | Dashboard de indicadores                                  |

Solicitantes só enxergam as próprias ocorrências (ocorrências de terceiros retornam `404`).

### Modelo de dados

`users` · `categories` · `occurrences` · `comments` · `status_history` — ver [`001_init.sql`](backend/internal/database/migrations/001_init.sql).

## API

Todas as rotas (exceto `health`, `register`, `login`) exigem `Authorization: Bearer <token>`.

| Método | Rota                                  | Quem        | Descrição                                   |
|--------|---------------------------------------|-------------|---------------------------------------------|
| GET    | `/api/health`                         | público     | Health check                                |
| POST   | `/api/auth/register`                  | público     | `{name, email, password}` → cria solicitante |
| POST   | `/api/auth/login`                     | público     | `{email, password}` → `{token, user}`       |
| GET    | `/api/auth/me`                        | autenticado | Usuário atual                               |
| GET    | `/api/categories`                     | autenticado | Categorias                                  |
| GET    | `/api/occurrences?status=&priority=&category_id=&q=` | autenticado | Lista (solicitante: só as suas) |
| POST   | `/api/occurrences`                    | autenticado | `{title, description, category_id, location}` |
| GET    | `/api/occurrences/{id}`               | dono/gestor | Detalhe + comentários + histórico           |
| POST   | `/api/occurrences/{id}/image`         | dono/gestor | multipart `image` (JPG/PNG/WEBP/GIF, ≤ 5 MB) |
| POST   | `/api/occurrences/{id}/comments`      | dono/gestor | `{body}`                                    |
| PATCH  | `/api/occurrences/{id}/status`        | gestor / dono (cancelar) | `{status, note, solution?}`    |
| POST   | `/api/occurrences/{id}/rating`        | dono        | `{rating: 1-5, comment}` (só resolvida)     |
| PATCH  | `/api/occurrences/{id}/priority`      | gestor      | `{priority: baixa\|media\|alta\|critica}`   |
| PATCH  | `/api/occurrences/{id}/assignee`      | gestor      | `{assignee_id \| null}`                     |
| PATCH  | `/api/occurrences/{id}/solution`      | gestor      | `{solution}`                                |
| GET    | `/api/users/gestores`                 | gestor      | Possíveis responsáveis                      |
| GET    | `/api/dashboard`                      | gestor      | Totais, por status/prioridade/categoria, tempo médio de resolução, satisfação média |

Erros seguem o formato `{"error": "mensagem"}` com `400`, `401`, `403`, `404`, `409` ou `422`.

## Variáveis de ambiente (backend)

| Variável         | Padrão                         | Descrição                              |
|------------------|--------------------------------|----------------------------------------|
| `PORT`           | `8080`                         |                                        |
| `DATABASE_URL`   | `postgres://resolveai:resolveai@localhost:5432/resolveai?sslmode=disable` | |
| `JWT_SECRET`     | — (obrigatório, ≥ 16 chars)    | Chave de assinatura dos tokens         |
| `JWT_TTL`        | `24h`                          | Validade do token                      |
| `UPLOAD_DIR`     | `./uploads`                    | Onde as imagens são gravadas           |
| `CORS_ORIGINS`   | `http://localhost:5173`        | Lista separada por vírgula             |
| `ADMIN_NAME` / `ADMIN_EMAIL` / `ADMIN_PASSWORD` | — | Seed do gestor inicial (idempotente) |

## Deploy em Cloud

As imagens são independentes e sem estado (exceto uploads), então funcionam em qualquer provedor de containers. Um caminho simples:

1. **Banco**: PostgreSQL gerenciado (AWS RDS, GCP Cloud SQL, Azure Database, Neon, Render…). Configure `DATABASE_URL` (com `sslmode=require`).
2. **Backend**: publique `backend/Dockerfile` em um serviço de containers (Cloud Run, ECS/Fargate, Azure Container Apps, Render, Fly.io). As migrações rodam automaticamente na subida. Monte um volume persistente em `/app/uploads` ou troque `LocalFileStore` por uma implementação de object storage (S3/GCS) da interface `service.FileStore`.
3. **Frontend**: publique `frontend/Dockerfile` e defina `BACKEND_URL` com a URL interna do backend (o Nginx faz proxy de `/api` e `/uploads`, sem CORS). Alternativa: build estático (`VITE_API_URL=https://api.seudominio`) em um CDN, adicionando o domínio em `CORS_ORIGINS`.
4. Defina `JWT_SECRET` e `ADMIN_PASSWORD` fortes via secrets do provedor.

O workflow `.github/workflows/ci.yml` roda lint, testes e build das imagens a cada push/PR.
