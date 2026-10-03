import { expect, it } from 'vitest';
import { connections } from '../domains/catalog/src/index';
import type { Concept, Problem } from '../domains/catalog/src/index';

const concept = (id: string, related: readonly string[] = []): Concept => ({ id, title: id, requires: [], related });
const problem = (slug: string, requires: readonly Concept[], teaches: readonly Concept[]): Problem => ({ slug, title: slug, category: 'web', kind: 'service', learning: { requires, teaches } });

it('distinguishes prerequisites, progression and related practice without using shared CLI or difficulty', () => {
  const first = { ...problem('ownership', [], [concept('ownership')]), cli: ['curl'], difficulty: 3 as const };
  const current = problem('export', [concept('ownership')], [concept('read-paths', ['policy'])]);
  const next = problem('history', [concept('read-paths')], [concept('history')]);
  const peer = problem('peer', [], [concept('read-paths')]);
  const reading = problem('policy', [], [concept('policy')]);
  const unrelated = { ...problem('unrelated', [], [concept('other')]), cli: ['curl'] };
  const legacy: Problem = { slug: 'legacy', title: 'Legacy', category: 'web', kind: 'service' };
  const catalog = [peer, unrelated, current, legacy, next, reading, first];
  expect(connections(current, catalog)).toEqual({ before: [first], after: [next], related: [peer, reading] });
  expect(connections(first, catalog).after).toEqual([current]);
  expect(connections(legacy, catalog)).toEqual({ before: [], after: [], related: [] });
  expect(catalog[0]).toBe(peer);
});
