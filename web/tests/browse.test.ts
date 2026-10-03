import { expect, it } from 'vitest';
import { categoryLabel, filterProblems } from '../features/catalog/src/browse';
import type { Problem } from '../domains/catalog/src/index';

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

it('finds tool names and other public brief words while respecting category and difficulty', () => {
  const problems: readonly Problem[] = [
    { slug: 'diagnostic-port', title: '개발용 점검 포트', category: 'misc', kind: 'service', difficulty: 2, searchText: 'nmap --unprivileged -sT로 포트를 조사하고 ncat으로 응답을 읽는다.' },
    { slug: 'note-vault', title: '다른 사람의 메모', category: 'web', kind: 'service', difficulty: 1 },
  ];
  expect(filterProblems(problems, '  ＮＭＡＰ ', '').map(p => p.slug)).toEqual(['diagnostic-port']);
  expect(filterProblems(problems, 'nmap 포트', 'misc', '2').map(p => p.slug)).toEqual(['diagnostic-port']);
  expect(filterProblems(problems, 'ncat', '').map(p => p.slug)).toEqual(['diagnostic-port']);
  expect(filterProblems(problems, 'nmap', 'web')).toEqual([]);
  expect(filterProblems(problems, 'nmap', '', '1')).toEqual([]);
  expect(filterProblems(problems, 'undefined', '')).toEqual([]);
});

it('combines difficulty with search and category, sorts within categories and keeps unrated problems last', () => {
  const problems: readonly Problem[] = [
    { slug: 'a-hard', title: 'A', category: 'web', kind: 'service', difficulty: 4 },
    { slug: 'b-easy', title: 'B', category: 'web', kind: 'service', difficulty: 2 },
    { slug: 'c-intro', title: 'C', category: 'rev', kind: 'file', difficulty: 1 },
    { slug: 'd-legacy', title: 'D', category: 'web', kind: 'service' },
  ];
  expect(filterProblems(problems, '', 'web', '', 'easiest').map(p => p.slug)).toEqual(['b-easy', 'a-hard', 'd-legacy']);
  expect(filterProblems(problems, '', 'web', '', 'hardest').map(p => p.slug)).toEqual(['a-hard', 'b-easy', 'd-legacy']);
  expect(filterProblems(problems, 'b', 'web', '2').map(p => p.slug)).toEqual(['b-easy']);
  expect(filterProblems(problems, '', 'rev', '2')).toEqual([]);
  expect(filterProblems(problems, '', '', 'unrated').map(p => p.slug)).toEqual(['d-legacy']);
  expect(problems.map(p => p.slug)).toEqual(['a-hard', 'b-easy', 'c-intro', 'd-legacy']);
});
