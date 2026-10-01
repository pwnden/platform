<script setup lang="ts">
import { computed } from 'vue';
import type { UISelectProps } from './props';
const props = defineProps<UISelectProps>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
const selection = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) });
const selectedLabel = computed(() => props.options.find(option => option.value === props.modelValue)?.label ?? '');
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <div class="ui-input ui-select">
      <span class="ui-select-value" aria-hidden="true">{{ selectedLabel }}</span>
      <select :id="id" v-model="selection" :disabled="disabled">
        <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m6 9 6 6 6-6" /></svg>
    </div>
  </div>
</template>

<style scoped>
.ui-select { position: relative; display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: var(--ui-space-1); }
.ui-select-value { min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
select { position: absolute; inset: 0; min-width: 0; width: 100%; height: 100%; opacity: 0; padding: 0; margin: 0; border: 0; outline: none; color: inherit; font: inherit; cursor: pointer; }
select:disabled { cursor: not-allowed; }
option { color: var(--ui-foreground); background: var(--ui-surface); }
svg { width: 1.4rem; height: 1.4rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; pointer-events: none; }
</style>
