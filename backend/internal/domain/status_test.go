package domain

import "testing"

func TestStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusAberta, StatusEmAnalise, true},
		{StatusAberta, StatusCancelada, true},
		{StatusAberta, StatusEmAtendimento, false},
		{StatusAberta, StatusResolvida, false},
		{StatusEmAnalise, StatusEmAtendimento, true},
		{StatusEmAnalise, StatusCancelada, true},
		{StatusEmAnalise, StatusAberta, false},
		{StatusEmAtendimento, StatusResolvida, true},
		{StatusEmAtendimento, StatusCancelada, true},
		{StatusResolvida, StatusCancelada, false},
		{StatusResolvida, StatusAberta, false},
		{StatusCancelada, StatusAberta, false},
	}
	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("%s -> %s: got %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestFinalStatuses(t *testing.T) {
	for _, s := range []Status{StatusResolvida, StatusCancelada} {
		if !s.IsFinal() || len(s.NextStatuses()) != 0 {
			t.Errorf("%s deveria ser final sem transições", s)
		}
	}
	if StatusAberta.IsFinal() {
		t.Error("aberta não deveria ser final")
	}
}

func TestValidators(t *testing.T) {
	if Status("xyz").Valid() || !StatusEmAnalise.Valid() {
		t.Error("Status.Valid incorreto")
	}
	if Priority("urgente").Valid() || !PriorityCritica.Valid() {
		t.Error("Priority.Valid incorreto")
	}
}
