import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { expect, it, vi } from 'vitest';
import { browserState } from '../packages/ui/src/browser-channel';
import { createAPI } from '../packages/api/src/index';

const target = 'http://127.0.0.1:8000';
const src = 'http://127.0.0.1:9000/__pwnden_browser/' + 'a'.repeat(64);
const channel = new URL(src).pathname;

it('confines navigation states to the current wrapper and problem origin', () => {
  const state = { type: 'pwnden.browser.state.v1', channel, url: target + '/notes?id=2#entry', canBack: true, canForward: false, busy: false, error: '' };
  expect(browserState(state, src, target)?.url).toBe(state.url);
  for (const patch of [{ channel: '/other' }, { type: 'other' }, { url: 'http://evil.test/' }, { url: 'javascript:alert(1)' }, { url: 'http://user@127.0.0.1:8000/' }, { canBack: 'true' }, { busy: 0 }]) {
    expect(browserState({ ...state, ...patch }, src, target)).toBeUndefined();
  }
});

it('prepares only declared names through the authenticated relative API', async () => {
  const fetch = vi.fn(async () => new Response(JSON.stringify({ slug: 'example', name: 'web', url: src, target }), { headers: { 'Content-Type': 'application/json' } }));
  const client = createAPI({ token: 'f'.repeat(64), fetch });
  expect(await client.player.browser('example', 'web')).toEqual({ url: src, target });
  expect(fetch).toHaveBeenCalledWith('/api/v1/problems/example/browser', expect.objectContaining({ method: 'POST', body: '{"name":"web"}', redirect: 'error', credentials: 'omit' }));
  await expect(client.player.browser('example', '')).rejects.toMatchObject({ code: 'invalid_argument' });
  await expect(client.player.browser('../example', 'web')).rejects.toMatchObject({ code: 'invalid_argument' });
  for (const patch of [{ url: 'http://evil.test/' }, { target: 'http://evil.test/' }, { url: target + channel }, { target: 'http://127.0.0.1:65536' }, { name: 'other' }, { slug: 'other' }]) {
    fetch.mockImplementationOnce(async () => new Response(JSON.stringify({ slug: 'example', name: 'web', url: src, target, ...patch })));
    await expect(client.player.browser('example', 'web')).rejects.toMatchObject({ code: 'invalid_response' });
  }
});

function controller() {
  const listeners: Record<string, (event?: any) => void> = {};
  const childEvents: Record<string, (event?: any) => void> = {};
  const finished = { finished: Promise.resolve() };
  const navigation = { canGoBack: true, canGoForward: false, back: vi.fn(() => finished), forward: vi.fn(() => finished), addEventListener: vi.fn((name, listener) => { childEvents[name] = listener; }) };
  const child = { navigation, location: { href: 'http://127.0.0.1:9000/notes?id=2', replace: vi.fn(), reload: vi.fn(), assign: vi.fn() } };
  const parent = { postMessage: vi.fn() };
  const frame = { contentWindow: child, addEventListener: vi.fn((name, listener) => { listeners['frame:' + name] = listener; }) };
  const window = { addEventListener: vi.fn((name, listener) => { listeners[name] = listener; }) };
  const poll = vi.fn();
  runInNewContext(readFileSync(new URL('../../internal/httpapi/browser-controller.js', import.meta.url), 'utf8'), {
    document: { body: { dataset: { parent: 'http://127.0.0.1:7000', target } }, getElementById: () => frame },
    window, parent, location: new URL(src), URL, setInterval: (callback: () => void) => { poll.mockImplementation(callback); return 1; }, clearInterval: vi.fn(),
  });
  const send = (action: string, url?: string, origin = 'http://127.0.0.1:7000', source = parent) => listeners.message!({ origin, source, data: { type: 'pwnden.browser.command.v1', channel, action, url } });
  return { navigation, child, parent, send, poll, listeners, childEvents };
}

it('uses native frame history, reload and navigation without modifying problem documents', () => {
  const c = controller();
  c.listeners['frame:load']!();
  expect(c.parent.postMessage).toHaveBeenLastCalledWith(expect.objectContaining({ url: target + '/notes?id=2', canBack: true, canForward: false, busy: false }), 'http://127.0.0.1:7000');
  c.send('back'); expect(c.navigation.back).toHaveBeenCalledOnce();
  c.send('forward'); expect(c.navigation.forward).not.toHaveBeenCalled();
  c.navigation.canGoForward = true;
  c.send('forward'); expect(c.navigation.forward).toHaveBeenCalledOnce();
  c.send('reload'); expect(c.child.location.reload).toHaveBeenCalledOnce();
  c.send('navigate', target + '/notes?id=3#fragment');
  expect(c.child.location.assign).toHaveBeenCalledWith('http://127.0.0.1:9000/notes?id=3#fragment');
  c.send('navigate', 'http://evil.test/');
  expect(c.child.location.assign).toHaveBeenCalledOnce();
  c.send('back', undefined, 'http://evil.test'); c.send('back', undefined, 'http://127.0.0.1:7000', { postMessage: vi.fn() });
  expect(c.navigation.back).toHaveBeenCalledOnce();
  c.child.location.href = 'http://127.0.0.1:9000/notes?id=4#spa';
  c.poll();
  expect(c.parent.postMessage).toHaveBeenLastCalledWith(expect.objectContaining({ url: target + '/notes?id=4#spa' }), 'http://127.0.0.1:7000');
  expect(c.navigation.addEventListener).toHaveBeenCalledTimes(4);
  const count = c.parent.postMessage.mock.calls.length;
  c.poll();
  expect(c.parent.postMessage).toHaveBeenCalledTimes(count);
  c.listeners['frame:load']!();
  c.childEvents.navigate!({ downloadRequest: 'notes.txt' });
  expect(c.parent.postMessage).toHaveBeenLastCalledWith(expect.objectContaining({ busy: false }), 'http://127.0.0.1:7000');
});

it('reports missing native navigation without inventing a history stack', () => {
  const c = controller();
  (c.child as { navigation: unknown }).navigation = undefined;
  c.listeners['frame:load']!();
  expect(c.parent.postMessage).toHaveBeenLastCalledWith(expect.objectContaining({ canBack: false, canForward: false, error: expect.stringContaining('새 탭') }), 'http://127.0.0.1:7000');
  c.send('back'); c.send('forward');
  expect(c.navigation.back).not.toHaveBeenCalled();
  expect(c.navigation.forward).not.toHaveBeenCalled();
});
