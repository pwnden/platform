export function takeSessionToken(): string | undefined {
  const token = location.hash.slice(1);
  history.replaceState(null, '', location.pathname);
  return /^[a-f0-9]{64}$/.test(token) ? token : undefined;
}
