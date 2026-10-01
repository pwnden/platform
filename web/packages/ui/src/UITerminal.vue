<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { terminalDocument } from './terminal-document';
import { copyTerminalSelection, terminalClipboardKeys, terminalPlatform } from './terminal-clipboard';
import type { UITerminalProps, UITerminalHandle } from './props';

const props = defineProps<UITerminalProps>();
const emit = defineEmits<{ input: [data: Uint8Array]; resize: [size: { cols: number; rows: number }] }>();
const container = ref<HTMLElement>();
const clipboardError = ref('');
let terminal: Terminal | undefined;
let observer: ResizeObserver | undefined;
let alive = true;
let clipboardRevision = 0;
const handle: UITerminalHandle = {
  write(data, rendered) { if (alive && terminal) terminal.write(data, rendered); },
  clear() { clipboardRevision++; clipboardError.value = ''; terminal?.reset(); },
  focus() { terminal?.focus(); },
};
defineExpose(handle);
onMounted(() => {
  if (!container.value) return;
  const style = getComputedStyle(document.documentElement);
  const color = (token: string) => style.getPropertyValue(token).trim();
  terminal = new Terminal({
    documentOverride: terminalDocument(document),
    disableStdin: !props.enabled, cursorBlink: props.enabled, screenReaderMode: true,
    scrollback: 2000, fontSize: 14, lineHeight: 1.4,
    fontFamily: color('--ui-font-mono'),
    theme: {
      background: color('--ui-terminal-background'), foreground: color('--ui-foreground'),
      cursor: color('--ui-accent'), selectionBackground: color('--ui-terminal-selection'),
      black: color('--ui-terminal-background'), brightBlack: color('--ui-muted'),
      red: color('--ui-danger'), brightRed: color('--ui-danger'),
      green: color('--ui-success'), brightGreen: color('--ui-success'),
      yellow: color('--ui-warning'), brightYellow: color('--ui-warning'),
      blue: color('--ui-accent'), brightBlue: color('--ui-accent'),
      magenta: '#c5acff', brightMagenta: '#c5acff',
      cyan: '#8cdce6', brightCyan: '#8cdce6',
      white: color('--ui-foreground'), brightWhite: '#f1f6ff',
    },
  });
  const fit = new FitAddon();
  terminal.loadAddon(fit);
  terminal.open(container.value);
  const current = terminal;
  terminal.attachCustomKeyEventHandler(terminalClipboardKeys(terminalPlatform(navigator), terminal, () => {
    const revision = ++clipboardRevision;
    clipboardError.value = '';
    void copyTerminalSelection(document, navigator.clipboard, current.getSelection()).then(copied => {
      if (alive && terminal === current && revision === clipboardRevision && !copied) clipboardError.value = '복사하지 못했습니다. 터미널에서 우클릭한 뒤 복사를 선택해 주세요.';
    });
  }));
  terminal.onData(data => { if (props.enabled) emit('input', new TextEncoder().encode(data)); });
  terminal.onBinary(data => { if (props.enabled) emit('input', Uint8Array.from(data, char => char.charCodeAt(0))); });
  terminal.onResize(({ cols, rows }) => emit('resize', { cols, rows }));
  const resize = () => {
    if (!alive || !container.value?.clientWidth) return;
    const size = fit.proposeDimensions();
    if (size) terminal?.resize(Math.max(2, Math.min(500, size.cols)), Math.max(1, Math.min(200, size.rows)));
  };
  observer = new ResizeObserver(resize);
  observer.observe(container.value);
  resize();
  void document.fonts.ready.then(() => { if (alive) resize(); });
});
watch(() => props.enabled, enabled => {
  if (terminal) { terminal.options.disableStdin = !enabled; terminal.options.cursorBlink = enabled; }
});
onUnmounted(() => { alive = false; observer?.disconnect(); terminal?.dispose(); terminal = undefined; });
</script>

<template>
  <div class="ui-terminal">
    <div ref="container" class="ui-terminal-screen" :aria-label="label" />
    <p v-if="clipboardError" class="ui-terminal-message" role="status">{{ clipboardError }}</p>
  </div>
</template>

<style scoped>
.ui-terminal { display: flex; flex-direction: column; min-width: 0; width: 100%; min-height: 18rem; height: 100%; padding: var(--ui-space-2); background: var(--ui-terminal-background); overflow: hidden; }
.ui-terminal-screen { flex: 1; min-width: 0; min-height: 0; }
.ui-terminal-message { margin: var(--ui-space-2) 0 0; color: var(--ui-danger); font-family: var(--ui-font-body); }
</style>
