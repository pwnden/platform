import { expect, it } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UIButton from '../packages/ui/src/UIButton.vue';
import UITextField from '../packages/ui/src/UITextField.vue';

it('uses a native non-submit button and disables it while busy', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UIButton, { busy: true }, () => '실행') }));
  expect(html).toContain('type="button"');
  expect(html).toContain('disabled');
  expect(html).toContain('aria-busy="true"');
});

it('renders the Sectile wrapper with a real label and controlled string value', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UITextField, { id: 'flag', label: '플래그', modelValue: 'test', required: true }) }));
  expect(html).toContain('<label for="flag">플래그</label>');
  expect(html).toContain('id="flag"');
  expect(html).toContain('value="test"');
  expect(html).toContain('required');
  expect(html).toContain('autocomplete="off"');
});
