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

  it('renders G.E.A.R. branding and Ortsverband Singen identifier', () => {
    render(
      <ThemeProvider>
        <Header />
      </ThemeProvider>,
    )

    expect(screen.getByRole('heading', { level: 1, name: 'G.E.A.R.' })).toBeInTheDocument()
    expect(screen.getByText('Ortsverband Singen')).toBeInTheDocument()
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
})
