import { afterEach, expect, it, vi } from 'vitest';
import { createApp, h, nextTick, ref } from 'vue';
import type { App } from 'vue';
import type { Catalog } from '../domains/catalog/src/index';
import ProblemList from '../features/catalog/src/ProblemList.vue';
import UISelect from '../packages/ui/src/UISelect.vue';

const apps: App[] = [];
afterEach(() => { for (const app of apps.splice(0)) app.unmount(); document.body.replaceChildren(); vi.restoreAllMocks(); });
async function settle() { for (let i = 0; i < 12; i++) { await Promise.resolve(); await nextTick(); } }

it('loads and displays the catalog with real Sectile controls mounted in the DOM', async () => {
  const errors: unknown[] = [];
  const problems = Array.from({ length: 25 }, (_, index) => ({ slug: `problem-${index}`, title: `Problem ${index}`, category: 'web', kind: 'service' as const, difficulty: 1 as const }));
  const catalog: Catalog = { list: vi.fn(async () => problems), detail: vi.fn(), download: vi.fn(), guidance: vi.fn() };
  const root = document.createElement('div'); document.body.append(root);
  const app = createApp({ render: () => h(ProblemList, { catalog }) });
  app.config.errorHandler = error => { errors.push(error); };
  apps.push(app); app.mount(root); await settle();
  expect(errors).toEqual([]);
  expect(catalog.list).toHaveBeenCalledOnce();
  expect(root.textContent).not.toContain('등록된 문제가 없습니다.');
  expect(root.querySelectorAll('.ui-button--row')).toHaveLength(20);
  expect(root.textContent).toContain('1–20 / 25개');
  const next = Array.from(root.querySelectorAll('button')).find(button => button.textContent === '다음')!;
  next.click(); await settle();
  expect(errors).toEqual([]);
  expect(root.querySelectorAll('.ui-button--row')).toHaveLength(5);
  expect(root.textContent).toContain('21–25 / 25개');
});

it('keeps empty selection values usable for clicks, keyboard selection and external updates', async () => {
  const errors: unknown[] = [];
  const selected = ref('');
  const options = [{ value: '', label: '전체 분야' }, { value: 'web', label: '웹' }, { value: 'option:', label: '다른 값' }];
  const root = document.createElement('div'); document.body.append(root);
  const app = createApp({ render: () => h(UISelect, { id: 'category', label: '분야', modelValue: selected.value, options, 'onUpdate:modelValue': (value: string) => { selected.value = value; } }) });
  app.config.errorHandler = error => { errors.push(error); };
  apps.push(app); app.mount(root); await settle();
  expect(errors).toEqual([]);
  const trigger = root.querySelector<HTMLButtonElement>('button')!;
  expect(trigger.textContent).toBe('전체 분야');
  trigger.click(); await settle();
  const option = (label: string) => Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find(option => option.textContent === label)!;
  option('웹').click(); await settle();
  expect(selected.value).toBe('web');
  expect(trigger.textContent).toBe('웹');
  trigger.click(); await settle();
  option('전체 분야').click(); await settle();
  expect(selected.value).toBe('');
  trigger.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true })); await settle();
  document.activeElement!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true })); await settle();
  expect(selected.value).toBe('web');
  selected.value = 'option:'; await settle();
  expect(trigger.textContent).toBe('다른 값');
  selected.value = ''; await settle();
  expect(trigger.textContent).toBe('전체 분야');
  expect(errors).toEqual([]);
});

it('shows a loading failure without claiming that the catalog is empty', async () => {
  const errors: unknown[] = [];
  const catalog: Catalog = { list: vi.fn(async () => { throw new Error('offline'); }), detail: vi.fn(), download: vi.fn(), guidance: vi.fn() };
  const root = document.createElement('div'); document.body.append(root);
  const app = createApp({ render: () => h(ProblemList, { catalog }) });
  app.config.errorHandler = error => { errors.push(error); };
  apps.push(app); app.mount(root); await settle();
  expect(errors).toEqual([]);
  expect(root.querySelector('[role="alert"]')?.textContent).toContain('목록을 불러오지 못했습니다.');
  expect(root.textContent).not.toContain('등록된 문제가 없습니다.');
  expect(root.textContent).not.toContain('검색 결과가 없습니다.');
});
