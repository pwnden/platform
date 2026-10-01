<script setup lang="ts">
import type { UIPanelProps } from './props';
withDefaults(defineProps<UIPanelProps>(), { headingLevel: 2 });
</script>

<template>
  <section class="ui-panel" :aria-labelledby="headingID">
    <header class="ui-panel-heading">
      <component :is="`h${headingLevel}`" :id="headingID">{{ title }}</component>
      <div v-if="$slots.actions" class="ui-panel-actions"><slot name="actions" /></div>
    </header>
    <div class="ui-panel-body"><slot /></div>
  </section>
</template>

<style scoped>
.ui-panel { min-width: 0; min-height: 0; display: flex; flex-direction: column; background: var(--ui-surface); }
.ui-panel-heading { display: flex; flex: none; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: var(--ui-space-1); min-height: 3.5rem; padding: 0.8rem var(--ui-space-2); border-bottom: 1px solid var(--ui-border); }
h2, h3 { font-size: 1rem; font-weight: 600; word-break: keep-all; }
.ui-panel-actions { display: flex; align-items: center; gap: var(--ui-space-1); }
.ui-panel-body { min-width: 0; min-height: 0; padding: var(--ui-space-3); flex: 1; }
@media (max-width: 48rem) { .ui-panel-heading { padding-inline: var(--ui-space-2); } .ui-panel-body { padding: var(--ui-space-2); } }
</style>
