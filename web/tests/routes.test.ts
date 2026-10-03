import { expect, it } from 'vitest';
import { createMemoryHistory } from 'vue-router';
import { createPlayerRouter } from '../apps/player/src/router';
import { browseFromQuery, browseToQuery } from '../apps/player/src/browse-query';

it('generates nested challenge pages and redirects the entry URL with filters preserved', async () => {
  const router = createPlayerRouter(createMemoryHistory());
  expect(router.getRoutes().find(record => record.path === '/')?.redirect).toBe('/challenges');
  await router.push('/?q=nmap');
  expect(router.currentRoute.value.fullPath).toBe('/challenges?q=nmap');
  await router.push('/challenges/note-vault?tool=web');
  expect(router.currentRoute.value.params).toEqual({ slug: 'note-vault' });
  expect(router.currentRoute.value.matched.map(record => record.path)).toEqual(['/challenges', '/challenges/:slug']);
  await router.push('/not-a-page');
  expect(router.currentRoute.value.name).toBe('/[...path]');
});

it('parses URL filters conservatively and serializes defaults without losing tool selection', () => {
  expect(browseFromQuery({ q: 'nmap', category: 'web', difficulty: '2', order: 'easiest', page: '3' })).toEqual({ query: 'nmap', category: 'web', difficulty: '2', order: 'easiest', page: 3 });
  expect(browseFromQuery({ q: ['a', 'b'], difficulty: '99', order: 'anything', page: '-2' })).toEqual({ query: '', category: '', difficulty: '', order: 'title', page: 1 });
  const router = createPlayerRouter(createMemoryHistory());
  const query = browseToQuery({ query: 'nmap', category: '', difficulty: '', order: 'title', page: 1 }, { tool: 'web' });
  expect(router.resolve({ path: '/challenges/note-vault', query }).fullPath).toBe('/challenges/note-vault?tool=web&q=nmap');
});
