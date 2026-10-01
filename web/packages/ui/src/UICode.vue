<script setup lang="ts">
import { h, onServerPrefetch, onUnmounted, shallowRef, watch } from 'vue';
import type { UICodeProps } from './props';
import { highlightCode, type CodeLines } from './syntax';
const props = defineProps<UICodeProps>();
const lines = shallowRef<CodeLines>();
let revision = 0;
let rendering: Promise<void>;
async function highlight() {
  const current = ++revision;
  lines.value = undefined;
  try {
    const result = await highlightCode(props.source, props.language, props.label);
    if (current === revision) lines.value = result;
  } catch { /* Escaped source remains available when highlighting cannot load. */ }
}
watch(() => [props.source, props.language, props.label], () => { rendering = highlight(); }, { immediate: true });
onServerPrefetch(() => rendering);
onUnmounted(() => { revision++; });
const CodeContent = () => lines.value
  ? lines.value.flatMap((line, index) => [...line.map(token => h('span', { class: token.className }, token.content)), index < lines.value!.length - 1 ? '\n' : ''])
  : props.source;
</script>

<template>
  <pre class="ui-code" tabindex="0" :aria-label="label"><code><CodeContent /></code></pre>
</template>

<style scoped>
.ui-code { max-height: 32rem; overflow: auto; margin: 0; padding: var(--ui-space-2); background: var(--ui-background); border: 1px solid var(--ui-border); color: var(--ui-foreground); font-family: var(--ui-font-mono); font-size: 0.82rem; line-height: 1.65; white-space: pre; tab-size: 4; }
.ui-code :deep(.ui-syntax--text) { color: var(--ui-foreground); }
.ui-code :deep(.ui-syntax--comment) { color: var(--ui-muted); }
.ui-code :deep(.ui-syntax--keyword) { color: var(--ui-syntax-keyword); }
.ui-code :deep(.ui-syntax--string) { color: var(--ui-success); }
.ui-code :deep(.ui-syntax--number) { color: var(--ui-warning); }
.ui-code :deep(.ui-syntax--function) { color: var(--ui-syntax-function); }
.ui-code :deep(.ui-syntax--type) { color: var(--ui-accent); }
</style>
