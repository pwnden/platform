<script setup lang="ts">
import { computed } from 'vue';
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@sectile/vue/tabs';
import type { UITabsProps } from './props';
import UIIcon from './UIIcon.vue';
const props = defineProps<UITabsProps>();
defineEmits<{ 'update:modelValue': [value: string] }>();
const values = computed(() => props.items.map(item => item.value));
</script>

<template>
  <TabsRoot class="ui-tabs" :items="values" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <div class="ui-tabs-heading">
      <TabsList class="ui-tabs-list" :label="label">
        <TabsTrigger v-for="item in items" :key="item.value" class="ui-tab" :value="item.value"><span class="ui-tab-label"><UIIcon v-if="item.icon" :name="item.icon" /><span>{{ item.label }}</span></span></TabsTrigger>
      </TabsList>
      <div v-if="$slots.actions" class="ui-tabs-actions"><slot name="actions" /></div>
    </div>
    <TabsContent v-for="item in items" :key="item.value" class="ui-tab-panel" :value="item.value" force-present>
      <slot :name="item.value" :selected="modelValue === item.value" />
    </TabsContent>
  </TabsRoot>
</template>

<style scoped>
.ui-tabs { display: flex; flex-direction: column; min-width: 0; min-height: 0; height: 100%; background: var(--ui-surface); }
.ui-tabs-heading { display: flex; flex: none; align-items: center; min-width: 0; min-height: var(--ui-workspace-header-size); border-bottom: 1px solid var(--ui-border); }
.ui-tabs-list { display: flex; flex: 1; min-width: 0; align-items: stretch; padding: 0; overflow-x: auto; }
.ui-tabs-actions { display: flex; flex: none; align-items: center; gap: var(--ui-space-1); padding: var(--ui-space-2); }
.ui-tab { position: relative; display: inline-flex; flex: none; align-items: center; justify-content: center; height: calc(var(--ui-workspace-header-size) - 1px); padding: var(--ui-space-2); border: 0; border-radius: var(--ui-radius-flush); color: var(--ui-muted); background: transparent; font-size: 0.85rem; line-height: 1.4; white-space: nowrap; cursor: pointer; transition: color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.ui-tab::after { content: ''; position: absolute; inset-inline: 0; bottom: 0; height: var(--ui-focus-width); background: transparent; }
.ui-tab[aria-selected='true'] { color: var(--ui-accent); }
.ui-tab[aria-selected='true']::after { background: var(--ui-accent); }
.ui-tab:focus-visible { outline: none; }
.ui-tab-label { display: inline-flex; align-items: center; gap: var(--ui-space-1); border-radius: var(--ui-radius-inline); }
.ui-tab:focus-visible .ui-tab-label { outline: var(--ui-focus-width) solid var(--ui-focus-color); outline-offset: 3px; }
@media (hover: hover) { .ui-tab:hover:enabled { color: var(--ui-foreground); background: var(--ui-surface-raised); } }
.ui-tab:active:enabled { background: var(--ui-surface-pressed); }
.ui-tab:disabled { opacity: 0.5; cursor: not-allowed; }
.ui-tab-panel { flex: 1; min-width: 0; min-height: 0; overflow: hidden; }
.ui-tab-panel[hidden], .ui-tab-panel[aria-hidden='true'] { display: none; }
</style>
