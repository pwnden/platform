<script setup lang="ts">
import { computed } from 'vue';
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from '@sectile/vue/tabs';
import type { UITabsProps } from './props';
const props = defineProps<UITabsProps>();
defineEmits<{ 'update:modelValue': [value: string] }>();
const values = computed(() => props.items.map(item => item.value));
</script>

<template>
  <TabsRoot class="ui-tabs" :items="values" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <TabsList class="ui-tabs-list" :label="label">
      <TabsTrigger v-for="item in items" :key="item.value" class="ui-button ui-button--compact ui-tab" :value="item.value">{{ item.label }}</TabsTrigger>
    </TabsList>
    <TabsContent v-for="item in items" :key="item.value" class="ui-tab-panel" :value="item.value" force-present>
      <slot :name="item.value" :selected="modelValue === item.value" />
    </TabsContent>
  </TabsRoot>
</template>

<style scoped>
.ui-tabs { display: flex; flex-direction: column; min-width: 0; min-height: 0; height: 100%; background: var(--ui-surface); }
.ui-tabs-list { display: flex; flex: none; align-items: center; gap: var(--ui-space-1); min-height: var(--ui-workspace-header-size); padding: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); overflow-x: auto; }
.ui-tab { color: var(--ui-muted); background: transparent; border-color: transparent; }
.ui-tab[aria-selected='true'] { color: var(--ui-accent); background: var(--ui-accent-surface); border-color: var(--ui-border-active); }
.ui-tab-panel { flex: 1; min-width: 0; min-height: 0; overflow: hidden; }
.ui-tab-panel[hidden], .ui-tab-panel[aria-hidden='true'] { display: none; }
</style>
