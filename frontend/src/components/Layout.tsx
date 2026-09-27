import { NavLink, Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ROLE_LABEL } from '../lib/labels'

export function Layout() {
  const { user, loading, logout } = useAuth()

  if (loading) return <div className="center muted">Carregando…</div>
  if (!user) return <Navigate to="/login" replace />

  return (
    <>
      <header className="topbar">
        <div className="topbar-inner">
          <NavLink to="/" className="brand">
            Resolve <span>Aí</span>
          </NavLink>
          <nav>
            {user.role === 'gestor' && <NavLink to="/dashboard">Dashboard</NavLink>}
            <NavLink to="/" end>
              Ocorrências
            </NavLink>
            <NavLink to="/nova">Nova ocorrência</NavLink>
          </nav>
          <div className="user">
            <span>
              {user.name} <small className="muted">· {ROLE_LABEL[user.role]}</small>
            </span>
            <button className="btn-link" onClick={logout}>
              Sair
            </button>
          </div>
        </div>
      </header>
      <main className="container">
        <Outlet />
      </main>
    </>
  )
}

export function GestorOnly({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  return user?.role === 'gestor' ? children : <Navigate to="/" replace />
}
