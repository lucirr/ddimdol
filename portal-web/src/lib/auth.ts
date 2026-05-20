const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const ID_TOKEN_KEY = 'id_token'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'
const OIDC_STATE_KEY = 'oidc_state'
const PKCE_VERIFIER_KEY = 'pkce_code_verifier'

const keycloakUrl = import.meta.env.VITE_KEYCLOAK_URL || 'http://localhost:8180'
const keycloakRealm = import.meta.env.VITE_KEYCLOAK_REALM || 'edgedip'
const keycloakClientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID || 'portal-web'

export const authConfig = {
  keycloakUrl,
  realm: keycloakRealm,
  clientId: keycloakClientId,
  redirectUri: `${window.location.origin}/auth/callback`,
}

export interface TokenSet {
  access_token: string
  refresh_token?: string
  id_token?: string
  expires_in?: number
}

export interface TokenClaims {
  sub?: string
  preferred_username?: string
  name?: string
  email?: string
  exp?: number
}

export class AuthError extends Error {
  readonly code = 'AUTH_ERROR' as const
  readonly name = 'AuthError'
  constructor(public readonly status: number, public readonly detail: string) {
    super(`AuthError(${status}): ${detail}`)
  }
}

function base64UrlEncodeBytes(bytes: Uint8Array) {
  let value = ''
  bytes.forEach((byte) => {
    value += String.fromCharCode(byte)
  })

  return btoa(value)
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/g, '')
}

function base64UrlDecode(value: string) {
  const padded = value.replace(/-/g, '+').replace(/_/g, '/').padEnd(
    Math.ceil(value.length / 4) * 4,
    '=',
  )

  return atob(padded)
}

function randomBase64Url(byteLength = 32) {
  const bytes = new Uint8Array(byteLength)
  crypto.getRandomValues(bytes)
  return base64UrlEncodeBytes(bytes)
}

async function sha256Base64Url(value: string) {
  const encoded = new TextEncoder().encode(value)
  const digest = await crypto.subtle.digest('SHA-256', encoded)
  return base64UrlEncodeBytes(new Uint8Array(digest))
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function getIdToken() {
  return localStorage.getItem(ID_TOKEN_KEY)
}

export function isTokenValid() {
  if (!getAccessToken()) return false
  const expiresAt = localStorage.getItem(TOKEN_EXPIRES_AT_KEY)
  if (!expiresAt) return true
  return Date.now() < Number(expiresAt) - 30_000
}

export function hasAccessToken() {
  return isTokenValid()
}

export function clearAuth() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(ID_TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)
  sessionStorage.removeItem(OIDC_STATE_KEY)
  sessionStorage.removeItem(PKCE_VERIFIER_KEY)
}

export function storeTokens(tokens: TokenSet) {
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token)
  if (tokens.refresh_token) localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token)
  if (tokens.id_token) localStorage.setItem(ID_TOKEN_KEY, tokens.id_token)
  if (tokens.expires_in) {
    localStorage.setItem(
      TOKEN_EXPIRES_AT_KEY,
      String(Date.now() + tokens.expires_in * 1000),
    )
  }
}

export function parseTokenClaims(token = getAccessToken()): TokenClaims | null {
  if (!token) return null
  const [, payload] = token.split('.')
  if (!payload) return null

  try {
    return JSON.parse(base64UrlDecode(payload)) as TokenClaims
  } catch {
    return null
  }
}

export async function startLogin() {
  const state = randomBase64Url(16)
  const codeVerifier = randomBase64Url(64)
  const codeChallenge = await sha256Base64Url(codeVerifier)

  sessionStorage.setItem(OIDC_STATE_KEY, state)
  sessionStorage.setItem(PKCE_VERIFIER_KEY, codeVerifier)

  const params = new URLSearchParams({
    client_id: authConfig.clientId,
    redirect_uri: authConfig.redirectUri,
    response_type: 'code',
    scope: 'openid profile email',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
  })

  window.location.href = `${authConfig.keycloakUrl}/realms/${authConfig.realm}/protocol/openid-connect/auth?${params}`
}

export async function completeLogin(code: string, state: string) {
  const expectedState = sessionStorage.getItem(OIDC_STATE_KEY)
  const codeVerifier = sessionStorage.getItem(PKCE_VERIFIER_KEY)

  if (!expectedState || state !== expectedState) {
    throw new Error('로그인 상태 값이 일치하지 않습니다.')
  }
  if (!codeVerifier) {
    throw new Error('PKCE verifier를 찾을 수 없습니다.')
  }

  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: authConfig.clientId,
    redirect_uri: authConfig.redirectUri,
    code,
    code_verifier: codeVerifier,
  })

  const response = await fetch(
    `${authConfig.keycloakUrl}/realms/${authConfig.realm}/protocol/openid-connect/token`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body,
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => 'token_exchange_failed')
    throw new AuthError(response.status, detail)
  }

  const tokens = (await response.json()) as TokenSet
  storeTokens(tokens)
  sessionStorage.removeItem(OIDC_STATE_KEY)
  sessionStorage.removeItem(PKCE_VERIFIER_KEY)
}

export function logout() {
  const idToken = getIdToken()
  clearAuth()

  const params = new URLSearchParams({
    client_id: authConfig.clientId,
    post_logout_redirect_uri: `${window.location.origin}/login`,
  })
  if (idToken) params.set('id_token_hint', idToken)

  window.location.href = `${authConfig.keycloakUrl}/realms/${authConfig.realm}/protocol/openid-connect/logout?${params}`
}
