import type { UIWebNavigation } from './props';

export function browserState(data: unknown, src: string, target: string): UIWebNavigation | undefined {
  if (!data || typeof data !== 'object') return;
  const state = data as Record<string, unknown>;
  try {
    if (state.type !== 'pwnden.browser.state.v1' || state.channel !== new URL(src).pathname || typeof state.url !== 'string' ||
      typeof state.canBack !== 'boolean' || typeof state.canForward !== 'boolean' || typeof state.busy !== 'boolean' ||
      typeof state.error !== 'string' || state.error.length > 500) return;
    const url = new URL(state.url);
    if (url.origin !== target || url.username || url.password || url.protocol !== 'http:') return;
    return { url: url.href, canBack: state.canBack, canForward: state.canForward, busy: state.busy, error: state.error };
  } catch { return; }
}
