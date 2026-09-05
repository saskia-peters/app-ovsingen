// Auth state helpers (review finding 1.6-12): the login response carries the
// user's is_mfa_enabled flag (wired from LoginUser.IsMFAEnabled) and it is
// persisted so the SPA can show the "MFA aktiv" indicator. The server remains
// the source of truth; this is only a client cache.
const MFA_ENABLED_KEY = 'gear.is_mfa_enabled'

export function saveAuthState(token: string, isMfaEnabled: boolean): void {
  localStorage.setItem('gear.session_token', token)
  if (isMfaEnabled) {
    localStorage.setItem(MFA_ENABLED_KEY, 'true')
  } else {
    localStorage.removeItem(MFA_ENABLED_KEY)
  }
}

export function isMfaEnabled(): boolean {
  return localStorage.getItem(MFA_ENABLED_KEY) === 'true'
}

export function clearAuthState(): void {
  localStorage.removeItem('gear.session_token')
  localStorage.removeItem(MFA_ENABLED_KEY)
}
