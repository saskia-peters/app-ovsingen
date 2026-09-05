import { useState, useEffect, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Header } from '../components/Header.tsx'
import styles from './LoginPage.module.css'

const TOKEN_STORAGE_KEY = 'gear.session_token'

interface LoginErrors {
  email?: string
  password?: string
  general?: string
}

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<LoginErrors>({})
  const [isSubmitting, setIsSubmitting] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    const prevTitle = document.title
    document.title = 'Anmeldung | G.E.A.R.'
    return () => {
      document.title = prevTitle
    }
  }, [])

  const validate = (): boolean => {
    const nextErrors: LoginErrors = {}
    let isValid = true

    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      nextErrors.email = 'Bitte gib deine E-Mail-Adresse ein.'
      isValid = false
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail)) {
      nextErrors.email = 'Bitte gib eine gültige E-Mail-Adresse ein.'
      isValid = false
    }

    if (!password) {
      nextErrors.password = 'Bitte gib dein Passwort ein.'
      isValid = false
    }

    setErrors(nextErrors)
    return isValid
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!validate()) {
      return
    }

    setIsSubmitting(true)
    setErrors({})

    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: email.trim(),
          password,
        }),
      })

      const data = await response.json().catch(() => null)

      if (response.ok && data?.token) {
        localStorage.setItem(TOKEN_STORAGE_KEY, data.token)
        navigate('/')
        return
      }

      if (response.status === 401) {
        // Anti-enumeration: identical microcopy for any invalid credentials
        // (UX-DR7) — the server never distinguishes wrong password, unknown
        // email or a non-active account.
        setErrors({
          general: 'E-Mail oder Passwort ist falsch.',
        })
        return
      }

      // 5xx and other unexpected failures are a server problem, not bad input.
      setErrors({
        general: 'Ein interner Fehler ist aufgetreten. Bitte versuche es erneut.',
      })
    } catch {
      setErrors({
        general: 'Verbindung zum Server fehlgeschlagen. Bitte prüfe deine Internetverbindung.',
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <div className={styles.card}>
          <h2 className={styles.title}>Anmeldung</h2>
          <p className={styles.subtitle}>
            Melde dich an, um auf die Geräteverwaltung zuzugreifen.
          </p>

          {errors.general && (
            <div className={styles.generalError} role="alert">
              {errors.general}
            </div>
          )}

          <form className={styles.form} onSubmit={handleSubmit} noValidate>
            <div className={styles.fieldGroup}>
              <label htmlFor="email" className={styles.label}>
                E-Mail-Adresse
              </label>
              <input
                id="email"
                type="email"
                className={`${styles.input} ${errors.email ? styles.inputError : ''}`}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                aria-invalid={!!errors.email}
                aria-describedby={errors.email ? 'email-error' : undefined}
                disabled={isSubmitting}
                autoComplete="email"
                required
              />
              {errors.email && (
                <p id="email-error" className={styles.errorText} role="alert">
                  {errors.email}
                </p>
              )}
            </div>

            <div className={styles.fieldGroup}>
              <label htmlFor="password" className={styles.label}>
                Passwort
              </label>
              <input
                id="password"
                type="password"
                className={`${styles.input} ${errors.password ? styles.inputError : ''}`}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                aria-invalid={!!errors.password}
                aria-describedby={errors.password ? 'password-error' : undefined}
                disabled={isSubmitting}
                autoComplete="current-password"
                required
              />
              {errors.password && (
                <p id="password-error" className={styles.errorText} role="alert">
                  {errors.password}
                </p>
              )}
            </div>

            <button type="submit" className={styles.submitButton} disabled={isSubmitting}>
              {isSubmitting ? 'Wird gesendet...' : 'Anmelden'}
            </button>
          </form>

          <div className={styles.links}>
            <Link to="/register" className={styles.link}>
              Noch kein Konto? Jetzt registrieren
            </Link>
            <Link to="/" className={styles.link}>
              Zurück zur Übersicht
            </Link>
          </div>
        </div>
      </main>
    </div>
  )
}
