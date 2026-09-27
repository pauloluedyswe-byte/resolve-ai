import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { ApiError, api, setUnauthorizedHandler, tokenStore } from '../lib/api'
import type { AuthResult, User } from '../lib/types'

interface AuthState {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(() => tokenStore.get() !== null)

  const logout = useCallback(() => {
    tokenStore.clear()
    setUser(null)
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(logout)
    if (!tokenStore.get()) return
    api
      .me()
      .then(setUser)
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) logout()
      })
      .finally(() => setLoading(false))
  }, [logout])

  const accept = (res: AuthResult) => {
    tokenStore.set(res.token)
    setUser(res.user)
  }

  const value: AuthState = {
    user,
    loading,
    logout,
    login: async (email, password) => accept(await api.login(email, password)),
    register: async (name, email, password) => accept(await api.register(name, email, password)),
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth deve ser usado dentro de AuthProvider')
  return ctx
}
