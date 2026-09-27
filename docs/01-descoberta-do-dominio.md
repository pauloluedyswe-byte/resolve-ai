# 1. Descoberta do domínio

## 1.1 Contexto

Condomínios, empresas, bairros e organizações lidam diariamente com problemas de infraestrutura e convivência: lâmpadas queimadas, vazamentos, equipamentos quebrados, falta de acessibilidade, limpeza, segurança.

Hoje essas solicitações chegam por **mensagens, e-mails ou conversas informais**. O resultado:

- **Sem registro único** — pedidos se perdem entre canais e pessoas.
- **Sem priorização** — não há critério claro do que atender primeiro.
- **Sem transparência** — quem reportou não sabe se o problema está sendo tratado.
- **Sem rastreabilidade** — não se sabe quem mudou o quê, nem quando.
- **Sem indicadores** — a gestão não mede volume, tempo de resolução ou satisfação.

## 1.2 Proposta de valor

O **Resolve Aí** centraliza o registro e o acompanhamento de ocorrências, do relato até a resolução e avaliação:

- Um único lugar para **registrar** o problema com título, descrição, categoria, localização e foto.
- Um **ciclo de vida explícito** (Aberta → Em análise → Em atendimento → Resolvida / Cancelada).
- **Histórico auditável** de toda mudança de status.
- **Comunicação** entre solicitante e gestão por comentários na própria ocorrência.
- **Dashboard** com indicadores para a gestão e **avaliação** da resolução pelo solicitante.

## 1.3 Atores

| Ator | Objetivo | Principais capacidades |
|------|----------|------------------------|
| **Solicitante** | Ter o problema resolvido e saber em que pé está | Criar conta, autenticar-se, registrar ocorrência (com imagem), acompanhar, comentar, consultar histórico, avaliar a resolução |
| **Gestor** | Organizar, priorizar e resolver as ocorrências | Ver e filtrar todas as ocorrências, alterar prioridade, atribuir responsável, atualizar status, comentar, registrar solução, acompanhar indicadores |

> O primeiro gestor é provisionado pela configuração do ambiente (`ADMIN_EMAIL`/`ADMIN_PASSWORD`); o cadastro público cria sempre um Solicitante, evitando que qualquer pessoa se autopromova a gestor.

## 1.4 Eventos de domínio

Levantados em uma sessão de *event storming* simplificada, na ordem em que acontecem no fluxo principal:

| Evento | Disparado por | Consequência |
|--------|---------------|--------------|
| `ContaCriada` | Solicitante | Pode autenticar-se |
| `OcorrenciaRegistrada` | Solicitante | Ocorrência nasce **Aberta**, prioridade **Média**; histórico recebe o primeiro registro |
| `ImagemAnexada` | Solicitante / Gestor | Evidência visual associada à ocorrência |
| `PrioridadeAlterada` | Gestor | Ocorrência reordenada na fila de atendimento |
| `ResponsavelAtribuido` | Gestor | Um gestor passa a responder pela ocorrência |
| `StatusAlterado` | Gestor (ou Solicitante, ao cancelar) | Registro no histórico com status anterior, novo, data/hora, usuário e observação |
| `ComentarioAdicionado` | Solicitante / Gestor | Conversa registrada na ocorrência |
| `SolucaoRegistrada` | Gestor | Pré-condição para resolver |
| `OcorrenciaResolvida` | Gestor | Estado final; habilita a avaliação |
| `OcorrenciaCancelada` | Gestor / Solicitante | Estado final, com motivo |
| `ResolucaoAvaliada` | Solicitante | Nota de 1 a 5 alimenta o indicador de satisfação |

## 1.5 Glossário (linguagem ubíqua)

| Termo | Definição |
|-------|-----------|
| **Ocorrência** | Objeto central do sistema: um problema relatado que precisa de tratamento |
| **Solicitante** | Usuário que registra e acompanha as próprias ocorrências |
| **Gestor** | Usuário que analisa, prioriza, atribui e resolve ocorrências |
| **Responsável** | Gestor atribuído para conduzir uma ocorrência específica |
| **Categoria** | Classificação do problema: Iluminação, Equipamentos quebrados, Acessibilidade, Limpeza, Vazamentos, Segurança, Manutenção, Outros |
| **Localização** | Onde o problema está (ex.: "Bloco B, 3º andar") |
| **Prioridade** | Urgência definida pelo gestor: Baixa, Média, Alta, Crítica |
| **Status** | Etapa do ciclo de vida: Aberta, Em análise, Em atendimento, Resolvida, Cancelada |
| **Estado final** | Resolvida ou Cancelada — não admite novas transições nem alterações de gestão |
| **Histórico** | Registro imutável de cada mudança de status |
| **Solução aplicada** | Descrição do que foi feito para resolver a ocorrência |
| **Avaliação** | Nota de 1 a 5 (e comentário opcional) dada pelo solicitante após a resolução |

## 1.6 Regras de negócio

| ID | Regra |
|----|-------|
| RN-01 | Toda ocorrência nasce com status **Aberta** e prioridade **Média**. |
| RN-02 | Título, descrição, categoria e localização são obrigatórios; imagem é opcional (JPG, PNG, WEBP ou GIF, até 5 MB). |
| RN-03 | Transições permitidas: Aberta → Em análise → Em atendimento → Resolvida; Aberta, Em análise e Em atendimento → Cancelada. Qualquer outra é rejeitada. |
| RN-04 | Resolvida e Cancelada são **estados finais**: não mudam de status nem de prioridade/responsável. |
| RN-05 | **Toda** mudança de status gera um registro no histórico com status anterior, novo status, data/hora, usuário responsável e observação — gravado na mesma transação da mudança. |
| RN-06 | A criação da ocorrência também é registrada no histórico (sem status anterior → Aberta). |
| RN-07 | O **solicitante** só pode **cancelar** a própria ocorrência, e apenas enquanto ela estiver **Aberta**. As demais transições são exclusivas do gestor. |
| RN-08 | **Cancelar exige observação** (o motivo do cancelamento). |
| RN-09 | **Resolver exige solução aplicada** registrada (antes ou no momento da resolução). |
| RN-10 | O **responsável** atribuído deve ser um usuário com perfil Gestor. |
| RN-11 | O solicitante só visualiza as **próprias** ocorrências; o gestor visualiza todas. |
| RN-12 | Somente o solicitante que registrou pode **avaliar**, apenas quando a ocorrência estiver **Resolvida** e **uma única vez** (nota de 1 a 5). |
| RN-13 | Se duas pessoas alterarem o status ao mesmo tempo, apenas a primeira é aplicada; a segunda recebe um aviso de conflito (evita histórico inconsistente). |

> RN-07 a RN-10 e RN-13 são decisões do grupo para cobrir lacunas do enunciado.

## 1.7 Rastreabilidade dos requisitos

| Requisito (enunciado) | Onde está atendido |
|-----------------------|--------------------|
| Criar conta / autenticar-se | `POST /api/auth/register`, `POST /api/auth/login` · telas Cadastro e Login |
| Registrar ocorrência (título, descrição, categoria, localização) | `POST /api/occurrences` · tela Nova ocorrência |
| Anexar imagem | `POST /api/occurrences/{id}/image` · tela Nova ocorrência e Detalhe |
| Acompanhar andamento / consultar histórico | `GET /api/occurrences`, `GET /api/occurrences/{id}` · Lista e Detalhe (linha do tempo) |
| Adicionar comentários | `POST /api/occurrences/{id}/comments` · Detalhe |
| Avaliar resolução | `POST /api/occurrences/{id}/rating` · Detalhe |
| Visualizar todas / filtrar por categoria, status e prioridade | `GET /api/occurrences?status=&priority=&category_id=&q=` · Lista |
| Alterar prioridade | `PATCH /api/occurrences/{id}/priority` · painel Gestão |
| Atribuir responsável | `PATCH /api/occurrences/{id}/assignee` · painel Gestão |
| Atualizar status | `PATCH /api/occurrences/{id}/status` · painel Gestão |
| Registrar solução aplicada | `PATCH /api/occurrences/{id}/solution` · painel Gestão |
| Dashboard de indicadores | `GET /api/dashboard` · tela Dashboard |
| Histórico de mudanças de status | Tabela `status_history` (RN-05, RN-06) |

## 1.8 Requisitos não funcionais

| Requisito | Como é atendido |
|-----------|-----------------|
| Segurança | Senhas com bcrypt; autenticação JWT (HS256) com expiração; autorização por perfil no backend; validação de tipo e tamanho de upload |
| Auditabilidade | Histórico imutável e transacional (RN-05) |
| Consistência | Controle otimista na mudança de status (RN-13); restrições `CHECK` e chaves estrangeiras no banco |
| Portabilidade | Containers Docker; configuração 100% por variáveis de ambiente |
| Operação | Migrações automáticas na subida; health check em `/api/health`; deploy contínuo a cada push |
| Usabilidade | Interface responsiva, com modo claro/escuro, em português |
