import { expect, it, vi } from 'vitest';
import { connectWorkspace } from '../packages/api/src/workspace';

const path = '/api/v1/problems/test/workspace';
const token = 'a'.repeat(64);
const status = { slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:43123' }] };
const encode = (value: unknown) => new TextEncoder().encode(JSON.stringify(value) + '\n');

it('keeps an authenticated view stream until close and handles split messages and heartbeat', async () => {
  const failed = vi.fn();
  const cancel = vi.fn();
  const stream = new ReadableStream<Uint8Array>({ start(controller) {
    const bytes = encode({ type: 'ready', status });
    controller.enqueue(bytes.slice(0, 10)); controller.enqueue(bytes.slice(10)); controller.enqueue(encode({ type: 'heartbeat' }));
  }, cancel });
  const request = vi.fn(async () => new Response(stream, { headers: { 'Content-Type': 'application/x-ndjson' } }));
  const connection = connectWorkspace(path, token, request, failed);
  expect(await connection.ready).toEqual(status);
  expect(request.mock.calls[0]?.[0]).toBe(path);
  expect(request.mock.calls[0]?.[1]).toMatchObject({ redirect: 'error', credentials: 'omit', headers: { Authorization: 'Bearer ' + token } });
  connection.close(); connection.close();
  expect(cancel).toHaveBeenCalledOnce();
  expect(failed).not.toHaveBeenCalled();
});

it('rejects invalid identities and unavailable preparation without admitting an environment', async () => {
  for (const payload of [{ type: 'ready', status: { ...status, slug: 'other' } }, { type: 'ready', status: { ...status, kind: 'file' } }, { type: 'ready', status: { ...status, endpoints: [{ name: 'web', url: 'javascript:alert(1)' }] } }]) {
    const failed = vi.fn();
    const connection = connectWorkspace(path, token, async () => new Response(encode(payload), { headers: { 'Content-Type': 'application/x-ndjson' } }), failed);
    await expect(connection.ready).rejects.toThrow('invalid_response');
    expect(failed).toHaveBeenCalledWith('invalid_response');
  }
  for (const code of ['workspace_full', 'unauthorized']) {
    const failed = vi.fn();
    const connection = connectWorkspace(path, token, async () => new Response(JSON.stringify({ error: { code } }), { status: code === 'unauthorized' ? 401 : 409 }), failed);
    await expect(connection.ready).rejects.toThrow(code);
    expect(failed).toHaveBeenCalledOnce();
  }
});

it('ignores a late response after the player leaves while preparation is pending', async () => {
  let resolve: (response: Response) => void = () => {};
  const failed = vi.fn();
  const connection = connectWorkspace(path, token, () => new Promise(done => { resolve = done; }), failed);
  const result = expect(connection.ready).rejects.toThrow('canceled');
  connection.close();
  resolve(new Response(encode({ type: 'ready', status }), { headers: { 'Content-Type': 'application/x-ndjson' } }));
  await result;
  expect(failed).not.toHaveBeenCalled();
});
