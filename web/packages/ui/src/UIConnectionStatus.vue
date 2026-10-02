<script setup lang="ts">
import type { UIConnectionStatusProps } from './props';
import UIIcon from './UIIcon.vue';
defineProps<UIConnectionStatusProps>();
const labels = { connected: '연결됨', connecting: '준비·연결 중', disconnected: '연결 해제됨', error: '연결 오류' };
</script>

<template>
  <span class="ui-connection-status" :class="`ui-connection-status--${state}`" role="status" :title="labels[state]">
    <UIIcon :name="state === 'connected' ? 'plug' : state === 'connecting' ? 'loader' : state === 'error' ? 'error' : 'unplug'" :class="{ 'connection-progress': state === 'connecting' }" />
    <span class="connection-label">{{ labels[state] }}</span>
  </span>
</template>

<style scoped>
.ui-connection-status { display: inline-flex; align-items: center; justify-content: center; flex: none; width: 2rem; min-height: 2.5rem; color: var(--ui-muted); }
.ui-connection-status svg { width: 1.5rem; height: 1.5rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.ui-connection-status--connected { color: var(--ui-accent); }
.ui-connection-status--connecting { color: var(--ui-warning); }
.ui-connection-status--error { color: var(--ui-danger); }
.connection-progress { animation: connection-progress 1s linear infinite; }
.connection-label { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@keyframes connection-progress { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .connection-progress { animation: none; } }
</style>
