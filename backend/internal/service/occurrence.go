package service

import (
	"context"
	"errors"
	"io"
	"strings"

	"resolveai/internal/domain"
)

// FileStore persiste arquivos enviados (imagens) e devolve a URL pública.
type FileStore interface {
	Save(ctx context.Context, contentType, ext string, r io.Reader) (url string, err error)
}

type OccurrenceService struct {
	occurrences OccurrenceRepository
	categories  CategoryRepository
	users       UserRepository
	files       FileStore
}

func NewOccurrenceService(o OccurrenceRepository, c CategoryRepository, u UserRepository, f FileStore) *OccurrenceService {
	return &OccurrenceService{occurrences: o, categories: c, users: u, files: f}
}

type CreateOccurrenceInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CategoryID  int64  `json:"category_id"`
	Location    string `json:"location"`
}

type ChangeStatusInput struct {
	Status   domain.Status `json:"status"`
	Note     string        `json:"note"`
	Solution *string       `json:"solution"`
}

func (s *OccurrenceService) Categories(ctx context.Context) ([]domain.Category, error) {
	return s.categories.List(ctx)
}

func (s *OccurrenceService) Create(ctx context.Context, actor Actor, in CreateOccurrenceInput) (*domain.Occurrence, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Location = strings.TrimSpace(in.Location)
	switch {
	case in.Title == "":
		return nil, domain.Invalid("título é obrigatório")
	case len(in.Title) > 150:
		return nil, domain.Invalid("título deve ter no máximo 150 caracteres")
	case in.Description == "":
		return nil, domain.Invalid("descrição é obrigatória")
	case in.Location == "":
		return nil, domain.Invalid("localização é obrigatória")
	}
	ok, err := s.categories.Exists(ctx, in.CategoryID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.Invalid("categoria inválida")
	}
	o := &domain.Occurrence{
		Title:       in.Title,
		Description: in.Description,
		CategoryID:  in.CategoryID,
		Location:    in.Location,
		Priority:    domain.PriorityMedia,
		Status:      domain.StatusAberta,
		RequesterID: actor.ID,
	}
	if err := s.occurrences.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, o.ID)
}

// List devolve todas as ocorrências para o gestor e apenas as próprias para o solicitante.
func (s *OccurrenceService) List(ctx context.Context, actor Actor, f domain.OccurrenceFilter) ([]domain.Occurrence, error) {
	if !actor.IsGestor() {
		f.RequesterID = &actor.ID
	}
	return s.occurrences.List(ctx, f)
}

func (s *OccurrenceService) Get(ctx context.Context, actor Actor, id int64) (*domain.OccurrenceDetail, error) {
	o, err := s.load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	comments, err := s.occurrences.ListComments(ctx, id)
	if err != nil {
		return nil, err
	}
	history, err := s.occurrences.ListHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.OccurrenceDetail{Occurrence: *o, Comments: comments, History: history}, nil
}

func (s *OccurrenceService) ChangeStatus(ctx context.Context, actor Actor, id int64, in ChangeStatusInput) (*domain.Occurrence, error) {
	o, err := s.load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if !in.Status.Valid() {
		return nil, domain.Invalid("status inválido")
	}
	if !o.Status.CanTransitionTo(in.Status) {
		return nil, domain.Invalid("transição de status não permitida: " + string(o.Status) + " → " + string(in.Status))
	}
	// O solicitante só pode cancelar a própria ocorrência enquanto ela ainda está aberta.
	if !actor.IsGestor() && !(in.Status == domain.StatusCancelada && o.Status == domain.StatusAberta) {
		return nil, domain.ErrForbidden
	}
	in.Note = strings.TrimSpace(in.Note)
	if in.Status == domain.StatusCancelada && in.Note == "" {
		return nil, domain.Invalid("informe o motivo do cancelamento")
	}
	var solution *string
	if in.Solution != nil {
		trimmed := strings.TrimSpace(*in.Solution)
		if trimmed != "" {
			solution = &trimmed
		}
	}
	if in.Status == domain.StatusResolvida && solution == nil && (o.Solution == nil || *o.Solution == "") {
		return nil, domain.Invalid("registre a solução aplicada antes de resolver a ocorrência")
	}
	if err := s.occurrences.ChangeStatus(ctx, id, o.Status, in.Status, actor.ID, in.Note, solution); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

func (s *OccurrenceService) UpdatePriority(ctx context.Context, actor Actor, id int64, p domain.Priority) (*domain.Occurrence, error) {
	o, err := s.loadForGestor(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if !p.Valid() {
		return nil, domain.Invalid("prioridade inválida")
	}
	if o.Status.IsFinal() {
		return nil, domain.Invalid("ocorrência finalizada não pode ser alterada")
	}
	if err := s.occurrences.UpdatePriority(ctx, id, p); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

func (s *OccurrenceService) Assign(ctx context.Context, actor Actor, id int64, assigneeID *int64) (*domain.Occurrence, error) {
	o, err := s.loadForGestor(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if o.Status.IsFinal() {
		return nil, domain.Invalid("ocorrência finalizada não pode ser alterada")
	}
	if assigneeID != nil {
		u, err := s.users.GetByID(ctx, *assigneeID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.Invalid("responsável não encontrado")
		}
		if err != nil {
			return nil, err
		}
		if u.Role != domain.RoleGestor {
			return nil, domain.Invalid("o responsável deve ser um gestor")
		}
	}
	if err := s.occurrences.UpdateAssignee(ctx, id, assigneeID); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

func (s *OccurrenceService) RegisterSolution(ctx context.Context, actor Actor, id int64, solution string) (*domain.Occurrence, error) {
	o, err := s.loadForGestor(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	solution = strings.TrimSpace(solution)
	if solution == "" {
		return nil, domain.Invalid("descreva a solução aplicada")
	}
	if o.Status == domain.StatusCancelada {
		return nil, domain.Invalid("ocorrência cancelada não aceita solução")
	}
	if err := s.occurrences.UpdateSolution(ctx, id, solution); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

func (s *OccurrenceService) AddComment(ctx context.Context, actor Actor, id int64, body string) (*domain.Comment, error) {
	if _, err := s.load(ctx, actor, id); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.Invalid("comentário vazio")
	}
	c := &domain.Comment{OccurrenceID: id, AuthorID: actor.ID, AuthorName: actor.Name, AuthorRole: actor.Role, Body: body}
	if err := s.occurrences.AddComment(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *OccurrenceService) Rate(ctx context.Context, actor Actor, id int64, rating int, comment string) (*domain.Occurrence, error) {
	o, err := s.load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if o.RequesterID != actor.ID {
		return nil, domain.ErrForbidden
	}
	if o.Status != domain.StatusResolvida {
		return nil, domain.Invalid("só é possível avaliar ocorrências resolvidas")
	}
	if o.Rating != nil {
		return nil, domain.Invalid("ocorrência já avaliada")
	}
	if rating < 1 || rating > 5 {
		return nil, domain.Invalid("a nota deve estar entre 1 e 5")
	}
	if err := s.occurrences.SetRating(ctx, id, rating, strings.TrimSpace(comment)); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (s *OccurrenceService) AttachImage(ctx context.Context, actor Actor, id int64, contentType string, r io.Reader) (*domain.Occurrence, error) {
	o, err := s.load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if o.RequesterID != actor.ID && !actor.IsGestor() {
		return nil, domain.ErrForbidden
	}
	if o.Status.IsFinal() {
		return nil, domain.Invalid("ocorrência finalizada não pode ser alterada")
	}
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		return nil, domain.Invalid("formato de imagem não suportado (use JPG, PNG, WEBP ou GIF)")
	}
	url, err := s.files.Save(ctx, contentType, ext, r)
	if err != nil {
		return nil, err
	}
	if err := s.occurrences.UpdateImage(ctx, id, url); err != nil {
		return nil, err
	}
	return s.occurrences.GetByID(ctx, id)
}

func (s *OccurrenceService) Dashboard(ctx context.Context, actor Actor) (*domain.DashboardStats, error) {
	if !actor.IsGestor() {
		return nil, domain.ErrForbidden
	}
	return s.occurrences.Stats(ctx)
}

// load busca a ocorrência garantindo que o ator tenha permissão de leitura.
// Para o solicitante, ocorrências de terceiros aparecem como inexistentes.
func (s *OccurrenceService) load(ctx context.Context, actor Actor, id int64) (*domain.Occurrence, error) {
	o, err := s.occurrences.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !actor.IsGestor() && o.RequesterID != actor.ID {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *OccurrenceService) loadForGestor(ctx context.Context, actor Actor, id int64) (*domain.Occurrence, error) {
	if !actor.IsGestor() {
		return nil, domain.ErrForbidden
	}
	return s.occurrences.GetByID(ctx, id)
}
