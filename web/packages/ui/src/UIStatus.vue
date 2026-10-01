<script setup lang="ts">
import type { UIStatusProps } from './props';
withDefaults(defineProps<UIStatusProps>(), { tone: 'muted' });
</script>

<template>
  <span class="ui-status" :class="`ui-status--${tone}`"><slot /></span>
</template>

<style scoped>
.ui-status { display: inline-flex; align-items: center; gap: 0.5rem; color: var(--ui-muted); font-size: 0.8rem; white-space: nowrap; }
.ui-status--info { color: var(--ui-accent); }
.ui-status--success { color: var(--ui-success); }
.ui-status--danger { color: var(--ui-danger); }
</style>
