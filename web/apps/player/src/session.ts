const sessionKey = 'pwnden.session';
const validToken = (value: string | null): value is string => value !== null && /^[a-f0-9]{64}$/.test(value);

export function takeSessionToken(): string | undefined {
  const token = location.hash.slice(1);
  history.replaceState(null, '', location.pathname);
  if (token) {
    try {
      if (validToken(token)) sessionStorage.setItem(sessionKey, token);
      else sessionStorage.removeItem(sessionKey);
    } catch { /* The full session URL still works when tab storage is disabled. */ }
    return validToken(token) ? token : undefined;
  }
  try {
    const stored = sessionStorage.getItem(sessionKey);
    if (validToken(stored)) return stored;
    sessionStorage.removeItem(sessionKey);
  } catch { /* A page without retained credentials displays connection recovery. */ }
  return undefined;
}

export function forgetSessionToken(token: string): void {
  try {
    if (sessionStorage.getItem(sessionKey) === token) sessionStorage.removeItem(sessionKey);
  } catch { /* Memory credentials are invalidated by the caller regardless. */ }
}
