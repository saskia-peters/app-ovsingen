import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach } from 'vitest'
import { Header } from './Header.tsx'
import { ThemeProvider } from '../context/ThemeContext.tsx'

describe('Header', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('renders G.E.A.R. branding', () => {
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )

    expect(screen.getByRole('heading', { level: 1, name: 'G.E.A.R.' })).toBeInTheDocument()
  })

  it('toggles light/dark theme when theme button is clicked', async () => {
    const user = userEvent.setup()

    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )

    const toggleButton = screen.getByRole('button', { name: /dunkelmodus/i })
    expect(toggleButton).toBeInTheDocument()

    // Click to toggle to dark mode
    await user.click(toggleButton)
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(screen.getByRole('button', { name: /hellmodus/i })).toBeInTheDocument()

    // Click again to toggle back to light mode
    await user.click(screen.getByRole('button', { name: /hellmodus/i }))
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('MFA_ACTIVE: shows the "MFA aktiv" badge when the auth state says MFA is enabled', () => {
    localStorage.setItem('gear.is_mfa_enabled', 'true')
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )
    expect(screen.getByText('MFA aktiv')).toBeInTheDocument()
  })

  it('MFA_INACTIVE: hides the badge when MFA is not enabled', () => {
    localStorage.clear()
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )
    expect(screen.queryByText('MFA aktiv')).not.toBeInTheDocument()
  })

  it('USER_LOGGED_IN: shows the logged-in user name in the header', () => {
    localStorage.setItem('gear.display_name', 'Erika Musterfrau')
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )
    expect(screen.getByText('Erika Musterfrau')).toBeInTheDocument()
  })

  it('USER_LOGGED_OUT: hides the user name when not authenticated', () => {
    localStorage.clear()
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )
    expect(screen.queryByText('Erika Musterfrau')).not.toBeInTheDocument()
  })
})
