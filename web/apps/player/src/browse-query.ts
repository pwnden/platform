import type { LocationQuery, LocationQueryRaw } from 'vue-router';
import type { BrowseState } from '@pwnden/catalog-feature';

const scalar = (value: LocationQuery[string] | undefined) => typeof value === 'string' ? value : '';
export function browseFromQuery(query: LocationQuery): BrowseState {
  const difficulty = scalar(query.difficulty);
  const order = scalar(query.order);
  const page = Number(scalar(query.page));
  return {
    query: scalar(query.q), category: scalar(query.category),
    difficulty: /^(?:[1-5]|unrated)$/.test(difficulty) ? difficulty : '',
    order: order === 'easiest' || order === 'hardest' ? order : 'title',
    page: Number.isSafeInteger(page) && page >= 1 ? page : 1,
  };
}
export function browseToQuery(state: BrowseState, query: LocationQuery): LocationQueryRaw {
  return { ...query, q: state.query || undefined, category: state.category || undefined,
    difficulty: state.difficulty || undefined, order: state.order === 'title' ? undefined : state.order,
    page: state.page === 1 ? undefined : String(state.page) };
}
