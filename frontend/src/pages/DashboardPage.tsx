import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { PRIORITY_LABEL, STATUS_LABEL } from '../lib/labels'
import type { CountItem, DashboardStats, Priority, Status } from '../lib/types'

export function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api.dashboard().then(setStats).catch((e) => setError(e.message))
  }, [])

  if (error) return <p className="error">{error}</p>
  if (!stats) return <p className="muted">Carregando…</p>

  const resolved = stats.by_status.find((s) => s.key === 'resolvida')?.count ?? 0

  return (
    <>
      <div className="page-head">
        <h1>Dashboard</h1>
      </div>
      <div className="tiles">
        <Tile label="Total de ocorrências" value={stats.total} />
        <Tile label="Em aberto" value={stats.open} hint={`${stats.unassigned} sem responsável`} />
        <Tile label="Resolvidas" value={resolved} />
        <Tile
          label="Tempo médio de resolução"
          value={stats.avg_resolution_hours != null ? formatHours(stats.avg_resolution_hours) : '—'}
        />
        <Tile label="Satisfação média" value={stats.avg_rating != null ? `${stats.avg_rating.toFixed(1)} / 5` : '—'} />
      </div>
      <div className="charts">
        <Bars title="Por status" items={stats.by_status} label={(k) => STATUS_LABEL[k as Status]} cls={(k) => `status-${k}`} />
        <Bars
          title="Por prioridade"
          items={stats.by_priority}
          label={(k) => PRIORITY_LABEL[k as Priority]}
          cls={(k) => `priority-${k}`}
        />
        <Bars title="Por categoria" items={stats.by_category} label={(k) => k} cls={() => 'bar-default'} />
      </div>
    </>
  )
}

function Tile({ label, value, hint }: { label: string; value: string | number; hint?: string }) {
  return (
    <div className="card tile">
      <div className="muted small">{label}</div>
      <div className="tile-value">{value}</div>
      {hint && <div className="muted small">{hint}</div>}
    </div>
  )
}

function Bars({
  title,
  items,
  label,
  cls,
}: {
  title: string
  items: CountItem[]
  label: (k: string) => string
  cls: (k: string) => string
}) {
  const max = Math.max(1, ...items.map((i) => i.count))
  return (
    <section className="card">
      <h3>{title}</h3>
      {items.length === 0 && <p className="muted small">Sem dados.</p>}
      <ul className="bars">
        {items.map((i) => (
          <li key={i.key}>
            <span className="bar-label">{label(i.key)}</span>
            <span className="bar-track">
              <span className={`bar ${cls(i.key)}`} style={{ width: `${(i.count / max) * 100}%` }} />
            </span>
            <span className="bar-count">{i.count}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}

function formatHours(h: number): string {
  if (h < 1) return `${Math.round(h * 60)} min`
  if (h < 48) return `${h.toFixed(1)} h`
  return `${(h / 24).toFixed(1)} dias`
}
