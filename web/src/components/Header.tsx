import { useTheme } from '../context/useTheme.ts'
import { isMfaEnabled } from '../auth/authState.ts'
import styles from './Header.module.css'

export function Header() {
  const { resolvedTheme, toggleTheme } = useTheme()

  const isDark = resolvedTheme === 'dark'
  const toggleLabel = isDark ? 'Hellmodus aktivieren' : 'Dunkelmodus aktivieren'
  const mfaActive = isMfaEnabled()

  return (
    <header className={styles.header}>
      <div className={styles.headerInner}>
        <div className={styles.brandGroup}>
          <h1 className={styles.title}>G.E.A.R.</h1>
          {mfaActive && (
            <span className={styles.mfaBadge} role="status" aria-live="polite">
              <span className={styles.mfaBadgeDot} aria-hidden="true" />
              MFA aktiv
            </span>
          )}
        </div>
        <button
          type="button"
          className={styles.themeToggle}
          onClick={toggleTheme}
          aria-label={toggleLabel}
          title={toggleLabel}
        >
          <span aria-hidden="true">{isDark ? '☀️' : '🌙'}</span>
          <span>{isDark ? 'Hell' : 'Dunkel'}</span>
        </button>
      </div>
    </header>
  )
}