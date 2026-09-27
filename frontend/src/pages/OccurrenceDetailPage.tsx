import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { PriorityBadge, Stars, StatusBadge } from '../components/Badges'
import { api, imageUrl } from '../lib/api'
import { PRIORITIES, PRIORITY_LABEL, ROLE_LABEL, STATUS_LABEL, formatDate, isFinal, nextStatuses } from '../lib/labels'
import type { OccurrenceDetail, Priority, Status, User } from '../lib/types'

export function OccurrenceDetailPage() {
  const { id } = useParams()
  const occId = Number(id)
  const { user } = useAuth()
  const [occ, setOcc] = useState<OccurrenceDetail | null>(null)
  const [error, setError] = useState('')
  const [flash, setFlash] = useState(() => {
    const msg = sessionStorage.getItem('resolveai.flash')
    sessionStorage.removeItem('resolveai.flash')
    return msg ?? ''
  })

  const reload = useCallback(() => {
    api
      .getOccurrence(occId)
      .then((d) => {
        setOcc(d)
        setError('')
      })
      .catch((e) => setError(e.message))
  }, [occId])

  useEffect(reload, [reload])

  // Executa uma ação, recarrega o detalhe e mostra o erro (se houver) no topo.
  const run = async (fn: () => Promise<unknown>) => {
    try {
      await fn()
      setFlash('')
      reload()
      return true
    } catch (e) {
      setFlash((e as Error).message)
      return false
    }
  }

  if (error) {
    return (
      <div className="card empty">
        <p className="error">{error}</p>
        <Link to="/">Voltar</Link>
      </div>
    )
  }
  if (!occ || !user) return <p className="muted">Carregando…</p>

  const isGestor = user.role === 'gestor'
  const isOwner = occ.requester_id === user.id
  const img = imageUrl(occ.image_url)

  return (
    <>
      <Link to="/" className="muted small">
        ← Voltar
      </Link>
      <div className="page-head">
        <div>
          <h1>
            <span className="muted">#{occ.id}</span> {occ.title}
          </h1>
          <div className="badges">
            <StatusBadge status={occ.status} />
            <PriorityBadge priority={occ.priority} />
            <span className="badge neutral">{occ.category_name}</span>
          </div>
        </div>
      </div>

      {flash && <p className="error">{flash}</p>}

      <div className="detail-grid">
        <div className="stack">
          <section className="card">
            <dl className="info">
              <dt>Localização</dt>
              <dd>{occ.location}</dd>
              <dt>Solicitante</dt>
              <dd>{occ.requester_name}</dd>
              <dt>Responsável</dt>
              <dd>{occ.assignee_name ?? <span className="muted">Não atribuído</span>}</dd>
              <dt>Aberta em</dt>
              <dd>{formatDate(occ.created_at)}</dd>
              {occ.resolved_at && (
                <>
                  <dt>Resolvida em</dt>
                  <dd>{formatDate(occ.resolved_at)}</dd>
                </>
              )}
            </dl>
            <h3>Descrição</h3>
            <p className="prewrap">{occ.description}</p>
            {img && (
              <a href={img} target="_blank" rel="noreferrer">
                <img src={img} alt="Imagem anexada" className="attachment" />
              </a>
            )}
            {!img && (isOwner || isGestor) && !isFinal(occ.status) && (
              <ImageUpload onUpload={(f) => run(() => api.uploadImage(occ.id, f))} />
            )}
          </section>

          {occ.solution && (
            <section className="card solution">
              <h3>Solução aplicada</h3>
              <p className="prewrap">{occ.solution}</p>
            </section>
          )}

          {occ.status === 'resolvida' &&
            (occ.rating ? (
              <section className="card">
                <h3>Avaliação do solicitante</h3>
                <Stars value={occ.rating} />
                {occ.rating_comment && <p className="prewrap">{occ.rating_comment}</p>}
              </section>
            ) : (
              isOwner && <RatingForm onSubmit={(r, c) => run(() => api.rate(occ.id, r, c))} />
            ))}

          <Comments occ={occ} onSubmit={(body) => run(() => api.addComment(occ.id, body))} />
        </div>

        <div className="stack">
          {isGestor && !isFinal(occ.status) && <GestorPanel occ={occ} run={run} />}
          {!isGestor && isOwner && occ.status === 'aberta' && (
            <section className="card">
              <StatusChanger occ={occ} role="solicitante" run={run} />
            </section>
          )}
          <History occ={occ} />
        </div>
      </div>
    </>
  )
}

type Run = (fn: () => Promise<unknown>) => Promise<boolean>

function GestorPanel({ occ, run }: { occ: OccurrenceDetail; run: Run }) {
  const [gestores, setGestores] = useState<User[]>([])
  const [solution, setSolution] = useState(occ.solution ?? '')

  useEffect(() => {
    api.gestores().then(setGestores).catch(() => {})
  }, [])

  return (
    <section className="card">
      <h3>Gestão</h3>
      <div className="form">
        <label>
          Prioridade
          <select
            value={occ.priority}
            onChange={(e) => run(() => api.updatePriority(occ.id, e.target.value as Priority))}
          >
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {PRIORITY_LABEL[p]}
              </option>
            ))}
          </select>
        </label>
        <label>
          Responsável
          <select
            value={occ.assignee_id ?? ''}
            onChange={(e) => run(() => api.assign(occ.id, e.target.value ? Number(e.target.value) : null))}
          >
            <option value="">Não atribuído</option>
            {gestores.map((g) => (
              <option key={g.id} value={g.id}>
                {g.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Solução aplicada
          <textarea rows={3} value={solution} onChange={(e) => setSolution(e.target.value)} />
        </label>
        <button
          className="btn secondary"
          disabled={!solution.trim() || solution === (occ.solution ?? '')}
          onClick={() => run(() => api.registerSolution(occ.id, solution))}
        >
          Salvar solução
        </button>
      </div>
      <hr />
      <StatusChanger key={occ.status} occ={occ} role="gestor" run={run} pendingSolution={solution} />
    </section>
  )
}

function StatusChanger({
  occ,
  role,
  run,
  pendingSolution,
}: {
  occ: OccurrenceDetail
  role: 'gestor' | 'solicitante'
  run: Run
  pendingSolution?: string
}) {
  const options = nextStatuses(occ.status, role)
  const [next, setNext] = useState<Status>(options[0])
  const [note, setNote] = useState('')

  if (options.length === 0) return null

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const solution = next === 'resolvida' ? pendingSolution : undefined
    if (await run(() => api.changeStatus(occ.id, next, note, solution))) setNote('')
  }

  return (
    <form className="form" onSubmit={submit}>
      <h3>{role === 'gestor' ? 'Atualizar status' : 'Cancelar ocorrência'}</h3>
      {role === 'gestor' && (
        <label>
          Novo status
          <select value={next} onChange={(e) => setNext(e.target.value as Status)}>
            {options.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABEL[s]}
              </option>
            ))}
          </select>
        </label>
      )}
      <label>
        Observação {next === 'cancelada' && <small className="muted">(obrigatória)</small>}
        <textarea rows={2} value={note} onChange={(e) => setNote(e.target.value)} required={next === 'cancelada'} />
      </label>
      {next === 'resolvida' && !pendingSolution?.trim() && (
        <p className="muted small">Preencha a solução aplicada antes de resolver.</p>
      )}
      <button className={`btn ${next === 'cancelada' ? 'danger' : ''}`}>
        {role === 'gestor' ? `Mover para "${STATUS_LABEL[next]}"` : 'Cancelar ocorrência'}
      </button>
    </form>
  )
}

function Comments({ occ, onSubmit }: { occ: OccurrenceDetail; onSubmit: (body: string) => Promise<boolean> }) {
  const [body, setBody] = useState('')
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (await onSubmit(body)) setBody('')
  }
  return (
    <section className="card">
      <h3>Comentários ({occ.comments.length})</h3>
      {occ.comments.length === 0 && <p className="muted small">Nenhum comentário ainda.</p>}
      <ul className="comments">
        {occ.comments.map((c) => (
          <li key={c.id} className={c.author_role === 'gestor' ? 'from-gestor' : ''}>
            <div className="small">
              <strong>{c.author_name}</strong>{' '}
              <span className="muted">
                · {ROLE_LABEL[c.author_role]} · {formatDate(c.created_at)}
              </span>
            </div>
            <p className="prewrap">{c.body}</p>
          </li>
        ))}
      </ul>
      <form onSubmit={submit} className="comment-form">
        <textarea rows={2} placeholder="Escreva um comentário…" value={body} onChange={(e) => setBody(e.target.value)} required />
        <button className="btn" disabled={!body.trim()}>
          Comentar
        </button>
      </form>
    </section>
  )
}

function History({ occ }: { occ: OccurrenceDetail }) {
  return (
    <section className="card">
      <h3>Histórico</h3>
      <ol className="timeline">
        {[...occ.history].reverse().map((h) => (
          <li key={h.id}>
            <div className="small">
              {h.from_status ? (
                <>
                  <StatusBadge status={h.from_status} /> → <StatusBadge status={h.to_status} />
                </>
              ) : (
                <StatusBadge status={h.to_status} />
              )}
            </div>
            <div className="muted small">
              {formatDate(h.created_at)} · {h.changed_by_name}
            </div>
            {h.note && <p className="small prewrap">{h.note}</p>}
          </li>
        ))}
      </ol>
    </section>
  )
}

function RatingForm({ onSubmit }: { onSubmit: (rating: number, comment: string) => Promise<boolean> }) {
  const [rating, setRating] = useState(0)
  const [comment, setComment] = useState('')
  return (
    <section className="card">
      <h3>Avalie a resolução</h3>
      <div className="star-picker" role="radiogroup" aria-label="Nota">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={rating === n}
            aria-label={`${n} estrela(s)`}
            className={n <= rating ? 'on' : ''}
            onClick={() => setRating(n)}
          >
            ★
          </button>
        ))}
      </div>
      <textarea rows={2} placeholder="Comentário (opcional)" value={comment} onChange={(e) => setComment(e.target.value)} />
      <button className="btn" disabled={!rating} onClick={() => onSubmit(rating, comment)}>
        Enviar avaliação
      </button>
    </section>
  )
}

function ImageUpload({ onUpload }: { onUpload: (f: File) => Promise<boolean> }) {
  const [busy, setBusy] = useState(false)
  return (
    <label className="upload small">
      {busy ? 'Enviando…' : '+ Anexar imagem'}
      <input
        type="file"
        accept="image/jpeg,image/png,image/webp,image/gif"
        hidden
        disabled={busy}
        onChange={async (e) => {
          const f = e.target.files?.[0]
          if (!f) return
          setBusy(true)
          await onUpload(f)
          setBusy(false)
        }}
      />
    </label>
  )
}
