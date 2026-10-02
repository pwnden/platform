<script setup lang="ts">
import { computed } from 'vue';
import UIIcon from './UIIcon.vue';
import { SelectRoot, SelectTrigger, SelectValue, SelectPortal, SelectContent, SelectViewport, SelectItem, SelectItemText, SelectItemIndicator } from '@sectile/vue/select';
import type { UISelectProps } from './props';
const props = defineProps<UISelectProps>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
function optionID(value: string) { return `option:${value}`; }
const values = computed(() => props.options.map(option => optionID(option.value)));
function textValue(id: string) { return props.options.find(option => optionID(option.value) === id)?.label ?? ''; }
function select(id: string | null) {
  const option = props.options.find(option => optionID(option.value) === id);
  if (option) emit('update:modelValue', option.value);
}
</script>

<template>
  <SelectRoot class="ui-field" :items="values" :model-value="optionID(modelValue)" :disabled="disabled" :label="label" :text-value="textValue" strategy="fixed" hide-when-detached @update:model-value="select">
    <label :for="id">{{ label }}</label>
    <SelectTrigger :id="id" class="ui-input ui-select">
      <SelectValue class="ui-select-value" />
      <UIIcon name="chevron-down" />
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="ui-select-content ui-radius-outer">
        <SelectViewport class="ui-select-viewport">
          <SelectItem v-for="option in options" :key="option.value" :value="optionID(option.value)" class="ui-select-item ui-radius-inner">
            <SelectItemText class="ui-select-item-text">{{ option.label }}</SelectItemText>
            <span class="ui-select-indicator" aria-hidden="true"><SelectItemIndicator><UIIcon name="check" /></SelectItemIndicator></span>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>

<style scoped>
.ui-select { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: var(--ui-space-1); text-align: start; cursor: pointer; }
.ui-select-value { min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.ui-select[data-state='open'] { border-color: var(--ui-accent); }
.ui-select-content { --ui-radius-outer: var(--ui-radius-surface); --ui-select-inset: calc(var(--ui-space-1) / 2); --ui-radius-inset: calc(var(--ui-select-inset) + 1px); z-index: 10; width: min(var(--sectile-position-anchor-width), var(--sectile-position-available-width)); padding: var(--ui-select-inset); border: 1px solid var(--ui-border-active); background: var(--ui-surface-raised); color: var(--ui-foreground); }
.ui-select-viewport { max-height: min(20rem, calc(var(--sectile-position-available-height) - 2 * var(--ui-radius-inset))); overflow-y: auto; overscroll-behavior: contain; }
.ui-select-item { display: grid; grid-template-columns: minmax(0, 1fr) 1.4rem; align-items: center; gap: var(--ui-space-1); min-height: var(--ui-control-size); padding: var(--ui-inset-control); cursor: pointer; font-size: 1rem; line-height: 1.4rem; transition: background-color var(--ui-state-duration) var(--ui-state-easing), color var(--ui-state-duration) var(--ui-state-easing); }
.ui-select-item-text { min-width: 0; overflow-wrap: anywhere; }
.ui-select-item[data-selected] { color: var(--ui-accent); background: var(--ui-accent-surface); }
.ui-select-item[data-highlighted] { background: var(--ui-surface-hover); }
@media (hover: hover) { .ui-select-item:hover { background: var(--ui-surface-hover); } }
.ui-select-item:active { background: var(--ui-surface-pressed); }
.ui-select-indicator { display: flex; align-items: center; justify-content: center; width: 1.4rem; height: 1.4rem; }
svg { display: block; width: 1.4rem; height: 1.4rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; pointer-events: none; }
</style>
