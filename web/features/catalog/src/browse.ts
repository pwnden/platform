import type { Problem } from '@pwnden/catalog';

const categories: Readonly<Record<string, string>> = {
  web: '웹', pwn: '시스템 해킹', rev: '리버싱', crypto: '암호', forensics: '포렌식', misc: '기타',
};
export function categoryLabel(category: string): string { return categories[category] ?? category; }

function normalize(value: string): string { return value.normalize('NFKC').toLocaleLowerCase('ko-KR').trim(); }
export function filterProblems(problems: readonly Problem[], query: string, category: string): readonly Problem[] {
  const terms = normalize(query).split(/\s+/).filter(Boolean);
  return problems.filter(problem => (!category || problem.category === category) &&
    terms.every(term => normalize(`${problem.title} ${problem.slug} ${categoryLabel(problem.category)} ${problem.category}`).includes(term)))
    .toSorted((a, b) => categoryLabel(a.category).localeCompare(categoryLabel(b.category), 'ko') || a.title.localeCompare(b.title, 'ko') || a.slug.localeCompare(b.slug));
}
