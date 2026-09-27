# 2. Fluxos

## 2.1 Perfis de usuário e ocorrência

Capacidades de cada perfil e como se relacionam com a ocorrência. As setas dentro de cada perfil indicam a jornada típica — após o registro, acompanhar, comentar e consultar o histórico podem acontecer em qualquer ordem.

```mermaid
flowchart LR

    %% =========================
    %% SOLICITANTE
    %% =========================
    subgraph SOL["👤 Solicitante"]
        direction TB

        S1["Criar conta"]
        S2["Autenticar-se"]
        S3["Registrar ocorrência"]
        S4["Informar título, descrição<br/>e categoria"]
        S5["Informar localização"]
        S6["Anexar imagem"]
        S7["Acompanhar andamento"]
        S8["Adicionar comentários"]
        S9["Consultar histórico"]
        S10["Avaliar resolução"]

        S1 --> S2
        S2 --> S3
        S3 --> S4
        S4 --> S5
        S5 --> S6
        S6 --> S7
        S7 --> S8
        S8 --> S9
        S9 --> S10
    end

    %% =========================
    %% OCORRÊNCIA
    %% =========================
    O["📋 OCORRÊNCIA<br/><br/>
    Título<br/>
    Descrição<br/>
    Categoria<br/>
    Localização<br/>
    Imagem<br/>
    Prioridade<br/>
    Status<br/>
    Responsável<br/>
    Comentários<br/>
    Histórico<br/>
    Solução aplicada<br/>
    Avaliação"]

    %% =========================
    %% GESTOR
    %% =========================
    subgraph GES["👔 Gestor"]
        direction TB

        G1["Visualizar ocorrências"]
        G2["Filtrar por categoria,<br/>status e prioridade"]
        G3["Alterar prioridade"]
        G4["Atribuir responsável"]
        G5["Atualizar status"]
        G6["Adicionar comentários"]
        G7["Registrar solução aplicada"]
        G8["Visualizar dashboard"]

        G1 --> G2
        G2 --> G3
        G3 --> G4
        G4 --> G5
        G5 --> G6
        G6 --> G7
        G7 --> G8
    end

    S3 -->|"cria"| O
    S7 -.->|"acompanha"| O
    S10 -->|"avalia"| O

    G1 -->|"consulta"| O
    G3 -->|"administra"| O
    G5 -->|"atualiza"| O
    G7 -->|"resolve"| O

    %% =========================
    %% ESTILOS
    %% =========================
    classDef solicitante fill:#EFF6FF,stroke:#2563EB,stroke-width:2px,color:#172554;
    classDef gestor fill:#F0F9FF,stroke:#0369A1,stroke-width:2px,color:#0C4A6E;
    classDef ocorrencia fill:#F0FDF4,stroke:#16A34A,stroke-width:3px,color:#14532D;

    class S1,S2,S3,S4,S5,S6,S7,S8,S9,S10 solicitante;
    class G1,G2,G3,G4,G5,G6,G7,G8 gestor;
    class O ocorrencia;
```

## 2.2 Ciclo de vida da ocorrência

Estados, transições permitidas (com quem pode executá-las e suas pré-condições) e o registro de histórico gerado em cada mudança. Regras detalhadas em [Regras de negócio](01-descoberta-do-dominio.md#16-regras-de-negócio).

```mermaid
flowchart LR

    N(["➕ Registrar<br/>(solicitante)"])
    A["📂 Aberta"]
    B["🔍 Em análise"]
    C["🔧 Em atendimento"]
    D["✅ Resolvida"]
    E["❌ Cancelada"]
    AV["⭐ Avaliação do<br/>solicitante (1 a 5)"]

    N --> A
    A -->|"Analisar<br/>(gestor)"| B
    B -->|"Iniciar atendimento<br/>(gestor)"| C
    C -->|"Concluir (gestor)<br/>exige solução aplicada"| D
    D -->|"Avaliar<br/>(solicitante, 1 vez)"| AV

    A -.->|"Cancelar (gestor ou solicitante)<br/>motivo obrigatório"| E
    B -.->|"Cancelar (gestor)<br/>motivo obrigatório"| E
    C -.->|"Cancelar (gestor)<br/>motivo obrigatório"| E

    %% Histórico
    H["🕘 Histórico da alteração<br/><br/>
    Status anterior<br/>
    Novo status<br/>
    Data e horário<br/>
    Usuário responsável<br/>
    Observação da alteração"]

    A -.-> H
    B -.-> H
    C -.-> H
    D -.-> H
    E -.-> H

    %% Estilos
    classDef normal fill:#EFF6FF,stroke:#2563EB,stroke-width:2px,color:#172554;
    classDef success fill:#F0FDF4,stroke:#16A34A,stroke-width:3px,color:#14532D;
    classDef cancel fill:#FEF2F2,stroke:#DC2626,stroke-width:3px,color:#991B1B;
    classDef history fill:#F8FAFC,stroke:#64748B,stroke-width:2px,color:#334155;
    classDef rating fill:#FFFBEB,stroke:#D97706,stroke-width:2px,color:#78350F;

    class N,A,B,C normal;
    class D success;
    class E cancel;
    class H history;
    class AV rating;
```

**Resumo das regras do ciclo:**

- Não é possível pular etapas (ex.: Aberta → Resolvida é rejeitada).
- Resolvida e Cancelada são estados finais.
- O histórico registra também a criação (sem status anterior → Aberta).
- Mudanças simultâneas de status: só a primeira é aplicada; a segunda recebe conflito (HTTP 409).

## 2.3 Sequência do fluxo principal

Como as camadas do sistema colaboram do registro à avaliação.

```mermaid
sequenceDiagram
    autonumber
    actor S as Solicitante
    actor G as Gestor
    participant W as Frontend (React)
    participant A as API (Go)
    participant DB as PostgreSQL

    S->>W: Preenche nova ocorrência (+ imagem)
    W->>A: POST /api/occurrences
    A->>DB: INSERT occurrence + INSERT status_history (∅ → aberta)
    A-->>W: 201 Ocorrência #id
    W->>A: POST /api/occurrences/{id}/image
    A->>DB: INSERT files · UPDATE image_url

    G->>W: Filtra, prioriza e atribui
    W->>A: PATCH /priority · PATCH /assignee
    G->>W: Avança o status
    W->>A: PATCH /status {em_analise → em_atendimento}
    A->>DB: UPDATE status (se ainda for o anterior) + INSERT status_history

    G->>W: Registra solução e resolve
    W->>A: PATCH /status {resolvida, solution}
    A->>DB: UPDATE status, solution, resolved_at + INSERT status_history

    S->>W: Consulta histórico e avalia
    W->>A: POST /api/occurrences/{id}/rating {1..5}
    A->>DB: UPDATE rating (somente se ainda não avaliada)
    A-->>W: 200 Ocorrência avaliada
```
