<script setup lang="ts">
import { onServerPrefetch, onUnmounted, ref, shallowRef, useId, watch } from 'vue';
import type { ComarkNode } from 'md4x/standalone';
import type { UIMarkdownProps } from './props';
import { parseMarkdown, renderMarkdownNode } from './markdown';
import UIButton from './UIButton.vue';

const props = defineProps<UIMarkdownProps>();
const nodes = shallowRef<ComarkNode[]>([]);
const pending = ref(true);
const failed = ref(false);
const prefix = `markdown-${useId()}`;
let revision = 0;
async function load() {
  const current = ++revision;
  const source = props.source;
  pending.value = true;
  failed.value = false;
  nodes.value = [];
  try {
    const result = await parseMarkdown(source);
    if (current === revision) nodes.value = result;
  } catch {
    if (current === revision) failed.value = true;
  } finally {
    if (current === revision) pending.value = false;
  }
}
let rendering: Promise<void>;
watch(() => props.source, () => { rendering = load(); }, { immediate: true });
onServerPrefetch(() => rendering);
onUnmounted(() => { revision++; });
const MarkdownContent = () => nodes.value.map(node => renderMarkdownNode(node, prefix, props.headingOffset ?? 3));
</script>

<template>
  <div class="ui-markdown" :aria-busy="pending">
    <p v-if="pending" role="status">설명을 표시하는 중…</p>
    <template v-else-if="failed">
      <p role="alert">설명 서식을 표시하지 못했습니다. 원문을 확인하거나 다시 시도하세요.</p>
      <UIButton variant="ghost" size="compact" @click="load">설명 다시 표시</UIButton>
      <pre class="markdown-source">{{ source }}</pre>
    </template>
    <MarkdownContent v-else />
  </div>
</template>

<style scoped>
.ui-markdown { min-width: 0; max-width: 72ch; font-family: var(--ui-font-body); font-size: 1rem; word-break: keep-all; overflow-wrap: break-word; line-height: 1.85; }
.ui-markdown :deep(p), .ui-markdown :deep(ul), .ui-markdown :deep(ol), .ui-markdown :deep(blockquote), .ui-markdown :deep(pre), .ui-markdown :deep(.markdown-table) { margin-block: 0 var(--ui-space-2); }
.ui-markdown :deep(h2), .ui-markdown :deep(h3), .ui-markdown :deep(h4), .ui-markdown :deep(h5), .ui-markdown :deep(h6) { margin-block: var(--ui-space-3) var(--ui-space-1); color: var(--ui-foreground); line-height: 1.5; scroll-margin-top: var(--ui-space-2); }
.ui-markdown :deep(h2), .ui-markdown :deep(h3) { font-size: 1.2rem; }
.ui-markdown :deep(h4) { font-size: 1.05rem; }
.ui-markdown :deep(h5) { font-size: 1rem; }
.ui-markdown :deep(h6) { font-size: 0.9rem; }
.ui-markdown :deep(h2:first-child), .ui-markdown :deep(h3:first-child), .ui-markdown :deep(h4:first-child), .ui-markdown :deep(h5:first-child), .ui-markdown :deep(h6:first-child) { margin-top: 0; }
.ui-markdown :deep(ul), .ui-markdown :deep(ol) { padding: var(--ui-space-2); }
.ui-markdown :deep(li > ul), .ui-markdown :deep(li > ol) { margin-block: var(--ui-space-1); }
.ui-markdown :deep(blockquote) { margin-inline: 0; padding: var(--ui-space-2); border-left: 1px solid var(--ui-border-active); color: var(--ui-muted); }
.ui-markdown :deep(.markdown-message) { padding: var(--ui-space-3); border: 0; border-radius: var(--ui-radius-surface); color: var(--ui-foreground); background: var(--ui-accent-surface); }
.ui-markdown :deep(.markdown-message-from) { margin-bottom: var(--ui-space-1); color: var(--ui-accent); font-weight: 600; }
.ui-markdown :deep(.markdown-message-body) { font-size: 1.05rem; }
.ui-markdown :deep(.markdown-section) { margin-block: var(--ui-space-3); padding: var(--ui-space-3); }
.ui-markdown :deep(.markdown-section-title) { margin-block: 0 var(--ui-space-1); font-size: 1.05rem; font-weight: 600; }
.ui-markdown :deep(.markdown-section--objective) { border-radius: var(--ui-radius-surface); background: var(--ui-surface-hover); }
.ui-markdown :deep(.markdown-section--objective > .markdown-section-title) { color: var(--ui-accent); }
.ui-markdown :deep(.markdown-section--objective > .markdown-section-body) { font-size: 1.05rem; }
.ui-markdown :deep(.markdown-section--resources) { border-radius: var(--ui-radius-surface); background: var(--ui-surface-raised); }
.ui-markdown :deep(.markdown-section--knowledge) { border-top: 1px solid var(--ui-border); }
.ui-markdown :deep(.markdown-section--submission) { display: flex; flex-wrap: wrap; align-items: baseline; gap: var(--ui-space-1) var(--ui-space-2); padding: var(--ui-space-2); border-top: 1px solid var(--ui-border); }
.ui-markdown :deep(.markdown-section--submission > .markdown-section-title) { margin: 0; color: var(--ui-muted); font-size: 0.9rem; }
.ui-markdown :deep(.markdown-section--submission > .markdown-section-body) { flex: 1 1 16rem; min-width: 0; font-size: 0.9rem; }
.ui-markdown :deep(pre) { min-width: 0; overflow: auto; padding: var(--ui-space-2); border: 1px solid var(--ui-border); border-radius: var(--ui-radius-surface); background: var(--ui-terminal-background); white-space: pre; line-height: 1.6; }
.ui-markdown :deep(code) { padding: 0.2rem; border-radius: var(--ui-radius-inline); color: var(--ui-accent); background: var(--ui-surface-raised); }
.ui-markdown :deep(pre code) { padding: 0; color: var(--ui-foreground); background: transparent; }
.ui-markdown :deep(hr) { margin-block: var(--ui-space-3); border: 0; border-top: 1px solid var(--ui-border); }
.ui-markdown :deep(mark) { color: var(--ui-foreground); background: var(--ui-accent-surface); }
.ui-markdown :deep(.markdown-table) { max-width: 100%; overflow-x: auto; }
.ui-markdown :deep(table) { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
.ui-markdown :deep(th), .ui-markdown :deep(td) { padding: var(--ui-space-1); border: 1px solid var(--ui-border); text-align: left; }
.ui-markdown :deep(th) { font-weight: 500; background: var(--ui-surface-raised); }
.ui-markdown :deep(.markdown-align-center) { text-align: center; }
.ui-markdown :deep(.markdown-align-right) { text-align: right; }
.ui-markdown :deep(.markdown-task) { list-style: none; }
.ui-markdown :deep(.markdown-task > p:first-of-type) { display: inline; }
.ui-markdown :deep(input[type='checkbox']) { margin-inline: -1.2rem var(--ui-space-1); accent-color: var(--ui-accent); }
.ui-markdown :deep(.markdown-image) { color: var(--ui-muted); }
.ui-markdown .markdown-source { white-space: pre-wrap; }
.ui-markdown :deep(:last-child) { margin-bottom: 0; }
</style>
