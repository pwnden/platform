<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from '@sectile/vue/switch';
import type { UISwitchProps } from './props';
defineOptions({ inheritAttrs: false });
defineProps<UISwitchProps>();
defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <SwitchRoot v-bind="$attrs" class="ui-switch ui-radius-outer" :class="{ 'ui-switch--compact': compact }" :model-value="modelValue" :disabled="disabled" :aria-label="label" :aria-busy="busy || undefined" @update:model-value="$emit('update:modelValue', $event)">
    <span class="ui-switch-track ui-radius-outer" aria-hidden="true"><SwitchThumb class="ui-switch-thumb ui-radius-inner"><slot name="thumb" :checked="modelValue" :busy="busy" /></SwitchThumb></span>
    <span class="ui-switch-label" :class="{ 'ui-sr-only': compact }"><slot>{{ label }}</slot></span>
  </SwitchRoot>
</template>

<style scoped>
.ui-switch { --ui-radius-outer: var(--ui-radius-control); --ui-switch-thumb-size: 1.75rem; --ui-switch-padding: calc((var(--ui-control-size-compact) - var(--ui-switch-thumb-size)) / 2 - 1px); --ui-radius-inset: calc(var(--ui-switch-padding) + 1px); position: relative; display: inline-flex; align-items: center; gap: var(--ui-space-1); height: var(--ui-control-size-compact); padding: 0; border: 0; background: transparent; color: var(--ui-muted); cursor: pointer; font-size: 0.85rem; white-space: nowrap; }
.ui-switch--compact { flex: none; }
.ui-switch-track { display: flex; align-items: center; flex: none; width: calc(var(--ui-switch-thumb-size) + var(--ui-switch-thumb-size) + var(--ui-radius-inset) + var(--ui-radius-inset)); height: var(--ui-control-size-compact); padding: var(--ui-switch-padding); border: 1px solid var(--ui-border); background: var(--ui-surface-raised); transition: border-color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-switch-label { margin: 0; }
.ui-switch-thumb { display: flex; align-items: center; justify-content: center; flex: none; width: var(--ui-switch-thumb-size); height: var(--ui-switch-thumb-size); background: var(--ui-muted); color: var(--ui-background); transition: transform var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-switch[aria-checked='true'] { color: var(--ui-accent); }
.ui-switch[aria-checked='true'] .ui-switch-track { background: var(--ui-accent-surface); border-color: var(--ui-border-active); }
.ui-switch[aria-checked='true'] .ui-switch-thumb { transform: translateX(100%); background: var(--ui-accent); }
.ui-switch[aria-busy='true'] { color: var(--ui-warning); }
.ui-switch[aria-busy='true'] .ui-switch-track { border-color: var(--ui-warning); }
.ui-switch[aria-busy='true'] .ui-switch-thumb { background: var(--ui-warning); }
.ui-switch--error { color: var(--ui-danger); }
.ui-switch--error .ui-switch-track { border-color: var(--ui-danger); }
.ui-switch--error .ui-switch-thumb { background: var(--ui-danger); }
@media (hover: hover) { .ui-switch:hover:enabled .ui-switch-track { background: var(--ui-surface-hover); border-color: var(--ui-border-active); } }
.ui-switch:active:enabled .ui-switch-track { background: var(--ui-surface-pressed); border-color: var(--ui-accent); }
.ui-switch:disabled { opacity: 0.5; cursor: not-allowed; }
@media (forced-colors: active) { .ui-switch-track { border-color: ButtonText; } .ui-switch-thumb { forced-color-adjust: none; background: ButtonText; color: Canvas; } .ui-switch[aria-checked='true'] .ui-switch-thumb { background: Highlight; color: HighlightText; } }
</style>
