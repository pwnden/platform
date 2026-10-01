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
    <summary class="ui-disclosure"><span class="ui-disclosure-label"><svg class="ui-disclosure-chevron" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m9 5 7 7-7 7" /></svg><span>{{ label }}</span></span></summary>
    <div v-if="modelValue" class="ui-reveal-content"><slot /></div>
  </details>
</template>

<style scoped>
.ui-reveal { min-width: 0; border-block-start: 1px solid var(--ui-border); }
.ui-reveal-content { min-width: 0; padding: var(--ui-space-1) 0 var(--ui-space-2); }
</style>
