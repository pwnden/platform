import type { Problem } from '@pwnden/catalog';

export interface BrowseState {
  query: string;
  category: string;
  difficulty: string;
  order: string;
  page: number;
}

const categories: Readonly<Record<string, string>> = {
  web: '웹', pwn: '시스템 해킹', rev: '리버싱', crypto: '암호', forensics: '포렌식', misc: '기타',
};
export function categoryLabel(category: string): string { return categories[category] ?? category; }

function normalize(value: string): string { return value.normalize('NFKC').toLocaleLowerCase('ko-KR').trim(); }
export function filterProblems(problems: readonly Problem[], query: string, category: string, difficulty = '', order = 'title'): readonly Problem[] {
  const terms = normalize(query).split(/\s+/).filter(Boolean);
  return problems.filter(problem => (!category || problem.category === category) &&
    (!difficulty || (difficulty === 'unrated' ? problem.difficulty === undefined : String(problem.difficulty) === difficulty)) &&
    terms.every(term => normalize(`${problem.title} ${problem.slug} ${categoryLabel(problem.category)} ${problem.category} ${(problem.cli ?? []).join(' ')}`).includes(term)))
    .toSorted((a, b) => {
      const categoryOrder = categoryLabel(a.category).localeCompare(categoryLabel(b.category), 'ko');
      if (categoryOrder) return categoryOrder;
      if (order === 'easiest' || order === 'hardest') {
        if (a.difficulty === undefined && b.difficulty !== undefined) return 1;
        if (b.difficulty === undefined && a.difficulty !== undefined) return -1;
        const difference = (a.difficulty ?? 0) - (b.difficulty ?? 0);
        if (difference) return order === 'easiest' ? difference : -difference;
      }
      return a.title.localeCompare(b.title, 'ko') || a.slug.localeCompare(b.slug);
    });
}
