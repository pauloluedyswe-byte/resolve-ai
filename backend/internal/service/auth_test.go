package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"resolveai/internal/domain"
)

func TestRegisterLoginAndToken(t *testing.T) {
	svc := NewAuthService(newFakeUsers(), "segredo-de-teste-123", time.Hour)
	ctx := context.Background()

	res, err := svc.Register(ctx, "Ana", " Ana@Email.com ", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if res.User.Role != domain.RoleSolicitante || res.User.Email != "ana@email.com" {
		t.Fatalf("usuário inesperado: %+v", res.User)
	}
	if _, err := svc.Register(ctx, "Ana 2", "ana@email.com", "123456"); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("esperava ErrEmailTaken, veio %v", err)
	}
	if _, err := svc.Login(ctx, "ana@email.com", "errada"); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("esperava credencial inválida, veio %v", err)
	}
	login, err := svc.Login(ctx, "ANA@email.com", "123456")
	if err != nil {
		t.Fatal(err)
	}
	actor, err := svc.ParseToken(login.Token)
	if err != nil || actor.ID != res.User.ID || actor.Role != domain.RoleSolicitante {
		t.Fatalf("token inválido: %+v %v", actor, err)
	}
	if _, err := svc.ParseToken(login.Token + "x"); err == nil {
		t.Fatal("token adulterado deveria ser rejeitado")
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := NewAuthService(newFakeUsers(), "segredo-de-teste-123", time.Hour)
	ctx := context.Background()
	for _, c := range [][3]string{{"", "a@a.com", "123456"}, {"A", "invalido", "123456"}, {"A", "a@a.com", "123"}} {
		if _, err := svc.Register(ctx, c[0], c[1], c[2]); !isValidation(err) {
			t.Errorf("%v: esperava erro de validação, veio %v", c, err)
		}
	}
}

func TestEnsureGestorIsIdempotent(t *testing.T) {
	users := newFakeUsers()
	svc := NewAuthService(users, "segredo-de-teste-123", time.Hour)
	ctx := context.Background()
	for range 2 {
		if err := svc.EnsureGestor(ctx, "Admin", "admin@resolveai.com", "admin123"); err != nil {
			t.Fatal(err)
		}
	}
	g, _ := svc.ListGestores(ctx)
	if len(g) != 1 {
		t.Fatalf("esperava 1 gestor, veio %d", len(g))
	}
}
