import { expect, it } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UIButton from '../packages/ui/src/UIButton.vue';
import UITextField from '../packages/ui/src/UITextField.vue';
import UISelect from '../packages/ui/src/UISelect.vue';
import UIStatus from '../packages/ui/src/UIStatus.vue';
import UIReveal from '../packages/ui/src/UIReveal.vue';
import UIFile from '../packages/ui/src/UIFile.vue';
import UICode from '../packages/ui/src/UICode.vue';
import { terminalDocument } from '../packages/ui/src/terminal-document';
import UITerminalControls from '../packages/ui/src/UITerminalControls.vue';

it('renders labeled native category choices and a plain status without decorative markers', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UISelect, { id: 'category', label: '분야', modelValue: 'web', options: [{ value: '', label: '전체 분야' }, { value: 'web', label: '웹' }] }) }));
  expect(html).toContain('<label for="category">분야</label>');
  expect(html).toContain('<select id="category"');
  expect(html).toContain('<option value="web" selected>웹</option>');
  const status = await renderToString(createSSRApp({ render: () => h(UIStatus, {}, () => '준비됨') }));
  expect(status).toContain('준비됨');
  expect(status).not.toContain('aria-hidden');
});

it('omits hidden spoilers and renders material source as escaped selectable code', async () => {
  const render = (open: boolean) => renderToString(createSSRApp({ render: () => h(UIReveal, { label: '해설 보기 · 정답 포함', modelValue: open }, () => 'answer spoiler') }));
  const closed = await render(false);
  expect(closed).toContain('<summary');
  expect(closed).not.toContain('answer spoiler');
  expect(await render(true)).toContain('answer spoiler');
  const source = await renderToString(createSSRApp({ render: () => h(UICode, { label: 'checker.py', source: '<script>run()</script>\n  indent' }) }));
  expect(source.replace(/<[^>]*>/g, '')).toContain('&lt;script&gt;run()&lt;/script&gt;\n  indent');
  expect(source).not.toContain('<script>');
  expect(source).toContain('tabindex="0"');
});

it('groups the accessible visual connection state and connection controls together in every state', async () => {
  for (const [state, label, action] of [
    ['connected', '연결됨', '연결 해제'], ['connecting', '준비·연결 중', '연결 취소'],
    ['disconnected', '연결 해제됨', '터미널 다시 연결'], ['error', '연결 오류', '터미널 다시 연결'],
  ] as const) {
    const html = await renderToString(createSSRApp({ render: () => h(UITerminalControls, { state }) }));
    expect(html).toContain('role="group" aria-label="터미널 연결 제어"');
    expect(html).toContain(`ui-connection-status--${state}`);
    expect(html).toContain(`role="status" title="${label}"`);
    expect(html).toContain(`aria-label="${action}"`);
    expect(html).toContain('aria-label="문제 환경 종료"');
    expect(html).toContain('aria-hidden="true" focusable="false"');
  }
  const disabled = await renderToString(createSSRApp({ render: () => h(UITerminalControls, { state: 'disconnected', busy: true }) }));
  expect(disabled.match(/ disabled/g)).toHaveLength(2);
});

it('keeps disclosure and download as separate named buttons with stable preview relationships', async () => {
  const name = `files/${'nested/'.repeat(15)}checker.py`;
  const render = (open: boolean, size = 883, busy = false) => renderToString(createSSRApp({
    render: () => h(UIFile, { name, size, modelValue: open, busy }, () => 'private source'),
  }));
  const closed = await render(false);
  expect(closed).toContain(`aria-label="${name} 미리보기 열기"`);
  expect(closed).toContain(`aria-label="${name} 다운로드"`);
  expect(closed.match(/<button /g)).toHaveLength(2);
  expect(closed).not.toMatch(/<summary/);
  expect(closed).not.toContain('private source');
  const previewID = closed.match(/aria-controls="([^"]+)"/)![1];
  expect(closed).toContain(`id="${previewID}"`);
  expect(closed).toContain('hidden');
  expect(closed).toMatch(/title="883 바이트"[^>]*>883 B/);
  expect(closed).toContain('title="' + name + '"');
  const open = await render(true);
  expect(open).toContain('aria-expanded="true"');
  expect(open).toContain('private source');
  expect(open).not.toContain(' hidden');
  expect(open.indexOf('다운로드</span>')).toBeLessThan(open.indexOf('private source'));
  const busy = await render(false, 1024, true);
  expect(busy).toContain('1 KiB');
  expect(busy).toContain('aria-busy="true"');
  expect(busy.match(/ disabled/g)).toHaveLength(1);
  expect(await render(false, 0)).toContain('0 B');
});

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
