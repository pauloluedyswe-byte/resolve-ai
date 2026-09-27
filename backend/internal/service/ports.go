package service

import (
	"context"

	"resolveai/internal/domain"
)

// Interfaces de persistência consumidas pelos serviços. A implementação
// PostgreSQL fica em internal/repository; os testes usam fakes em memória.

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	ListByRole(ctx context.Context, role domain.Role) ([]domain.User, error)
}

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Exists(ctx context.Context, id int64) (bool, error)
}

type OccurrenceRepository interface {
	// Create insere a ocorrência e o registro inicial de histórico na mesma transação.
	Create(ctx context.Context, o *domain.Occurrence) error
	GetByID(ctx context.Context, id int64) (*domain.Occurrence, error)
	List(ctx context.Context, f domain.OccurrenceFilter) ([]domain.Occurrence, error)
	// ChangeStatus atualiza o status (se ainda for `from`) e grava o histórico atomicamente.
	// Retorna domain.ErrConflict se o status atual não for mais `from`.
	ChangeStatus(ctx context.Context, id int64, from, to domain.Status, userID int64, note string, solution *string) error
	UpdatePriority(ctx context.Context, id int64, p domain.Priority) error
	UpdateAssignee(ctx context.Context, id int64, assigneeID *int64) error
	UpdateSolution(ctx context.Context, id int64, solution string) error
	UpdateImage(ctx context.Context, id int64, url string) error
	SetRating(ctx context.Context, id int64, rating int, comment string) error
	AddComment(ctx context.Context, c *domain.Comment) error
	ListComments(ctx context.Context, occurrenceID int64) ([]domain.Comment, error)
	ListHistory(ctx context.Context, occurrenceID int64) ([]domain.StatusChange, error)
	Stats(ctx context.Context) (*domain.DashboardStats, error)
}

// Actor é o usuário autenticado que executa a operação.
type Actor struct {
	ID   int64
	Name string
	Role domain.Role
}

func (a Actor) IsGestor() bool { return a.Role == domain.RoleGestor }
