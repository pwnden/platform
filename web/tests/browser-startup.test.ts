// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { createApp, h, nextTick, ref } from 'vue';
import UIWebFrame from '../packages/ui/src/UIWebFrame.vue';
import type { UIWebFrameHandle, UIWebNavigation } from '../packages/ui/src/props';

const origin = 'http://127.0.0.1:9000';
const src = origin + '/__pwnden_browser/' + 'a'.repeat(64);
const apps: ReturnType<typeof createApp>[] = [];
beforeEach(() => {
  // Exercise real frame identity and Vue lifecycle; protocol messages are supplied
  // by the test, so iframe documents stay local and make no network requests.
  const setAttribute = HTMLIFrameElement.prototype.setAttribute;
  vi.spyOn(HTMLIFrameElement.prototype, 'setAttribute').mockImplementation(function (name, value) {
    return setAttribute.call(this, name, name === 'src' ? 'about:blank' : value);
  });
});
afterEach(() => { for (const app of apps.splice(0)) app.unmount(); document.body.innerHTML = ''; vi.useRealTimers(); vi.restoreAllMocks(); });

function mount() {
  vi.useFakeTimers();
  const events = vi.fn();
  const handle = ref<UIWebFrameHandle>();
  const source = ref(src);
  const root = document.createElement('div'); document.body.append(root);
  const app = createApp({ render: () => h(UIWebFrame, { ref: handle, src: source.value, target: origin, title: '문제 웹', onNavigation: events }) });
  app.mount(root); apps.push(app);
  function message(frame: HTMLIFrameElement, patch = {}, sender = origin) {
    window.dispatchEvent(new MessageEvent('message', { source: frame.contentWindow, origin: sender, data: {
      type: 'pwnden.browser.state.v1', channel: new URL(source.value).pathname,
      url: origin + '/login', canBack: false, canForward: false, busy: false, error: '', ...patch,
    } }));
  }
  return { root, events, handle, source, message, frame: () => root.querySelector('iframe')! };
}

it('recovers the first failed iframe connection and stops retrying after a validated response', async () => {
  const view = mount();
  const first = view.frame();
  expect(view.root.textContent).toContain('연결 중');
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: true, error: '' }));
  first.dispatchEvent(new Event('load'));
  await vi.advanceTimersByTimeAsync(1000); await nextTick();
  expect(view.frame()).not.toBe(first);
  view.message(view.frame()); await nextTick();
  const connected = view.frame();
  expect(view.root.textContent).not.toContain('연결 중');
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ url: origin + '/login', busy: false, error: '' }));
  await vi.advanceTimersByTimeAsync(16000); await nextTick();
  expect(view.frame()).toBe(connected);
  const post = vi.spyOn(connected.contentWindow!, 'postMessage').mockImplementation(() => {});
  view.handle.value!.reload();
  expect(post).toHaveBeenCalledWith(expect.objectContaining({ action: 'reload' }), origin);
  expect(view.frame()).toBe(connected);
});

it('bounds failed startup and lets reload reconnect the frame after timeout', async () => {
  const view = mount();
  await vi.advanceTimersByTimeAsync(15000); await nextTick();
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: false, error: expect.stringContaining('새로고침') }));
  expect(vi.getTimerCount()).toBe(0);
  const failed = view.frame();
  view.handle.value!.reload(); await nextTick();
  expect(view.frame()).not.toBe(failed);
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: true, error: '' }));
  view.message(view.frame()); await nextTick();
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: false, error: '' }));
});

it('ignores old frame, wrong origin and wrong channel messages during recovery', async () => {
  const view = mount();
  const old = view.frame();
  await vi.advanceTimersByTimeAsync(1000); await nextTick();
  view.message(old);
  view.message(view.frame(), {}, 'http://evil.test');
  view.message(view.frame(), { channel: '/other' });
  view.message(view.frame(), { url: 'http://evil.test/' });
  expect(view.events).toHaveBeenCalledTimes(1);
  await vi.advanceTimersByTimeAsync(2000); await nextTick();
  view.message(view.frame()); await nextTick();
  expect(view.events).toHaveBeenCalledTimes(2);
});

it('cancels startup work on unmount and confirms the new address independently', async () => {
  const view = mount();
  const first = view.frame();
  view.message(first); await nextTick();
  view.source.value = src.replace('a'.repeat(64), 'b'.repeat(64)); await nextTick();
  expect(view.frame()).not.toBe(first);
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: true }));
  view.message(first); await nextTick();
  expect((view.events.mock.lastCall![0] as UIWebNavigation).busy).toBe(true);
  apps.pop()!.unmount();
  await vi.advanceTimersByTimeAsync(16000);
  expect(vi.getTimerCount()).toBe(0);
});

it('ignores a late reply from the frame being replaced before Vue updates its ref', async () => {
  const view = mount();
  const retired = view.frame();
  vi.advanceTimersByTime(1000);
  view.message(retired);
  expect(view.events).toHaveBeenCalledTimes(1);
  await nextTick();
  view.message(view.frame());
  expect(view.events).toHaveBeenLastCalledWith(expect.objectContaining({ busy: false }));
  const connected = view.frame();
  await vi.advanceTimersByTimeAsync(16000); await nextTick();
  expect(view.frame()).toBe(connected);
  expect(view.events).toHaveBeenCalledTimes(2);
  expect(vi.getTimerCount()).toBe(0);
});
