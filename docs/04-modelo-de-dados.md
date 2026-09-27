# 4. Modelo de dados

Esquema definido nas migrações em [`backend/internal/database/migrations`](../backend/internal/database/migrations), aplicadas automaticamente na subida da API.

```mermaid
erDiagram
    USERS ||--o{ OCCURRENCES : "registra (requester_id)"
    USERS |o--o{ OCCURRENCES : "é responsável (assignee_id)"
    CATEGORIES ||--o{ OCCURRENCES : classifica
    OCCURRENCES ||--o{ COMMENTS : possui
    OCCURRENCES ||--|{ STATUS_HISTORY : "registra mudanças"
    USERS ||--o{ COMMENTS : escreve
    USERS ||--o{ STATUS_HISTORY : altera

    USERS {
        bigint id PK
        text name
        text email UK
        text password_hash "bcrypt"
        text role "solicitante | gestor"
        timestamptz created_at
    }
    CATEGORIES {
        bigint id PK
        text name UK
    }
    OCCURRENCES {
        bigint id PK
        text title
        text description
        bigint category_id FK
        text location
        text image_url "nullable"
        text priority "baixa | media | alta | critica"
        text status "aberta | em_analise | em_atendimento | resolvida | cancelada"
        bigint requester_id FK
        bigint assignee_id FK "nullable"
        text solution "nullable"
        smallint rating "1..5, nullable"
        text rating_comment "nullable"
        timestamptz created_at
        timestamptz updated_at
        timestamptz resolved_at "nullable"
    }
    COMMENTS {
        bigint id PK
        bigint occurrence_id FK
        bigint author_id FK
        text body
        timestamptz created_at
    }
    STATUS_HISTORY {
        bigint id PK
        bigint occurrence_id FK
        text from_status "nullable (criação)"
        text to_status
        bigint changed_by FK
        text note
        timestamptz created_at
    }
    FILES {
        text name PK
        text content_type
        bytea data
        timestamptz created_at
    }
```

## Dicionário de dados

| Tabela | Finalidade | Observações |
|--------|------------|-------------|
| `users` | Solicitantes e gestores | E-mail único; senha só como hash bcrypt; `role` restrito por `CHECK` |
| `categories` | Categorias de ocorrência | Semeada com as 8 categorias do domínio |
| `occurrences` | Objeto central | `status` e `priority` restritos por `CHECK`; `rating` entre 1 e 5; `resolved_at` preenchido ao resolver (usado no tempo médio de resolução) |
| `comments` | Conversa na ocorrência | Removidos em cascata com a ocorrência |
| `status_history` | Auditoria de mudanças de status | Uma linha por transição (inclusive a criação, com `from_status` nulo); gravada na mesma transação da mudança |
| `files` | Imagens anexadas | Servidas em `/uploads/<name>`; `occurrences.image_url` aponta para elas |
| `schema_migrations` | Controle de migrações | Versões já aplicadas |

**Índices:** `occurrences(requester_id)`, `(status)`, `(category_id)`, `(priority)` para os filtros da listagem; `comments(occurrence_id)` e `status_history(occurrence_id)` para o detalhe.
