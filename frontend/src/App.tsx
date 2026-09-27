import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth/AuthContext'
import { GestorOnly, Layout } from './components/Layout'
import { LoginPage, RegisterPage } from './pages/AuthPages'
import { DashboardPage } from './pages/DashboardPage'
import { NewOccurrencePage } from './pages/NewOccurrencePage'
import { OccurrenceDetailPage } from './pages/OccurrenceDetailPage'
import { OccurrencesPage } from './pages/OccurrencesPage'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/cadastro" element={<RegisterPage />} />
          <Route element={<Layout />}>
            <Route index element={<OccurrencesPage />} />
            <Route path="/nova" element={<NewOccurrencePage />} />
            <Route path="/ocorrencias/:id" element={<OccurrenceDetailPage />} />
            <Route
              path="/dashboard"
              element={
                <GestorOnly>
                  <DashboardPage />
                </GestorOnly>
              }
            />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
