export type Role = 'solicitante' | 'gestor'
export type Status = 'aberta' | 'em_analise' | 'em_atendimento' | 'resolvida' | 'cancelada'
export type Priority = 'baixa' | 'media' | 'alta' | 'critica'

export interface User {
  id: number
  name: string
  email: string
  role: Role
  created_at: string
}

export interface Category {
  id: number
  name: string
}

export interface Occurrence {
  id: number
  title: string
  description: string
  category_id: number
  category_name: string
  location: string
  image_url: string | null
  priority: Priority
  status: Status
  requester_id: number
  requester_name: string
  assignee_id: number | null
  assignee_name: string | null
  solution: string | null
  rating: number | null
  rating_comment: string | null
  created_at: string
  updated_at: string
  resolved_at: string | null
}

export interface Comment {
  id: number
  occurrence_id: number
  author_id: number
  author_name: string
  author_role: Role
  body: string
  created_at: string
}

export interface StatusChange {
  id: number
  from_status: Status | null
  to_status: Status
  changed_by: number
  changed_by_name: string
  note: string
  created_at: string
}

export interface OccurrenceDetail extends Occurrence {
  comments: Comment[]
  history: StatusChange[]
}

export interface CountItem {
  key: string
  count: number
}

export interface DashboardStats {
  total: number
  open: number
  unassigned: number
  avg_rating: number | null
  avg_resolution_hours: number | null
  by_status: CountItem[]
  by_priority: CountItem[]
  by_category: CountItem[]
}

export interface AuthResult {
  token: string
  user: User
}
