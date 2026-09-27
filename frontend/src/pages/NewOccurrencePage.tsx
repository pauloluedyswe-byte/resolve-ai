import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import type { Category } from '../lib/types'

const MAX_IMAGE = 5 * 1024 * 1024

export function NewOccurrencePage() {
  const navigate = useNavigate()
  const [categories, setCategories] = useState<Category[]>([])
  const [form, setForm] = useState({ title: '', description: '', category_id: '', location: '' })
  const [image, setImage] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api.categories().then(setCategories).catch((e) => setError(e.message))
  }, [])

  const preview = useMemo(() => (image ? URL.createObjectURL(image) : null), [image])
  useEffect(() => () => void (preview && URL.revokeObjectURL(preview)), [preview])

  const set =
    (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) =>
      setForm({ ...form, [k]: e.target.value })

  const pickImage = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null
    if (f && f.size > MAX_IMAGE) {
      setError('A imagem deve ter no máximo 5 MB.')
      e.target.value = ''
      return
    }
    setError('')
    setImage(f)
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const o = await api.createOccurrence({ ...form, category_id: Number(form.category_id) })
      if (image) {
        try {
          await api.uploadImage(o.id, image)
        } catch (err) {
          // A ocorrência já existe; segue para o detalhe e avisa sobre a imagem.
          alertLater(`Ocorrência criada, mas a imagem não foi enviada: ${(err as Error).message}`)
        }
      }
      navigate(`/ocorrencias/${o.id}`)
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <div className="narrow">
      <h1>Nova ocorrência</h1>
      <p className="muted">Descreva o problema com o máximo de detalhes possível.</p>
      <form className="card form" onSubmit={submit}>
        <label>
          Título
          <input value={form.title} onChange={set('title')} maxLength={150} required placeholder="Ex.: Lâmpada queimada no corredor" />
        </label>
        <label>
          Categoria
          <select value={form.category_id} onChange={set('category_id')} required>
            <option value="" disabled>
              Selecione…
            </option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Localização
          <input value={form.location} onChange={set('location')} required placeholder="Ex.: Bloco B, 3º andar" />
        </label>
        <label>
          Descrição
          <textarea value={form.description} onChange={set('description')} rows={5} required />
        </label>
        <label>
          Imagem <small className="muted">(opcional · JPG, PNG, WEBP ou GIF · até 5 MB)</small>
          <input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={pickImage} />
        </label>
        {preview && <img src={preview} alt="Pré-visualização" className="preview" />}
        {error && <p className="error">{error}</p>}
        <div className="row-end">
          <button type="button" className="btn secondary" onClick={() => navigate(-1)}>
            Cancelar
          </button>
          <button className="btn" disabled={busy}>
            {busy ? 'Registrando…' : 'Registrar ocorrência'}
          </button>
        </div>
      </form>
    </div>
  )
}

// Guarda um aviso para ser exibido na próxima página (sem bloquear a navegação).
function alertLater(msg: string) {
  sessionStorage.setItem('resolveai.flash', msg)
}
