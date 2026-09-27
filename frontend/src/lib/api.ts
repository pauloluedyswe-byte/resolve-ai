import type {
  AuthResult,
  Category,
  Comment,
  DashboardStats,
  Occurrence,
  OccurrenceDetail,
  Priority,
  Status,
  User,
} from './types'

const BASE = import.meta.env.VITE_API_URL ?? ''
const TOKEN_KEY = 'resolveai.token'

export const tokenStore = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

let onUnauthorized: () => void = () => {}
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const token = tokenStore.get()
  if (token) headers.Authorization = `Bearer ${token}`
  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  let res: Response
  try {
    // 90 s cobre o "cold start" do plano gratuito do Render (~50 s).
    res = await fetch(`${BASE}/api${path}`, { method, headers, body: payload, signal: AbortSignal.timeout(90_000) })
  } catch {
    throw new ApiError(0, 'O servidor não respondeu. Verifique sua conexão e tente novamente.')
  }
  const data = res.status === 204 ? null : await res.json().catch(() => null)
  if (!res.ok) {
    if (res.status === 401 && token) onUnauthorized()
    throw new ApiError(res.status, data?.error ?? `Erro ${res.status}`)
  }
  return data as T
}

export function imageUrl(path: string | null): string | null {
  return path ? `${BASE}${path}` : null
}

export interface OccurrenceFilters {
  status?: Status | ''
  priority?: Priority | ''
  category_id?: string
  q?: string
}

export const api = {
  login: (email: string, password: string) => request<AuthResult>('POST', '/auth/login', { email, password }),
  register: (name: string, email: string, password: string) =>
    request<AuthResult>('POST', '/auth/register', { name, email, password }),
  me: () => request<User>('GET', '/auth/me'),

  categories: () => request<Category[]>('GET', '/categories'),
  gestores: () => request<User[]>('GET', '/users/gestores'),
  dashboard: () => request<DashboardStats>('GET', '/dashboard'),

  listOccurrences: (f: OccurrenceFilters = {}) => {
    const qs = new URLSearchParams(Object.entries(f).filter(([, v]) => v) as [string, string][])
    return request<Occurrence[]>('GET', `/occurrences${qs.size ? `?${qs}` : ''}`)
  },
  getOccurrence: (id: number) => request<OccurrenceDetail>('GET', `/occurrences/${id}`),
  createOccurrence: (input: { title: string; description: string; category_id: number; location: string }) =>
    request<Occurrence>('POST', '/occurrences', input),
  uploadImage: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('image', file)
    return request<Occurrence>('POST', `/occurrences/${id}/image`, fd)
  },
  addComment: (id: number, body: string) => request<Comment>('POST', `/occurrences/${id}/comments`, { body }),
  changeStatus: (id: number, status: Status, note: string, solution?: string) =>
    request<Occurrence>('PATCH', `/occurrences/${id}/status`, { status, note, solution }),
  updatePriority: (id: number, priority: Priority) =>
    request<Occurrence>('PATCH', `/occurrences/${id}/priority`, { priority }),
  assign: (id: number, assignee_id: number | null) =>
    request<Occurrence>('PATCH', `/occurrences/${id}/assignee`, { assignee_id }),
  registerSolution: (id: number, solution: string) =>
    request<Occurrence>('PATCH', `/occurrences/${id}/solution`, { solution }),
  rate: (id: number, rating: number, comment: string) =>
    request<Occurrence>('POST', `/occurrences/${id}/rating`, { rating, comment }),
}
