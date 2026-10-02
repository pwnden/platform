<script setup lang="ts">
import type { UIBadgeProps } from './props';
withDefaults(defineProps<UIBadgeProps>(), { tone: 'accent' });
</script>

<template>
  <span class="ui-badge" :class="`ui-badge--${tone}`"><slot /></span>
</template>

<style scoped>
.ui-badge { display: inline-flex; flex: none; align-items: center; padding: 0.25rem; border: 1px solid var(--ui-border-active); border-radius: var(--ui-radius-control); background: var(--ui-accent-surface); color: var(--ui-accent); font-size: 0.8rem; line-height: 1.4; white-space: nowrap; }
.ui-badge--quartz { --ui-badge-color: var(--ui-difficulty-intro); }
.ui-badge--emerald { --ui-badge-color: var(--ui-difficulty-easy); }
.ui-badge--sapphire { --ui-badge-color: var(--ui-difficulty-medium); }
.ui-badge--amethyst { --ui-badge-color: var(--ui-difficulty-hard); }
.ui-badge--ruby { --ui-badge-color: var(--ui-difficulty-expert); }
.ui-badge:not(.ui-badge--accent) { color: var(--ui-badge-color); background: color-mix(in srgb, var(--ui-badge-color) 10%, transparent); border-color: color-mix(in srgb, var(--ui-badge-color) 30%, transparent); }
</style>
