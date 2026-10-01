import { afterEach, expect, it, vi } from 'vitest';
import { createRenderer, h, nextTick } from 'vue';
import type { Catalog } from '../domains/catalog/src/index';
import type { Player, RunStatus } from '../domains/play/src/index';
import ProblemDetail from '../features/catalog/src/ProblemDetail.vue';
import ProblemFiles from '../features/catalog/src/ProblemFiles.vue';
import ProblemList from '../features/catalog/src/ProblemList.vue';
import PlayPanel from '../features/play/src/PlayPanel.vue';
import ProblemWeb from '../features/play/src/ProblemWeb.vue';
import type { ProblemWebHandle } from '../features/play/src/props';
import TerminalPanel from '../features/terminal/src/TerminalPanel.vue';
import type { TerminalPanelHandle } from '../features/terminal/src/props';
import type { ProblemFilesHandle } from '../features/catalog/src/props';
import App from '../apps/player/src/App.vue';
import UICode from '../packages/ui/src/UICode.vue';
import * as syntax from '../packages/ui/src/syntax';

function browserSession(target: string) { return { target, url: target + '/__pwnden_browser/' + 'a'.repeat(64) }; }
const unusedBrowser = vi.fn();

// Exercise feature lifecycle and real tool tabs on a small host renderer.
vi.mock('../packages/ui/src/index.ts', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    UIPanel: (await import('../packages/ui/src/UIPanel.vue')).default,
    UIStatus: (await import('../packages/ui/src/UIStatus.vue')).default,
    UIBadge: (await import('../packages/ui/src/UIBadge.vue')).default,
    UILink: (await import('../packages/ui/src/UILink.vue')).default,
    UIIconButton: (await import('../packages/ui/src/UIIconButton.vue')).default,
    UITerminalControls: (await import('../packages/ui/src/UITerminalControls.vue')).default,
    UIFile: (await import('../packages/ui/src/UIFile.vue')).default,
    UITabs: (await import('../packages/ui/src/UITabs.vue')).default,
    UIWebFrame: defineComponent({ props: ['src', 'title', 'target'], emits: ['navigation'], setup: (props, { attrs, expose, emit }) => {
      const controls = { back: vi.fn(), forward: vi.fn(), reload: vi.fn(), navigate: vi.fn() };
      expose(controls);
      return () => h('iframe', { ...attrs, ...controls, src: props.src, onWebNavigation: (state: unknown) => emit('navigation', state) });
    } }),
    UISplit: defineComponent({ setup: (_, { attrs, slots }) => () => h('split', attrs, [slots.before?.(), slots.after?.()]) }),
    UIMarkdown: defineComponent({ props: ['source'], setup: props => () => h('markdown', props.source) }),
    UICode: defineComponent({ props: ['source'], setup: props => () => h('code', props.source) }),
    UIReveal: defineComponent({ props: ['label', 'modelValue'], setup: (props, { attrs, slots }) => () => h('reveal', { ...attrs, label: props.label, modelValue: props.modelValue }, [props.label, props.modelValue ? slots.default?.() : null]) }),
    UIButton: defineComponent({ setup: (_, { attrs, slots }) => () => h('button', attrs, slots.default?.()) }),
    UITextField: defineComponent({ setup: (_, { attrs }) => () => h('input', attrs) }),
    UISelect: defineComponent({ setup: (_, { attrs }) => () => h('select', attrs) }),
    UITerminal: defineComponent({ setup: (_, { attrs, expose }) => {
      expose({ write: (_data: Uint8Array, rendered: () => void) => rendered(), clear: () => {}, focus: () => {} });
      return () => h('terminal', attrs);
    } }),
  };
});

it('automatically connects and releases pending and ready attachments on unmount', async () => {
  const close = vi.fn();
  let resolve: (value: unknown) => void = () => {};
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  const terminals = { connect: vi.fn(() => ({ ready: new Promise<any>(done => { resolve = done; }), close })), list: vi.fn(), stop: vi.fn() };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  await settle();
  expect(terminals.connect).toHaveBeenCalledOnce();
  expect(text(root)).toContain('풀이 환경에 연결하는 중');
  renderer.render(null, root);
  resolve(session); await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(session.close).toHaveBeenCalledOnce();
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  resolve(session); await settle();
  expect(flatten(root).find(item => item.type === 'terminal')!.props.enabled).toBe(true);
  const terminal = flatten(root).find(item => item.type === 'terminal')!;
  (terminal.props.onInput as (data: Uint8Array) => void)(new Uint8Array([3]));
  expect(session.input).toHaveBeenCalledWith(new Uint8Array([3]));
  renderer.render(null, root);
  expect(close).toHaveBeenCalledTimes(2);
});

it('automatically enters Note Vault and preserves its attachment across column resize', async () => {
  const problem = { slug: 'note-vault', title: 'Note Vault', category: 'web', kind: 'service' as const };
  let status: RunStatus = { ...problem, state: 'stopped', endpoints: [] };
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  const close = vi.fn();
  const client = {
    catalog: { list: vi.fn(async () => [problem]), detail: vi.fn(async () => ({ ...problem, description: '설명', files: [], hintCount: 0, walkthrough: false })), download: vi.fn(), guidance: vi.fn() },
    player: {
      browser: unusedBrowser,
      status: vi.fn(async () => status), run: vi.fn(async () => { status = { ...status, state: 'running' }; return { ...status, fileCount: 0 }; }),
      stop: vi.fn(async () => { status = { ...status, state: 'stopped' }; }), submit: vi.fn(),
    },
    terminals: { connect: vi.fn(() => { status = { ...status, state: 'running' }; return { ready: Promise.resolve(session), close }; }), list: vi.fn(), stop: vi.fn(async () => { status = { ...status, state: 'stopped' }; }) },
  };
  const root = node('root');
  renderer.render(h(App, { client }), root);
  await settle();
  await click(flatten(root).find(item => item.type === 'button' && text(item).includes('Note Vault'))!);
  await settle();
  expect(client.terminals.connect).toHaveBeenCalledOnce();
  expect(client.player.run).not.toHaveBeenCalled();
  expect(button(root, '터미널 새로고침')).toBeDefined();
  const heading = flatten(root).find(item => item.props.class === 'problem-header')!;
  expect(text(heading)).toBe('Note Vault분야: 웹');
  expect(flatten(heading).find(item => item.props.class === 'ui-badge')?.props.title).toBe('분야: 웹');
  expect(text(flatten(heading).find(item => item.props.class === 'ui-sr-only')!)).toBe('분야: ');
  for (const split of flatten(root).filter(item => item.type === 'split')) {
    (split.props['onUpdate:modelValue'] as (value: number) => void)(30);
  }
  await settle();
  expect(client.terminals.connect).toHaveBeenCalledOnce();
  expect(close).not.toHaveBeenCalled();
  const terminal = flatten(root).find(item => item.type === 'terminal')!;
  expect(terminal.props.enabled).toBe(true);
  (terminal.props.onInput as (bytes: Uint8Array) => void)(new Uint8Array([112, 119, 100, 13]));
  expect(session.input).toHaveBeenCalledWith(new Uint8Array([112, 119, 100, 13]));
  (terminal.props.onResize as (size: { cols: number; rows: number }) => void)({ cols: 100, rows: 30 });
  expect(session.resize).toHaveBeenLastCalledWith({ cols: 100, rows: 30 });
  expect(button(root, '문제 실행')).toBeUndefined();
  expect(button(root, '문제 중지')).toBeUndefined();
  expect(button(root, '문제 환경 종료')).toBeUndefined();
  expect(button(root, '터미널 연결')).toBeUndefined();
  expect(text(root)).not.toContain('풀이 터미널');
  await click(button(root, '터미널 새로고침')); await settle();
  expect(client.terminals.stop).not.toHaveBeenCalled();
  expect(client.player.stop).not.toHaveBeenCalled();
  expect(close).toHaveBeenCalledOnce();
  expect(client.terminals.connect).toHaveBeenCalledTimes(2);
  expect(terminal.props.enabled).toBe(true);
  renderer.render(null, root);
});

it('keeps terminal, source and web state across tool tabs and resets tools on problem change', async () => {
  const id = 'f'.repeat(64);
  const problems = [
    { slug: 'note-vault', title: 'Note Vault', category: 'web', kind: 'service' as const },
    { slug: 'rotor-lock', title: 'Rotor Lock', category: 'rev', kind: 'file' as const },
  ];
  const close = vi.fn();
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  let state: RunStatus = { slug: 'note-vault', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:43123' }] };
  const client = {
    catalog: {
      list: vi.fn(async () => problems),
      detail: vi.fn(async (slug: string) => ({ ...problems.find(problem => problem.slug === slug)!, description: '설명', files: [{ id, name: 'checker.py', size: 30 }], hintCount: 0, walkthrough: false })),
      download: vi.fn(async () => new TextEncoder().encode('<script>plain source</script>')), guidance: vi.fn(),
    },
    player: { browser: vi.fn(async () => browserSession(state.endpoints[0]!.url)), status: vi.fn(async (slug: string) => slug === 'note-vault' ? state : { slug, kind: 'file', state: 'ready', endpoints: [] }), run: vi.fn(), stop: vi.fn(), submit: vi.fn() },
    terminals: {
      connect: vi.fn(() => ({ ready: Promise.resolve(session), close })), list: vi.fn(),
      stop: vi.fn(async () => { state = { ...state, state: 'stopped', endpoints: [] }; }),
    },
  };
  const root = node('root');
  renderer.render(h(App, { client }), root); await settle();
  await click(flatten(root).find(item => item.type === 'button' && text(item).includes('Note Vault'))!); await settle();
  const terminal = flatten(root).find(item => item.type === 'terminal')!;
  expect(client.catalog.detail).toHaveBeenCalledOnce();
  expect(flatten(root).filter(item => item.props.role === 'tab').map(text)).toEqual(['터미널', '파일', '웹']);
  expect(flatten(root).filter(item => item.type === 'iframe')).toHaveLength(0);
  const list = flatten(root).find(item => item.props.role === 'tablist')!;
  const key = new Event('keydown', { cancelable: true });
  Object.defineProperties(key, { key: { value: 'ArrowRight' }, currentTarget: { value: list } });
  (list.props.onKeydown as (event: Event) => void)(key); await settle();
  expect(button(root, '파일').props['aria-selected']).toBe('true');
  expect(button(root, 'checker.py 미리보기 열기')).toBeUndefined();
  const actions = flatten(root).find(item => item.props.class === 'ui-tabs-actions')!;
  expect(button(root, 'checker.py 다운로드').parent).toBe(actions);
  expect(button(root, '터미널 새로고침')).toBeUndefined();
  expect(client.catalog.download).toHaveBeenCalledWith('note-vault', id, 1 << 20);
  expect(text(root)).toContain('<script>plain source</script>');
  expect(flatten(root).some(item => item.type === 'script')).toBe(false);
  await click(button(root, '웹')); await settle();
  const frame = flatten(root).find(item => item.type === 'iframe')!;
  expect(frame.props.src).toBe(browserSession('http://127.0.0.1:43123').url);
  expect(button(root, '문제 웹 새로고침').parent?.props['aria-label']).toBe('웹 탐색');
  expect(button(root, 'checker.py 다운로드')).toBeUndefined();
  expect(flatten(root).find(item => item.type === 'a')?.props.rel).toBe('noopener noreferrer');
  await click(button(root, '터미널')); await settle();
  expect(flatten(root).find(item => item.type === 'terminal')).toBe(terminal);
  expect(client.terminals.connect).toHaveBeenCalledOnce();
  expect(close).not.toHaveBeenCalled();
  await click(button(root, '파일')); await settle();
  expect(text(root)).toContain('<script>plain source</script>');
  expect(client.catalog.download).toHaveBeenCalledOnce();
  await click(button(root, '웹')); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')).toBe(frame);
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')).toBe(frame);
  await click(button(root, '터미널')); await settle();
  state = { ...state, state: 'stopped', endpoints: [] };
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(flatten(root).some(item => item.type === 'iframe')).toBe(false);
  expect(button(root, '웹')).toBeDefined();
  state = { ...state, state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:43124' }] };
  await click(button(root, '터미널 새로고침')); await settle();
  await click(button(root, '웹')); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')?.props.src).toBe(browserSession('http://127.0.0.1:43124').url);
  await click(flatten(root).find(item => item.type === 'button' && text(item).includes('Rotor Lock'))!); await settle();
  expect(flatten(root).filter(item => item.props.role === 'tab').map(text)).toEqual(['터미널', '파일']);
  expect(button(root, '터미널').props['aria-selected']).toBe('true');
  expect(flatten(root).some(item => item.type === 'iframe')).toBe(false);
  expect(client.catalog.download).toHaveBeenCalledOnce();
  expect(button(root, '터미널 새로고침')).toBeDefined();
  renderer.render(null, root);
});

it('retains multiple web documents and reloads only the explicitly selected one', async () => {
  const endpoints = [{ name: 'first', url: 'http://127.0.0.1:43123' }, { name: 'second', url: 'http://127.0.0.1:43124' }];
  const state: RunStatus = { slug: 'test', kind: 'service', state: 'running', endpoints };
  const root = node('root');
  const player: Player = { browser: vi.fn(async (_slug: string, name: string) => browserSession(endpoints.find(endpoint => endpoint.name === name)!.url)), status: vi.fn(), run: vi.fn(), stop: vi.fn(), submit: vi.fn() };
  const component = h(ProblemWeb, { player, slug: 'test', status: state, active: true });
  renderer.render(component, root); await settle();
  const web = component.component!.exposed as ProblemWebHandle;
  const first = flatten(root).find(item => item.type === 'iframe')!;
  const selector = flatten(root).find(item => item.type === 'select')!;
  const choose = selector.props['onUpdate:modelValue'] as (url: string) => void;
  choose(endpoints[1]!.url); await settle();
  const second = flatten(root).find(item => item.type === 'iframe' && item.props.src === browserSession(endpoints[1]!.url).url)!;
  expect(first.props.hidden).toBe(true);
  expect(second.props.hidden).toBe(false);
  choose(endpoints[0]!.url); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')).toBe(first);
  expect(web!.url).toBe(endpoints[0]!.url);
  (first.props.onWebNavigation as (state: unknown) => void)({ url: endpoints[0]!.url + '/notes?id=2', canBack: true, canForward: true, busy: false, error: '' });
  await settle();
  expect(web!.url).toBe(endpoints[0]!.url + '/notes?id=2');
  expect(web!.canBack).toBe(true);
  expect(web!.canForward).toBe(true);
  web!.back(); web!.forward();
  expect(first.props.back).toHaveBeenCalledOnce();
  expect(first.props.forward).toHaveBeenCalledOnce();
  web!.reload(); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')).toBe(first);
  expect(first.props.reload).toHaveBeenCalledOnce();
  expect(second.props.reload).not.toHaveBeenCalled();
  expect(player.browser).toHaveBeenCalledTimes(2);
  expect(flatten(root).find(item => item.type === 'iframe' && item.props.src === browserSession(endpoints[1]!.url).url)).toBe(second);
  renderer.render(h(ProblemWeb, { player, slug: 'test', status: undefined, active: true }), root); await settle();
  expect(flatten(root).some(item => item.type === 'iframe' || item.type === 'a')).toBe(false);
  renderer.render(null, root);
});

it('resolves web paths within the current problem and restores unusable input without navigation', async () => {
  const origin = 'http://127.0.0.1:43123';
  const status: RunStatus = { slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: origin }] };
  const player: Player = { browser: vi.fn(async () => browserSession(origin)), status: vi.fn(), run: vi.fn(), stop: vi.fn(), submit: vi.fn() };
  const root = node('root');
  renderer.render(h(ProblemWeb, { player, slug: 'test', status, active: true }), root); await settle();
  const frame = flatten(root).find(item => item.type === 'iframe')!;
  const field = () => flatten(root).find(item => item.props.id === 'web-address')!;
  const form = flatten(root).find(item => item.type === 'form')!;
  const submit = async (input: string) => {
    (field().props['onUpdate:modelValue'] as (value: string) => void)(input);
    (form.props.onSubmit as (event: { preventDefault(): void }) => void)({ preventDefault() {} });
    await settle();
  };
  expect(field().props.modelValue).toBe('/');
  (frame.props.onWebNavigation as (state: unknown) => void)({ url: origin + '/notes/view?id=2', canBack: true, canForward: false, busy: false, error: '' });
  await settle();
  expect(field().props.modelValue).toBe('/notes/view?id=2');
  for (const [input, path] of [
    [' /notes?id=3 ', '/notes?id=3'], ['?id=4', '/notes/view?id=4'],
    ['#entry', '/notes/view?id=2#entry'], ['../login', '/login'],
    [origin + '/notes?id=5', '/notes?id=5'],
  ]) {
    await submit(input!);
    expect(frame.props.navigate).toHaveBeenLastCalledWith(origin + path);
    expect(field().props.modelValue).toBe(path);
  }
  const navigation = frame.props.navigate as ReturnType<typeof vi.fn>;
  navigation.mockClear();
  for (const input of ['', ' ', 'https://example.com/notes', '//example.com/notes',
    'http://127.0.0.1:43124/notes', 'javascript:alert(1)', 'http://[invalid',
    'http://user:password@127.0.0.1:43123/notes']) {
    await submit(input);
    expect(navigation).not.toHaveBeenCalled();
    expect(field().props.modelValue).toBe('/notes/view?id=2');
    expect(flatten(root).some(item => item.props.role === 'alert')).toBe(false);
  }
  expect(flatten(root).find(item => item.type === 'iframe')).toBe(frame);
  renderer.render(null, root);
});

it('prepares web lazily, retries once and ignores a response for a removed endpoint', async () => {
  const state: RunStatus = { slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:43123' }] };
  let resolve: (value: { url: string; target: string }) => void = () => {};
  const browser = vi.fn().mockRejectedValueOnce(new Error('offline')).mockImplementationOnce(() => new Promise(done => { resolve = done; }));
  const player: Player = { browser, status: vi.fn(), run: vi.fn(), stop: vi.fn(), submit: vi.fn() };
  const root = node('root');
  const component = h(ProblemWeb, { player, slug: 'test', status: state, active: false });
  renderer.render(component, root); await settle();
  expect(browser).not.toHaveBeenCalled();
  renderer.render(h(ProblemWeb, { player, slug: 'test', status: state, active: true }), root); await settle();
  expect(text(root)).toContain('새로고침으로 다시 시도');
  const web = component.component!.exposed as ProblemWebHandle;
  web.reload(); web.reload(); await settle();
  expect(browser).toHaveBeenCalledTimes(2);
  renderer.render(h(ProblemWeb, { player, slug: 'test', active: true }), root); await settle();
  resolve(browserSession(state.endpoints[0]!.url)); await settle();
  expect(flatten(root).some(item => item.type === 'iframe')).toBe(false);
  renderer.render(null, root);
});

it('observes a newly prepared web service after the initial status request finishes', async () => {
  const problem = { slug: 'note-vault', title: 'Note Vault', category: 'web', kind: 'service' as const };
  let resolve: (status: RunStatus) => void = () => {};
  const status = vi.fn().mockImplementationOnce(() => new Promise<RunStatus>(done => { resolve = done; }))
    .mockResolvedValue({ slug: problem.slug, kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:43123' }] });
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  const client = {
    catalog: { list: vi.fn(async () => [problem]), detail: vi.fn(async () => ({ ...problem, description: '', files: [], hintCount: 0, walkthrough: false })), download: vi.fn(), guidance: vi.fn() },
    player: { browser: vi.fn(async () => browserSession('http://127.0.0.1:43123')), status, run: vi.fn(), stop: vi.fn(), submit: vi.fn() },
    terminals: { connect: vi.fn(() => ({ ready: Promise.resolve(session), close: vi.fn() })), list: vi.fn(), stop: vi.fn() },
  };
  const root = node('root');
  renderer.render(h(App, { client }), root); await settle();
  await click(flatten(root).find(item => item.type === 'button' && text(item).includes('Note Vault'))!); await settle();
  expect(status).toHaveBeenCalledOnce();
  resolve({ slug: problem.slug, kind: 'service', state: 'stopped', endpoints: [] }); await settle();
  expect(status).toHaveBeenCalledTimes(2);
  expect(button(root, '웹')).toBeDefined();
  await click(button(root, '웹')); await settle();
  expect(flatten(root).find(item => item.type === 'iframe')?.props.src).toBe(browserSession('http://127.0.0.1:43123').url);
  renderer.render(null, root);
});

it('retries failed preparation explicitly and exposes environments when the limit is reached', async () => {
  const stop = vi.fn(async () => {});
  const list = vi.fn(async () => [{ slug: 'other', title: '다른 문제', connected: false, expiresAt: null }]);
  const terminals = { connect: vi.fn((_slug, _size, receive) => {
    receive({ type: 'error', code: 'workspace_full' });
    return { ready: Promise.reject(new Error('workspace_full')), close: vi.fn() };
  }), list, stop };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'note-vault' }), root);
  await settle();
  expect(terminals.connect).toHaveBeenCalledOnce();
  expect(text(root)).toContain('최대 10개');
  expect(text(root)).toContain('다른 문제');
  await click(button(root, '종료')); await settle();
  expect(stop).toHaveBeenCalledWith('other');
  expect(terminals.connect).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});

it('detaches on hidden pages and automatically reattaches when visible', async () => {
  const target = new EventTarget();
  const document = { visibilityState: 'visible', addEventListener: target.addEventListener.bind(target), removeEventListener: target.removeEventListener.bind(target) };
  vi.stubGlobal('document', document);
  const close = vi.fn();
  const terminals = { connect: vi.fn(() => ({ ready: Promise.resolve({ input: vi.fn(), resize: vi.fn(), close }), close })), list: vi.fn(), stop: vi.fn() };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  await settle();
  document.visibilityState = 'hidden'; target.dispatchEvent(new Event('visibilitychange')); await settle();
  expect(close).toHaveBeenCalledOnce();
  document.visibilityState = 'visible'; target.dispatchEvent(new Event('visibilitychange')); await settle();
  expect(terminals.connect).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});

it('refreshes the attachment without stopping the retained environment', async () => {
  const target = new EventTarget();
  const document = { visibilityState: 'visible', addEventListener: target.addEventListener.bind(target), removeEventListener: target.removeEventListener.bind(target) };
  vi.stubGlobal('document', document);
  const close = vi.fn();
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn(), reused: true };
  const terminals = { connect: vi.fn(() => ({ ready: Promise.resolve(session), close })), list: vi.fn(), stop: vi.fn(async () => {}) };
  const root = node('root');
  const component = h(TerminalPanel, { terminals, slug: 'test' });
  renderer.render(component, root);
  const terminal = component.component!.exposed as TerminalPanelHandle;
  await settle();
  expect(flatten(root).some(item => item.type === 'header')).toBe(false);
  await terminal!.refresh(); await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(terminals.stop).not.toHaveBeenCalled();
  expect(flatten(root).find(item => item.type === 'terminal')!.props.enabled).toBe(true);
  expect(terminals.connect).toHaveBeenCalledTimes(2);
  document.visibilityState = 'hidden'; target.dispatchEvent(new Event('visibilitychange'));
  document.visibilityState = 'visible'; target.dispatchEvent(new Event('visibilitychange')); await settle();
  expect(terminals.connect).toHaveBeenCalledTimes(3);
  expect(terminals.stop).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('keeps a single pending preparation when refresh is requested repeatedly', async () => {
  const close = vi.fn();
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  let resolve: (value: unknown) => void = () => {};
  const terminals = { connect: vi.fn(() => ({ ready: new Promise<any>(done => { resolve = done; }), close })), list: vi.fn(), stop: vi.fn() };
  const root = node('root');
  const component = h(TerminalPanel, { terminals, slug: 'test' });
  renderer.render(component, root);
  const terminal = component.component!.exposed as TerminalPanelHandle;
  await settle();
  expect(terminal!.busy).toBe(true);
  await terminal!.refresh(); await terminal!.refresh(); await settle();
  expect(close).not.toHaveBeenCalled();
  expect(terminals.connect).toHaveBeenCalledOnce();
  resolve(session); await settle();
  expect(terminal!.busy).toBe(false);
  expect(flatten(root).find(item => item.type === 'terminal')!.props.enabled).toBe(true);
  expect(terminals.stop).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('drops an old capacity list when a new connection is already ready', async () => {
  let resolveList: (value: unknown) => void = () => {};
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  const terminals = {
    connect: vi.fn((_slug, _size, receive) => {
      receive({ type: 'error', code: 'workspace_full' });
      return { ready: Promise.reject(new Error('workspace_full')), close: vi.fn() };
    }),
    list: vi.fn(() => new Promise<any>(done => { resolveList = done; })), stop: vi.fn(),
  };
  const root = node('root');
  const component = h(TerminalPanel, { terminals, slug: 'test' });
  renderer.render(component, root);
  const terminal = component.component!.exposed as TerminalPanelHandle;
  await settle();
  terminals.connect.mockImplementationOnce(() => ({ ready: Promise.resolve(session), close: vi.fn() }));
  await terminal!.refresh(); await settle();
  resolveList([{ slug: 'old', title: '오래된 유지 환경', connected: false, expiresAt: null }]); await settle();
  expect(flatten(root).find(item => item.type === 'terminal')!.props.enabled).toBe(true);
  expect(text(root)).not.toContain('오래된 유지 환경');
  renderer.render(null, root);
});

it('keeps the current material source when an old highlight result arrives late', async () => {
  const pending: ((value: syntax.CodeLines) => void)[] = [];
  vi.spyOn(syntax, 'highlightCode').mockImplementation(() => new Promise(done => { pending.push(done); }));
  const root = node('root');
  renderer.render(h(UICode, { source: 'old', label: 'old.py' }), root);
  await settle();
  renderer.render(h(UICode, { source: 'current', label: 'current.py' }), root);
  await settle();
  pending[1]!([[{ content: 'current', className: 'ui-syntax--keyword' }]]); await settle();
  pending[0]!([[{ content: 'old', className: 'ui-syntax--string' }]]); await settle();
  expect(text(root)).toBe('current');
  expect(flatten(root).find(item => item.type === 'span')!.props.class).toBe('ui-syntax--keyword');
  renderer.render(h(UICode, { source: 'last', label: 'last.py' }), root); await settle();
  renderer.render(null, root);
  pending[2]!([[{ content: 'last', className: 'ui-syntax--string' }]]); await settle();
  expect(root.children).toHaveLength(0);
});

interface Node {
  type: string;
  text: string;
  props: Record<string, unknown>;
  children: Node[];
  parent: Node | null;
  readonly dataset: { tabsId?: unknown };
  closest(selector: string): Node | null;
  querySelectorAll(selector: string): Node[];
  focus(): void;
}
const node = (type: string, text = ''): Node => ({
  type, text, props: {}, children: [], parent: null,
  get dataset() { return { tabsId: this.props['data-tabs-id'] }; },
  closest() { let current: Node | null = this; while (current && current.props.role !== 'tablist') current = current.parent; return current; },
  querySelectorAll() { return flatten(this).filter(item => item.props['data-tabs-id'] !== undefined); },
  focus() {},
});
const renderer = createRenderer<Node, Node>({
  createElement: type => node(type), createText: text => node('text', text), createComment: text => node('comment', text),
  setText: (item, text) => { item.text = text; }, setElementText: (item, text) => { item.text = text; item.children = []; },
  patchProp: (item, key, _, value) => { item.props[key] = value; },
  insert: (item, parent, anchor) => {
    if (item.parent) item.parent.children.splice(item.parent.children.indexOf(item), 1);
    const index = anchor ? parent.children.indexOf(anchor) : -1;
    parent.children.splice(index < 0 ? parent.children.length : index, 0, item);
    item.parent = parent;
  },
  remove: item => { item.parent?.children.splice(item.parent.children.indexOf(item), 1); item.parent = null; },
  parentNode: item => item.parent,
  nextSibling: item => item.parent?.children[item.parent.children.indexOf(item) + 1] ?? null,
});
const text = (item: Node): string => (item.type === 'comment' ? '' : item.text) + item.children.map(text).join('');
const flatten = (item: Node): Node[] => [item, ...item.children.flatMap(flatten)];
const button = (root: Node, label: string) => flatten(root).find(item => item.type === 'button' && (text(item) === label || item.props['aria-label'] === label))!;
const click = (item: Node) => {
  const event = new Event('click');
  Object.defineProperty(event, 'currentTarget', { value: item });
  return (item.props.onClick as (event: Event) => unknown)(event);
};
async function settle() { for (let i = 0; i < 8; i++) { await Promise.resolve(); await nextTick(); } }
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

it('browses a large catalog by category, search and bounded pages without repeating identifiers', async () => {
  const problems = Array.from({ length: 501 }, (_, i) => ({ slug: `exercise-${i}`, title: `Exercise ${String(i).padStart(3, '0')}`, category: i % 2 ? 'web' : 'rev', kind: 'file' as const }));
  const catalog: Catalog = { list: vi.fn(async () => problems), detail: vi.fn(), download: vi.fn(), guidance: vi.fn() };
  const selected = vi.fn();
  const root = node('root');
  renderer.render(h(ProblemList, { catalog, onSelect: selected }), root);
  await settle();
  const rows = () => flatten(root).filter(item => item.type === 'button' && item.props.variant === 'row');
  expect(rows()).toHaveLength(20);
  expect(text(root)).toContain('1–20 / 501개');
  expect(text(root)).not.toContain('exercise-');
  const first = text(rows()[0]!);
  await click(button(root, '다음')); await settle();
  expect(text(root)).toContain('21–40 / 501개');
  expect(text(rows()[0]!)).not.toBe(first);
  const search = flatten(root).find(item => item.props.id === 'problem-search')!;
  const category = flatten(root).find(item => item.props.id === 'problem-category')!;
  (category.props['onUpdate:modelValue'] as (value: string) => void)('web');
  (search.props['onUpdate:modelValue'] as (value: string) => void)('EXERCISE-101');
  await settle();
  expect(rows()).toHaveLength(1);
  expect(text(root)).toContain('웹');
  expect(text(root)).toContain('1–1 / 1개');
  await click(rows()[0]!);
  expect(selected).toHaveBeenCalledWith(problems[101]);
  (search.props['onUpdate:modelValue'] as (value: string) => void)('missing');
  await settle();
  expect(rows()).toHaveLength(0);
  expect(text(root)).toContain('검색 결과가 없습니다.');
  await click(button(root, '검색 조건 초기화')); await settle();
  expect(rows()).toHaveLength(20);
  expect(text(root)).toContain('1–20 / 501개');
  renderer.render(null, root);
});

it('loads spoiler documents on demand, retries failures and caches reopened hints', async () => {
  const id = 'd'.repeat(64);
  const catalog: Catalog = {
    list: vi.fn(),
    detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: 'Find the key', files: [{ id, name: 'checker.py', size: 30 }], hintCount: 2, walkthrough: true })),
    download: vi.fn(async () => new TextEncoder().encode('<script>plain source</script>')),
    guidance: vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce('first clue').mockResolvedValueOnce('full answer'),
  };
  const root = node('root');
  renderer.render(h(ProblemDetail, { catalog, slug: 'test', title: 'Test' }), root);
  await settle();
  const reveal = async (label: string, open = true) => {
    const item = flatten(root).find(item => item.type === 'reveal' && item.props.label === label)!;
    await (item.props['onUpdate:modelValue'] as (open: boolean) => unknown)(open);
    await settle();
  };
  expect(catalog.guidance).not.toHaveBeenCalled();
  expect(catalog.download).not.toHaveBeenCalled();
  await reveal('힌트 1');
  expect(text(root)).toContain('힌트를 불러오지 못했습니다.');
  await click(button(root, '힌트 다시 불러오기')); await settle();
  expect(text(root)).toContain('first clue');
  await reveal('힌트 1', false);
  expect(text(root)).not.toContain('first clue');
  await reveal('힌트 1');
  expect(catalog.guidance).toHaveBeenCalledTimes(2);
  expect(text(root)).not.toContain('full answer');
  await reveal('해설 보기 · 정답 포함');
  expect(catalog.guidance).toHaveBeenLastCalledWith('test', 'walkthrough');
  expect(text(root)).toContain('full answer');
  expect(button(root, 'checker.py 미리보기 열기')).toBeUndefined();
  expect(catalog.download).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('routes binary and large materials to the prepared terminal without fetching a large preview', async () => {
  const small = 'a'.repeat(64), large = 'b'.repeat(64);
  const catalog: Catalog = {
    list: vi.fn(),
    detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: 'Find the key', files: [{ id: small, name: 'binary', size: 2 }, { id: large, name: 'large', size: (1 << 20) + 1 }], hintCount: 0, walkthrough: false })),
    download: vi.fn(async () => new Uint8Array([0, 255])), guidance: vi.fn(),
  };
  const root = node('root');
  renderer.render(h(ProblemFiles, { catalog, slug: 'test', files: [{ id: small, name: 'binary', size: 2 }, { id: large, name: 'large', size: (1 << 20) + 1 }] }), root);
  await settle();
  expect(text(root)).toContain('텍스트로 표시할 수 없는 자료');
  (flatten(root).find(item => item.type === 'select')!.props['onUpdate:modelValue'] as (id: string) => void)(large); await settle();
  expect(text(root)).toContain('큰 자료는 터미널 탭');
  expect(catalog.download).toHaveBeenCalledOnce();
  renderer.render(null, root);
});

it('observes service endpoints without duplicating terminal environment controls', async () => {
  let state: RunStatus = { slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] };
  const player: Player = { browser: unusedBrowser,
    status: vi.fn(async () => state),
    run: vi.fn(), stop: vi.fn(),
    submit: vi.fn(async () => ({ slug: 'test', accepted: false })),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'service' }), root);
  await settle();
  expect(text(root)).toContain('실행 중');
  expect(flatten(root).some(item => item.type === 'a')).toBe(false);
  expect(button(root, '문제 실행')).toBeUndefined();
  expect(button(root, '문제 중지')).toBeUndefined();
  state = { ...state, state: 'stopped', endpoints: [] };
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('오른쪽 터미널을 연결하면');
  expect(flatten(root).some(item => item.type === 'a')).toBe(false);
  expect(flatten(root).some(item => item.props.class === 'actions')).toBe(false);
  state = { ...state, state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:9000' }] };
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('실행 중');
  expect(player.status).toHaveBeenCalledTimes(3);
  expect(player.run).not.toHaveBeenCalled();
  expect(player.stop).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('directs unavailable service recovery to the terminal environment controls', async () => {
  const player: Player = { browser: unusedBrowser,
    status: vi.fn().mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'stopped', endpoints: [] })
      .mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] })
      .mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'unavailable', endpoints: [] }),
    run: vi.fn(), stop: vi.fn(), submit: vi.fn(),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'service' }), root);
  await settle();
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('실행 중');
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('문제에서 나간 뒤 10분 후 다시 열면 환경을 새로 준비합니다.');
  expect(button(root, '문제 실행')).toBeUndefined();
  expect(button(root, '문제 중지')).toBeUndefined();
  expect(player.run).not.toHaveBeenCalled();
  expect(player.stop).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('shows file problems without service controls', async () => {
  const player: Player = { browser: unusedBrowser, status: vi.fn(async () => ({ slug: 'test', kind: 'file', state: 'ready', endpoints: [] })), run: vi.fn(), stop: vi.fn(), submit: vi.fn() };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'file' }), root);
  await settle();
  expect(text(root)).toContain('준비됨');
  expect(button(root, '문제 실행')).toBeUndefined();
  const refresh = button(root, '실행 상태 새로고침');
  expect(refresh.parent?.props.class).toBe('ui-panel-actions');
  expect(text(refresh.parent!)).toContain('준비됨');
  expect(flatten(root).some(item => item.props.class === 'actions')).toBe(false);
  await click(refresh); await settle();
  expect(player.status).toHaveBeenCalledTimes(2);
  expect(player.run).not.toHaveBeenCalled();
  expect(player.stop).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('clears stale endpoints on failed refresh and ignores a response after unmount', async () => {
  let resolve: (value: RunStatus) => void = () => {};
  const player: Player = { browser: unusedBrowser,
    status: vi.fn().mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] })
      .mockRejectedValueOnce(new Error('offline')).mockImplementationOnce(() => new Promise(value => { resolve = value; })),
    run: vi.fn(), stop: vi.fn(), submit: vi.fn(),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'service' }), root);
  await settle();
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('실행 상태를 확인하지 못했습니다.');
  expect(flatten(root).some(item => item.type === 'a')).toBe(false);
  expect(button(root, '문제 중지')).toBeUndefined();
  expect(button(root, '문제 실행')).toBeUndefined();
  const previousText = text(root);
  const pending = click(button(root, '실행 상태 새로고침'));
  await settle();
  expect(text(root)).toBe(previousText);
  renderer.render(null, root);
  resolve({ slug: 'test', kind: 'service', state: 'running', endpoints: [] });
  await pending; await settle();
  expect(root.children).toHaveLength(0);
});

it.each(['file', 'service'] as const)('keeps the %s submission and body stable during status refresh', async kind => {
  const state: RunStatus = { slug: 'test', kind, state: kind === 'file' ? 'ready' : 'running', endpoints: kind === 'file' ? [] : [{ name: 'web', url: 'http://127.0.0.1:8000' }] };
  let resolve: (value: RunStatus) => void = () => {};
  const player: Player = { browser: unusedBrowser,
    status: vi.fn(async () => state), run: vi.fn(), stop: vi.fn(),
    submit: vi.fn(async () => ({ slug: 'test', accepted: true })),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind }), root);
  await settle();
  const input = flatten(root).find(item => item.type === 'input')!;
  const form = flatten(root).find(item => item.type === 'form')!;
  const setFlag = input.props['onUpdate:modelValue'] as (value: string) => void;
  setFlag('pwnden{answer}'); await settle();
  await (form.props.onSubmit as (event: Event) => Promise<unknown>)(new Event('submit'));
  await settle();
  expect(text(root)).toContain('정답입니다.');
  setFlag('pwnden{next}'); await settle();
  const previousText = text(root);
  const previousStructure = flatten(root).map(item => item.type);
  player.status = vi.fn(() => new Promise<RunStatus>(done => { resolve = done; }));
  const refresh = button(root, '실행 상태 새로고침');
  const pending = click(refresh); await settle();
  expect(text(root)).toBe(previousText);
  expect(flatten(root).map(item => item.type)).toEqual(previousStructure);
  expect(flatten(root).find(item => item.type === 'form')).toBe(form);
  expect(input.props.modelValue).toBe('pwnden{next}');
  expect(input.props.disabled).toBe(false);
  expect(refresh.props.busy).toBe(true);
  expect(button(root, '정답 확인').props.disabled).toBe(true);
  void click(refresh); await settle();
  expect(player.status).toHaveBeenCalledOnce();
  resolve(state); await pending; await settle();
  expect(text(root)).toBe(previousText);
  expect(flatten(root).map(item => item.type)).toEqual(previousStructure);
  expect(input.props.modelValue).toBe('pwnden{next}');
  expect(button(root, '정답 확인').props.disabled).toBe(false);
  renderer.render(null, root);
});

it('downloads only the selected declared file from the materials panel', async () => {
  const id = 'a'.repeat(64);
  const catalog: Catalog = {
    list: vi.fn(), detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: '<script>alert(1)</script>', files: [{ id, name: 'files/data.bin', size: 3 }], hintCount: 0, walkthrough: false })),
    download: vi.fn(async () => new Uint8Array([0, 1, 255])),
    guidance: vi.fn(),
  };
  const link = { href: '', download: '', click: vi.fn(), remove: vi.fn() };
  vi.stubGlobal('document', { createElement: () => link, body: { append: vi.fn() } });
  const create = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:local');
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  const root = node('root');
  const component = h(ProblemFiles, { catalog, slug: 'test', files: [{ id, name: 'files/data.bin', size: 3 }] });
  renderer.render(component, root);
  const materials = component.component!.exposed as ProblemFilesHandle;
  await settle();
  expect(flatten(root).some(item => item.type === 'script')).toBe(false);
  await materials!.download(); await settle();
  expect(catalog.download).toHaveBeenCalledWith('test', id);
  expect(link.download).toBe('data.bin');
  expect(link.click).toHaveBeenCalledOnce();
  expect(new Uint8Array(await create.mock.calls[0]![0].arrayBuffer())).toEqual(new Uint8Array([0, 1, 255]));
  expect(button(root, 'files/data.bin 미리보기 열기')).toBeUndefined();
  expect(catalog.download).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});

it('automatically displays source, retries failures and caches it across tool visibility', async () => {
  const id = 'e'.repeat(64);
  let resolve: (bytes: Uint8Array) => void = () => {};
  const catalog: Catalog = {
    list: vi.fn(), guidance: vi.fn(),
    detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: '', files: [{ id, name: 'files/checker.py', size: 883 }], hintCount: 0, walkthrough: false })),
    download: vi.fn().mockRejectedValueOnce(new Error('offline')).mockImplementationOnce(() => new Promise<Uint8Array>(done => { resolve = done; })),
  };
  const root = node('root');
  const files = [{ id, name: 'files/checker.py', size: 883 }];
  renderer.render(h(ProblemFiles, { catalog, slug: 'test', files }), root);
  await settle();
  expect(text(root)).toContain('files/checker.py');
  expect(text(root)).not.toContain('883 B');
  expect(button(root, 'files/checker.py 미리보기 열기')).toBeUndefined();
  expect(text(root)).toContain('자료를 불러오지 못했습니다.');
  void click(button(root, '자료 다시 불러오기')); await settle();
  expect(text(root)).toContain('자료를 불러오는 중');
  renderer.render(h(ProblemFiles, { catalog, slug: 'test', files, foreground: false }), root); await settle();
  resolve(new TextEncoder().encode('print("ready")')); await settle();
  renderer.render(h(ProblemFiles, { catalog, slug: 'test', files, foreground: true }), root); await settle();
  expect(text(root)).toContain('print("ready")');
  expect(catalog.download).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});

it('keeps the selected file visible when another preview arrives late and downloads only that selection', async () => {
  const files = [{ id: 'a'.repeat(64), name: 'first.py', size: 20 }, { id: 'b'.repeat(64), name: 'second.py', size: 20 }];
  let resolve: (bytes: Uint8Array) => void = () => {};
  const catalog: Catalog = {
    list: vi.fn(), detail: vi.fn(), guidance: vi.fn(),
    download: vi.fn().mockImplementationOnce(() => new Promise<Uint8Array>(done => { resolve = done; }))
      .mockResolvedValue(new TextEncoder().encode('second source')),
  };
  const root = node('root');
  const component = h(ProblemFiles, { catalog, slug: 'test', files });
  renderer.render(component, root); await settle();
  const materials = component.component!.exposed as ProblemFilesHandle;
  const select = flatten(root).find(item => item.type === 'select')!.props['onUpdate:modelValue'] as (id: string) => void;
  select(files[1]!.id); await settle();
  expect(text(root)).toContain('second source');
  expect(materials.filename).toBe('second.py');
  resolve(new TextEncoder().encode('first source')); await settle();
  expect(text(root)).not.toContain('first source');
  const link = { href: '', download: '', click: vi.fn(), remove: vi.fn() };
  vi.stubGlobal('document', { createElement: () => link, body: { append: vi.fn() } });
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:local');
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});
  await materials.download(); await settle();
  expect(catalog.download).toHaveBeenLastCalledWith('test', files[1]!.id);
  expect(link.download).toBe('second.py');
  select(files[0]!.id); await settle();
  expect(text(root)).toContain('first source');
  expect(catalog.download).toHaveBeenCalledTimes(3);
  renderer.render(null, root);
});
