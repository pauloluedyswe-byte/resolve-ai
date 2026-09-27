package domain

import "errors"

var (
	ErrNotFound          = errors.New("recurso não encontrado")
	ErrUnauthorized      = errors.New("não autenticado")
	ErrForbidden         = errors.New("acesso negado")
	ErrInvalidCredential = errors.New("e-mail ou senha inválidos")
	ErrEmailTaken        = errors.New("e-mail já cadastrado")
	ErrConflict          = errors.New("a ocorrência foi alterada por outro usuário; recarregue e tente novamente")
)

// ValidationError representa entrada inválida do cliente (HTTP 422).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func Invalid(msg string) error { return &ValidationError{Msg: msg} }
