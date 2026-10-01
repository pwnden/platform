import { afterEach, expect, it, vi } from 'vitest';
import { createAPI } from '../packages/api/src/index';
import type { TerminalTransport } from '../packages/api/src/terminal';
import type { TerminalEvent } from '../domains/terminal/src/index';

class Socket implements TerminalTransport {
  bufferedAmount = 0;
  binaryType: BinaryType = 'blob';
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  send = vi.fn();
  close = vi.fn();
  constructor(readonly url: string) {}
  message(data: unknown) { this.onmessage?.({ data } as MessageEvent); }
  open() { this.onopen?.({} as Event); }
}
function fixture(size = { cols: 80, rows: 24 }, onUnauthorized?: () => void) {
  vi.stubGlobal('location', { origin: 'http://127.0.0.1:12345' });
  let socket: Socket;
  const create = vi.fn((url: string) => { socket = new Socket(url); return socket; });
  const client = createAPI({ token: 'a'.repeat(64), socket: create, ...(onUnauthorized ? { onUnauthorized } : {}) });
  const events: TerminalEvent[] = [];
  const connection = client.terminals.connect('test', size, event => events.push(event));
  return { socket: socket!, create, connection, events };
}
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });

it('sends only protocol dimensions even when an xterm 6 resize event has extra fields', async () => {
  const dimensions = { cols: 100, rows: 30, colsChanged: true, rowsChanged: false };
  const { socket, connection } = fixture(dimensions);
  socket.open();
  expect(JSON.parse(socket.send.mock.calls[0]![0])).toEqual({ type: 'authenticate', token: 'a'.repeat(64), cols: 100, rows: 30 });
  socket.message('{"type":"ready"}');
  const session = await connection.ready;
  session.resize(dimensions);
  expect(JSON.parse(socket.send.mock.calls.at(-1)![0])).toEqual({ type: 'resize', cols: 100, rows: 30 });
  session.close();
});

it.each(['unauthorized', 'invalid_request', 'network_error'])('invalidates credentials only for terminal authentication rejection: %s', async code => {
  const unauthorized = vi.fn();
  const { socket, connection } = fixture(undefined, unauthorized);
  const failed = expect(connection.ready).rejects.toThrow(code);
  socket.message(JSON.stringify({ type: 'error', code }));
  await failed;
  expect(unauthorized).toHaveBeenCalledTimes(code === 'unauthorized' ? 1 : 0);
});

it('authenticates inside the fixed-origin protocol and preserves terminal bytes with render acknowledgement', async () => {
  const { socket, create, connection, events } = fixture();
  expect(create).toHaveBeenCalledWith('ws://127.0.0.1:12345/api/v1/problems/test/terminal', 'pwnden.terminal.v1');
  expect(socket.url).not.toContain('a'.repeat(64));
  expect(socket.binaryType).toBe('arraybuffer');
  socket.open();
  expect(JSON.parse(socket.send.mock.calls[0]![0])).toEqual({ type: 'authenticate', token: 'a'.repeat(64), cols: 80, rows: 24 });
  socket.message('{"type":"ready"}');
  const session = await connection.ready;
  socket.message(new Uint8Array([0, 255, 3]).buffer);
  const event = events[0]!;
  expect(event.type).toBe('output');
  if (event.type === 'output') {
    expect(event.data).toEqual(new Uint8Array([0, 255, 3]));
    expect(socket.send).toHaveBeenCalledTimes(1);
    event.acknowledge(); event.acknowledge();
    expect(socket.send).toHaveBeenLastCalledWith('{"type":"ack"}');
    expect(socket.send).toHaveBeenCalledTimes(2);
  }
  session.input(new Uint8Array([3]));
  expect(socket.send).toHaveBeenLastCalledWith(new Uint8Array([3]));
  session.resize({ cols: 100, rows: 40 });
  expect(socket.send).toHaveBeenLastCalledWith('{"type":"resize","cols":100,"rows":40}');
  socket.message('{"type":"exit","code":7}');
  expect(events.slice(-2)).toEqual([{ type: 'exit', code: 7 }, { type: 'closed' }]);
  expect(socket.close).toHaveBeenCalledOnce();
});

it('cancels a connecting socket and rejects unexpected output before authentication', async () => {
  let current = fixture();
  const rejected = expect(current.connection.ready).rejects.toThrow('canceled');
  current.connection.close(); current.socket.open();
  await rejected;
  expect(current.socket.send).not.toHaveBeenCalled();
  current = fixture();
  const invalid = expect(current.connection.ready).rejects.toThrow('invalid_response');
  current.socket.message(new Uint8Array([1]).buffer);
  await invalid;
});

it('limits buffered input and handles setup timeout without retaining a socket', async () => {
  vi.useFakeTimers();
  let current = fixture();
  const timeout = expect(current.connection.ready).rejects.toThrow('deadline_exceeded');
  await vi.advanceTimersByTimeAsync(15 * 60 * 1000); await timeout;
  expect(current.socket.close).toHaveBeenCalledOnce();
  current = fixture();
  current.socket.message('{"type":"ready"}');
  const session = await current.connection.ready;
  current.socket.bufferedAmount = 65537;
  session.input(new Uint8Array([1]));
  expect(current.events).toEqual([{ type: 'error', code: 'input_backpressure' }]);
  expect(current.socket.close).toHaveBeenCalledOnce();
});
