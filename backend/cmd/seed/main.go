// Comando seed popula o banco com clientes e ocorrências de demonstração.
//
//	DATABASE_URL=... SEED_PASSWORD=... [SEED_RESET=true] go run ./cmd/seed
//
// Cria 20 ocorrências em todos os estados do ciclo de vida, cada uma aberta
// por um reclamante diferente com nome aleatório (e-mail @exemplo.com), com
// histórico, comentários, soluções e avaliações coerentes com as regras de
// negócio. As ocorrências são atribuídas ao primeiro gestor cadastrado.
//
// É idempotente: não faz nada se já houver reclamantes de demonstração.
// Com SEED_RESET=true, remove antes apenas os dados de demonstração
// (usuários @exemplo.com e suas ocorrências), preservando os demais.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"resolveai/internal/database"
	"resolveai/internal/domain"
)

const demoDomain = "@exemplo.com"

var (
	firstNames = []string{
		"Adriana", "Alexandre", "Aline", "André", "Beatriz", "Caio", "Camila", "Carlos", "Cecília", "Daniel",
		"Débora", "Eduardo", "Elaine", "Fábio", "Fernanda", "Gustavo", "Helena", "Igor", "Isabela", "João",
		"Juliana", "Larissa", "Leandro", "Letícia", "Lucas", "Marcela", "Marcos", "Mariana", "Mateus", "Natália",
		"Otávio", "Patrícia", "Paula", "Rafael", "Renata", "Ricardo", "Rodrigo", "Sabrina", "Sérgio", "Tatiane",
		"Thiago", "Valéria", "Vinícius", "Vitória", "Wagner",
	}
	lastNames = []string{
		"Almeida", "Andrade", "Araújo", "Barbosa", "Barros", "Cardoso", "Carvalho", "Castro", "Correia", "Costa",
		"Cunha", "Dias", "Farias", "Ferreira", "Freitas", "Gomes", "Lopes", "Machado", "Martins", "Melo",
		"Monteiro", "Moraes", "Moreira", "Nascimento", "Oliveira", "Pereira", "Pinto", "Ramos", "Ribeiro", "Rocha",
		"Santos", "Silva", "Soares", "Souza", "Teixeira", "Vieira",
	}
	accents = strings.NewReplacer("á", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "ú", "u", "ç", "c")
)

type client struct{ name, email string }

// staff é a equipe de manutenção fictícia (perfil gestor) que assume as
// ocorrências abertas, para que nenhuma fique sem responsável.
var staff = []client{
	{"Roberto Siqueira", "roberto.siqueira" + demoDomain},
	{"Sandra Figueiredo", "sandra.figueiredo" + demoDomain},
	{"Marcelo Tavares", "marcelo.tavares" + demoDomain},
}

// randomClients gera n reclamantes com nomes aleatórios e e-mails únicos.
func randomClients(n int) []client {
	seen := map[string]bool{}
	out := make([]client, 0, n)
	for len(out) < n {
		first := firstNames[rand.IntN(len(firstNames))]
		last := lastNames[rand.IntN(len(lastNames))]
		email := accents.Replace(strings.ToLower(first+"."+last)) + demoDomain
		if seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, client{first + " " + last, email})
	}
	return out
}

// comment: from "r" = solicitante, "g" = gestor.
type comment struct{ from, body string }

type occurrence struct {
	title, description, category, location string
	priority                               domain.Priority
	status                                 domain.Status
	daysAgo                                float64
	comments                               []comment
	solution                               string
	rating                                 int
	ratingComment                          string
	cancelNote                             string
	cancelledByRequester                   bool
}

var occurrences = []occurrence{
	// ---- Resolvidas ----
	{title: "Lâmpada queimada no corredor do 3º andar", category: "Iluminação", location: "Bloco A, 3º andar",
		description: "A lâmpada em frente ao apartamento 32 está queimada há dois dias. À noite o corredor fica totalmente escuro.",
		priority:    domain.PriorityMedia, status: domain.StatusResolvida, daysAgo: 20,
		comments: []comment{{"g", "Equipe de manutenção agendada para amanhã pela manhã."}, {"r", "Agradeço o retorno rápido!"}},
		solution: "Lâmpada substituída por modelo LED de 12 W.", rating: 5, ratingComment: "Resolveram no dia seguinte, excelente."},
	{title: "Vazamento no teto da garagem", category: "Vazamentos", location: "Subsolo, próximo à vaga 12",
		description: "Água pingando do teto da garagem, formando poça e deixando o piso escorregadio.",
		priority:    domain.PriorityAlta, status: domain.StatusResolvida, daysAgo: 18,
		comments: []comment{{"r", "Piorou com a chuva de ontem."}, {"g", "Encanador identificou cano rompido na laje; reparo em andamento."}},
		solution: "Troca de 2 m de tubulação rompida e impermeabilização do trecho da laje.", rating: 4, ratingComment: "Demorou um pouco, mas ficou bem feito."},
	{title: "Portão da garagem não fecha sozinho", category: "Equipamentos quebrados", location: "Portão principal da garagem",
		description: "O portão abre normalmente, mas não fecha automaticamente. Precisa acionar o controle duas vezes.",
		priority:    domain.PriorityCritica, status: domain.StatusResolvida, daysAgo: 16,
		comments: []comment{{"g", "Técnico da empresa do portão acionado em caráter de urgência."}},
		solution: "Sensor fotoelétrico desalinhado; realinhado e placa de comando reprogramada.", rating: 5, ratingComment: "Muito importante para a segurança, obrigado pela agilidade."},
	{title: "Lixeira do térreo sem recolhimento", category: "Limpeza", location: "Área de lixo, térreo do Bloco B",
		description: "O lixo não foi recolhido no fim de semana e está com mau cheiro.",
		priority:    domain.PriorityMedia, status: domain.StatusResolvida, daysAgo: 12,
		solution: "Recolhimento realizado e escala de limpeza do fim de semana reforçada.", rating: 3, ratingComment: "Resolveu, mas já é a segunda vez que acontece."},
	{title: "Interfone do apartamento 104 sem áudio", category: "Manutenção", location: "Bloco B, apartamento 104",
		description: "Consigo ouvir a portaria, mas eles não me ouvem quando atendo o interfone.",
		priority:    domain.PriorityBaixa, status: domain.StatusResolvida, daysAgo: 9,
		comments: []comment{{"g", "Aparelho será testado na próxima visita técnica."}},
		solution: "Microfone do monofone substituído."},

	// ---- Em atendimento ----
	{title: "Elevador social parando entre andares", category: "Equipamentos quebrados", location: "Bloco A, elevador social",
		description: "O elevador parou entre o 5º e o 6º andar duas vezes esta semana. Tem idosos no prédio que dependem dele.",
		priority:    domain.PriorityCritica, status: domain.StatusEmAtendimento, daysAgo: 6,
		comments: []comment{{"r", "Hoje o elevador parou comigo dentro por uns 10 minutos."}, {"g", "Elevador interditado; empresa de manutenção trocando o quadro de comando."}}},
	{title: "Rampa de acesso com piso quebrado", category: "Acessibilidade", location: "Entrada principal, rampa lateral",
		description: "Há placas soltas no piso da rampa. Cadeirantes e carrinhos de bebê têm dificuldade para passar.",
		priority:    domain.PriorityAlta, status: domain.StatusEmAtendimento, daysAgo: 5,
		comments: []comment{{"g", "Orçamento aprovado; obra começa esta semana."}}},
	{title: "Infiltração na parede da área de lazer", category: "Vazamentos", location: "Salão de festas",
		description: "Mancha de umidade crescendo na parede ao lado da churrasqueira, com mofo.",
		priority:    domain.PriorityMedia, status: domain.StatusEmAtendimento, daysAgo: 4},
	{title: "Câmera da portaria sem imagem", category: "Segurança", location: "Portaria, câmera da entrada de pedestres",
		description: "O monitor da portaria mostra tela preta para a câmera do portão de pedestres.",
		priority:    domain.PriorityAlta, status: domain.StatusEmAtendimento, daysAgo: 3,
		comments: []comment{{"g", "Fonte da câmera queimada; peça encomendada."}}},

	// ---- Em análise ----
	{title: "Refletor da quadra apagado", category: "Iluminação", location: "Quadra poliesportiva",
		description: "Dois dos quatro refletores da quadra não acendem, impossível jogar à noite.",
		priority:    domain.PriorityBaixa, status: domain.StatusEmAnalise, daysAgo: 3},
	{title: "Barulho excessivo na casa de máquinas", category: "Manutenção", location: "Cobertura do Bloco B",
		description: "Ruído metálico constante vindo da casa de máquinas, dá para ouvir no último andar.",
		priority:    domain.PriorityMedia, status: domain.StatusEmAnalise, daysAgo: 2.5,
		comments: []comment{{"g", "Vamos verificar se é o motor do elevador ou a bomba d'água."}}},
	{title: "Falta de corrimão na escada do bloco C", category: "Acessibilidade", location: "Bloco C, escada de emergência",
		description: "A escada de emergência não tem corrimão entre o 1º e o 2º andar.",
		priority:    domain.PriorityAlta, status: domain.StatusEmAnalise, daysAgo: 2},
	{title: "Piscina com água turva", category: "Limpeza", location: "Área da piscina",
		description: "A água da piscina está turva e esverdeada desde o último fim de semana.",
		priority:    domain.PriorityMedia, status: domain.StatusEmAnalise, daysAgo: 1.5},

	// ---- Abertas ----
	{title: "Porta corta-fogo não fecha", category: "Segurança", location: "Bloco A, 7º andar",
		description: "A mola da porta corta-fogo está fraca e a porta fica aberta.",
		priority:    domain.PriorityMedia, status: domain.StatusAberta, daysAgo: 1},
	{title: "Vazamento na torneira do jardim", category: "Vazamentos", location: "Jardim interno, próximo ao playground",
		description: "Torneira pingando sem parar, desperdiçando água.",
		priority:    domain.PriorityMedia, status: domain.StatusAberta, daysAgo: 0.8},
	{title: "Brinquedo quebrado no playground", category: "Equipamentos quebrados", location: "Playground",
		description: "O balanço está com a corrente solta de um lado, risco para as crianças.",
		priority:    domain.PriorityMedia, status: domain.StatusAberta, daysAgo: 0.5,
		comments: []comment{{"r", "Coloquei uma fita para ninguém usar até o conserto."}}},
	{title: "Sujeira acumulada na escada do bloco B", category: "Limpeza", location: "Bloco B, escadas",
		description: "As escadas não são varridas há alguns dias, com folhas e poeira.",
		priority:    domain.PriorityMedia, status: domain.StatusAberta, daysAgo: 0.3},
	{title: "Sugestão: bicicletário coberto", category: "Outros", location: "Área externa, lateral do Bloco C",
		description: "As bicicletas ficam expostas à chuva. Sugiro instalar uma cobertura simples.",
		priority:    domain.PriorityMedia, status: domain.StatusAberta, daysAgo: 0.1},

	// ---- Canceladas ----
	{title: "Lâmpada da escada piscando", category: "Iluminação", location: "Bloco C, escada, 2º andar",
		description: "A lâmpada fica piscando o tempo todo.",
		priority:    domain.PriorityMedia, status: domain.StatusCancelada, daysAgo: 7,
		cancelNote: "Era mau contato; o próprio morador ajustou o soquete.", cancelledByRequester: true},
	{title: "Portão de pedestres emperrado", category: "Equipamentos quebrados", location: "Portão de pedestres",
		description: "O portão de pedestres está emperrando ao abrir.",
		priority:    domain.PriorityMedia, status: domain.StatusCancelada, daysAgo: 10,
		comments:   []comment{{"g", "Já existe um chamado aberto para este portão com a empresa de manutenção."}},
		cancelNote: "Ocorrência duplicada: o problema já está sendo tratado em outro chamado."},
}

// Notas padrão registradas no histórico de cada transição.
var stepNotes = map[domain.Status]string{
	domain.StatusEmAnalise:     "Ocorrência recebida e em análise pela administração.",
	domain.StatusEmAtendimento: "Equipe/fornecedor acionado para o atendimento.",
	domain.StatusResolvida:     "Serviço concluído e conferido.",
}

var path = []domain.Status{domain.StatusAberta, domain.StatusEmAnalise, domain.StatusEmAtendimento, domain.StatusResolvida}

func main() {
	if err := run(); err != nil {
		slog.Error("seed falhou", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	url, password := os.Getenv("DATABASE_URL"), os.Getenv("SEED_PASSWORD")
	if url == "" || len(password) < 6 {
		return errors.New("defina DATABASE_URL e SEED_PASSWORD (mín. 6 caracteres)")
	}
	pool, err := database.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	if os.Getenv("SEED_RESET") == "true" {
		if err := resetDemo(ctx, pool); err != nil {
			return err
		}
	}

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email LIKE '%'||$1)`, demoDomain).Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Println("dados de demonstração já existem; nada a fazer (use SEED_RESET=true para recriá-los)")
		return nil
	}

	var gestorID int64
	err = pool.QueryRow(ctx, `SELECT id FROM users WHERE role = 'gestor' ORDER BY id LIMIT 1`).Scan(&gestorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("nenhum gestor cadastrado: crie um gestor antes de rodar o seed")
	}
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		categories := map[string]int64{}
		rows, err := tx.Query(ctx, `SELECT id, name FROM categories`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				return err
			}
			categories[name] = id
		}
		if err := rows.Err(); err != nil {
			return err
		}

		now := time.Now()
		clients := randomClients(len(occurrences))
		clientIDs := make([]int64, len(clients))
		for i, c := range clients {
			created := now.Add(-22 * 24 * time.Hour).Add(time.Duration(i) * time.Hour)
			if err := tx.QueryRow(ctx,
				`INSERT INTO users (name, email, password_hash, role, created_at) VALUES ($1, $2, $3, 'solicitante', $4) RETURNING id`,
				c.name, c.email, string(hash), created).Scan(&clientIDs[i]); err != nil {
				return err
			}
		}

		staffIDs := make([]int64, len(staff))
		for i, c := range staff {
			if err := tx.QueryRow(ctx,
				`INSERT INTO users (name, email, password_hash, role, created_at) VALUES ($1, $2, $3, 'gestor', $4) RETURNING id`,
				c.name, c.email, string(hash), now.Add(-30*24*time.Hour)).Scan(&staffIDs[i]); err != nil {
				return err
			}
		}

		// Cada ocorrência é aberta por um reclamante diferente. As abertas são
		// distribuídas entre a equipe; as demais ficam com o gestor principal.
		// Só a cancelada pelo próprio reclamante fica sem responsável.
		opened := 0
		for i, o := range occurrences {
			catID, ok := categories[o.category]
			if !ok {
				return fmt.Errorf("categoria desconhecida: %s", o.category)
			}
			var assignee *int64
			switch {
			case o.cancelledByRequester:
			case o.status == domain.StatusAberta:
				assignee = &staffIDs[opened%len(staffIDs)]
				opened++
			default:
				assignee = &gestorID
			}
			if err := insertOccurrence(ctx, tx, o, catID, clientIDs[i], gestorID, assignee, now); err != nil {
				return fmt.Errorf("%q: %w", o.title, err)
			}
		}
		fmt.Printf("criados %d reclamantes, %d gestores da equipe e %d ocorrências (gestor principal: #%d)\n",
			len(clients), len(staff), len(occurrences), gestorID)
		return nil
	})
}

// resetDemo remove somente os dados de demonstração: usuários @exemplo.com e as
// ocorrências abertas por eles (comentários e histórico caem em cascata).
func resetDemo(ctx context.Context, pool *pgxpool.Pool) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		occ, err := tx.Exec(ctx, `DELETE FROM occurrences WHERE requester_id IN
			(SELECT id FROM users WHERE email LIKE '%'||$1)`, demoDomain)
		if err != nil {
			return err
		}
		// Ocorrências reais atribuídas à equipe de demonstração ficam sem responsável.
		if _, err := tx.Exec(ctx, `UPDATE occurrences SET assignee_id = NULL WHERE assignee_id IN
			(SELECT id FROM users WHERE email LIKE '%'||$1)`, demoDomain); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM comments WHERE author_id IN
			(SELECT id FROM users WHERE email LIKE '%'||$1)`, demoDomain); err != nil {
			return err
		}
		users, err := tx.Exec(ctx, `DELETE FROM users WHERE email LIKE '%'||$1`, demoDomain)
		if err != nil {
			return err
		}
		fmt.Printf("removidos %d reclamantes e %d ocorrências de demonstração\n", users.RowsAffected(), occ.RowsAffected())
		return nil
	})
}

func insertOccurrence(ctx context.Context, tx pgx.Tx, o occurrence, catID, requesterID, gestorID int64, assignee *int64, now time.Time) error {
	created := now.Add(-time.Duration(o.daysAgo * float64(24*time.Hour)))
	// Etapas espaçadas proporcionalmente à idade da ocorrência, limitadas a 18 h
	// para um tempo de resolução realista (~2 dias).
	step := min(time.Duration(o.daysAgo*float64(24*time.Hour)/4), 18*time.Hour)
	at := func(n int) time.Time { return created.Add(time.Duration(n) * step) }

	// Sequência de status percorrida até o estado final.
	var steps []domain.Status
	if o.status == domain.StatusCancelada {
		steps = []domain.Status{domain.StatusAberta}
		if !o.cancelledByRequester {
			steps = append(steps, domain.StatusEmAnalise)
		}
		steps = append(steps, domain.StatusCancelada)
	} else {
		for _, s := range path {
			steps = append(steps, s)
			if s == o.status {
				break
			}
		}
	}
	finalAt := at(len(steps) - 1)

	var solution, ratingComment *string
	var rating *int
	var resolvedAt *time.Time
	if o.solution != "" {
		solution = &o.solution
	}
	if o.status == domain.StatusResolvida {
		resolvedAt = &finalAt
		if o.rating > 0 {
			rating = &o.rating
			if o.ratingComment != "" {
				ratingComment = &o.ratingComment
			}
		}
	}

	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO occurrences (title, description, category_id, location, priority, status, requester_id,
		                          assignee_id, solution, rating, rating_comment, created_at, updated_at, resolved_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`,
		o.title, o.description, catID, o.location, o.priority, o.status, requesterID,
		assignee, solution, rating, ratingComment, created, finalAt, resolvedAt,
	).Scan(&id); err != nil {
		return err
	}

	// Histórico: criação (∅ → aberta) + uma linha por transição.
	var from *domain.Status
	for i, s := range steps {
		by, note := gestorID, stepNotes[s]
		switch {
		case i == 0:
			by, note = requesterID, "Ocorrência registrada"
		case s == domain.StatusCancelada:
			note = o.cancelNote
			if o.cancelledByRequester {
				by = requesterID
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO status_history (occurrence_id, from_status, to_status, changed_by, note, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`, id, from, s, by, note, at(i)); err != nil {
			return err
		}
		prev := s
		from = &prev
	}

	// Comentários distribuídos entre a criação e o estado atual.
	span := finalAt.Sub(created)
	if span <= 0 {
		span = time.Hour
	}
	for i, c := range o.comments {
		author := requesterID
		if c.from == "g" {
			author = gestorID
		}
		when := created.Add(span * time.Duration(i+1) / time.Duration(len(o.comments)+1))
		if _, err := tx.Exec(ctx,
			`INSERT INTO comments (occurrence_id, author_id, body, created_at) VALUES ($1, $2, $3, $4)`,
			id, author, c.body, when); err != nil {
			return err
		}
	}
	return nil
}
