import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { PriorityBadge, StatusBadge } from '../components/Badges'
import { api, type OccurrenceFilters } from '../lib/api'
import { PRIORITIES, PRIORITY_LABEL, STATUSES, STATUS_LABEL, formatDate } from '../lib/labels'
import type { Category, Occurrence } from '../lib/types'

export function OccurrencesPage() {
  const { user } = useAuth()
  const isGestor = user?.role === 'gestor'
  const [items, setItems] = useState<Occurrence[] | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [filters, setFilters] = useState<OccurrenceFilters>({ status: '', priority: '', category_id: '', q: '' })
  const [error, setError] = useState('')

  useEffect(() => {
    api.categories().then(setCategories).catch(() => {})
  }, [])

  useEffect(() => {
    const t = setTimeout(() => {
      api
        .listOccurrences(filters)
        .then((list) => {
          setItems(list)
          setError('')
        })
        .catch((e) => setError(e.message))
    }, 250)
    return () => clearTimeout(t)
  }, [filters])

  const set = (k: keyof OccurrenceFilters) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setFilters({ ...filters, [k]: e.target.value })

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{isGestor ? 'Todas as ocorrências' : 'Minhas ocorrências'}</h1>
          <p className="muted">{items ? `${items.length} encontrada(s)` : 'Carregando…'}</p>
        </div>
        <Link to="/nova" className="btn">
          + Nova ocorrência
        </Link>
      </div>

      <div className="filters card">
        <input placeholder="Buscar por título, descrição ou local" value={filters.q} onChange={set('q')} />
        <select value={filters.status} onChange={set('status')} aria-label="Status">
          <option value="">Todos os status</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {STATUS_LABEL[s]}
            </option>
          ))}
        </select>
        <select value={filters.priority} onChange={set('priority')} aria-label="Prioridade">
          <option value="">Todas as prioridades</option>
          {PRIORITIES.map((p) => (
            <option key={p} value={p}>
              {PRIORITY_LABEL[p]}
            </option>
          ))}
        </select>
        <select value={filters.category_id} onChange={set('category_id')} aria-label="Categoria">
          <option value="">Todas as categorias</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>

      {error && <p className="error">{error}</p>}

      {items && items.length === 0 && (
        <div className="card empty">
          <p>Nenhuma ocorrência encontrada.</p>
          <Link to="/nova">Registrar a primeira</Link>
        </div>
      )}

      <div className="list">
        {items?.map((o) => (
          <Link to={`/ocorrencias/${o.id}`} key={o.id} className="card occ-row">
            <div className="occ-main">
              <div className="occ-title">
                <span className="muted">#{o.id}</span> {o.title}
              </div>
              <div className="muted small">
                {o.category_name} · {o.location}
                {isGestor && <> · por {o.requester_name}</>} · {formatDate(o.created_at)}
              </div>
            </div>
            <div className="occ-meta">
              {isGestor && <span className="muted small">{o.assignee_name ?? 'Sem responsável'}</span>}
              <PriorityBadge priority={o.priority} />
              <StatusBadge status={o.status} />
            </div>
          </Link>
        ))}
      </div>
    </>
  )
}
