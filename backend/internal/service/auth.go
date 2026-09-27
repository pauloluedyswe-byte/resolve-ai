package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"resolveai/internal/domain"
)

type AuthService struct {
	users  UserRepository
	secret []byte
	ttl    time.Duration
}

func NewAuthService(users UserRepository, secret string, ttl time.Duration) *AuthService {
	return &AuthService{users: users, secret: []byte(secret), ttl: ttl}
}

type Claims struct {
	Name string      `json:"name"`
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type AuthResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// Register cria sempre um Solicitante; gestores são provisionados pelo administrador.
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*AuthResult, error) {
	u, err := s.createUser(ctx, name, email, password, domain.RoleSolicitante)
	if err != nil {
		return nil, err
	}
	return s.issue(u)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrInvalidCredential
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, domain.ErrInvalidCredential
	}
	return s.issue(u)
}

func (s *AuthService) Me(ctx context.Context, id int64) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// EnsureGestor cria o gestor inicial caso ainda não exista (seed via variáveis de ambiente).
func (s *AuthService) EnsureGestor(ctx context.Context, name, email, password string) error {
	_, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	_, err = s.createUser(ctx, name, email, password, domain.RoleGestor)
	return err
}

func (s *AuthService) ListGestores(ctx context.Context) ([]domain.User, error) {
	return s.users.ListByRole(ctx, domain.RoleGestor)
}

func (s *AuthService) ParseToken(token string) (*Actor, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	var id int64
	if _, err := fmt.Sscan(claims.Subject, &id); err != nil {
		return nil, domain.ErrUnauthorized
	}
	return &Actor{ID: id, Name: claims.Name, Role: claims.Role}, nil
}

func (s *AuthService) createUser(ctx context.Context, name, email, password string, role domain.Role) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" {
		return nil, domain.Invalid("nome é obrigatório")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, domain.Invalid("e-mail inválido")
	}
	if len(password) < 6 {
		return nil, domain.Invalid("a senha deve ter ao menos 6 caracteres")
	}
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Name: name, Email: email, PasswordHash: string(hash), Role: role}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) issue(u *domain.User) (*AuthResult, error) {
	now := time.Now()
	claims := Claims{
		Name: u.Name,
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprint(u.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: u}, nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }
