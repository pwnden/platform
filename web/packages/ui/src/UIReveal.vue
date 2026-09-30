<script setup lang="ts">
import type { UIRevealProps } from './props';
defineProps<UIRevealProps>();
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>();
function toggle(event: Event) {
  emit('update:modelValue', (event.currentTarget as HTMLDetailsElement).open);
}
</script>

<template>
  <details class="ui-reveal" :open="modelValue" @toggle="toggle">
    <summary>{{ label }}</summary>
    <div v-if="modelValue" class="ui-reveal-content"><slot /></div>
  </details>
</template>

<style scoped>
.ui-reveal { min-width: 0; border-block-start: 1px solid var(--ui-border); }
summary { cursor: pointer; padding-block: var(--ui-space-2); color: var(--ui-foreground); font-size: 0.9rem; word-break: keep-all; overflow-wrap: break-word; }
summary:hover { color: var(--ui-accent); }
summary:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
.ui-reveal-content { min-width: 0; padding-bottom: var(--ui-space-2); }
</style>
