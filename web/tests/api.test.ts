import { describe, expect, it, vi } from 'vitest';
import { APIError, createAPI } from '../packages/api/src/index';

const token = 'a'.repeat(64);
const response = (payload: unknown, status = 200) => new Response(JSON.stringify(payload), { status, headers: { 'Content-Type': 'application/json' } });

it.each([1, 2, 3, 4, 5] as const)('maps difficulty %i for list and detail', async difficulty => {
  const problem = { slug: 'test', title: 'Test', category: 'web', kind: 'service', difficulty };
  const replies = [response({ problems: [problem] }), response({ ...problem, description: '', files: [], tools: ['web'], hint_count: 0, walkthrough: false })];
  const client = createAPI({ token, fetch: async () => replies.shift()! });
  expect(await client.catalog.list()).toEqual([problem]);
  expect((await client.catalog.detail('test')).difficulty).toBe(difficulty);
});

it.each([0, 6, -1, 2.5, '2', true, null])('rejects invalid difficulty %j in list and detail', async difficulty => {
  const problem = { slug: 'test', title: 'Test', category: 'web', kind: 'service', difficulty };
  const replies = [response({ problems: [problem] }), response({ ...problem, description: '', files: [], tools: ['web'], hint_count: 0, walkthrough: false })];
  const client = createAPI({ token, fetch: async () => replies.shift()! });
  await expect(client.catalog.list()).rejects.toMatchObject({ code: 'invalid_response' });
  await expect(client.catalog.detail('test')).rejects.toMatchObject({ code: 'invalid_response' });
});

it('invalidates a rejected credential even for malformed 401 bodies and retains it on network failures', async () => {
  const onUnauthorized = vi.fn();
  const client = createAPI({ token, onUnauthorized, fetch: vi.fn().mockResolvedValueOnce(new Response('unauthorized', { status: 401 }))
    .mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(response({ error: { code: 'not_found' } }, 404)) });
  await expect(client.catalog.list()).rejects.toMatchObject({ status: 401 });
  expect(onUnauthorized).toHaveBeenCalledOnce();
  await expect(client.catalog.list()).rejects.toMatchObject({ code: 'network_error' });
  await expect(client.catalog.list()).rejects.toMatchObject({ code: 'not_found' });
  expect(onUnauthorized).toHaveBeenCalledOnce();
});

it('requests only explicit guidance IDs and cancels oversized preview streams', async () => {
  const request = vi.fn(async (_input: RequestInfo | URL, _options?: RequestInit) => response({ id: 'hint-1', content: 'One clue', path: 'private' }));
  const client = createAPI({ token, fetch: request });
  expect(await client.catalog.guidance('test', 'hint-1')).toBe('One clue');
  expect(request.mock.calls[0]?.[0]).toBe('/api/v1/problems/test/guidance/hint-1');
  await expect(client.catalog.guidance('test', '../solve.py')).rejects.toMatchObject({ code: 'invalid_argument' });
  expect(request).toHaveBeenCalledOnce();
  const cancel = vi.fn();
  const stream = new ReadableStream<Uint8Array>({ start(controller) { controller.enqueue(new Uint8Array(5)); }, cancel });
  const download = createAPI({ token, fetch: async () => new Response(stream, { headers: { 'Content-Type': 'application/octet-stream' } }) });
  await expect(download.catalog.download('test', 'a'.repeat(64), 4)).rejects.toMatchObject({ code: 'preview_too_large' });
  expect(cancel).toHaveBeenCalledOnce();
});

it('maps details, binary downloads and restored run status without exposing internal fields', async () => {
  const id = 'b'.repeat(64);
  const calls: string[] = [];
  const replies = [
    response({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', tools: ['files', 'terminal'], description: '<script>text</script>', files: [{ id, name: 'files/data.bin', size: 3 }], hint_count: 3, walkthrough: true, private: 'hidden' }),
    new Response(new Uint8Array([0, 1, 255]), { headers: { 'Content-Type': 'application/octet-stream' } }),
    response({ slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }], flag: 'hidden' }),
  ];
  const client = createAPI({ token, fetch: async (input, options) => {
    calls.push(String(input));
    expect(options?.headers).toHaveProperty('Authorization', `Bearer ${token}`);
    expect(options?.redirect).toBe('error');
    return replies.shift()!;
  } });
  expect(await client.catalog.detail('test')).toEqual({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', tools: ['files', 'terminal'], description: '<script>text</script>', files: [{ id, name: 'files/data.bin', size: 3 }], hintCount: 3, walkthrough: true });
  expect(await client.catalog.download('test', id)).toEqual(new Uint8Array([0, 1, 255]));
  expect(await client.player.status('test')).toEqual({ slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] });
  expect(calls).toEqual(['/api/v1/problems/test', `/api/v1/problems/test/files/${id}`, '/api/v1/problems/test/status']);
});

it('rejects undeclared path inputs and malformed detail/status/download responses', async () => {
  let calls = 0;
  const invalid = createAPI({ token, fetch: async () => { calls++; return response({}); } });
  await expect(invalid.catalog.download('test', '../../solve.py')).rejects.toMatchObject({ code: 'invalid_argument' });
  await expect(invalid.catalog.detail('../test')).rejects.toMatchObject({ code: 'invalid_argument' });
  expect(calls).toBe(0);
  for (const item of [
    { slug: 'other', kind: 'service', state: 'running', endpoints: [] },
    { slug: 'test', kind: 'file', state: 'running', endpoints: [] },
    { slug: 'test', kind: 'service', state: 'ready', endpoints: [] },
    { slug: 'test', kind: 'service', state: 'unknown', endpoints: [] },
    { slug: 'test', kind: 'service', state: 'stopped', endpoints: [{ name: 'bad', url: 'http://127.0.0.1:8000' }] },
  ]) {
    const client = createAPI({ token, fetch: async () => response(item) });
    await expect(client.player.status('test')).rejects.toMatchObject({ code: 'invalid_response' });
  }
  const badFile = createAPI({ token, fetch: async () => response({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: '', files: [{ id: 'bad', name: 'data', size: -1 }], hint_count: 0, walkthrough: false }) });
  await expect(badFile.catalog.detail('test')).rejects.toMatchObject({ code: 'invalid_response' });
  await expect(invalid.catalog.download('test', 'a'.repeat(64))).rejects.toMatchObject({ code: 'invalid_response' });
  const missing = createAPI({ token, fetch: async () => response({ error: { code: 'not_found' } }, 404) });
  await expect(missing.catalog.download('test', 'a'.repeat(64))).rejects.toMatchObject({ code: 'not_found', status: 404 });
});

describe('Go player adapter', () => {
  it('maps all four operations, keeps extra response fields private and sends only the submission body', async () => {
    const requests: { url: string; options: RequestInit }[] = [];
    const responses = [
      { problems: [{ slug: 'test', title: 'Test', category: 'web', kind: 'service', extra: 1 }], extra: 1 },
      { slug: 'test', kind: 'service', file_count: 2, endpoints: [{ name: 'web', url: 'http://127.0.0.1:1234/' }], project: 'private' },
      { slug: 'test', accepted: false },
      { slug: 'test' },
    ];
    const client = createAPI({ token, fetch: async (input, options) => {
      requests.push({ url: String(input), options: options ?? {} });
      return response(responses.shift());
    } });
    expect(await client.catalog.list()).toEqual([{ slug: 'test', title: 'Test', category: 'web', kind: 'service' }]);
    expect(await client.player.run('test')).toEqual({ slug: 'test', kind: 'service', fileCount: 2, endpoints: [{ name: 'web', url: 'http://127.0.0.1:1234/' }] });
    expect(await client.player.submit('test', 'incorrect')).toEqual({ slug: 'test', accepted: false });
    expect(await client.player.stop('test')).toBeUndefined();
    expect(requests.map(item => [item.url, item.options.method, item.options.body])).toEqual([
      ['/api/v1/problems', 'GET', undefined],
      ['/api/v1/problems/test/run', 'POST', undefined],
      ['/api/v1/problems/test/submissions', 'POST', '{"flag":"incorrect"}'],
      ['/api/v1/problems/test/run', 'DELETE', undefined],
    ]);
    for (const { options } of requests) {
      expect(options.headers).toHaveProperty('Authorization', `Bearer ${token}`);
      expect(options.redirect).toBe('error');
      expect(options.credentials).toBe('omit');
    }
  });

  it('keeps error codes and excludes server diagnostics and fetch exceptions', async () => {
    const client = createAPI({ token, fetch: async () => response({ error: { code: 'already_running', message: 'private message' } }, 409) });
    await expect(client.player.run('test')).rejects.toMatchObject({ code: 'already_running', status: 409, message: 'The player request failed.' });
    const disconnected = createAPI({ token, fetch: async () => { throw new Error('secret'); } });
    await expect(disconnected.catalog.list()).rejects.toMatchObject({ code: 'network_error', message: 'The player request failed.' });
  });

  it('rejects bad slugs, empty/oversized flags and invalid tokens before transport', async () => {
    let called = false;
    const client = createAPI({ token, fetch: async () => { called = true; return response({}); } });
    await expect(client.player.run('../other')).rejects.toMatchObject({ code: 'invalid_argument' });
    await expect(client.player.submit('test', ' ')).rejects.toMatchObject({ code: 'invalid_argument' });
    await expect(client.player.submit('test', '한'.repeat(2000))).rejects.toMatchObject({ code: 'invalid_argument' });
    expect(called).toBe(false);
    expect(() => createAPI({ token: 'bad' })).toThrow(APIError);
  });

  it.each([
    { problems: null },
    { problems: [{ slug: 'test', title: 'Test', category: 'web', kind: 'other' }] },
  ])('rejects a response incompatible with the port', async payload => {
    const client = createAPI({ token, fetch: async () => response(payload) });
    await expect(client.catalog.list()).rejects.toMatchObject({ code: 'invalid_response' });
  });

  it('rejects unsafe endpoint schemes and mismatched response identities', async () => {
    const client = createAPI({ token, fetch: async () => response({ slug: 'test', kind: 'service', file_count: 0, endpoints: [{ name: 'web', url: 'javascript:alert(1)' }] }) });
    await expect(client.player.run('test')).rejects.toMatchObject({ code: 'invalid_response' });
    const mismatch = createAPI({ token, fetch: async () => response({ slug: 'other', accepted: true }) });
    await expect(mismatch.player.submit('test', 'flag')).rejects.toMatchObject({ code: 'invalid_response' });
  });
});
