import { afterEach, expect, it, vi } from 'vitest';
import { createRenderer, h, nextTick } from 'vue';
import type { Catalog } from '../domains/catalog/src/index';
import type { Player, RunStatus } from '../domains/play/src/index';
import ProblemDetail from '../features/catalog/src/ProblemDetail.vue';
import ProblemList from '../features/catalog/src/ProblemList.vue';
import PlayPanel from '../features/play/src/PlayPanel.vue';
import TerminalPanel from '../features/terminal/src/TerminalPanel.vue';
import App from '../apps/player/src/App.vue';
import UICode from '../packages/ui/src/UICode.vue';
import * as syntax from '../packages/ui/src/syntax';

// Exercise feature lifecycle and async handlers; UI/Sectile have their own tests.
vi.mock('../packages/ui/src/index.ts', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    UIPanel: (await import('../packages/ui/src/UIPanel.vue')).default,
    UIStatus: (await import('../packages/ui/src/UIStatus.vue')).default,
    UITerminalControls: (await import('../packages/ui/src/UITerminalControls.vue')).default,
    UIFile: (await import('../packages/ui/src/UIFile.vue')).default,
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
  expect(text(root)).toContain('준비·연결 중');
  renderer.render(null, root);
  resolve(session); await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(session.close).toHaveBeenCalledOnce();
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  resolve(session); await settle();
  expect(text(root)).toContain('연결됨');
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
      status: vi.fn(async () => status), run: vi.fn(async () => { status = { ...status, state: 'running' }; return { ...status, fileCount: 0 }; }),
      stop: vi.fn(async () => { status = { ...status, state: 'stopped' }; }), submit: vi.fn(),
    },
    terminals: { connect: vi.fn(() => { status = { ...status, state: 'running' }; return { ready: Promise.resolve(session), close }; }), list: vi.fn(), stop: vi.fn() },
  };
  const root = node('root');
  renderer.render(h(App, { client }), root);
  await settle();
  await click(flatten(root).find(item => item.type === 'button' && text(item).includes('Note Vault'))!);
  await settle();
  expect(client.terminals.connect).toHaveBeenCalledOnce();
  expect(client.player.run).not.toHaveBeenCalled();
  expect(text(root)).toContain('연결됨');
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
  await click(button(root, '문제 중지')); await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(terminal.props.enabled).toBe(false);
  expect(client.terminals.connect).toHaveBeenCalledOnce();
  await click(button(root, '문제 실행')); await settle();
  expect(client.terminals.connect).toHaveBeenCalledTimes(2);
  expect(terminal.props.enabled).toBe(true);
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

it('disconnects through the header without stopping the environment and waits for explicit reconnect', async () => {
  const target = new EventTarget();
  const document = { visibilityState: 'visible', addEventListener: target.addEventListener.bind(target), removeEventListener: target.removeEventListener.bind(target) };
  vi.stubGlobal('document', document);
  const close = vi.fn();
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn(), reused: true };
  const terminals = { connect: vi.fn(() => ({ ready: Promise.resolve(session), close })), list: vi.fn(), stop: vi.fn(async () => {}) };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  await settle();
  const header = flatten(root).find(item => item.type === 'header')!;
  expect(text(header)).toContain('연결됨');
  expect(button(root, '터미널 연결').props.role).toBe('switch');
  expect(button(root, '터미널 연결').props['aria-checked']).toBe('true');
  await click(button(root, '터미널 연결')); await settle();
  expect(button(root, '터미널 연결').props['aria-checked']).toBe('false');
  expect(close).toHaveBeenCalledOnce();
  expect(terminals.stop).not.toHaveBeenCalled();
  expect(flatten(root).find(item => item.type === 'terminal')!.props.enabled).toBe(false);
  document.visibilityState = 'hidden'; target.dispatchEvent(new Event('visibilitychange'));
  document.visibilityState = 'visible'; target.dispatchEvent(new Event('visibilitychange')); await settle();
  expect(terminals.connect).toHaveBeenCalledOnce();
  await click(button(root, '터미널 연결')); await settle();
  expect(terminals.connect).toHaveBeenCalledTimes(2);
  expect(text(root)).toContain('연결됨');
  await click(button(root, '문제 환경 종료')); await settle();
  expect(terminals.stop).toHaveBeenCalledWith('test');
  document.visibilityState = 'hidden'; target.dispatchEvent(new Event('visibilitychange'));
  document.visibilityState = 'visible'; target.dispatchEvent(new Event('visibilitychange')); await settle();
  expect(terminals.connect).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});

it('cancels pending preparation and releases a late ready session without changing the disconnected state', async () => {
  const close = vi.fn();
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  let resolve: (value: unknown) => void = () => {};
  const terminals = { connect: vi.fn(() => ({ ready: new Promise<any>(done => { resolve = done; }), close })), list: vi.fn(), stop: vi.fn() };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  await settle();
  expect(button(root, '터미널 연결').props['aria-busy']).toBe(true);
  await click(button(root, '터미널 연결')); await settle();
  expect(close).toHaveBeenCalledOnce();
  resolve(session); await settle();
  expect(session.close).toHaveBeenCalledOnce();
  expect(text(root)).toContain('연결 해제됨');
  expect(text(root)).not.toContain('연결됨');
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
  renderer.render(h(TerminalPanel, { terminals, slug: 'test' }), root);
  await settle();
  terminals.connect.mockImplementationOnce(() => ({ ready: Promise.resolve(session), close: vi.fn() }));
  await click(button(root, '터미널 연결')); await settle();
  resolveList([{ slug: 'old', title: '오래된 유지 환경', connected: false, expiresAt: null }]); await settle();
  expect(text(root)).toContain('연결됨');
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
}
const node = (type: string, text = ''): Node => ({ type, text, props: {}, children: [], parent: null });
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
const click = (item: Node) => (item.props.onClick as (event: Event) => unknown)(new Event('click'));
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

it('loads material and spoiler documents on demand, retries failures and caches reopened hints', async () => {
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
  await click(button(root, 'checker.py 미리보기 열기')); await settle();
  expect(catalog.download).toHaveBeenCalledWith('test', id, 1 << 20);
  expect(text(root)).toContain('<script>plain source</script>');
  expect(flatten(root).some(item => item.type === 'script')).toBe(false);
  await click(button(root, 'checker.py 미리보기 닫기')); await settle();
  expect(text(root)).not.toContain('<script>plain source</script>');
  await click(button(root, 'checker.py 미리보기 열기')); await settle();
  expect(catalog.download).toHaveBeenCalledOnce();
  expect(text(root)).toContain('<script>plain source</script>');
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
  renderer.render(h(ProblemDetail, { catalog, slug: 'test', title: 'Test' }), root);
  await settle();
  for (const name of ['binary', 'large']) {
    await click(button(root, `${name} 미리보기 열기`)); await settle();
  }
  expect(text(root)).toContain('텍스트로 표시할 수 없는 자료');
  expect(text(root)).toContain('큰 자료는 오른쪽 터미널');
  expect(catalog.download).toHaveBeenCalledOnce();
  renderer.render(null, root);
});

it('restores an existing service on mount and refreshes endpoints after stop/start', async () => {
  let state: RunStatus = { slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] };
  const player: Player = {
    status: vi.fn(async () => state),
    run: vi.fn(async () => { state = { ...state, state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:9000' }] }; return { ...state, fileCount: 0 }; }),
    stop: vi.fn(async () => { state = { ...state, state: 'stopped', endpoints: [] }; }),
    submit: vi.fn(async () => ({ slug: 'test', accepted: false })),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'service' }), root);
  await settle();
  expect(text(root)).toContain('실행 중');
  expect(flatten(root).find(item => item.type === 'a')?.props.href).toBe('http://127.0.0.1:8000');
  expect(button(root, '문제 실행').props.disabled).toBe(true);
  await click(button(root, '문제 중지')); await settle();
  expect(text(root)).toContain('문제를 실행하면');
  expect(flatten(root).some(item => item.type === 'a')).toBe(false);
  await click(button(root, '문제 실행')); await settle();
  expect(flatten(root).find(item => item.type === 'a')?.props.href).toBe('http://127.0.0.1:9000');
  expect(player.status).toHaveBeenCalledTimes(3);
  renderer.render(null, root);
});

it('recovers a recorded service after a start conflict and marks unavailable resources for cleanup', async () => {
  const player: Player = {
    status: vi.fn().mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'stopped', endpoints: [] })
      .mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'running', endpoints: [{ name: 'web', url: 'http://127.0.0.1:8000' }] })
      .mockResolvedValueOnce({ slug: 'test', kind: 'service', state: 'unavailable', endpoints: [] }),
    run: vi.fn().mockRejectedValue(new Error('already running')), stop: vi.fn(), submit: vi.fn(),
  };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'service' }), root);
  await settle();
  await click(button(root, '문제 실행')); await settle();
  expect(flatten(root).find(item => item.type === 'a')?.props.href).toBe('http://127.0.0.1:8000');
  await click(button(root, '실행 상태 새로고침')); await settle();
  expect(text(root)).toContain('중지해 정리한 뒤 다시 실행하세요.');
  expect(button(root, '문제 실행').props.disabled).toBe(true);
  expect(button(root, '문제 중지').props.disabled).toBe(false);
  renderer.render(null, root);
});

it('shows file problems without service controls', async () => {
  const player: Player = { status: vi.fn(async () => ({ slug: 'test', kind: 'file', state: 'ready', endpoints: [] })), run: vi.fn(), stop: vi.fn(), submit: vi.fn() };
  const root = node('root');
  renderer.render(h(PlayPanel, { player, slug: 'test', kind: 'file' }), root);
  await settle();
  expect(text(root)).toContain('준비됨');
  expect(button(root, '문제 실행')).toBeUndefined();
  expect(player.run).not.toHaveBeenCalled();
  renderer.render(null, root);
});

it('clears stale endpoints on failed refresh and ignores a response after unmount', async () => {
  let resolve: (value: RunStatus) => void = () => {};
  const player: Player = {
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
  expect(button(root, '문제 중지').props.disabled).toBe(false);
  expect(button(root, '문제 실행').props.disabled).toBe(true);
  const pending = click(button(root, '실행 상태 새로고침'));
  renderer.render(null, root);
  resolve({ slug: 'test', kind: 'service', state: 'running', endpoints: [] });
  await pending; await settle();
  expect(root.children).toHaveLength(0);
});

it('loads a description as text and downloads only the selected declared file', async () => {
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
  renderer.render(h(ProblemDetail, { catalog, slug: 'test', title: 'Test' }), root);
  await settle();
  expect(text(root)).toContain('<script>alert(1)</script>');
  expect(flatten(root).some(item => item.type === 'script')).toBe(false);
  await click(button(root, '다운로드')); await settle();
  expect(catalog.download).toHaveBeenCalledWith('test', id);
  expect(link.download).toBe('data.bin');
  expect(link.click).toHaveBeenCalledOnce();
  expect(new Uint8Array(await create.mock.calls[0]![0].arrayBuffer())).toEqual(new Uint8Array([0, 1, 255]));
  expect(button(root, 'files/data.bin 미리보기 열기').props['aria-expanded']).toBe(false);
  expect(catalog.download).toHaveBeenCalledOnce();
  renderer.render(null, root);
});

it('keeps file actions in the header while preview loading fails, retries and completes', async () => {
  const id = 'e'.repeat(64);
  let resolve: (bytes: Uint8Array) => void = () => {};
  const catalog: Catalog = {
    list: vi.fn(), guidance: vi.fn(),
    detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: '', files: [{ id, name: 'files/checker.py', size: 883 }], hintCount: 0, walkthrough: false })),
    download: vi.fn().mockRejectedValueOnce(new Error('offline')).mockImplementationOnce(() => new Promise<Uint8Array>(done => { resolve = done; })),
  };
  const root = node('root');
  renderer.render(h(ProblemDetail, { catalog, slug: 'test' }), root);
  await settle();
  const action = button(root, 'files/checker.py 다운로드');
  const header = action.parent!;
  expect(text(header)).toContain('checker.py');
  expect(text(header)).toContain('883 B');
  await click(button(root, 'files/checker.py 미리보기 열기')); await settle();
  expect(text(root)).toContain('자료를 불러오지 못했습니다.');
  const preview = flatten(root).find(item => item.props.id === button(root, 'files/checker.py 미리보기 닫기').props['aria-controls'])!;
  expect(preview.parent).toBe(header.parent);
  expect(preview.props.hidden).toBe(false);
  void click(button(root, '자료 다시 불러오기')); await settle();
  expect(text(root)).toContain('자료를 불러오는 중');
  expect(button(root, 'files/checker.py 다운로드').parent).toBe(header);
  await click(button(root, 'files/checker.py 미리보기 닫기')); await settle();
  resolve(new TextEncoder().encode('print("ready")')); await settle();
  expect(text(root)).not.toContain('print("ready")');
  await click(button(root, 'files/checker.py 미리보기 열기')); await settle();
  expect(text(root)).toContain('print("ready")');
  expect(button(root, 'files/checker.py 다운로드').parent).toBe(header);
  expect(catalog.download).toHaveBeenCalledTimes(2);
  renderer.render(null, root);
});
