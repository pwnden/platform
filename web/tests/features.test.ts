import { afterEach, expect, it, vi } from 'vitest';
import { createRenderer, h, nextTick } from 'vue';
import type { Catalog } from '../domains/catalog/src/index';
import type { Player, RunStatus } from '../domains/play/src/index';
import ProblemDetail from '../features/catalog/src/ProblemDetail.vue';
import PlayPanel from '../features/play/src/PlayPanel.vue';
import TerminalPanel from '../features/terminal/src/TerminalPanel.vue';

// Exercise feature lifecycle and async handlers; UI/Sectile have their own tests.
vi.mock('../packages/ui/src/index.ts', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    UIPanel: (await import('../packages/ui/src/UIPanel.vue')).default,
    UIStatus: (await import('../packages/ui/src/UIStatus.vue')).default,
    UIMarkdown: defineComponent({ props: ['source'], setup: props => () => h('markdown', props.source) }),
    UIButton: defineComponent({ setup: (_, { attrs, slots }) => () => h('button', attrs, slots.default?.()) }),
    UITextField: defineComponent({ setup: (_, { attrs }) => () => h('input', attrs) }),
    UITerminal: defineComponent({ setup: (_, { attrs, expose }) => {
      expose({ write: (_data: Uint8Array, rendered: () => void) => rendered(), clear: () => {}, focus: () => {} });
      return () => h('terminal', attrs);
    } }),
  };
});

it('owns pending and ready terminal connections across cancellation and unmount', async () => {
  const close = vi.fn();
  let resolve: (value: unknown) => void = () => {};
  const session = { input: vi.fn(), resize: vi.fn(), close: vi.fn() };
  const terminals = { connect: vi.fn(() => ({ ready: new Promise<any>(done => { resolve = done; }), close })) };
  const root = node('root');
  renderer.render(h(TerminalPanel, { terminals, slug: 'test', enabled: true, busy: false }), root);
  const pending = click(button(root, '터미널 연결')); await settle();
  expect(text(root)).toContain('연결하는 중');
  await click(button(root, '연결 종료')); resolve(session); await pending; await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(session.close).toHaveBeenCalledOnce();
  expect(text(root)).not.toContain('연결됨');
  const next = click(button(root, '터미널 연결')); resolve(session); await next; await settle();
  expect(text(root)).toContain('연결됨');
  const terminal = flatten(root).find(item => item.type === 'terminal')!;
  (terminal.props.onInput as (data: Uint8Array) => void)(new Uint8Array([3]));
  expect(session.input).toHaveBeenCalledWith(new Uint8Array([3]));
  renderer.render(null, root);
  expect(close).toHaveBeenCalledTimes(2);
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
const button = (root: Node, label: string) => flatten(root).find(item => item.type === 'button' && text(item) === label)!;
const click = (item: Node) => (item.props.onClick as () => unknown)();
async function settle() { for (let i = 0; i < 8; i++) { await Promise.resolve(); await nextTick(); } }
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

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
  expect(text(root)).toContain('중지됨');
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
  expect(text(root)).toContain('서비스 실행 없이');
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
    list: vi.fn(), detail: vi.fn(async () => ({ slug: 'test', title: 'Test', category: 'rev', kind: 'file', description: '<script>alert(1)</script>', files: [{ id, name: 'files/data.bin', size: 3 }] })),
    download: vi.fn(async () => new Uint8Array([0, 1, 255])),
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
  renderer.render(null, root);
});
