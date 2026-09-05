import { Link } from 'react-router-dom'
import { Header } from '../components/Header.tsx'
import styles from './PlaceholderPage.module.css'

export function LoginPage() {
  return (
    <div className={styles.page}>
      <Header />
      <main className={styles.main}>
        <div className={styles.card}>
          <h2 className={styles.title}>Anmeldung</h2>
          <p className={styles.description}>
            Die Authentifizierung folgt in Story 1.4.
          </p>
          <Link to="/" className={styles.backLink}>
            Zurück zur Übersicht
          </Link>
        </div>
      </main>
    </div>
  )
}
