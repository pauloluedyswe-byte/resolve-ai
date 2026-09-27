package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// LocalFileStore grava arquivos em disco e os expõe em /uploads/<nome>.
// Em produção, o diretório deve ser um volume persistente (ou trocado por
// uma implementação de object storage, como S3/GCS, com a mesma interface).
type LocalFileStore struct{ dir string }

func NewLocalFileStore(dir string) (*LocalFileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &LocalFileStore{dir: dir}, nil
}

func (s *LocalFileStore) Dir() string { return s.dir }

func (s *LocalFileStore) Save(_ context.Context, ext string, r io.Reader) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	name := hex.EncodeToString(buf) + ext
	f, err := os.Create(filepath.Join(s.dir, name))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}
