import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { takeSessionToken, forgetSessionToken } from '../apps/player/src/session';

const first = 'a'.repeat(64), second = 'b'.repeat(64);
let location: { hash: string; pathname: string; search: string };
let stored: Map<string, string>;
let replaceState: ReturnType<typeof vi.fn>;
beforeEach(() => {
  location = { hash: `#${first}`, pathname: '/', search: '' };
  stored = new Map();
  replaceState = vi.fn(() => { location.hash = ''; });
  vi.stubGlobal('location', location);
  vi.stubGlobal('history', { replaceState, state: null });
  vi.stubGlobal('sessionStorage', {
    getItem: (key: string) => stored.get(key) ?? null,
    setItem: (key: string, value: string) => stored.set(key, value),
    removeItem: (key: string) => stored.delete(key),
  });
});
afterEach(() => vi.unstubAllGlobals());

it('removes the URL credential and retains it for reloads and HMR in the same tab', () => {
  expect(takeSessionToken()).toBe(first);
  expect(replaceState).toHaveBeenCalledWith(null, '', '/');
  expect(location.hash).toBe('');
  expect(takeSessionToken()).toBe(first);
});

it('replaces a previous server credential when a fresh full URL is opened', () => {
  takeSessionToken();
  location.hash = `#${second}`;
  expect(takeSessionToken()).toBe(second);
  forgetSessionToken(first);
  expect(takeSessionToken()).toBe(second);
  forgetSessionToken(second);
  expect(takeSessionToken()).toBeUndefined();
});

it('preserves the challenge path, filters and router history while removing the credential', () => {
  location.pathname = '/challenges/note-vault';
  location.search = '?q=nmap&page=2';
  const state = { position: 3 };
  vi.stubGlobal('history', { replaceState, state });
  expect(takeSessionToken()).toBe(first);
  expect(replaceState).toHaveBeenCalledWith(state, '', '/challenges/note-vault?q=nmap&page=2');
  replaceState.mockClear();
  expect(takeSessionToken()).toBe(first);
  expect(replaceState).not.toHaveBeenCalled();
});

it.each(['#bad', `#${'a'.repeat(63)}`, `#${'A'.repeat(64)}`])('rejects an invalid explicit fragment and clears old credentials: %s', hash => {
  takeSessionToken();
  location.hash = hash;
  expect(takeSessionToken()).toBeUndefined();
  expect(location.hash).toBe('');
  expect(takeSessionToken()).toBeUndefined();
});

it('shows recovery for an uninitialized tab or corrupted stored credential', () => {
  location.hash = '';
  expect(takeSessionToken()).toBeUndefined();
  stored.set('pwnden.session', 'bad');
  expect(takeSessionToken()).toBeUndefined();
  expect(stored.size).toBe(0);
});

it('opens a full URL even when tab storage is unavailable', () => {
  vi.stubGlobal('sessionStorage', { getItem() { throw new Error('disabled'); }, setItem() { throw new Error('disabled'); }, removeItem() { throw new Error('disabled'); } });
  expect(takeSessionToken()).toBe(first);
  expect(takeSessionToken()).toBeUndefined();
  expect(() => forgetSessionToken(first)).not.toThrow();
});
