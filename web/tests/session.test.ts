import { afterEach, expect, it, vi } from 'vitest';
import { takeSessionToken } from '../apps/player/src/session';

afterEach(() => vi.unstubAllGlobals());

it('takes the token once and removes it from the current URL before rendering', () => {
  const location = { hash: `#${'a'.repeat(64)}`, pathname: '/' };
  const replaceState = vi.fn(() => { location.hash = ''; });
  vi.stubGlobal('location', location);
  vi.stubGlobal('history', { replaceState });
  expect(takeSessionToken()).toBe('a'.repeat(64));
  expect(replaceState).toHaveBeenCalledWith(null, '', '/');
  expect(takeSessionToken()).toBeUndefined();
});

it.each(['', '#bad', `#${'a'.repeat(63)}`, `#${'A'.repeat(64)}`])('clears and rejects an invalid fragment %s', hash => {
  const replaceState = vi.fn();
  vi.stubGlobal('location', { hash, pathname: '/' });
  vi.stubGlobal('history', { replaceState });
  expect(takeSessionToken()).toBeUndefined();
  expect(replaceState).toHaveBeenCalledOnce();
});
