import { describe, expect, it } from 'vitest';
import { APIError, createAPI } from '../packages/api/src/index';

const token = 'a'.repeat(64);
const response = (payload: unknown, status = 200) => new Response(JSON.stringify(payload), { status, headers: { 'Content-Type': 'application/json' } });

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
