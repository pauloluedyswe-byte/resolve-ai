package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resolveai/internal/domain"
)

type OccurrenceRepo struct{ db *pgxpool.Pool }

func NewOccurrenceRepo(db *pgxpool.Pool) *OccurrenceRepo { return &OccurrenceRepo{db: db} }

const occurrenceSelect = `
SELECT o.id, o.title, o.description, o.category_id, c.name, o.location, o.image_url,
       o.priority, o.status, o.requester_id, r.name, o.assignee_id, a.name,
       o.solution, o.rating, o.rating_comment, o.created_at, o.updated_at, o.resolved_at
FROM occurrences o
JOIN categories c ON c.id = o.category_id
JOIN users r      ON r.id = o.requester_id
LEFT JOIN users a ON a.id = o.assignee_id`

func scanOccurrence(row pgx.Row) (*domain.Occurrence, error) {
	var o domain.Occurrence
	var rating *int16
	err := row.Scan(&o.ID, &o.Title, &o.Description, &o.CategoryID, &o.CategoryName, &o.Location, &o.ImageURL,
		&o.Priority, &o.Status, &o.RequesterID, &o.RequesterName, &o.AssigneeID, &o.AssigneeName,
		&o.Solution, &rating, &o.RatingComment, &o.CreatedAt, &o.UpdatedAt, &o.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if rating != nil {
		v := int(*rating)
		o.Rating = &v
	}
	return &o, err
}

func (r *OccurrenceRepo) Create(ctx context.Context, o *domain.Occurrence) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO occurrences (title, description, category_id, location, priority, status, requester_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`,
			o.Title, o.Description, o.CategoryID, o.Location, o.Priority, o.Status, o.RequesterID,
		).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO status_history (occurrence_id, from_status, to_status, changed_by, note)
			 VALUES ($1, NULL, $2, $3, 'Ocorrência registrada')`,
			o.ID, o.Status, o.RequesterID)
		return err
	})
}

func (r *OccurrenceRepo) GetByID(ctx context.Context, id int64) (*domain.Occurrence, error) {
	return scanOccurrence(r.db.QueryRow(ctx, occurrenceSelect+` WHERE o.id = $1`, id))
}

func (r *OccurrenceRepo) List(ctx context.Context, f domain.OccurrenceFilter) ([]domain.Occurrence, error) {
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.RequesterID != nil {
		add("o.requester_id = $%d", *f.RequesterID)
	}
	if f.CategoryID != nil {
		add("o.category_id = $%d", *f.CategoryID)
	}
	if f.Status != nil {
		add("o.status = $%d", *f.Status)
	}
	if f.Priority != nil {
		add("o.priority = $%d", *f.Priority)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		add("(o.title ILIKE $%[1]d OR o.description ILIKE $%[1]d OR o.location ILIKE $%[1]d)", "%"+s+"%")
	}
	q := occurrenceSelect
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += ` ORDER BY o.created_at DESC`

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []domain.Occurrence{}
	for rows.Next() {
		o, err := scanOccurrence(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *o)
	}
	return list, rows.Err()
}

func (r *OccurrenceRepo) ChangeStatus(ctx context.Context, id int64, from, to domain.Status, userID int64, note string, solution *string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE occurrences
			 SET status = $3,
			     solution = COALESCE($4, solution),
			     resolved_at = CASE WHEN $3 = 'resolvida' THEN now() ELSE resolved_at END,
			     updated_at = now()
			 WHERE id = $1 AND status = $2`,
			id, from, to, solution)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrConflict
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO status_history (occurrence_id, from_status, to_status, changed_by, note)
			 VALUES ($1, $2, $3, $4, $5)`,
			id, from, to, userID, note)
		return err
	})
}

func (r *OccurrenceRepo) exec(ctx context.Context, sql string, args ...any) error {
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OccurrenceRepo) UpdatePriority(ctx context.Context, id int64, p domain.Priority) error {
	return r.exec(ctx, `UPDATE occurrences SET priority = $2, updated_at = now() WHERE id = $1`, id, p)
}

func (r *OccurrenceRepo) UpdateAssignee(ctx context.Context, id int64, assigneeID *int64) error {
	return r.exec(ctx, `UPDATE occurrences SET assignee_id = $2, updated_at = now() WHERE id = $1`, id, assigneeID)
}

func (r *OccurrenceRepo) UpdateSolution(ctx context.Context, id int64, solution string) error {
	return r.exec(ctx, `UPDATE occurrences SET solution = $2, updated_at = now() WHERE id = $1`, id, solution)
}

func (r *OccurrenceRepo) UpdateImage(ctx context.Context, id int64, url string) error {
	return r.exec(ctx, `UPDATE occurrences SET image_url = $2, updated_at = now() WHERE id = $1`, id, url)
}

func (r *OccurrenceRepo) SetRating(ctx context.Context, id int64, rating int, comment string) error {
	return r.exec(ctx,
		`UPDATE occurrences SET rating = $2, rating_comment = NULLIF($3, ''), updated_at = now()
		 WHERE id = $1 AND rating IS NULL`, id, rating, comment)
}

func (r *OccurrenceRepo) AddComment(ctx context.Context, c *domain.Comment) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO comments (occurrence_id, author_id, body) VALUES ($1, $2, $3) RETURNING id, created_at`,
		c.OccurrenceID, c.AuthorID, c.Body,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *OccurrenceRepo) ListComments(ctx context.Context, occurrenceID int64) ([]domain.Comment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.id, c.occurrence_id, c.author_id, u.name, u.role, c.body, c.created_at
		 FROM comments c JOIN users u ON u.id = c.author_id
		 WHERE c.occurrence_id = $1 ORDER BY c.created_at`, occurrenceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Comment])
}

func (r *OccurrenceRepo) ListHistory(ctx context.Context, occurrenceID int64) ([]domain.StatusChange, error) {
	rows, err := r.db.Query(ctx,
		`SELECT h.id, h.occurrence_id, h.from_status, h.to_status, h.changed_by, u.name, h.note, h.created_at
		 FROM status_history h JOIN users u ON u.id = h.changed_by
		 WHERE h.occurrence_id = $1 ORDER BY h.created_at, h.id`, occurrenceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.StatusChange])
}

func (r *OccurrenceRepo) Stats(ctx context.Context) (*domain.DashboardStats, error) {
	s := &domain.DashboardStats{}
	err := r.db.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status NOT IN ('resolvida', 'cancelada')),
		       count(*) FILTER (WHERE assignee_id IS NULL AND status NOT IN ('resolvida', 'cancelada')),
		       avg(rating)::float8,
		       (avg(EXTRACT(EPOCH FROM (resolved_at - created_at))) / 3600)::float8
		FROM occurrences`,
	).Scan(&s.Total, &s.Open, &s.Unassigned, &s.AvgRating, &s.AvgResolutionHours)
	if err != nil {
		return nil, err
	}
	groups := []struct {
		dst *[]domain.CountItem
		sql string
	}{
		{&s.ByStatus, `SELECT status, count(*) FROM occurrences GROUP BY status ORDER BY 2 DESC`},
		{&s.ByPriority, `SELECT priority, count(*) FROM occurrences GROUP BY priority ORDER BY 2 DESC`},
		{&s.ByCategory, `SELECT c.name, count(o.id) FROM categories c
		                 LEFT JOIN occurrences o ON o.category_id = c.id
		                 GROUP BY c.name ORDER BY 2 DESC, 1`},
	}
	for _, g := range groups {
		rows, err := r.db.Query(ctx, g.sql)
		if err != nil {
			return nil, err
		}
		items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.CountItem])
		if err != nil {
			return nil, err
		}
		*g.dst = items
	}
	return s, nil
}
