// Minimal application shell (Story 1.1). The G.E.A.R. UI has no business
// logic; server-side authorization is the only source of truth (AD-6) and
// grows in with the dashboard story (1.2).
function App() {
  return (
    <main className="shell">
      <h1>G.E.A.R.</h1>
      <p className="tagline">Geräteverwaltung &amp; Einsatzbereitschaft</p>
      <p className="muted">Das Dashboard folgt in Story 1.2.</p>
    </main>
  )
}

export default App