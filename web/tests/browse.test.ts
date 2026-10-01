import { expect, it } from 'vitest';
import { categoryLabel, filterProblems } from '../features/catalog/src/browse';

it('searches all words across titles, identifiers and readable categories without mutating input', () => {
  const problems = [
    { slug: 'rotor-lock', title: 'Rotor Lock', category: 'rev', kind: 'file' as const },
    { slug: 'note-vault', title: 'Note Vault', category: 'web', kind: 'service' as const },
    { slug: 'test-new', title: '새 문제', category: 'new-category', kind: 'file' as const },
  ];
  expect(filterProblems(problems, '  ＮＯＴＥ 웹 ', '').map(p => p.slug)).toEqual(['note-vault']);
  expect(filterProblems(problems, '리버싱', '').map(p => p.slug)).toEqual(['rotor-lock']);
  expect(filterProblems(problems, 'lock', 'web')).toEqual([]);
  expect(filterProblems(problems, '', 'new-category')).toEqual([problems[2]]);
  expect(categoryLabel('new-category')).toBe('new-category');
  expect(problems.map(p => p.slug)).toEqual(['rotor-lock', 'note-vault', 'test-new']);
});
