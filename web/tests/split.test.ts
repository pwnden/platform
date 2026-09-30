import { expect, it } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UISplit from '../packages/ui/src/UISplit.vue';

it('uses Sectile separator semantics and bounded values without exposing upstream types', async () => {
  const html = await renderToString(createSSRApp({
    render: () => h(UISplit, { label: '설명과 터미널 너비', modelValue: 50, min: 30, max: 70 }, {
      before: () => h('section', '설명'), after: () => h('section', '터미널'),
    }),
  }));
  expect(html).toContain('role="separator"');
  expect(html).toContain('aria-label="설명과 터미널 너비"');
  expect(html).toContain('aria-valuemin="30"');
  expect(html).toContain('aria-valuemax="70"');
  expect(html).toContain('aria-valuenow="50"');
  expect(html).toContain('tabindex="0"');
  expect(html).toContain('<section>설명</section>');
  expect(html).toContain('<section>터미널</section>');
});
