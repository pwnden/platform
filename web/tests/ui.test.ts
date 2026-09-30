import { expect, it } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UIButton from '../packages/ui/src/UIButton.vue';
import UITextField from '../packages/ui/src/UITextField.vue';
import { terminalDocument } from '../packages/ui/src/terminal-document';

it('uses a native non-submit button and disables it while busy', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UIButton, { busy: true }, () => '실행') }));
  expect(html).toContain('type="button"');
  expect(html).toContain('disabled');
  expect(html).toContain('aria-busy="true"');
});

it('assigns the page nonce only to terminal styles and keeps native DOM receivers', () => {
  const elements: { name: string; attributes: Record<string, string> }[] = [];
  const document = {
    querySelector() { return { content: 'b'.repeat(64) }; },
    createElement(name: string) {
      expect(this).toBe(document);
      const element = { name, attributes: {} as Record<string, string>, setAttribute(key: string, value: string) { this.attributes[key] = value; }, appendChild<T>(node: T) { expect(this).toBe(element); return node; } };
      elements.push(element);
      return element;
    },
    hasFocus() { expect(this).toBe(document); return true; },
  } as unknown as Document;
  const create = document.createElement;
  const scoped = terminalDocument(document);
  scoped.createElement('style'); scoped.createElement('div');
  expect(elements.map(element => element.attributes)).toEqual([{ nonce: 'b'.repeat(64) }, {}]);
  expect(scoped.hasFocus()).toBe(true);
  const viewport = scoped.createElement('div');
  const nativeStyle = document.createElement('style');
  Object.assign(nativeStyle, { nodeName: 'STYLE' });
  expect(viewport.appendChild(nativeStyle)).toBe(nativeStyle);
  expect(elements.at(-1)?.attributes.nonce).toBe('b'.repeat(64));
  expect(document.createElement).toBe(create);
  expect(() => terminalDocument({ querySelector: () => null } as unknown as Document)).toThrow('nonce');
});

it('renders the Sectile wrapper with a real label and controlled string value', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UITextField, { id: 'flag', label: '플래그', modelValue: 'test', required: true }) }));
  expect(html).toContain('<label for="flag">플래그</label>');
  expect(html).toContain('id="flag"');
  expect(html).toContain('value="test"');
  expect(html).toContain('required');
  expect(html).toContain('autocomplete="off"');
});
