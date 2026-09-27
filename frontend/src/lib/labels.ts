import type { Priority, Role, Status } from './types'

export const STATUS_LABEL: Record<Status, string> = {
  aberta: 'Aberta',
  em_analise: 'Em análise',
  em_atendimento: 'Em atendimento',
  resolvida: 'Resolvida',
  cancelada: 'Cancelada',
}

export const PRIORITY_LABEL: Record<Priority, string> = {
  baixa: 'Baixa',
  media: 'Média',
  alta: 'Alta',
  critica: 'Crítica',
}

export const ROLE_LABEL: Record<Role, string> = {
  solicitante: 'Solicitante',
  gestor: 'Gestor',
}

export const STATUSES = Object.keys(STATUS_LABEL) as Status[]
export const PRIORITIES = Object.keys(PRIORITY_LABEL) as Priority[]

// Espelha as regras de transição do backend (internal/domain/status.go).
const TRANSITIONS: Record<Status, Status[]> = {
  aberta: ['em_analise', 'cancelada'],
  em_analise: ['em_atendimento', 'cancelada'],
  em_atendimento: ['resolvida', 'cancelada'],
  resolvida: [],
  cancelada: [],
}

export function nextStatuses(current: Status, role: Role): Status[] {
  if (role === 'gestor') return TRANSITIONS[current]
  return current === 'aberta' ? ['cancelada'] : []
}

export function isFinal(s: Status): boolean {
  return s === 'resolvida' || s === 'cancelada'
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}
