import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { LoginPage } from './LoginPage.tsx'
import { ThemeProvider } from '../context/ThemeContext.tsx'

const TOKEN_STORAGE_KEY = 'gear.session_token'

function renderLoginPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<div>Übersicht</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

function stubFetch(response: {
  ok: boolean
  status: number
  body: unknown
}) {
  const mock = vi.fn().mockResolvedValue({
    ok: response.ok,
    status: response.status,
    json: async () => response.body,
  })
  vi.stubGlobal('fetch', mock)
  return mock
}

async function submitLogin(email: string, password: string) {
  const user = userEvent.setup()
  renderLoginPage()
  await user.type(screen.getByLabelText('E-Mail-Adresse'), email)
  await user.type(screen.getByLabelText('Passwort'), password)
  await user.click(screen.getByRole('button', { name: 'Anmelden' }))
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('HAPPY_PATH: renders login form with email, password and navigation links', () => {
    renderLoginPage()

    expect(screen.getByRole('heading', { level: 2, name: 'Anmeldung' })).toBeInTheDocument()
    expect(screen.getByLabelText('E-Mail-Adresse')).toBeInTheDocument()
    expect(screen.getByLabelText('Passwort')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Anmelden' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Noch kein Konto? Jetzt registrieren' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Zurück zur Übersicht' })).toBeInTheDocument()
  })

  it('MISSING_FIELDS: shows validation errors when submitting with empty fields', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.click(screen.getByRole('button', { name: 'Anmelden' }))

    expect(screen.getByText('Bitte gib deine E-Mail-Adresse ein.')).toBeInTheDocument()
    expect(screen.getByText('Bitte gib dein Passwort ein.')).toBeInTheDocument()
  })

  it('INVALID_EMAIL: shows validation error on invalid email format', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByLabelText('E-Mail-Adresse'), 'invalid-email')
    await user.type(screen.getByLabelText('Passwort'), 'geheim123456')
    await user.click(screen.getByRole('button', { name: 'Anmelden' }))

    expect(screen.getByText('Bitte gib eine gültige E-Mail-Adresse ein.')).toBeInTheDocument()
  })

  it('HAPPY_PATH: successful login stores token and navigates to the dashboard', async () => {
    const fetchMock = stubFetch({
      ok: true,
      status: 200,
      body: {
        token: 'opaque-session-token',
        user: { id: 'u-1', email: 'erika@example.com', display_name: 'Erika Musterfrau' },
      },
    })

    await submitLogin('erika@example.com', 'geheim123456')

    await waitFor(() => {
      expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBe('opaque-session-token')
    })

    expect(screen.getByText('Übersicht')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'erika@example.com', password: 'geheim123456' }),
    })
  })

  it('INVALID_CREDENTIALS: shows anti-enumeration microcopy on 401', async () => {
    stubFetch({
      ok: false,
      status: 401,
      body: { error: { code: 'invalid_credentials', message: 'E-Mail oder Passwort ist falsch.' } },
    })

    await submitLogin('nobody@example.com', 'falsches-passwort')

    await waitFor(() => {
      expect(screen.getByText('E-Mail oder Passwort ist falsch.')).toBeInTheDocument()
    })

    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()
  })

  it('SERVER_ERROR: a 5xx failure shows the server-error message, not bad-credentials microcopy', async () => {
    stubFetch({
      ok: false,
      status: 500,
      body: { error: { code: 'internal_error', message: 'Ein interner Fehler ist aufgetreten.' } },
    })

    await submitLogin('erika@example.com', 'geheim123456')

    await waitFor(() => {
      expect(
        screen.getByText('Ein interner Fehler ist aufgetreten. Bitte versuche es erneut.'),
      ).toBeInTheDocument()
    })

    expect(screen.queryByText('E-Mail oder Passwort ist falsch.')).not.toBeInTheDocument()
  })

  it('NETWORK_ERROR: displays connection error when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('Network error')))

    await submitLogin('erika@example.com', 'geheim123456')

    await waitFor(() => {
      expect(
        screen.getByText('Verbindung zum Server fehlgeschlagen. Bitte prüfe deine Internetverbindung.'),
      ).toBeInTheDocument()
    })
  })
})
