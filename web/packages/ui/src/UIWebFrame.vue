<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue';
import type { UIWebFrameProps, UIWebFrameHandle, UIWebNavigation } from './props';
import { browserState } from './browser-channel';
const props = defineProps<UIWebFrameProps>();
const emit = defineEmits<{ navigation: [state: UIWebNavigation] }>();
const frame = ref<HTMLIFrameElement>();
let last = '';
function receive(event: MessageEvent) {
  if (event.source !== frame.value?.contentWindow || event.origin !== new URL(props.src).origin || !props.target) return;
  const state = browserState(event.data, props.src, props.target);
  if (!state || JSON.stringify(state) === last) return;
  last = JSON.stringify(state);
  emit('navigation', state);
}
function send(action: string, url?: string) {
  frame.value?.contentWindow?.postMessage({ type: 'pwnden.browser.command.v1', channel: new URL(props.src).pathname, action, url }, new URL(props.src).origin);
}
onMounted(() => window.addEventListener('message', receive));
onBeforeUnmount(() => window.removeEventListener('message', receive));
const handle: UIWebFrameHandle = { back: () => send('back'), forward: () => send('forward'), reload: () => send('reload'), navigate: url => send('navigate', url) };
defineExpose(handle);
</script>

<template>
  <iframe ref="frame" class="ui-web-frame" :src="src" :title="title" referrerpolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-forms allow-modals allow-popups allow-downloads" />
</template>

<style scoped>
.ui-web-frame { display: block; width: 100%; height: 100%; min-height: 0; border: 0; border-radius: var(--ui-radius-flush); background: white; color-scheme: normal; }
</style>
