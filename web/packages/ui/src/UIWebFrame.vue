<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue';
import type { UIWebFrameProps, UIWebFrameHandle, UIWebNavigation } from './props';
import { browserState } from './browser-channel';
const props = defineProps<UIWebFrameProps>();
const emit = defineEmits<{ navigation: [state: UIWebNavigation] }>();
const frame = ref<HTMLIFrameElement>();
const generation = ref(0);
const starting = ref(!!props.target);
const retryDelays = [1000, 2000, 4000, 8000] as const;
let timer: ReturnType<typeof setTimeout> | undefined;
let mounted = false;
let connected = false;
let retired: Window | null | undefined;
let last = '';
function clearRetry() { clearTimeout(timer); timer = undefined; }
function replaceFrame() { retired = frame.value?.contentWindow; generation.value++; }
function reportStartup(error = '') {
  if (props.target) emit('navigation', { url: props.target + '/', canBack: false, canForward: false, busy: starting.value, error });
}
function awaitConnection(attempt: number) {
  timer = setTimeout(() => {
    if (!mounted || connected) return;
    if (attempt === retryDelays.length - 1) {
      starting.value = false;
      reportStartup('문제 웹에 연결하지 못했습니다. 새로고침으로 다시 연결하세요.');
      return;
    }
    replaceFrame();
    awaitConnection(attempt + 1);
  }, retryDelays[attempt]);
}
function start() {
  clearRetry(); connected = false; last = '';
  starting.value = !!props.target;
  if (props.target) { reportStartup(); awaitConnection(0); }
}
function receive(event: MessageEvent) {
  if (event.source === retired || event.source !== frame.value?.contentWindow || event.origin !== new URL(props.src).origin || !props.target) return;
  const state = browserState(event.data, props.src, props.target);
  if (!state) return;
  connected = true; starting.value = false; clearRetry();
  if (JSON.stringify(state) === last) return;
  last = JSON.stringify(state);
  emit('navigation', state);
}
function send(action: string, url?: string) {
  frame.value?.contentWindow?.postMessage({ type: 'pwnden.browser.command.v1', channel: new URL(props.src).pathname, action, url }, new URL(props.src).origin);
}
onMounted(() => { mounted = true; window.addEventListener('message', receive); start(); });
onBeforeUnmount(() => { mounted = false; clearRetry(); window.removeEventListener('message', receive); });
watch([() => props.src, () => props.target], () => { if (mounted) { replaceFrame(); start(); } });
function reload() {
  if (connected) send('reload');
  else { replaceFrame(); start(); }
}
const handle: UIWebFrameHandle = { back: () => send('back'), forward: () => send('forward'), reload, navigate: url => send('navigate', url) };
defineExpose(handle);
</script>

<template>
  <div class="ui-web-frame" :aria-busy="starting">
    <iframe :key="generation" ref="frame" :src="src" :title="title" :inert="starting" :aria-hidden="starting || undefined" referrerpolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-forms allow-modals allow-popups allow-downloads" />
    <p v-if="starting" class="ui-web-connecting" role="status">문제 웹 연결 중…</p>
  </div>
</template>

<style scoped>
.ui-web-frame { position: relative; display: block; width: 100%; height: 100%; min-height: 0; overflow: hidden; border-radius: var(--ui-radius-flush); background: var(--ui-surface); }
iframe { display: block; width: 100%; height: 100%; border: 0; background: var(--ui-surface); color-scheme: dark; }
.ui-web-connecting { position: absolute; inset: 0; margin: 0; display: grid; place-content: center; padding: var(--ui-space-2); color: var(--ui-muted); background: var(--ui-surface); }
</style>
