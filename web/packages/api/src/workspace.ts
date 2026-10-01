import type { RunStatus, WorkspaceConnection } from '@pwnden/play';

export function connectWorkspace(path: string, token: string, request: typeof fetch, failed: (code: string) => void): WorkspaceConnection {
  const abort = new AbortController();
  let active = true;
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
  let settled = false;
  let resolve: (status: RunStatus) => void = () => {};
  let reject: (error: Error) => void = () => {};
  const ready = new Promise<RunStatus>((ok, fail) => { resolve = ok; reject = fail; });
  const timer = setTimeout(() => stop('deadline_exceeded'), 15 * 60 * 1000);
  function stop(code?: string) {
    if (!active) return;
    active = false; clearTimeout(timer); abort.abort();
    void reader?.cancel().catch(() => {});
    if (!settled) { settled = true; reject(new Error(code ?? 'canceled')); }
    if (code) failed(code);
  }
  async function receive() {
    try {
      const response = await request(path, { credentials: 'omit', cache: 'no-store', redirect: 'error', signal: abort.signal, headers: { Authorization: `Bearer ${token}` } });
      if (!active) { await response.body?.cancel(); return; }
      if (!response.ok) {
        let code = response.status === 401 ? 'unauthorized' : 'execution_failed';
        try { const payload = await response.json(); if (response.status !== 401 && typeof payload?.error?.code === 'string') code = payload.error.code; } catch {}
        stop(code); return;
      }
      if (response.headers.get('Content-Type') !== 'application/x-ndjson' || !response.body) { stop('invalid_response'); return; }
      reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8', { fatal: true });
      let buffer = '';
      while (active) {
        const chunk = await reader.read();
        if (!active) return;
        if (chunk.done) { stop('network_error'); return; }
        buffer += decoder.decode(chunk.value, { stream: true });
        if (buffer.length > 65536) { stop('invalid_response'); return; }
        let newline: number;
        while ((newline = buffer.indexOf('\n')) !== -1 && active) {
          const message = JSON.parse(buffer.slice(0, newline));
          buffer = buffer.slice(newline + 1);
          if (message?.type === 'error' && typeof message.code === 'string') { stop(message.code); return; }
          if (message?.type === 'heartbeat' && settled) continue;
          const status = message?.status;
          const slug = path.split('/').at(-2);
          if (message?.type !== 'ready' || settled || status?.slug !== slug || !((status.kind === 'file' && status.state === 'ready') || (status.kind === 'service' && status.state === 'running')) || !Array.isArray(status.endpoints)) { stop('invalid_response'); return; }
          if (status.endpoints.some((item: { name?: unknown; url?: unknown }) => !item || typeof item.name !== 'string' || typeof item.url !== 'string' || !/^(http|tcp):\/\//.test(item.url))) { stop('invalid_response'); return; }
          settled = true; clearTimeout(timer); resolve(status as RunStatus);
        }
      }
    } catch { if (active) stop('network_error'); }
  }
  void receive();
  return { ready, close: () => stop() };
}
