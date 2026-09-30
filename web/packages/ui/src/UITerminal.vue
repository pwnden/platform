<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { terminalDocument } from './terminal-document';
import type { UITerminalProps, UITerminalHandle } from './props';

const props = defineProps<UITerminalProps>();
const emit = defineEmits<{ input: [data: Uint8Array]; resize: [size: { cols: number; rows: number }] }>();
const container = ref<HTMLElement>();
let terminal: Terminal | undefined;
let observer: ResizeObserver | undefined;
let alive = true;
const handle: UITerminalHandle = {
  write(data, rendered) { if (alive && terminal) terminal.write(data, rendered); },
  clear() { terminal?.reset(); },
  focus() { terminal?.focus(); },
};
defineExpose(handle);
onMounted(() => {
  if (!container.value) return;
  terminal = new Terminal({
    documentOverride: terminalDocument(document),
    disableStdin: !props.enabled, cursorBlink: true, screenReaderMode: true,
    scrollback: 2000, fontSize: 14,
    theme: { background: '#ffffff', foreground: '#20252b', cursor: '#174b7a', selectionBackground: '#cddfed' },
  });
  const fit = new FitAddon();
  terminal.loadAddon(fit);
  terminal.open(container.value);
  terminal.onData(data => { if (props.enabled) emit('input', new TextEncoder().encode(data)); });
  terminal.onBinary(data => { if (props.enabled) emit('input', Uint8Array.from(data, char => char.charCodeAt(0))); });
  terminal.onResize(size => emit('resize', size));
  const resize = () => {
    if (!alive || !container.value?.clientWidth) return;
    const size = fit.proposeDimensions();
    if (size) terminal?.resize(Math.max(2, Math.min(500, size.cols)), Math.max(1, Math.min(200, size.rows)));
  };
  observer = new ResizeObserver(resize);
  observer.observe(container.value);
  resize();
});
watch(() => props.enabled, enabled => { if (terminal) terminal.options.disableStdin = !enabled; });
onUnmounted(() => { alive = false; observer?.disconnect(); terminal?.dispose(); terminal = undefined; });
</script>

<template>
  <div ref="container" class="ui-terminal" :aria-label="label" />
</template>

<style scoped>
.ui-terminal { min-width: 0; width: 100%; height: clamp(16rem, 40vh, 28rem); padding: var(--ui-space-1); border: 1px solid var(--ui-border); border-radius: var(--ui-radius); background: var(--ui-surface); overflow: hidden; }
</style>
