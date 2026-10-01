<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from '@sectile/vue/switch';
import type { UISwitchProps } from './props';
defineOptions({ inheritAttrs: false });
defineProps<UISwitchProps>();
defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <SwitchRoot v-bind="$attrs" class="ui-switch" :model-value="modelValue" :disabled="disabled" :aria-label="label" :aria-busy="busy || undefined" @update:model-value="$emit('update:modelValue', $event)">
    <span class="ui-switch-track" aria-hidden="true"><SwitchThumb class="ui-switch-thumb" /></span>
    <span class="ui-switch-label"><slot>{{ label }}</slot></span>
  </SwitchRoot>
</template>

<style scoped>
.ui-switch { display: inline-flex; align-items: center; gap: var(--ui-space-1); min-height: 2.5rem; padding: var(--ui-space-1); border: 0; border-radius: var(--ui-radius); background: transparent; color: var(--ui-muted); cursor: pointer; font-size: 0.85rem; white-space: nowrap; transition: color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-switch-track { display: block; flex: none; width: 3rem; height: 1.5rem; padding: 0.125rem; border: 1px solid var(--ui-border-active); background: var(--ui-background); transition: border-color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-switch-thumb { display: block; width: calc(1.25rem - 2px); height: calc(1.25rem - 2px); background: var(--ui-muted); transition: transform var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-switch[aria-checked='true'] { color: var(--ui-accent); }
.ui-switch[aria-checked='true'] .ui-switch-track { background: var(--ui-accent-surface); border-color: var(--ui-accent); }
.ui-switch[aria-checked='true'] .ui-switch-thumb { transform: translateX(1.5rem); background: var(--ui-accent); }
.ui-switch[aria-busy='true'] { color: var(--ui-warning); }
.ui-switch[aria-busy='true'] .ui-switch-track { border-color: var(--ui-warning); }
.ui-switch[aria-busy='true'] .ui-switch-thumb { background: var(--ui-warning); }
.ui-switch--error { color: var(--ui-danger); }
.ui-switch--error .ui-switch-track { border-color: var(--ui-danger); }
@media (hover: hover) { .ui-switch:hover:enabled { background: var(--ui-surface-hover); } .ui-switch:hover:enabled .ui-switch-track { border-color: var(--ui-accent); } }
.ui-switch:active:enabled { background: var(--ui-surface-pressed); }
.ui-switch:disabled { opacity: 0.5; cursor: not-allowed; }
@media (forced-colors: active) { .ui-switch-track { border-color: ButtonText; } .ui-switch-thumb { background: ButtonText; } .ui-switch[aria-checked='true'] .ui-switch-thumb { background: Highlight; } }
</style>
