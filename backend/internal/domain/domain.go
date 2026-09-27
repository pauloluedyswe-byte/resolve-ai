// Package domain contém as entidades e regras de negócio centrais do Resolve Aí.
package domain

import "time"

type Role string

const (
	RoleSolicitante Role = "solicitante"
	RoleGestor      Role = "gestor"
)

type Priority string

const (
	PriorityBaixa   Priority = "baixa"
	PriorityMedia   Priority = "media"
	PriorityAlta    Priority = "alta"
	PriorityCritica Priority = "critica"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityBaixa, PriorityMedia, PriorityAlta, PriorityCritica:
		return true
	}
	return false
}

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Occurrence struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	CategoryID    int64      `json:"category_id"`
	CategoryName  string     `json:"category_name"`
	Location      string     `json:"location"`
	ImageURL      *string    `json:"image_url"`
	Priority      Priority   `json:"priority"`
	Status        Status     `json:"status"`
	RequesterID   int64      `json:"requester_id"`
	RequesterName string     `json:"requester_name"`
	AssigneeID    *int64     `json:"assignee_id"`
	AssigneeName  *string    `json:"assignee_name"`
	Solution      *string    `json:"solution"`
	Rating        *int       `json:"rating"`
	RatingComment *string    `json:"rating_comment"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
}

type Comment struct {
	ID           int64     `json:"id"`
	OccurrenceID int64     `json:"occurrence_id"`
	AuthorID     int64     `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	AuthorRole   Role      `json:"author_role"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"created_at"`
}

// StatusChange é o registro auditável de uma transição de status.
type StatusChange struct {
	ID            int64     `json:"id"`
	OccurrenceID  int64     `json:"occurrence_id"`
	FromStatus    *Status   `json:"from_status"`
	ToStatus      Status    `json:"to_status"`
	ChangedBy     int64     `json:"changed_by"`
	ChangedByName string    `json:"changed_by_name"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
}

type OccurrenceDetail struct {
	Occurrence
	Comments []Comment      `json:"comments"`
	History  []StatusChange `json:"history"`
}

type OccurrenceFilter struct {
	RequesterID *int64
	CategoryID  *int64
	Status      *Status
	Priority    *Priority
	Search      string
}

type CountItem struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type DashboardStats struct {
	Total              int         `json:"total"`
	Open               int         `json:"open"`
	Unassigned         int         `json:"unassigned"`
	AvgRating          *float64    `json:"avg_rating"`
	AvgResolutionHours *float64    `json:"avg_resolution_hours"`
	ByStatus           []CountItem `json:"by_status"`
	ByPriority         []CountItem `json:"by_priority"`
	ByCategory         []CountItem `json:"by_category"`
}
