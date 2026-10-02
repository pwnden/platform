import { expect, it } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UIButton from '../packages/ui/src/UIButton.vue';
import UIIconButton from '../packages/ui/src/UIIconButton.vue';
import UILink from '../packages/ui/src/UILink.vue';
import UITextField from '../packages/ui/src/UITextField.vue';
import UISelect from '../packages/ui/src/UISelect.vue';
import UIToggleButton from '../packages/ui/src/UIToggleButton.vue';
import UIForm from '../packages/ui/src/UIForm.vue';
import UISubmitButton from '../packages/ui/src/UISubmitButton.vue';
import UIPagination from '../packages/ui/src/UIPagination.vue';
import UIStatus from '../packages/ui/src/UIStatus.vue';
import UIReveal from '../packages/ui/src/UIReveal.vue';
import UIFile from '../packages/ui/src/UIFile.vue';
import UICode from '../packages/ui/src/UICode.vue';
import { terminalDocument } from '../packages/ui/src/terminal-document';
import UITerminalControls from '../packages/ui/src/UITerminalControls.vue';
import UITabs from '../packages/ui/src/UITabs.vue';
import UIWebFrame from '../packages/ui/src/UIWebFrame.vue';

it('keeps inactive tool panels mounted and inaccessible with linked Sectile tabs', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UITabs, {
    label: '풀이 도구', modelValue: 'terminal', items: [{ value: 'terminal', label: '터미널' }, { value: 'web', label: '웹' }],
  }, { terminal: () => 'shell state', web: () => 'web state', actions: () => h(UIIconButton, { label: '터미널 새로고침', icon: 'refresh' }) }) }));
  expect(html).toContain('role="tablist"');
  expect(html).toContain('aria-label="풀이 도구"');
  expect(html.match(/role="tab"/g)).toHaveLength(2);
  expect(html.match(/role="tabpanel"/g)).toHaveLength(2);
  expect(html).toContain('aria-selected="true"');
  expect(html).toContain('aria-selected="false"');
  expect(html).toContain('shell state');
  expect(html).toContain('web state');
  expect(html).toContain('inert');
  expect(html).toContain('aria-hidden="true"');
  expect(html.replace(/<!--[\s\S]*?-->/g, '')).toMatch(/<\/div><div class="ui-tabs-actions"[^>]*><button[^>]*aria-label="터미널 새로고침"/);
  for (const id of html.matchAll(/aria-controls="([^"]+)"/g)) expect(html).toContain(`id="${id[1]}"`);
});

it('gives toolbar actions stable names and a shared compact icon control', async () => {
  for (const icon of ['refresh', 'download'] as const) {
    const html = await renderToString(createSSRApp({ render: () => h(UIIconButton, { label: '도구 작업', icon, busy: true }) }));
    expect(html).toContain('ui-button--compact');
    expect(html).toContain('ui-icon-control');
    expect(html).toContain('aria-label="도구 작업"');
    expect(html).toContain('title="도구 작업"');
    expect(html).toContain('aria-busy="true"');
    expect(html).toContain(' disabled');
    expect(html).toContain('ui-icon-progress');
  }
  const link = await renderToString(createSSRApp({ render: () => h(UILink, { href: 'http://127.0.0.1:43123', newTab: true, iconOnly: true, size: 'compact', 'aria-label': '새 탭에서 열기' }) }));
  expect(link).toContain('ui-button--compact');
  expect(link).toContain('ui-icon-control');
  expect(link).toContain('rel="noopener noreferrer"');
});

it('frames problem documents separately with named, bounded browser capabilities', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UIWebFrame, { src: 'http://127.0.0.1:43123', title: 'web 문제 사이트' }) }));
  expect(html).toContain('<iframe');
  expect(html).toContain('src="http://127.0.0.1:43123"');
  expect(html).toContain('title="web 문제 사이트"');
  expect(html).toContain('referrerpolicy="no-referrer"');
  expect(html).toContain('allow-scripts allow-same-origin allow-forms');
  expect(html).not.toContain('allow-top-navigation');
  expect(html).not.toContain('srcdoc');
});

it('renders labeled Sectile category choices with a custom trigger and portaled listbox', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UISelect, { id: 'category', label: '분야', modelValue: 'web', options: [{ value: '', label: '전체 분야' }, { value: 'web', label: '웹' }] }) }));
  expect(html).toMatch(/<label for="category"[^>]*>분야<\/label>/);
  expect(html).toMatch(/<button id="category"[^>]*aria-haspopup="listbox"/);
  expect(html).toContain('aria-label="분야"');
  expect(html).toContain('aria-expanded="false"');
  expect(html).toMatch(/class="ui-select-value"[^>]*>웹<\/span>/);
  expect(html).toContain('<svg');
  expect(html).not.toContain('<select');
  const context: { teleports?: Record<string, string> } = {};
  await renderToString(createSSRApp({ render: () => h(UISelect, { id: 'difficulty', label: '난이도', modelValue: '', options: [{ value: '', label: '전체 난이도' }, { value: '1', label: '입문' }] }) }), context);
  const popup = context.teleports?.body ?? '';
  expect(popup).toContain('role="listbox"');
  expect(popup).toMatch(/aria-selected="true"[^>]*data-sectile-select-id(?:="")?\s/);
  expect(popup).toContain('전체 난이도');
  expect(popup).toContain('입문');
  expect(popup.match(/role="option"/g)).toHaveLength(2);
  const disabled = await renderToString(createSSRApp({ render: () => h(UISelect, { id: 'empty', label: '파일', modelValue: '', options: [], disabled: true }) }));
  expect(disabled).toMatch(/<button[^>]* disabled/);
});

it('renders a plain status without decorative markers', async () => {
  const status = await renderToString(createSSRApp({ render: () => h(UIStatus, {}, () => '준비됨') }));
  expect(status).toContain('준비됨');
  expect(status).not.toContain('aria-hidden');
});

it('uses Sectile for selected buttons and retains the shared busy and disabled states', async () => {
  for (const selected of [false, true]) {
    const html = await renderToString(createSSRApp({ render: () => h(UIToggleButton, { modelValue: selected, variant: 'row', busy: true }, () => '문제') }));
    expect(html).toContain(`aria-pressed="${selected}"`);
    expect(html).toContain('ui-button--row');
    expect(html).toContain(' disabled');
    expect(html).toContain('aria-busy="true"');
    expect(html.match(/<button/g)).toHaveLength(1);
  }
});

it('renders shared Sectile forms and submits with the existing input and button geometry', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UIForm, { submit: async () => {} }, () => [
    h(UITextField, { id: 'flag', label: '정답 플래그', modelValue: 'pwnden{solved}', required: true, readonly: true }),
    h(UISubmitButton, { disabled: true }, () => '완료'),
  ]) }));
  expect(html).toContain('data-scope="form"');
  expect(html).toContain('data-part="submit"');
  expect(html).toContain('type="submit"');
  expect(html).toContain('ui-button--primary');
  expect(html).toContain('value="pwnden{solved}"');
  expect(html).toContain(' readonly');
  expect(html).toContain(' required');
  expect(html.match(/<form/g)).toHaveLength(1);
  expect(html.match(/<button/g)).toHaveLength(1);
});

it('renders bounded Sectile pagination for empty, first, last and reduced result sets', async () => {
  const render = (total: number, page: number, disabled = false) => renderToString(createSSRApp({ render: () => h(UIPagination, { label: '문제 목록 페이지', total, modelValue: page, pageSize: 20, disabled }) }));
  const empty = await render(0, 1);
  expect(empty).toContain('0개');
  expect(empty).not.toContain('<button');
  const first = await render(45, 1);
  expect(first).toContain('role="navigation"');
  expect(first).toContain('data-scope="pagination"');
  expect(first).toContain('1–20 / 45개');
  expect(first.match(/<button[^>]* disabled/g)).toHaveLength(1);
  const last = await render(45, 3);
  expect(last).toContain('41–45 / 45개');
  expect(last.match(/<button[^>]* disabled/g)).toHaveLength(1);
  expect((await render(45, 2, true)).match(/<button[^>]* disabled/g)).toHaveLength(2);
  expect(await render(3, 3)).toContain('1–3 / 3개');
});

it('omits hidden spoilers and renders material source as escaped selectable code', async () => {
  const render = (open: boolean) => renderToString(createSSRApp({ render: () => h(UIReveal, { label: '해설 보기 · 정답 포함', modelValue: open }, () => 'answer spoiler') }));
  const closed = await render(false);
  expect(closed).toContain('aria-expanded="false"');
  expect(closed).toContain('data-scope="disclosure"');
  const panelID = closed.match(/aria-controls="([^"]+)"/)![1];
  expect(closed).toContain(`id="${panelID}"`);
  expect(closed).toContain('hidden');
  expect(closed).not.toContain('<details');
  expect(closed).not.toContain('<summary');
  expect(closed).not.toContain('answer spoiler');
  expect(await render(true)).toContain('answer spoiler');
  const source = await renderToString(createSSRApp({ render: () => h(UICode, { label: 'checker.py', source: '<script>run()</script>\n  indent' }) }));
  expect(source.replace(/<[^>]*>/g, '')).toContain('&lt;script&gt;run()&lt;/script&gt;\n  indent');
  expect(source).not.toContain('<script>');
  expect(source).toContain('tabindex="0"');
});

it('uses compact named terminal controls with distinct thumb icons in every state', async () => {
  const thumbIcons: string[] = [];
  for (const [state, label, checked, action] of [
    ['connected', '연결됨', true, '연결 해제'], ['connecting', '준비·연결 중', true, '연결 취소'],
    ['disconnected', '연결 해제됨', false, '터미널 다시 연결'], ['error', '연결 오류', false, '터미널 다시 연결'],
  ] as const) {
    const html = await renderToString(createSSRApp({ render: () => h(UITerminalControls, { state }) }));
    expect(html).toContain('role="group" aria-label="터미널 연결 제어"');
    expect(html).toContain('role="switch"');
    expect(html).toContain(`aria-checked="${checked}"`);
    expect(html).toContain('aria-label="터미널 연결"');
    expect(html).toContain(`title="${label} · ${action}"`);
    expect(html).toContain(`role="status"`);
    expect(html).toContain(label);
    expect(html.includes('aria-busy="true"')).toBe(state === 'connecting');
    expect(html).toContain('aria-label="문제 환경 종료"');
    expect(html).toContain('ui-switch--compact');
    expect(html).toMatch(/class="ui-switch-label ui-sr-only"/);
    expect(html).not.toContain('환경 종료</span>');
    expect(html).toContain('aria-hidden="true" focusable="false"');
    const thumb = html.match(/class="ui-switch-thumb ui-radius-inner"[^>]*>(.*?)<\/span>/s)![1];
    expect(thumb).toContain('<svg');
    thumbIcons.push(thumb.match(/<path d="([^"]+)"/)![1]);
  }
  expect(new Set(thumbIcons.slice(0, 3)).size).toBe(3);
  expect(thumbIcons[3]).toBe(thumbIcons[2]);
  const disabled = await renderToString(createSSRApp({ render: () => h(UITerminalControls, { state: 'disconnected', busy: true }) }));
  expect(disabled.match(/ disabled/g)).toHaveLength(2);
  const paused = await renderToString(createSSRApp({ render: () => h(UITerminalControls, { state: 'disconnected', disabled: true }) }));
  expect(paused.match(/ disabled/g)).toHaveLength(1);
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
  const downloadButton = (html: string) => html.match(/<button\b[^>]*>[\s\S]*?<\/button>/g)![1]!;
  const buttonText = (html: string) => downloadButton(html).replace(/<[^>]*>/g, '').trim();
  expect(buttonText(busy)).toBe(buttonText(closed));
  expect(downloadButton(busy).match(/<svg\b/g)).toHaveLength(1);
  expect(downloadButton(closed).match(/<svg\b/g)).toHaveLength(1);
  expect(downloadButton(busy)).toContain('ui-file-progress');
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
  expect(html).toMatch(/<label for="flag"(?: class="")?>플래그<\/label>/);
  expect(html).toContain('id="flag"');
  expect(html).toContain('value="test"');
  expect(html).toContain('required');
  expect(html).toContain('autocomplete="off"');
});
