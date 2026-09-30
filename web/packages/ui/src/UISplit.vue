<script setup lang="ts">
import { WindowSplitterRoot, WindowSplitterPane, WindowSplitterHandle } from '@sectile/vue';
import type { UISplitProps } from './props';

withDefaults(defineProps<UISplitProps>(), { min: 15, max: 85 });
const emit = defineEmits<{ 'update:modelValue': [value: number] }>();
</script>

<template>
  <WindowSplitterRoot class="ui-split" :model-value="modelValue" :min="min" :max="max" :label="label" :format-value="value => `${value}%`" @update:model-value="emit('update:modelValue', Number($event))">
    <WindowSplitterPane side="before" class="ui-split-pane"><slot name="before" /></WindowSplitterPane>
    <WindowSplitterHandle class="ui-split-handle" :title="`${label} — 드래그 또는 방향키로 조절`" />
    <WindowSplitterPane side="after" class="ui-split-pane"><slot name="after" /></WindowSplitterPane>
  </WindowSplitterRoot>
</template>

<style scoped>
.ui-split { display: flex; min-width: 0; min-height: 0; width: 100%; height: 100%; }
.ui-split-pane { min-width: 0; min-height: 0; overflow: hidden; }
.ui-split-pane[data-side='before'] { flex-shrink: 0; }
.ui-split-pane[data-side='after'] { flex: 1; }
.ui-split-handle { position: absolute; z-index: 1; inset-block: 0; left: var(--sectile-window-splitter-percentage); width: 12px; transform: translateX(-50%); cursor: col-resize; touch-action: none; }
.ui-split-handle::after { content: ''; position: absolute; inset-block: 0; left: 50%; width: 1px; background: var(--ui-border); }
.ui-split-handle:hover::after, .ui-split-handle:focus-visible::after, .ui-split-handle[data-dragging]::after { width: 2px; background: var(--ui-accent); box-shadow: var(--ui-glow); }
.ui-split-handle:focus-visible { outline-offset: -2px; }
@media (max-width: 48rem) {
  .ui-split { flex-direction: column; height: auto; }
  .ui-split-pane { flex-basis: auto !important; overflow: visible; }
  .ui-split-handle { display: none; }
}
</style>
