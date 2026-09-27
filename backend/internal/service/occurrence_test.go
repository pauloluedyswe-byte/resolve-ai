package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"resolveai/internal/domain"
)

type fixture struct {
	svc       *OccurrenceService
	repo      *fakeOccurrences
	requester Actor
	other     Actor
	gestor    Actor
}

func setup(t *testing.T) *fixture {
	t.Helper()
	users := newFakeUsers()
	ctx := context.Background()
	mk := func(name string, role domain.Role) Actor {
		u := &domain.User{Name: name, Email: strings.ToLower(name) + "@x.com", Role: role}
		users.Create(ctx, u)
		return Actor{ID: u.ID, Name: u.Name, Role: u.Role}
	}
	repo := newFakeOccurrences()
	return &fixture{
		svc:       NewOccurrenceService(repo, fakeCategories{}, users, fakeFiles{}),
		repo:      repo,
		requester: mk("Ana", domain.RoleSolicitante),
		other:     mk("Bruno", domain.RoleSolicitante),
		gestor:    mk("Gabi", domain.RoleGestor),
	}
}

func (f *fixture) create(t *testing.T) *domain.Occurrence {
	t.Helper()
	o, err := f.svc.Create(context.Background(), f.requester, CreateOccurrenceInput{
		Title: "Lâmpada queimada", Description: "Corredor escuro", CategoryID: 1, Location: "Bloco B",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return o
}

func isValidation(err error) bool {
	var ve *domain.ValidationError
	return errors.As(err, &ve)
}

func TestCreateValidatesInput(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	bad := []CreateOccurrenceInput{
		{Description: "d", CategoryID: 1, Location: "l"},
		{Title: "t", CategoryID: 1, Location: "l"},
		{Title: "t", Description: "d", CategoryID: 1},
		{Title: "t", Description: "d", CategoryID: 99, Location: "l"},
	}
	for i, in := range bad {
		if _, err := f.svc.Create(ctx, f.requester, in); !isValidation(err) {
			t.Errorf("caso %d: esperava erro de validação, veio %v", i, err)
		}
	}
}

func TestCreateStartsOpenWithHistory(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	if o.Status != domain.StatusAberta || o.Priority != domain.PriorityMedia {
		t.Fatalf("estado inicial inesperado: %s/%s", o.Status, o.Priority)
	}
	d, _ := f.svc.Get(context.Background(), f.requester, o.ID)
	if len(d.History) != 1 || d.History[0].FromStatus != nil {
		t.Fatalf("histórico inicial inesperado: %+v", d.History)
	}
}

func TestRequesterCannotSeeOthersOccurrences(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.Get(ctx, f.other, o.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, veio %v", err)
	}
	list, _ := f.svc.List(ctx, f.other, domain.OccurrenceFilter{})
	if len(list) != 0 {
		t.Fatalf("solicitante viu ocorrência alheia")
	}
	list, _ = f.svc.List(ctx, f.gestor, domain.OccurrenceFilter{})
	if len(list) != 1 {
		t.Fatalf("gestor deveria ver todas")
	}
}

func TestFullLifecycleRecordsHistory(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	steps := []domain.Status{domain.StatusEmAnalise, domain.StatusEmAtendimento}
	for _, s := range steps {
		if _, err := f.svc.ChangeStatus(ctx, f.gestor, o.ID, ChangeStatusInput{Status: s, Note: "ok"}); err != nil {
			t.Fatalf("-> %s: %v", s, err)
		}
	}
	// Resolver sem solução deve falhar.
	if _, err := f.svc.ChangeStatus(ctx, f.gestor, o.ID, ChangeStatusInput{Status: domain.StatusResolvida}); !isValidation(err) {
		t.Fatalf("esperava exigir solução, veio %v", err)
	}
	sol := "Lâmpada trocada"
	got, err := f.svc.ChangeStatus(ctx, f.gestor, o.ID, ChangeStatusInput{Status: domain.StatusResolvida, Solution: &sol})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusResolvida || *got.Solution != sol {
		t.Fatalf("resolução não aplicada: %+v", got)
	}
	d, _ := f.svc.Get(ctx, f.gestor, o.ID)
	if len(d.History) != 4 {
		t.Fatalf("esperava 4 registros de histórico, veio %d", len(d.History))
	}
	last := d.History[3]
	if *last.FromStatus != domain.StatusEmAtendimento || last.ToStatus != domain.StatusResolvida || last.ChangedBy != f.gestor.ID {
		t.Fatalf("registro de histórico incorreto: %+v", last)
	}
}

func TestInvalidTransitionRejected(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	_, err := f.svc.ChangeStatus(context.Background(), f.gestor, o.ID, ChangeStatusInput{Status: domain.StatusResolvida})
	if !isValidation(err) {
		t.Fatalf("aberta -> resolvida deveria falhar, veio %v", err)
	}
}

func TestRequesterPermissionsOnStatus(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.ChangeStatus(ctx, f.requester, o.ID, ChangeStatusInput{Status: domain.StatusEmAnalise}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("solicitante não pode avançar status, veio %v", err)
	}
	if _, err := f.svc.ChangeStatus(ctx, f.requester, o.ID, ChangeStatusInput{Status: domain.StatusCancelada}); !isValidation(err) {
		t.Fatalf("cancelamento sem motivo deveria falhar, veio %v", err)
	}
	got, err := f.svc.ChangeStatus(ctx, f.requester, o.ID, ChangeStatusInput{Status: domain.StatusCancelada, Note: "Duplicada"})
	if err != nil || got.Status != domain.StatusCancelada {
		t.Fatalf("solicitante deveria poder cancelar ocorrência aberta: %v", err)
	}
}

func TestGestorOnlyOperations(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.UpdatePriority(ctx, f.requester, o.ID, domain.PriorityAlta); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("esperava ErrForbidden, veio %v", err)
	}
	if got, err := f.svc.UpdatePriority(ctx, f.gestor, o.ID, domain.PriorityAlta); err != nil || got.Priority != domain.PriorityAlta {
		t.Fatalf("prioridade não alterada: %v", err)
	}
	if _, err := f.svc.Assign(ctx, f.gestor, o.ID, &f.requester.ID); !isValidation(err) {
		t.Fatalf("responsável solicitante deveria ser rejeitado, veio %v", err)
	}
	if got, err := f.svc.Assign(ctx, f.gestor, o.ID, &f.gestor.ID); err != nil || *got.AssigneeID != f.gestor.ID {
		t.Fatalf("atribuição falhou: %v", err)
	}
	if _, err := f.svc.Dashboard(ctx, f.requester); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("dashboard deveria ser restrito")
	}
}

func TestRating(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.Rate(ctx, f.requester, o.ID, 5, ""); !isValidation(err) {
		t.Fatalf("não deveria avaliar antes de resolver, veio %v", err)
	}
	sol := "feito"
	for _, s := range []domain.Status{domain.StatusEmAnalise, domain.StatusEmAtendimento, domain.StatusResolvida} {
		if _, err := f.svc.ChangeStatus(ctx, f.gestor, o.ID, ChangeStatusInput{Status: s, Solution: &sol}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Rate(ctx, f.gestor, o.ID, 5, ""); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("apenas o solicitante avalia, veio %v", err)
	}
	if _, err := f.svc.Rate(ctx, f.requester, o.ID, 6, ""); !isValidation(err) {
		t.Fatalf("nota fora do intervalo deveria falhar")
	}
	if got, err := f.svc.Rate(ctx, f.requester, o.ID, 4, "Rápido"); err != nil || *got.Rating != 4 {
		t.Fatalf("avaliação falhou: %v", err)
	}
	if _, err := f.svc.Rate(ctx, f.requester, o.ID, 5, ""); !isValidation(err) {
		t.Fatalf("avaliação dupla deveria falhar")
	}
}

func TestAttachImageValidatesType(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.AttachImage(ctx, f.requester, o.ID, "application/pdf", strings.NewReader("x")); !isValidation(err) {
		t.Fatalf("PDF deveria ser rejeitado")
	}
	got, err := f.svc.AttachImage(ctx, f.requester, o.ID, "image/png", strings.NewReader("x"))
	if err != nil || got.ImageURL == nil || !strings.HasSuffix(*got.ImageURL, ".png") {
		t.Fatalf("upload falhou: %v", err)
	}
}

func TestComments(t *testing.T) {
	f := setup(t)
	o := f.create(t)
	ctx := context.Background()
	if _, err := f.svc.AddComment(ctx, f.other, o.ID, "oi"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("terceiro não pode comentar, veio %v", err)
	}
	if _, err := f.svc.AddComment(ctx, f.gestor, o.ID, "  "); !isValidation(err) {
		t.Fatalf("comentário vazio deveria falhar")
	}
	f.svc.AddComment(ctx, f.gestor, o.ID, "Equipe a caminho")
	f.svc.AddComment(ctx, f.requester, o.ID, "Obrigado")
	d, _ := f.svc.Get(ctx, f.requester, o.ID)
	if len(d.Comments) != 2 {
		t.Fatalf("esperava 2 comentários, veio %d", len(d.Comments))
	}
}
