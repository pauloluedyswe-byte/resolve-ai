package domain

type Status string

const (
	StatusAberta        Status = "aberta"
	StatusEmAnalise     Status = "em_analise"
	StatusEmAtendimento Status = "em_atendimento"
	StatusResolvida     Status = "resolvida"
	StatusCancelada     Status = "cancelada"
)

// transitions define o ciclo de vida da ocorrência:
// Aberta → Em análise → Em atendimento → Resolvida, com cancelamento
// possível a partir de qualquer estado não final.
var transitions = map[Status][]Status{
	StatusAberta:        {StatusEmAnalise, StatusCancelada},
	StatusEmAnalise:     {StatusEmAtendimento, StatusCancelada},
	StatusEmAtendimento: {StatusResolvida, StatusCancelada},
}

func (s Status) Valid() bool {
	switch s {
	case StatusAberta, StatusEmAnalise, StatusEmAtendimento, StatusResolvida, StatusCancelada:
		return true
	}
	return false
}

func (s Status) IsFinal() bool {
	return s == StatusResolvida || s == StatusCancelada
}

func (s Status) NextStatuses() []Status {
	return transitions[s]
}

func (s Status) CanTransitionTo(next Status) bool {
	for _, t := range transitions[s] {
		if t == next {
			return true
		}
	}
	return false
}
