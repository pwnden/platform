import type { TerminalConnection, TerminalEvent, TerminalSession, TerminalSize } from '@pwnden/terminal';

export interface TerminalTransport {
  readonly url: string;
  readonly bufferedAmount: number;
  binaryType: BinaryType;
  onopen: ((event: Event) => void) | null;
  onmessage: ((event: MessageEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  onclose: ((event: CloseEvent) => void) | null;
  send(data: string | Uint8Array<ArrayBuffer>): void;
  close(): void;
}

export function validSize(size: TerminalSize): boolean {
  return Number.isInteger(size.cols) && size.cols >= 2 && size.cols <= 500 &&
    Number.isInteger(size.rows) && size.rows >= 1 && size.rows <= 200;
}

export function connectTerminal(
  path: string, token: string, size: TerminalSize, receive: (event: TerminalEvent) => void,
  create: (url: string, protocol: string) => TerminalTransport,
  origin: string,
): TerminalConnection {
  if (!validSize(size)) throw new Error('invalid_argument');
  const url = new URL(path, origin);
  if (url.origin !== origin || !['http:', 'https:'].includes(url.protocol)) throw new Error('invalid_origin');
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  const socket = create(url.href, 'pwnden.terminal.v2');
  socket.binaryType = 'arraybuffer';
  let opened = false;
  let active = true;
  let settled = false;
  let exited = false;
  let finish: (session: TerminalSession) => void = () => {};
  let reject: (error: Error) => void = () => {};
  const ready = new Promise<TerminalSession>((resolve, failure) => { finish = resolve; reject = failure; });
  const timer = setTimeout(() => terminate('deadline_exceeded'), 15 * 60 * 1000);

  function terminate(code?: string) {
    if (!active) return;
    active = false;
    clearTimeout(timer);
    if (!settled) { settled = true; reject(new Error(code ?? 'canceled')); }
    socket.close();
    receive(code ? { type: 'error', code } : { type: 'closed' });
  }
  function send(data: string | Uint8Array<ArrayBuffer>) {
    if (!active || !opened) return;
    if (socket.bufferedAmount > 64 * 1024) { terminate('input_backpressure'); return; }
    try { socket.send(data); } catch { terminate('network_error'); }
  }
  socket.onopen = () => {
    if (!active) { socket.close(); return; }
    if (socket.url !== url.href) { terminate('invalid_origin'); return; }
    try { socket.send(JSON.stringify({ type: 'authenticate', token, cols: size.cols, rows: size.rows })); }
    catch { terminate('network_error'); }
  };
  socket.onerror = () => terminate('network_error');
  socket.onclose = () => terminate();
  socket.onmessage = event => {
    if (!active) return;
    if (event.data instanceof ArrayBuffer) {
      if (!opened || event.data.byteLength > 16 * 1024) { terminate('invalid_response'); return; }
      let acknowledged = false;
      receive({ type: 'output', data: new Uint8Array(event.data), acknowledge() {
        if (!acknowledged) { acknowledged = true; send(JSON.stringify({ type: 'ack' })); }
      } });
      return;
    }
    try {
      if (typeof event.data !== 'string') throw new Error();
      const message = JSON.parse(event.data) as Record<string, unknown>;
      if (message?.type === 'ready' && !opened && typeof message.reused === 'boolean') {
        opened = true; settled = true; clearTimeout(timer);
        finish({
          reused: message.reused,
          input(data) {
            if (exited) return;
            if (data.byteLength > 16 * 1024) { terminate('invalid_argument'); return; }
            send(Uint8Array.from(data));
          },
          resize(next) {
            if (!validSize(next)) { terminate('invalid_argument'); return; }
            send(JSON.stringify({ type: 'resize', cols: next.cols, rows: next.rows }));
          },
          close: () => terminate(),
        });
      } else if (message?.type === 'exit' && opened && Number.isInteger(message.code)) {
        exited = true;
        receive({ type: 'exit', code: message.code as number });
      } else if (message?.type === 'error' && typeof message.code === 'string') {
        terminate(message.code);
      } else terminate('invalid_response');
    } catch { terminate('invalid_response'); }
  };
  return { ready, close: () => terminate() };
}
