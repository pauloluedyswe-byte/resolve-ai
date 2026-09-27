package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resolveai/internal/domain"
)

// FileRepo guarda os arquivos enviados no PostgreSQL e os expõe em /uploads/<nome>.
type FileRepo struct{ db *pgxpool.Pool }

func NewFileRepo(db *pgxpool.Pool) *FileRepo { return &FileRepo{db: db} }

func (r *FileRepo) Save(ctx context.Context, contentType, ext string, src io.Reader) (string, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	name := hex.EncodeToString(buf) + ext
	if _, err := r.db.Exec(ctx,
		`INSERT INTO files (name, content_type, data) VALUES ($1, $2, $3)`, name, contentType, data); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}

func (r *FileRepo) Get(ctx context.Context, name string) (string, []byte, error) {
	var contentType string
	var data []byte
	err := r.db.QueryRow(ctx, `SELECT content_type, data FROM files WHERE name = $1`, name).Scan(&contentType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, domain.ErrNotFound
	}
	return contentType, data, err
}
